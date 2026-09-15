package resources

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	pdh        = windows.NewLazySystemDLL("pdh.dll")
	pdhOpen    = pdh.NewProc("PdhOpenQueryW")
	pdhAdd     = pdh.NewProc("PdhAddEnglishCounterW")
	pdhCollect = pdh.NewProc("PdhCollectQueryData")
	pdhArray   = pdh.NewProc("PdhGetFormattedCounterArrayW")
	pdhClose   = pdh.NewProc("PdhCloseQuery")
)

type windowsGPUReader struct {
	query, engine, dedicated, shared uintptr
	last                             time.Time
}

func newGPUReader() gpuReader { return &windowsGPUReader{} }

func (r *windowsGPUReader) close() {
	if r.query != 0 {
		pdhClose.Call(r.query)
	}
	r.query, r.engine, r.dedicated, r.shared = 0, 0, 0, 0
	r.last = time.Time{}
}

func (r *windowsGPUReader) open() bool {
	if status, _, _ := pdhOpen.Call(0, 0, uintptr(unsafe.Pointer(&r.query))); status != 0 {
		return false
	}
	for _, counter := range []struct {
		path   string
		handle *uintptr
	}{
		{`\GPU Engine(*)\Utilization Percentage`, &r.engine},
		{`\GPU Adapter Memory(*)\Dedicated Usage`, &r.dedicated},
		{`\GPU Adapter Memory(*)\Shared Usage`, &r.shared},
	} {
		path, _ := windows.UTF16PtrFromString(counter.path)
		if status, _, _ := pdhAdd.Call(r.query, uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(counter.handle))); status != 0 {
			*counter.handle = 0
		}
	}
	if r.engine == 0 && r.dedicated == 0 && r.shared == 0 {
		r.close()
		return false
	}
	return true
}

// PDH counts all WDDM engines, including CUDA/compute. Sum each engine across
// processes, then use the busiest engine per adapter (as Task Manager does).
// Never average idle engines into the busy percentage or sum different engines.
func (r *windowsGPUReader) read(now time.Time) []GPU {
	if !r.last.IsZero() && (now.Sub(r.last) > 5*time.Second || now.Before(r.last)) {
		r.close()
	}
	warming := r.query == 0
	if warming && !r.open() {
		return nil
	}
	r.last = now
	if status, _, _ := pdhCollect.Call(r.query); status != 0 {
		r.close()
		return nil
	}
	var engine []counterValue
	if !warming {
		engine = formattedCounters(r.engine)
	}
	rows := aggregateGPUCounters(engine, formattedCounters(r.dedicated), formattedCounters(r.shared))
	metadata := adapterMetadata()
	keys := make([]string, 0, len(rows))
	for key := range rows {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	result := make([]GPU, 0, min(len(keys), 8))
	for _, key := range keys[:min(len(keys), 8)] {
		row := rows[key]
		row.Name = fmt.Sprintf("Graphics adapter %d", len(result)+1)
		if info, ok := metadata[key]; ok {
			if info.software {
				continue
			}
			row.Name, row.MemoryTotalBytes = info.name, info.capacity
		}
		result = append(result, row)
	}
	return result
}

type counterValue struct {
	name  string
	value float64
}

// PDH_FMT_COUNTERVALUE_ITEM_W on supported 64-bit Windows. Every returned
// array is bounded, validated and discarded after aggregate values are built;
// instance strings (which include process IDs) never cross the Wails boundary.
type formattedCounter struct {
	name    *uint16
	status  uint32
	padding uint32
	value   float64
}

func formattedCounters(handle uintptr) []counterValue {
	if handle == 0 {
		return nil
	}
	const format = 0x200 | 0x8000 // PDH_FMT_DOUBLE | PDH_FMT_NOCAP100
	var size, count uint32
	status, _, _ := pdhArray.Call(handle, format, uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), 0)
	if uint32(status) != 0x800007d2 || size == 0 || size > 2<<20 {
		return nil
	} // PDH_MORE_DATA
	buffer := make([]byte, size)
	status, _, _ = pdhArray.Call(handle, format, uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&buffer[0])))
	if status != 0 || count > 16384 || uint64(count)*uint64(unsafe.Sizeof(formattedCounter{})) > uint64(len(buffer)) {
		return nil
	}
	items := unsafe.Slice((*formattedCounter)(unsafe.Pointer(&buffer[0])), count)
	result := make([]counterValue, 0, count)
	for _, item := range items {
		if item.status > 1 || math.IsNaN(item.value) || math.IsInf(item.value, 0) || item.value < 0 {
			continue
		}
		// PDH names must point into this result buffer and terminate inside it.
		base, ptr := uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(item.name))
		if ptr < base || ptr >= base+uintptr(len(buffer)) || (ptr-base)%2 != 0 {
			continue
		}
		length := min((base+uintptr(len(buffer))-ptr)/2, 256)
		name := unsafe.Slice(item.name, length)
		end := slices.Index(name, uint16(0))
		if end < 0 {
			continue
		}
		result = append(result, counterValue{windows.UTF16ToString(name[:end]), item.value})
	}
	return result
}

func aggregateGPUCounters(engine, dedicated, shared []counterValue) map[string]GPU {
	rows := make(map[string]GPU)
	engines := make(map[string]float64)
	for _, item := range engine {
		_, instance, ok := strings.Cut(strings.ToLower(item.name), "luid_")
		if !ok {
			continue
		}
		adapter, _, ok := strings.Cut(instance, "_eng_")
		if !ok {
			continue
		}
		key := "luid_" + adapter
		engines[instance] += item.value
		row := rows[key]
		row.UtilizationAvailable = true
		row.UtilizationPercent = max(row.UtilizationPercent, min(100, engines[instance]))
		rows[key] = row
	}
	for _, item := range dedicated {
		item.name = strings.ToLower(item.name)
		if !strings.HasPrefix(item.name, "luid_") || item.value >= 1<<53 {
			continue
		}
		row := rows[item.name]
		row.MemoryAvailable, row.MemoryUsedBytes = true, uint64(item.value)
		rows[item.name] = row
	}
	for _, item := range shared {
		item.name = strings.ToLower(item.name)
		if !strings.HasPrefix(item.name, "luid_") || item.value >= 1<<53 {
			continue
		}
		row := rows[item.name]
		row.SharedMemoryAvailable, row.SharedMemoryUsedBytes = true, uint64(item.value)
		rows[item.name] = row
	}
	return rows
}

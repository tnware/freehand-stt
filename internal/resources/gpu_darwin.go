//go:build darwin && cgo

package resources

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/IOKitLib.h>
#include <stdint.h>
#include <stdio.h>

typedef struct {
    char name[128];
    double utilization;
    uint64_t memory_used;
    int utilization_ok, memory_ok, unified;
} freehand_gpu;

typedef struct {
    freehand_gpu values[8];
    int count;
} freehand_gpus;

static int freehand_gpu_number(CFDictionaryRef dictionary, CFStringRef key, CFNumberType type, void *output) {
    CFTypeRef value = CFDictionaryGetValue(dictionary, key);
    return value && CFGetTypeID(value) == CFNumberGetTypeID() && CFNumberGetValue((CFNumberRef)value, type, output);
}

static freehand_gpus freehand_read_gpus(void) {
    freehand_gpus result = {0};
    io_iterator_t iterator = IO_OBJECT_NULL;
    CFMutableDictionaryRef matching = IOServiceMatching("IOAccelerator");
    if (!matching || IOServiceGetMatchingServices(kIOMainPortDefault, matching, &iterator) != KERN_SUCCESS) return result;
    io_registry_entry_t entry;
    while (result.count < 8 && (entry = IOIteratorNext(iterator)) != IO_OBJECT_NULL) {
        freehand_gpu *gpu = &result.values[result.count++];
        gpu->unified = IOObjectConformsTo(entry, "AGXAccelerator");
        snprintf(gpu->name, sizeof(gpu->name), gpu->unified ? "Apple GPU" : "Graphics adapter %d", result.count);
        CFTypeRef model = IORegistryEntryCreateCFProperty(entry, CFSTR("model"), kCFAllocatorDefault, 0);
        if (model) {
            if (CFGetTypeID(model) == CFStringGetTypeID()) {
                char name[128] = {0};
                if (CFStringGetCString((CFStringRef)model, name, sizeof(name), kCFStringEncodingUTF8) && name[0]) {
                    snprintf(gpu->name, sizeof(gpu->name), "%s", name);
                }
            }
            CFRelease(model);
        }
        // Driver-published optional fields, not a guaranteed macOS API schema.
        // Read only this bounded set; never substitute zero for absent fields.
        CFTypeRef property = IORegistryEntryCreateCFProperty(entry, CFSTR("PerformanceStatistics"), kCFAllocatorDefault, 0);
        if (property) {
            if (CFGetTypeID(property) == CFDictionaryGetTypeID()) {
                CFDictionaryRef stats = (CFDictionaryRef)property;
                double busy = -1;
                if (freehand_gpu_number(stats, CFSTR("Device Utilization %"), kCFNumberDoubleType, &busy) && busy >= 0 && busy <= 100) {
                    gpu->utilization_ok = 1;
                    gpu->utilization = busy;
                }
                int64_t used = -1;
                if (gpu->unified && freehand_gpu_number(stats, CFSTR("In use system memory"), kCFNumberSInt64Type, &used) && used >= 0) {
                    gpu->memory_ok = 1;
                    gpu->memory_used = (uint64_t)used;
                }
            }
            CFRelease(property);
        }
        IOObjectRelease(entry);
    }
    IOObjectRelease(iterator);
    return result;
}
*/
import "C"

import "time"

type darwinGPUReader struct{}

func newGPUReader() gpuReader  { return darwinGPUReader{} }
func (darwinGPUReader) close() {}

func (darwinGPUReader) read(time.Time) []GPU {
	value := C.freehand_read_gpus()
	result := make([]GPU, 0, int(value.count))
	for i := range int(value.count) {
		gpu := &value.values[i]
		result = append(result, GPU{
			Name:                 C.GoString(&gpu.name[0]),
			UtilizationAvailable: gpu.utilization_ok != 0, UtilizationPercent: float64(gpu.utilization),
			MemoryAvailable: gpu.memory_ok != 0, MemoryUsedBytes: uint64(gpu.memory_used),
			UnifiedMemory: gpu.unified != 0,
		})
	}
	return result
}

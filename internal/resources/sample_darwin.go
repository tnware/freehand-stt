//go:build darwin && cgo

package resources

/*
#include <mach/mach.h>
#include <stdint.h>
#include <sys/sysctl.h>

typedef struct {
    uint64_t idle, total, memory_total, memory_available;
    int cpu_ok;
} freehand_resource_counters;

static freehand_resource_counters freehand_read_resources(void) {
    freehand_resource_counters result = {0};
    mach_port_t host = mach_host_self();
    host_cpu_load_info_data_t cpu = {0};
    mach_msg_type_number_t count = HOST_CPU_LOAD_INFO_COUNT;
    if (host_statistics(host, HOST_CPU_LOAD_INFO, (host_info_t)&cpu, &count) == KERN_SUCCESS) {
        result.cpu_ok = 1;
        result.idle = cpu.cpu_ticks[CPU_STATE_IDLE];
        for (int i = 0; i < CPU_STATE_MAX; i++) result.total += cpu.cpu_ticks[i];
    }
    uint64_t total = 0;
    size_t size = sizeof(total);
    vm_statistics64_data_t vm = {0};
    vm_size_t page_size = 0;
    count = HOST_VM_INFO64_COUNT;
    if (sysctlbyname("hw.memsize", &total, &size, NULL, 0) == 0 &&
        host_page_size(host, &page_size) == KERN_SUCCESS &&
        host_statistics64(host, HOST_VM_INFO64, (host_info64_t)&vm, &count) == KERN_SUCCESS) {
        result.memory_total = total;
        // An estimate including inactive (reclaimable) pages. Speculative
        // pages are already in free_count; purgeable can overlap other lists.
        result.memory_available = ((uint64_t)vm.free_count + vm.inactive_count) * page_size;
    }
    mach_port_deallocate(mach_task_self(), host);
    return result;
}
*/
import "C"

func readCounters() counters {
	value := C.freehand_read_resources()
	return counters{
		idle: uint64(value.idle), total: uint64(value.total), cpuOK: value.cpu_ok != 0,
		memoryTotal: uint64(value.memory_total), memoryAvailable: uint64(value.memory_available),
	}
}

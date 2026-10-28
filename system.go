package main

import (
	"fmt"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

var (
	modKernel32                 = windows.NewLazySystemDLL("kernel32.dll")
	procGlobalMemoryStatusEx    = modKernel32.NewProc("GlobalMemoryStatusEx")
	procGetTickCount64          = modKernel32.NewProc("GetTickCount64")
	procK32GetProcessMemoryInfo = modKernel32.NewProc("K32GetProcessMemoryInfo")
)

// MEMORYSTATUSEX
type memoryStatus struct {
	length, load                 uint32
	totalPhys, availPhys         uint64
	totalPageFile, availPageFile uint64
	totalVirtual, availVirtual   uint64
	availExtendedVirtual         uint64
}

func memStatus() memoryStatus {
	m := memoryStatus{length: uint32(unsafe.Sizeof(memoryStatus{}))}
	procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&m)))
	return m
}

func uptime() time.Duration {
	r, _, _ := procGetTickCount64.Call()
	return time.Duration(r) * time.Millisecond
}

func diskSpace(drive string) (total, used uint64) {
	p, err := windows.UTF16PtrFromString(drive + `\`)
	if err != nil {
		return 0, 0
	}
	var free, tot, totFree uint64
	if windows.GetDiskFreeSpaceEx(p, &free, &tot, &totFree) != nil {
		return 0, 0
	}
	return tot, tot - totFree
}

func regString(path, name string) string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		return ""
	}
	defer k.Close()
	v, _, _ := k.GetStringValue(name)
	return strings.TrimSpace(v)
}

// machineInfo returns the laptop model, a tidied CPU name and the GPU name.
func machineInfo() (model, cpu, gpu string) {
	model = regString(`HARDWARE\DESCRIPTION\System\BIOS`, "SystemProductName")
	cpu = regString(`HARDWARE\DESCRIPTION\System\CentralProcessor\0`, "ProcessorNameString")
	cpu = strings.NewReplacer("(R)", "", "(TM)", "", " CPU", "").Replace(cpu)
	if i := strings.Index(cpu, "@"); i > 0 {
		cpu = cpu[:i]
	}
	cpu = strings.Join(strings.Fields(cpu), " ")

	for i := 0; i < 8; i++ {
		name := regString(fmt.Sprintf(`SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}\%04d`, i), "DriverDesc")
		if name != "" && !strings.Contains(name, "Basic") {
			gpu = strings.NewReplacer("(R)", "", "(TM)", "").Replace(name)
			break
		}
	}
	return model, cpu, gpu
}

package main

import (
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type proc struct {
	pid     uint32
	name    string
	threads uint32
	cpu     float64 // percent of the whole machine, like Task Manager
	mem     uint64  // working set
	cpuTime time.Duration
}

// PROCESS_MEMORY_COUNTERS
type processMemoryCounters struct {
	cb                         uint32
	pageFaultCount             uint32
	peakWorkingSetSize         uintptr
	workingSetSize             uintptr
	quotaPeakPagedPoolUsage    uintptr
	quotaPagedPoolUsage        uintptr
	quotaPeakNonPagedPoolUsage uintptr
	quotaNonPagedPoolUsage     uintptr
	pagefileUsage              uintptr
	peakPagefileUsage          uintptr
}

type procTracker struct {
	last     map[uint32]time.Duration
	lastWall time.Time
}

func filetimeDur(f windows.Filetime) time.Duration {
	return time.Duration(uint64(f.HighDateTime)<<32|uint64(f.LowDateTime)) * 100
}

func (t *procTracker) sample(ncpu int) []proc {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil
	}
	defer windows.CloseHandle(snap)

	now := time.Now()
	wall := now.Sub(t.lastWall)
	seen := make(map[uint32]time.Duration, len(t.last))
	list := make([]proc, 0, len(t.last))

	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	for err = windows.Process32First(snap, &e); err == nil; err = windows.Process32Next(snap, &e) {
		if e.ProcessID == 0 {
			continue // System Idle Process
		}
		p := proc{pid: e.ProcessID, name: windows.UTF16ToString(e.ExeFile[:]), threads: e.Threads}
		if h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, e.ProcessID); err == nil {
			var c, x, k, u windows.Filetime
			if windows.GetProcessTimes(h, &c, &x, &k, &u) == nil {
				p.cpuTime = filetimeDur(k) + filetimeDur(u)
			}
			mc := processMemoryCounters{cb: uint32(unsafe.Sizeof(processMemoryCounters{}))}
			if r, _, _ := procK32GetProcessMemoryInfo.Call(uintptr(h), uintptr(unsafe.Pointer(&mc)), uintptr(mc.cb)); r != 0 {
				p.mem = uint64(mc.workingSetSize)
			}
			windows.CloseHandle(h)
		}
		if prev, ok := t.last[p.pid]; ok && wall > 0 && p.cpuTime >= prev {
			p.cpu = min(float64(p.cpuTime-prev)/float64(wall)/float64(ncpu)*100, 100)
		}
		seen[p.pid] = p.cpuTime
		list = append(list, p)
	}
	t.last, t.lastWall = seen, now
	return list
}

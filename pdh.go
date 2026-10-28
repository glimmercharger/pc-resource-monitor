package main

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Thin wrapper over the Performance Data Helper API, which is where Task
// Manager gets most of its numbers (temps, GPU engines, disk activity...).

var (
	modPdh                          = windows.NewLazySystemDLL("pdh.dll")
	procPdhOpenQuery                = modPdh.NewProc("PdhOpenQueryW")
	procPdhAddEnglishCounter        = modPdh.NewProc("PdhAddEnglishCounterW")
	procPdhCollectQueryData         = modPdh.NewProc("PdhCollectQueryData")
	procPdhGetFormattedCounterArray = modPdh.NewProc("PdhGetFormattedCounterArrayW")
)

const (
	pdhFmtDouble   = 0x00000200
	pdhFmtNoCap100 = 0x00008000
	pdhMoreData    = 0x800007D2
)

type pdhQuery uintptr
type pdhCounter uintptr

// PDH_FMT_COUNTERVALUE_ITEM_DOUBLE
type pdhItem struct {
	name   *uint16
	status uint32
	_      uint32
	value  float64
}

func openQuery() (pdhQuery, error) {
	var q pdhQuery
	if r, _, _ := procPdhOpenQuery.Call(0, 0, uintptr(unsafe.Pointer(&q))); r != 0 {
		return 0, fmt.Errorf("PdhOpenQuery failed: %#x", r)
	}
	return q, nil
}

// add registers a counter by its English path so it works on any Windows
// language. A counter that doesn't exist on this machine comes back as 0.
func (q pdhQuery) add(path string) pdhCounter {
	if q == 0 {
		return 0
	}
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0
	}
	var c pdhCounter
	if r, _, _ := procPdhAddEnglishCounter.Call(uintptr(q), uintptr(unsafe.Pointer(p)), 0, uintptr(unsafe.Pointer(&c))); r != 0 {
		return 0
	}
	return c
}

func (q pdhQuery) collect() {
	if q != 0 {
		procPdhCollectQueryData.Call(uintptr(q))
	}
}

// values returns every instance of the counter keyed by instance name.
func (c pdhCounter) values() map[string]float64 {
	if c == 0 {
		return nil
	}
	var size, count uint32
	r, _, _ := procPdhGetFormattedCounterArray.Call(uintptr(c), pdhFmtDouble|pdhFmtNoCap100,
		uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), 0)
	if uint32(r) != pdhMoreData || size == 0 {
		return nil
	}
	buf := make([]byte, size)
	r, _, _ = procPdhGetFormattedCounterArray.Call(uintptr(c), pdhFmtDouble|pdhFmtNoCap100,
		uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&buf[0])))
	if r != 0 || count == 0 {
		return nil
	}
	items := unsafe.Slice((*pdhItem)(unsafe.Pointer(&buf[0])), count)
	out := make(map[string]float64, count)
	for _, it := range items {
		// 0 = valid data, 1 = new data; anything else is a bad sample
		if it.status > 1 {
			continue
		}
		out[windows.UTF16PtrToString(it.name)] = it.value
	}
	return out
}

// value is for counters with a single instance like _Total.
func (c pdhCounter) value() (float64, bool) {
	for _, v := range c.values() {
		return v, true
	}
	return 0, false
}

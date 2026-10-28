package main

import (
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	modWlan                = windows.NewLazySystemDLL("wlanapi.dll")
	procWlanOpenHandle     = modWlan.NewProc("WlanOpenHandle")
	procWlanEnumInterfaces = modWlan.NewProc("WlanEnumInterfaces")
	procWlanQueryInterface = modWlan.NewProc("WlanQueryInterface")
	procWlanFreeMemory     = modWlan.NewProc("WlanFreeMemory")
)

const (
	wlanStateConnected       = 1
	wlanOpcodeCurrentConnect = 7
)

type wifiInfo struct {
	ok             bool
	adapter        string
	ssid           string
	signal         int // 0-100
	rxMbps, txMbps float64
	phy            string
}

// WLAN_INTERFACE_INFO
type wlanInterfaceInfo struct {
	guid  windows.GUID
	desc  [256]uint16
	state uint32
}

// WLAN_CONNECTION_ATTRIBUTES, minus the security part we don't read
type wlanConnAttrs struct {
	state    uint32
	mode     uint32
	profile  [256]uint16
	ssidLen  uint32
	ssid     [32]byte
	bssType  uint32
	bssid    [6]byte
	phyType  uint32
	phyIndex uint32
	signal   uint32
	rxRate   uint32 // kbps
	txRate   uint32
}

var phyNames = map[uint32]string{
	4: "802.11a", 5: "802.11b", 6: "802.11g", 7: "802.11n",
	8: "802.11ac", 10: "802.11ax", 11: "802.11be",
}

type wlanClient struct{ h windows.Handle }

func openWlan() *wlanClient {
	if modWlan.Load() != nil {
		return nil
	}
	var ver uint32
	var h windows.Handle
	if r, _, _ := procWlanOpenHandle.Call(2, 0, uintptr(unsafe.Pointer(&ver)), uintptr(unsafe.Pointer(&h))); r != 0 {
		return nil
	}
	return &wlanClient{h}
}

func (w *wlanClient) query() wifiInfo {
	var info wifiInfo
	if w == nil {
		return info
	}
	var list unsafe.Pointer
	if r, _, _ := procWlanEnumInterfaces.Call(uintptr(w.h), 0, uintptr(unsafe.Pointer(&list))); r != 0 {
		return info
	}
	defer procWlanFreeMemory.Call(uintptr(list))

	n := *(*uint32)(list)
	ifaces := unsafe.Slice((*wlanInterfaceInfo)(unsafe.Add(list, 8)), n)
	for i := range ifaces {
		ifc := &ifaces[i]
		info.adapter = windows.UTF16ToString(ifc.desc[:])
		if ifc.state != wlanStateConnected {
			continue
		}
		var size uint32
		var data unsafe.Pointer
		r, _, _ := procWlanQueryInterface.Call(uintptr(w.h), uintptr(unsafe.Pointer(&ifc.guid)), wlanOpcodeCurrentConnect, 0,
			uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&data)), 0)
		if r != 0 {
			continue
		}
		a := (*wlanConnAttrs)(data)
		info.ok = true
		info.ssid = string(a.ssid[:min(a.ssidLen, 32)])
		info.signal = int(a.signal)
		info.rxMbps = float64(a.rxRate) / 1000
		info.txMbps = float64(a.txRate) / 1000
		info.phy = phyNames[a.phyType]
		procWlanFreeMemory.Call(uintptr(data))
		break
	}
	return info
}

// pdhInstanceName turns an adapter description into the name PDH uses for it
// under \Network Interface(*).
func pdhInstanceName(s string) string {
	return strings.NewReplacer("(", "[", ")", "]", "#", "_", "/", "_", `\`, "_").Replace(s)
}

func netRates(in, out map[string]float64, adapter string) (down, up float64) {
	if adapter != "" {
		key := pdhInstanceName(adapter)
		if v, ok := in[key]; ok {
			return v, out[key]
		}
	}
	for name, v := range in {
		l := strings.ToLower(name)
		if strings.Contains(l, "loopback") || strings.Contains(l, "isatap") || strings.Contains(l, "teredo") {
			continue
		}
		down += v
		up += out[name]
	}
	return down, up
}

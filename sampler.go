package main

import (
	"os"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

const histLen = 400

type series []float64

func (s *series) push(v float64) {
	if len(*s) < histLen {
		*s = append(*s, v)
		return
	}
	copy(*s, (*s)[1:])
	(*s)[histLen-1] = v
}

// tail returns the last n points (or fewer if we don't have that many yet).
func (s series) tail(n int) series {
	if len(s) > n {
		return s[len(s)-n:]
	}
	return s
}

func (s series) max() float64 {
	m := 0.0
	for _, v := range s {
		m = max(m, v)
	}
	return m
}

func clamp(v, lo, hi float64) float64 { return max(lo, min(hi, v)) }

type stats struct {
	cpu, mhz float64
	cores    []float64
	temp     float64
	hasTemp  bool

	memUsed, memTotal, swapUsed, swapTotal uint64

	drive                           string
	diskActive, diskRead, diskWrite float64
	diskUsed, diskTotal             uint64

	netDown, netUp float64
	wifi           wifiInfo

	gpu                     map[string]float64
	gpuTypes                []string
	gpuShared, gpuDedicated uint64

	procs   []proc
	threads int
	uptime  time.Duration

	cpuH, tempH, memH, downH, upH, readH, writeH, diskH series
	gpuH                                                map[string]*series
}

type sampler struct {
	q pdhQuery

	cpu, cores, perf, freq, temp             pdhCounter
	diskIdle, diskRead, diskWrite, pageUsage pdhCounter
	netIn, netOut                            pdhCounter
	gpuEngine, gpuShared, gpuDedicated       pdhCounter

	wlan  *wlanClient
	pt    procTracker
	ncpu  int
	ticks int
	st    stats
}

func newSampler() *sampler {
	s := &sampler{ncpu: runtime.NumCPU(), wlan: openWlan()}
	s.st.gpu = map[string]float64{}
	s.st.gpuH = map[string]*series{}
	if s.st.drive = os.Getenv("SystemDrive"); s.st.drive == "" {
		s.st.drive = "C:"
	}

	q, err := openQuery()
	if err != nil {
		return s
	}
	s.q = q
	// "Processor Utility" is what Task Manager shows; older builds only have "Processor Time".
	if s.cpu = q.add(`\Processor Information(_Total)\% Processor Utility`); s.cpu == 0 {
		s.cpu = q.add(`\Processor(_Total)\% Processor Time`)
	}
	if s.cores = q.add(`\Processor Information(*)\% Processor Utility`); s.cores == 0 {
		s.cores = q.add(`\Processor(*)\% Processor Time`)
	}
	s.perf = q.add(`\Processor Information(_Total)\% Processor Performance`)
	s.freq = q.add(`\Processor Information(_Total)\Processor Frequency`)
	s.temp = q.add(`\Thermal Zone Information(*)\High Precision Temperature`)
	s.diskIdle = q.add(`\PhysicalDisk(_Total)\% Idle Time`)
	s.diskRead = q.add(`\PhysicalDisk(_Total)\Disk Read Bytes/sec`)
	s.diskWrite = q.add(`\PhysicalDisk(_Total)\Disk Write Bytes/sec`)
	s.pageUsage = q.add(`\Paging File(_Total)\% Usage`)
	s.netIn = q.add(`\Network Interface(*)\Bytes Received/sec`)
	s.netOut = q.add(`\Network Interface(*)\Bytes Sent/sec`)
	s.gpuEngine = q.add(`\GPU Engine(*)\Utilization Percentage`)
	s.gpuShared = q.add(`\GPU Adapter Memory(*)\Shared Usage`)
	s.gpuDedicated = q.add(`\GPU Adapter Memory(*)\Dedicated Usage`)
	return s
}

func (s *sampler) sample() {
	s.q.collect()
	st := &s.st

	if v, ok := s.cpu.value(); ok {
		st.cpu = clamp(v, 0, 100)
	}
	st.cores = coreValues(s.cores.values(), s.ncpu)
	perf, ok1 := s.perf.value()
	base, ok2 := s.freq.value()
	if ok1 && ok2 {
		st.mhz = base * perf / 100
	}

	st.hasTemp = false
	for _, v := range s.temp.values() {
		c := v/10 - 273.15 // tenths of a kelvin
		if !st.hasTemp || c > st.temp {
			st.temp = c
		}
		st.hasTemp = true
	}

	m := memStatus()
	st.memTotal, st.memUsed = m.totalPhys, m.totalPhys-m.availPhys
	if m.totalPageFile > m.totalPhys {
		st.swapTotal = m.totalPageFile - m.totalPhys
	}
	if v, ok := s.pageUsage.value(); ok {
		st.swapUsed = uint64(float64(st.swapTotal) * clamp(v, 0, 100) / 100)
	}

	if v, ok := s.diskIdle.value(); ok {
		st.diskActive = clamp(100-v, 0, 100)
	}
	st.diskRead, _ = s.diskRead.value()
	st.diskWrite, _ = s.diskWrite.value()
	st.diskTotal, st.diskUsed = diskSpace(st.drive)

	// the wlan api is slowish and signal doesn't change much, so skip most ticks
	if s.ticks%3 == 0 {
		st.wifi = s.wlan.query()
	}
	adapter := ""
	if st.wifi.ok {
		adapter = st.wifi.adapter
	}
	st.netDown, st.netUp = netRates(s.netIn.values(), s.netOut.values(), adapter)

	s.sampleGPU()

	st.procs = s.pt.sample(s.ncpu)
	st.threads = 0
	for _, p := range st.procs {
		st.threads += int(p.threads)
	}
	st.uptime = uptime()

	st.cpuH.push(st.cpu)
	st.tempH.push(st.temp)
	st.memH.push(float64(st.memUsed) / float64(max(st.memTotal, 1)) * 100)
	st.downH.push(st.netDown)
	st.upH.push(st.netUp)
	st.readH.push(st.diskRead)
	st.writeH.push(st.diskWrite)
	st.diskH.push(st.diskActive)
	s.ticks++
}

// coreValues sorts per-core instances ("0,0", "0,1"...) into a plain list.
func coreValues(m map[string]float64, n int) []float64 {
	type core struct {
		key int
		v   float64
	}
	var cs []core
	for name, v := range m {
		if strings.Contains(name, "_Total") {
			continue
		}
		g, i, ok := strings.Cut(name, ",")
		if !ok {
			g, i = "0", name
		}
		gn, _ := strconv.Atoi(g)
		in, err := strconv.Atoi(i)
		if err != nil {
			continue
		}
		cs = append(cs, core{gn*1000 + in, clamp(v, 0, 100)})
	}
	if len(cs) == 0 {
		return make([]float64, n)
	}
	sort.Slice(cs, func(a, b int) bool { return cs[a].key < cs[b].key })
	out := make([]float64, len(cs))
	for i, c := range cs {
		out[i] = c.v
	}
	return out
}

// Task Manager's default four, then whatever else the GPU has.
var gpuOrder = []string{"3D", "Copy", "VideoDecode", "VideoProcessing", "VideoEncode", "Compute"}

var gpuLabels = map[string]string{
	"VideoDecode": "Video Decode", "VideoProcessing": "Video Processing",
	"VideoEncode": "Video Encode", "LegacyOverlay": "Legacy Overlay",
}

func gpuLabel(t string) string {
	if l, ok := gpuLabels[t]; ok {
		return l
	}
	return t
}

func (s *sampler) sampleGPU() {
	st := &s.st
	// Instances look like pid_123_luid_0x0_0xD1B5_phys_0_eng_0_engtype_3D.
	// Add up every process per physical engine, then take the busiest engine
	// of each type - same as Task Manager.
	engines := map[string]float64{}
	types := map[string]string{}
	for name, v := range s.gpuEngine.values() {
		i, t := strings.Index(name, "luid_"), strings.Index(name, "engtype_")
		if i < 0 || t < 0 || t+len("engtype_") == len(name) {
			continue
		}
		key := name[i:]
		engines[key] += v
		types[key] = name[t+len("engtype_"):]
	}
	usage := map[string]float64{}
	for key, v := range engines {
		usage[types[key]] = max(usage[types[key]], v)
	}
	for t := range usage {
		if _, ok := st.gpuH[t]; !ok {
			st.gpuH[t] = &series{}
		}
	}
	for t, h := range st.gpuH {
		u := clamp(usage[t], 0, 100)
		st.gpu[t] = u
		h.push(u)
	}

	st.gpuTypes = st.gpuTypes[:0]
	for _, t := range gpuOrder {
		if _, ok := st.gpuH[t]; ok && len(st.gpuTypes) < 4 {
			st.gpuTypes = append(st.gpuTypes, t)
		}
	}
	var rest []string
	for t := range st.gpuH {
		if !slices.Contains(gpuOrder, t) {
			rest = append(rest, t)
		}
	}
	sort.Strings(rest)
	for _, t := range rest {
		if len(st.gpuTypes) < 4 {
			st.gpuTypes = append(st.gpuTypes, t)
		}
	}

	st.gpuShared, st.gpuDedicated = 0, 0
	for _, v := range s.gpuShared.values() {
		st.gpuShared += uint64(v)
	}
	for _, v := range s.gpuDedicated.values() {
		st.gpuDedicated += uint64(v)
	}
}

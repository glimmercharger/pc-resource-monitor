package main

import (
	"fmt"
	"time"
)

func main() {
	s := newSampler()
	s.sample()
	time.Sleep(time.Second)
	s.sample()
	st := &s.st
	fmt.Printf("cpu %.1f%% cores %v %.0f MHz\n", st.cpu, st.cores, st.mhz)
	fmt.Printf("temp %.1fC (%v)\n", st.temp, st.hasTemp)
	fmt.Printf("mem %d/%d swap %d/%d\n", st.memUsed>>20, st.memTotal>>20, st.swapUsed>>20, st.swapTotal>>20)
	fmt.Printf("disk %s active %.1f%% r %.0f w %.0f used %d/%d GB\n", st.drive, st.diskActive, st.diskRead, st.diskWrite, st.diskUsed>>30, st.diskTotal>>30)
	fmt.Printf("wifi %+v down %.0f up %.0f\n", st.wifi, st.netDown, st.netUp)
	fmt.Printf("gpu %v types %v shared %dMB\n", st.gpu, st.gpuTypes, st.gpuShared>>20)
	fmt.Printf("procs %d threads %d uptime %s\n", len(st.procs), st.threads, st.uptime.Round(time.Second))
	fmt.Println(machineInfo())
}

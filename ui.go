package main

import (
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	sortCPU = iota
	sortMem
	sortPID
	sortName
)

var sortCols = []string{"CPU%", "MEM%", "PID", "NAME"}

type app struct {
	s          *sampler
	theme      int
	showGraphs bool
	showProcs  bool
	help       bool
	sortBy     int
	sel        int
	scroll     int

	host, model, cpuName, gpuName string
}

func newApp(themeName string) *app {
	a := &app{s: newSampler(), showGraphs: true, showProcs: true}
	for i, t := range themes {
		if strings.EqualFold(t.name, themeName) {
			a.theme = i
		}
	}
	a.host, _ = os.Hostname()
	a.model, a.cpuName, a.gpuName = machineInfo()
	return a
}

// handleKey returns false when it's time to quit.
func (a *app) handleKey(k string) bool {
	if a.help {
		a.help = false
		return k != "q" && k != "f10" && k != "ctrl-c"
	}
	switch k {
	case "q", "Q", "f10", "ctrl-c":
		return false
	case "f1", "h", "?":
		a.help = true
	case "f2", "t":
		a.theme = (a.theme + 1) % len(themes)
	case "f3", "g":
		a.showGraphs = !a.showGraphs
	case "f4", "p":
		a.showProcs = !a.showProcs
	case "f6", "s":
		a.sortBy = (a.sortBy + 1) % len(sortCols)
	case "c":
		a.sortBy = sortCPU
	case "m":
		a.sortBy = sortMem
	case "i":
		a.sortBy = sortPID
	case "n":
		a.sortBy = sortName
	case "up", "k":
		a.sel--
	case "down", "j":
		a.sel++
	case "pgup":
		a.sel -= 10
	case "pgdn":
		a.sel += 10
	case "home":
		a.sel = 0
	case "end":
		a.sel = 1 << 30
	}
	return true
}

func (a *app) draw(w, h int) *canvas {
	th := &themes[a.theme]
	st := th.styles()
	cv := newCanvas(w, h, st.text)
	if w < 60 || h < 16 {
		msg := fmt.Sprintf("make the window bigger (need 60x16, got %dx%d)", w, h)
		cv.text(max(0, (w-width(msg))/2), h/2, clip(msg, w), st.accent)
		return cv
	}

	a.drawTitle(cv, &st)
	y := a.drawMeters(cv, &st, 1) + 1
	avail := h - 1 - y

	graphH, procH := 0, 0
	switch {
	case a.showGraphs && a.showProcs:
		graphH = min(avail*3/5, 45)
		procH = avail - graphH
		if procH < 4 {
			graphH, procH = avail, 0
		}
	case a.showGraphs:
		graphH = avail
	case a.showProcs:
		procH = avail
	}
	if graphH < 12 {
		// not enough room for three rows of graphs, give it to the list
		if a.showProcs {
			procH += graphH
		}
		graphH = 0
	}
	if graphH > 0 {
		a.drawGraphs(cv, &st, y, graphH)
	}
	if procH > 0 {
		a.drawProcs(cv, &st, y+graphH, procH)
	}
	a.drawFooter(cv, &st, h-1)
	if a.help {
		a.drawHelp(cv, &st)
	}
	return cv
}

func (a *app) drawTitle(cv *canvas, st *styles) {
	cv.fill(0, 0, cv.w, 1, ' ', st.bar)
	x := cv.text(0, 0, " ▓▒░ ", st.bar)
	x = cv.text(x, 0, "PC RESOURCE MONITOR", st.bar)
	x = cv.text(x, 0, " ░▒▓ ", st.bar)

	right := fmt.Sprintf(" up %s  %s ", fmtUptime(a.s.st.uptime), time.Now().Format("15:04:05"))
	info := strings.Join(nonEmpty(a.host, a.model, a.cpuName), " · ")
	room := cv.w - x - width(right) - 1
	cv.text(x+1, 0, clip(info, room), st.bar)
	cv.text(cv.w-width(right), 0, right, st.bar)
}

// meter is an htop bar: "Mem[|||||||       2.91G/3.82G]"
func (cv *canvas) meter(x, y, w int, label string, frac float64, text string, st *styles) {
	x = cv.text(x, y, label, st.label)
	inner := w - width(label) - 2
	if inner < 1 {
		return
	}
	cv.set(x, y, '[', st.bracket)
	x++
	n := int(math.Round(clamp(frac, 0, 1) * float64(inner)))
	tr := []rune(clip(text, inner))
	ts := inner - len(tr)
	for i := 0; i < inner; i++ {
		switch {
		case i >= ts:
			cv.set(x+i, y, tr[i-ts], st.dim)
		case i < n:
			cv.set(x+i, y, '|', st.level(float64(i+1)/float64(inner)))
		}
	}
	cv.set(x+inner, y, ']', st.bracket)
}

// line writes label/value pairs like htop's "Tasks: 213, 2891 thr".
func (cv *canvas) line(x, y, w int, st *styles, parts ...string) {
	end := x + w
	for i, p := range parts {
		s := st.label
		if i%2 == 1 {
			s = st.accent
		}
		x = cv.text(x, y, clip(p, end-x), s)
	}
}

func (a *app) drawMeters(cv *canvas, st *styles, y int) int {
	s := &a.s.st
	lw := (cv.w - 3) / 2
	lx, rx := 1, 2+lw
	rw := cv.w - rx - 1

	cols := 1
	if len(s.cores) > 8 {
		cols = 2
	}
	per := (len(s.cores) + cols - 1) / cols
	cw := (lw - (cols - 1)) / cols
	for i, v := range s.cores {
		c, r := i/per, i%per
		cv.meter(lx+c*(cw+1), y+r, cw, fmt.Sprintf("%3d", i), v/100, fmt.Sprintf("%.1f%%", v), st)
	}
	ly := y + per
	cv.meter(lx, ly, lw, "Mem", ratio(s.memUsed, s.memTotal), fmtBytes(float64(s.memUsed))+"/"+fmtBytes(float64(s.memTotal)), st)
	cv.meter(lx, ly+1, lw, "Swp", ratio(s.swapUsed, s.swapTotal), fmtBytes(float64(s.swapUsed))+"/"+fmtBytes(float64(s.swapTotal)), st)
	ly += 2

	cv.meter(rx, y, rw, "CPU", s.cpu/100, fmt.Sprintf("%.1f%%", s.cpu), st)
	temp := "n/a"
	if s.hasTemp {
		temp = fmt.Sprintf("%.1f°C", s.temp)
	}
	cv.meter(rx, y+1, rw, "Tmp", s.temp/100, temp, st)
	cv.meter(rx, y+2, rw, "Dsk", ratio(s.diskUsed, s.diskTotal), fmtBytes(float64(s.diskUsed))+"/"+fmtBytes(float64(s.diskTotal)), st)
	cv.line(rx, y+3, rw, st, "Tasks: ", fmt.Sprintf("%d", len(s.procs)), ", ", fmt.Sprintf("%d thr", s.threads),
		"; Clock: ", fmt.Sprintf("%.2f GHz", s.mhz/1000))
	cv.line(rx, y+4, rw, st, "Uptime: ", fmtUptime(s.uptime))
	if s.wifi.ok {
		x := cv.text(rx, y+5, "WiFi: ", st.label)
		x = cv.text(x, y+5, clip(s.wifi.ssid, 20)+" ", st.accent)
		x = signalBars(cv, x, y+5, s.wifi.signal, st)
		cv.line(x, y+5, rx+rw-x, st, fmt.Sprintf(" %d%%", s.wifi.signal), "",
			"  ", fmt.Sprintf("%.0f Mbps", s.wifi.rxMbps), "  ", s.wifi.phy)
	} else {
		cv.line(rx, y+5, rw, st, "WiFi: ", "not connected")
	}
	return max(ly, y+6)
}

func signalBars(cv *canvas, x, y, signal int, st *styles) int {
	for i, r := range "▂▄▆█" {
		s := st.dim
		if signal > i*25 {
			s = st.accent
		}
		cv.set(x, y, r, s)
		x++
	}
	return x
}

// panel draws a titled box and returns the space inside it.
func (cv *canvas) panel(x, y, w, h int, title, info string, st *styles) (int, int, int, int) {
	cv.box(x, y, w, h, st.border)
	t := " " + title + " "
	cv.text(x+2, y, clip(t, w-4), st.title)
	if info != "" {
		i := " " + info + " "
		if x+2+width(t)+width(i)+2 <= x+w {
			cv.text(x+w-2-width(i), y, i, st.accent)
		}
	}
	return x + 1, y + 1, w - 2, h - 2
}

func (a *app) drawGraphs(cv *canvas, st *styles, y, h int) {
	s := &a.s.st
	w := cv.w
	h1, h2 := h/3, h/3
	h3 := h - h1 - h2

	// row 1: cpu, temperature, memory
	w1, w2 := w/3, w/3
	w3 := w - w1 - w2
	ix, iy, iw, ih := cv.panel(0, y, w1, h1, "CPU", fmt.Sprintf("%.0f%% · %.2f GHz", s.cpu, s.mhz/1000), st)
	cv.graph(ix, iy, iw, ih, s.cpuH, 0, 100, st, "")

	tinfo := "no sensor"
	if s.hasTemp {
		tinfo = fmt.Sprintf("%.0f°C · peak %.0f°C", s.temp, s.tempH.tail(iw).max())
	}
	ix, iy, iw, ih = cv.panel(w1, y, w2, h1, "TEMPERATURE", tinfo, st)
	cv.graph(ix, iy, iw, ih, s.tempH, 20, 100, st, "")

	ix, iy, iw, ih = cv.panel(w1+w2, y, w3, h1, "MEMORY", fmt.Sprintf("%s / %s", fmtBytes(float64(s.memUsed)), fmtBytes(float64(s.memTotal))), st)
	cv.graph(ix, iy, iw, ih, s.memH, 0, 100, st, "")

	// row 2: wifi and disk, each split into two graphs
	y += h1
	wa := w / 2
	title, info := "NETWORK", ""
	if s.wifi.ok {
		title, info = "WIFI "+s.wifi.ssid, fmt.Sprintf("%d%% signal", s.wifi.signal)
	}
	ix, iy, iw, ih = cv.panel(0, y, wa, h2, title, info, st)
	cv.dual(ix, iy, iw, ih, s.downH, s.upH, "↓ "+fmtRate(s.netDown), "↑ "+fmtRate(s.netUp), st)

	ix, iy, iw, ih = cv.panel(wa, y, w-wa, h2, "DISK "+s.drive, fmt.Sprintf("%.0f%% active · %.0f%% full", s.diskActive, ratio(s.diskUsed, s.diskTotal)*100), st)
	cv.dual(ix, iy, iw, ih, s.readH, s.writeH, "R "+fmtRate(s.diskRead), "W "+fmtRate(s.diskWrite), st)

	// row 3: the four gpu engines
	y += h2
	gpuInfo := ""
	if s.gpuShared > 0 || s.gpuDedicated > 0 {
		gpuInfo = "shared mem " + fmtBytes(float64(s.gpuShared))
		if s.gpuDedicated > 0 {
			gpuInfo += " · dedicated " + fmtBytes(float64(s.gpuDedicated))
		}
	}
	ix, iy, iw, ih = cv.panel(0, y, w, h3, strings.TrimSpace("GPU "+a.gpuName), gpuInfo, st)
	if len(s.gpuTypes) == 0 {
		cv.text(ix+1, iy, "no GPU counters on this PC", st.dim)
		return
	}
	n := len(s.gpuTypes)
	gw := (iw - (n - 1)) / n
	for i, t := range s.gpuTypes {
		gx := ix + i*(gw+1)
		if i == n-1 {
			gw = ix + iw - gx
		}
		cv.text(gx, iy, clip(gpuLabel(t), gw-5), st.label)
		pct := fmt.Sprintf("%3.0f%%", s.gpu[t])
		cv.text(gx+gw-width(pct), iy, pct, st.accent)
		cv.graph(gx, iy+1, gw, ih-1, *s.gpuH[t], 0, 100, st, "")
		if i < n-1 {
			for j := iy; j < iy+ih; j++ {
				cv.set(gx+gw, j, '│', st.grid)
			}
		}
	}
}

// dual stacks two auto-scaled graphs, e.g. download over upload.
func (cv *canvas) dual(x, y, w, h int, a, b series, la, lb string, st *styles) {
	top := h / 2
	for _, g := range []struct {
		data  series
		label string
		y, h  int
		style string
	}{{a, la, y, top, st.in}, {b, lb, y + top, h - top, st.out}} {
		peak := max(g.data.tail(w).max()*1.15, 64<<10)
		cv.graph(x, g.y, w, g.h, g.data, 0, peak, st, g.style)
		cv.text(x+1, g.y, " "+g.label+" ", g.style)
	}
}

func (a *app) sortedProcs() []proc {
	ps := append([]proc(nil), a.s.st.procs...)
	sort.SliceStable(ps, func(i, j int) bool {
		p, q := ps[i], ps[j]
		switch a.sortBy {
		case sortMem:
			if p.mem != q.mem {
				return p.mem > q.mem
			}
		case sortPID:
			return p.pid < q.pid
		case sortName:
			if !strings.EqualFold(p.name, q.name) {
				return strings.ToLower(p.name) < strings.ToLower(q.name)
			}
		default:
			if p.cpu != q.cpu {
				return p.cpu > q.cpu
			}
			if p.mem != q.mem {
				return p.mem > q.mem
			}
		}
		return p.pid < q.pid
	})
	return ps
}

func (a *app) drawProcs(cv *canvas, st *styles, y, h int) {
	s := &a.s.st
	ps := a.sortedProcs()
	nameW := cv.w - 48
	header := fmt.Sprintf("%7s %-*s %5s %6s %6s %7s %10s", "PID", nameW, "NAME", "THR", "CPU%", "MEM%", "RES", "TIME+")
	cv.fill(0, y, cv.w, 1, ' ', st.head)
	cv.text(0, y, header, st.head)
	// highlight whichever column we're sorting by
	col := map[int][2]int{sortPID: {0, 7}, sortName: {8, 4}, sortCPU: {nameW + 15, 6}, sortMem: {nameW + 22, 6}}[a.sortBy]
	cv.text(col[0], y, header[col[0]:col[0]+col[1]], st.headSort)

	rows := h - 1
	a.sel = max(0, min(a.sel, len(ps)-1))
	if a.sel < a.scroll {
		a.scroll = a.sel
	}
	if a.sel >= a.scroll+rows {
		a.scroll = a.sel - rows + 1
	}
	a.scroll = max(0, min(a.scroll, len(ps)-rows))

	for i := 0; i < rows && a.scroll+i < len(ps); i++ {
		p := ps[a.scroll+i]
		ry := y + 1 + i
		line := fmt.Sprintf("%7d %-*s %5d %6.1f %6.1f %7s %10s", p.pid, nameW, clip(p.name, nameW), p.threads,
			p.cpu, ratio(p.mem, s.memTotal)*100, fmtBytes(float64(p.mem)), fmtCPUTime(p.cpuTime))
		if a.scroll+i == a.sel {
			cv.fill(0, ry, cv.w, 1, ' ', st.sel)
			cv.text(0, ry, line, st.sel)
			continue
		}
		cv.text(0, ry, line, st.text)
		if p.cpu >= 1 {
			cv.text(nameW+15, ry, string([]rune(line)[nameW+15:nameW+21]), st.level(p.cpu/100))
		}
	}
}

func (a *app) drawFooter(cv *canvas, st *styles, y int) {
	keys := [][2]string{{"F1", "Help"}, {"F2", "Theme"}, {"F3", "Graphs"}, {"F4", "Procs"}, {"F6", "SortBy"}, {"F10", "Quit"}}
	x := 0
	for _, k := range keys {
		x = cv.text(x, y, k[0], st.key)
		x = cv.text(x, y, fmt.Sprintf("%-7s", k[1]), st.bar)
	}
	right := fmt.Sprintf(" %s · sort %s ", themes[a.theme].name, sortCols[a.sortBy])
	if cv.w-width(right) > x {
		cv.text(cv.w-width(right), y, right, st.dim)
	}
}

func (a *app) drawHelp(cv *canvas, st *styles) {
	lines := []string{
		"pc-resource-monitor " + version,
		"",
		"F1  h ?        this help",
		"F2  t          next colour theme",
		"F3  g          show / hide graphs",
		"F4  p          show / hide process list",
		"F6  s          change sort column",
		"c m n i        sort by cpu / mem / name / pid",
		"↑ ↓ PgUp PgDn  move through processes",
		"F10 q          quit",
		"",
		"themes: htop, phosphor, amber, c64",
		"press any key to close",
	}
	w, h := 50, len(lines)+2
	x, y := (cv.w-w)/2, (cv.h-h)/2
	cv.fill(x, y, w, h, ' ', st.text)
	cv.box(x, y, w, h, st.accent)
	for i, l := range lines {
		s := st.text
		if i == 0 {
			s = st.title
		}
		cv.text(x+2, y+1+i, clip(l, w-4), s)
	}
}

func ratio(a, b uint64) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

// fmtBytes is htop style: 1024 based, one letter.
func fmtBytes(b float64) string {
	units := "KMGT"
	if b < 1024 {
		return fmt.Sprintf("%.0fB", b)
	}
	u := -1
	for b >= 1024 && u < len(units)-1 {
		b /= 1024
		u++
	}
	if b >= 100 || u == 0 {
		return fmt.Sprintf("%.0f%c", b, units[u])
	}
	return fmt.Sprintf("%.2f%c", b, units[u])
}

func fmtRate(b float64) string {
	switch {
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB/s", b/(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.0f KB/s", b/(1<<10))
	}
	return fmt.Sprintf("%.0f B/s", b)
}

func fmtUptime(d time.Duration) string {
	days := int(d.Hours()) / 24
	rest := fmt.Sprintf("%02d:%02d:%02d", int(d.Hours())%24, int(d.Minutes())%60, int(d.Seconds())%60)
	switch days {
	case 0:
		return rest
	case 1:
		return "1 day, " + rest
	}
	return fmt.Sprintf("%d days, %s", days, rest)
}

// fmtCPUTime matches htop's TIME+ column.
func fmtCPUTime(d time.Duration) string {
	if d >= time.Hour {
		return fmt.Sprintf("%dh%02d:%02d", int(d.Hours()), int(d.Minutes())%60, int(d.Seconds())%60)
	}
	return fmt.Sprintf("%d:%05.2f", int(d.Minutes()), math.Mod(d.Seconds(), 60))
}

func nonEmpty(xs ...string) []string {
	var out []string
	for _, x := range xs {
		if x != "" {
			out = append(out, x)
		}
	}
	return out
}

package main

import (
	"flag"
	"fmt"
	"os"
	"time"
)

var version = "dev"

func main() {
	themeName := flag.String("theme", "htop", "colour theme: htop, phosphor, amber or c64")
	interval := flag.Duration("delay", time.Second, "time between updates")
	once := flag.Bool("once", false, "print a single frame and exit")
	w := flag.Int("width", 120, "frame width for -once")
	h := flag.Int("height", 40, "frame height for -once")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("pc-resource-monitor", version)
		return
	}

	a := newApp(*themeName)
	if *once {
		a.s.sample()
		time.Sleep(time.Second)
		a.s.sample()
		os.Stdout.WriteString(a.draw(*w, *h).render(false))
		return
	}

	t, err := openTerminal()
	if err != nil {
		fmt.Fprintln(os.Stderr, "pc-resource-monitor has to run in a console window:", err)
		os.Exit(1)
	}
	defer t.restore()
	run(a, t, max(*interval, 250*time.Millisecond))
}

func run(a *app, t *terminal, interval time.Duration) {
	keys := make(chan string, 32)
	go readKeys(keys)
	tick := time.NewTicker(interval)
	defer tick.Stop()
	resize := time.NewTicker(150 * time.Millisecond)
	defer resize.Stop()

	w, h := t.size()
	redraw := func() { t.write(a.draw(w, h).render(true)) }
	a.s.sample()
	redraw()
	for {
		select {
		case <-tick.C:
			a.s.sample()
			redraw()
		case k, ok := <-keys:
			if !ok || !a.handleKey(k) {
				return
			}
			redraw()
		case <-resize.C:
			if nw, nh := t.size(); nw != w || nh != h {
				w, h = nw, nh
				t.write("\x1b[2J")
				redraw()
			}
		}
	}
}

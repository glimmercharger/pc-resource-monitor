package main

import (
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// The whole screen is drawn into a grid of cells and then printed in one go,
// which keeps the terminal from flickering.

type cell struct {
	r rune
	s string
}

type canvas struct {
	w, h  int
	cells []cell
}

func newCanvas(w, h int, base string) *canvas {
	cv := &canvas{w: w, h: h, cells: make([]cell, w*h)}
	for i := range cv.cells {
		cv.cells[i] = cell{' ', base}
	}
	return cv
}

func (cv *canvas) set(x, y int, r rune, s string) {
	if x < 0 || y < 0 || x >= cv.w || y >= cv.h {
		return
	}
	cv.cells[y*cv.w+x] = cell{r, s}
}

func (cv *canvas) empty(x, y int) bool {
	if x < 0 || y < 0 || x >= cv.w || y >= cv.h {
		return false
	}
	return cv.cells[y*cv.w+x].r == ' '
}

// text writes str at x,y and returns the column right after it.
func (cv *canvas) text(x, y int, str, s string) int {
	for _, r := range str {
		cv.set(x, y, r, s)
		x++
	}
	return x
}

func (cv *canvas) fill(x, y, w, h int, r rune, s string) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			cv.set(i, j, r, s)
		}
	}
}

func (cv *canvas) box(x, y, w, h int, s string) {
	if w < 2 || h < 2 {
		return
	}
	for i := x + 1; i < x+w-1; i++ {
		cv.set(i, y, '─', s)
		cv.set(i, y+h-1, '─', s)
	}
	for j := y + 1; j < y+h-1; j++ {
		cv.set(x, j, '│', s)
		cv.set(x+w-1, j, '│', s)
	}
	cv.set(x, y, '┌', s)
	cv.set(x+w-1, y, '┐', s)
	cv.set(x, y+h-1, '└', s)
	cv.set(x+w-1, y+h-1, '┘', s)
}

var blocks = []rune(" ▁▂▃▄▅▆▇█")

// graph draws a Task Manager style history graph, newest sample on the right.
// Each cell has 8 steps of height thanks to the block characters.
func (cv *canvas) graph(x, y, w, h int, data series, lo, hi float64, st *styles, colour string) {
	if w <= 0 || h <= 0 {
		return
	}
	if h >= 4 {
		for q := 1; q < 4; q++ {
			gy := y + h - h*q/4
			for i := 0; i < w; i += 2 {
				cv.set(x+i, gy, '·', st.grid)
			}
		}
	}
	data = data.tail(w)
	off := w - len(data)
	for i, v := range data {
		f := clamp((v-lo)/(hi-lo), 0, 1)
		steps := int(math.Round(f * float64(h*8)))
		if steps == 0 && f > 0.004 {
			steps = 1
		}
		for r := 0; r < h && steps > r*8; r++ {
			s := colour
			if s == "" {
				s = st.level(float64(r+1) / float64(h))
			}
			cv.set(x+off+i, y+h-1-r, blocks[min(steps-r*8, 8)], s)
		}
	}
}

func (cv *canvas) render(positioned bool) string {
	var b strings.Builder
	b.Grow(cv.w * cv.h * 3)
	cur := ""
	for y := 0; y < cv.h; y++ {
		if positioned {
			b.WriteString("\x1b[" + strconv.Itoa(y+1) + ";1H")
		} else if y > 0 {
			b.WriteString("\x1b[0m\n")
			cur = ""
		}
		for x := 0; x < cv.w; x++ {
			c := cv.cells[y*cv.w+x]
			if c.s != cur {
				b.WriteString(c.s)
				cur = c.s
			}
			b.WriteRune(c.r)
		}
	}
	b.WriteString("\x1b[0m")
	if !positioned {
		b.WriteString("\n")
	}
	return b.String()
}

func width(s string) int { return utf8.RuneCountInString(s) }

// clip cuts s down to n columns.
func clip(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if width(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n])
}

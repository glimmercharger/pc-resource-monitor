package main

import "fmt"

type rgb [3]uint8

func hex(s string) rgb {
	var c rgb
	fmt.Sscanf(s, "%02x%02x%02x", &c[0], &c[1], &c[2])
	return c
}

type theme struct {
	name                         string
	bg, fg, dim, label, accent   rgb
	border, grid                 rgb
	low, mid, high               rgb // meters and graphs go low -> high as they fill up
	in, out                      rgb // download/read and upload/write
	barBg, barFg, keyBg, keyFg   rgb
	headBg, headFg, selBg, selFg rgb
}

var themes = []theme{
	{
		name: "htop",
		bg:   hex("0c0c0c"), fg: hex("d0d0d0"), dim: hex("767676"), label: hex("00afaf"), accent: hex("ffffff"),
		border: hex("3a3a3a"), grid: hex("262626"),
		low: hex("5fd700"), mid: hex("ffd75f"), high: hex("ff5f5f"),
		in: hex("5fafff"), out: hex("ff87d7"),
		barBg: hex("00afaf"), barFg: hex("000000"), keyBg: hex("0c0c0c"), keyFg: hex("ffffff"),
		headBg: hex("5faf00"), headFg: hex("000000"), selBg: hex("00afaf"), selFg: hex("000000"),
	},
	{
		name: "phosphor",
		bg:   hex("030d06"), fg: hex("33ff66"), dim: hex("1c7a3a"), label: hex("2fd65a"), accent: hex("c8ffd6"),
		border: hex("145a2a"), grid: hex("0b2c15"),
		low: hex("1f9e45"), mid: hex("33ff66"), high: hex("c8ffd6"),
		in: hex("33ff66"), out: hex("1f9e45"),
		barBg: hex("33ff66"), barFg: hex("021a08"), keyBg: hex("030d06"), keyFg: hex("c8ffd6"),
		headBg: hex("1f9e45"), headFg: hex("021a08"), selBg: hex("145a2a"), selFg: hex("c8ffd6"),
	},
	{
		name: "amber",
		bg:   hex("0d0800"), fg: hex("ffb000"), dim: hex("7a5300"), label: hex("ffc640"), accent: hex("ffe3a8"),
		border: hex("5c3e00"), grid: hex("2b1d00"),
		low: hex("b37a00"), mid: hex("ffb000"), high: hex("ffe3a8"),
		in: hex("ffb000"), out: hex("b37a00"),
		barBg: hex("ffb000"), barFg: hex("1a0f00"), keyBg: hex("0d0800"), keyFg: hex("ffe3a8"),
		headBg: hex("b37a00"), headFg: hex("1a0f00"), selBg: hex("5c3e00"), selFg: hex("ffe3a8"),
	},
	{
		name: "c64",
		bg:   hex("352879"), fg: hex("a49dff"), dim: hex("6c5eb5"), label: hex("70a4b2"), accent: hex("ffffff"),
		border: hex("6c5eb5"), grid: hex("41349a"),
		low: hex("9ae29b"), mid: hex("e6ee7c"), high: hex("ee8a7e"),
		in: hex("70a4b2"), out: hex("e6ee7c"),
		barBg: hex("a49dff"), barFg: hex("352879"), keyBg: hex("352879"), keyFg: hex("ffffff"),
		headBg: hex("70a4b2"), headFg: hex("352879"), selBg: hex("6c5eb5"), selFg: hex("ffffff"),
	},
}

func sgr(fg, bg rgb, bold bool) string {
	b := "0"
	if bold {
		b = "0;1"
	}
	return fmt.Sprintf("\x1b[%s;38;2;%d;%d;%d;48;2;%d;%d;%dm", b, fg[0], fg[1], fg[2], bg[0], bg[1], bg[2])
}

// styles are a theme turned into ready-to-print escape codes.
type styles struct {
	text, dim, label, bracket, accent, border, grid, title string
	bar, key, head, headSort, sel                          string
	levels                                                 [3]string
	in, out                                                string
}

func (t *theme) styles() styles {
	return styles{
		text:     sgr(t.fg, t.bg, false),
		dim:      sgr(t.dim, t.bg, false),
		label:    sgr(t.label, t.bg, false),
		bracket:  sgr(t.accent, t.bg, true),
		accent:   sgr(t.accent, t.bg, true),
		border:   sgr(t.border, t.bg, false),
		grid:     sgr(t.grid, t.bg, false),
		title:    sgr(t.label, t.bg, true),
		bar:      sgr(t.barFg, t.barBg, false),
		key:      sgr(t.keyFg, t.keyBg, true),
		head:     sgr(t.headFg, t.headBg, false),
		headSort: sgr(t.headFg, t.selBg, true),
		sel:      sgr(t.selFg, t.selBg, false),
		levels:   [3]string{sgr(t.low, t.bg, false), sgr(t.mid, t.bg, false), sgr(t.high, t.bg, false)},
		in:       sgr(t.in, t.bg, false),
		out:      sgr(t.out, t.bg, false),
	}
}

// level picks the low/mid/high colour for how full something is (0..1).
func (s *styles) level(f float64) string {
	switch {
	case f > 0.8:
		return s.levels[2]
	case f > 0.5:
		return s.levels[1]
	}
	return s.levels[0]
}

package main

import (
	"reflect"
	"testing"
)

func TestParseKeys(t *testing.T) {
	cases := map[string][]string{
		"q":              {"q"},
		"\x1b[A\x1b[B":   {"up", "down"},
		"\x1bOQ":         {"f2"},
		"\x1b[21~":       {"f10"},
		"\x1b":           {"esc"},
		"\x03":           {"ctrl-c"},
		"j\x1b[6~k":      {"j", "pgdn", "k"},
		"\x1b[1;5A":      nil, // ctrl+up, not bound to anything
		"\x1b[200~x":     {"x"},
		"\x1b[5~\x1b[5~": {"pgup", "pgup"},
	}
	for in, want := range cases {
		if got := parseKeys([]byte(in)); !reflect.DeepEqual(got, want) {
			t.Errorf("parseKeys(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFormatting(t *testing.T) {
	for b, want := range map[float64]string{512: "512B", 2048: "2K", 3 << 30: "3.00G", 150 << 20: "150M"} {
		if got := fmtBytes(b); got != want {
			t.Errorf("fmtBytes(%v) = %q, want %q", b, got, want)
		}
	}
	if got := pdhInstanceName("Intel(R) Wireless-AC 9560 #2"); got != "Intel[R] Wireless-AC 9560 _2" {
		t.Errorf("pdhInstanceName = %q", got)
	}
}

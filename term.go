package main

import (
	"bytes"
	"os"

	"golang.org/x/sys/windows"
)

type terminal struct {
	in, out         windows.Handle
	inMode, outMode uint32
}

func openTerminal() (*terminal, error) {
	t := &terminal{in: windows.Handle(os.Stdin.Fd()), out: windows.Handle(os.Stdout.Fd())}
	if err := windows.GetConsoleMode(t.in, &t.inMode); err != nil {
		return nil, err
	}
	if err := windows.GetConsoleMode(t.out, &t.outMode); err != nil {
		return nil, err
	}
	// raw-ish input with keys sent as escape sequences, and no quick edit
	// so clicking the window doesn't freeze the screen
	in := t.inMode&^(windows.ENABLE_ECHO_INPUT|windows.ENABLE_LINE_INPUT|windows.ENABLE_PROCESSED_INPUT|
		windows.ENABLE_MOUSE_INPUT|windows.ENABLE_QUICK_EDIT_MODE) |
		windows.ENABLE_VIRTUAL_TERMINAL_INPUT | windows.ENABLE_EXTENDED_FLAGS
	if err := windows.SetConsoleMode(t.in, in); err != nil {
		return nil, err
	}
	if err := windows.SetConsoleMode(t.out, t.outMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING|windows.ENABLE_PROCESSED_OUTPUT); err != nil {
		windows.SetConsoleMode(t.in, t.inMode)
		return nil, err
	}
	// alternate screen, hide cursor, no line wrap
	t.write("\x1b[?1049h\x1b[?25l\x1b[?7l\x1b]0;pc-resource-monitor\x07")
	return t, nil
}

func (t *terminal) restore() {
	t.write("\x1b[0m\x1b[?7h\x1b[?25h\x1b[?1049l")
	windows.SetConsoleMode(t.in, t.inMode)
	windows.SetConsoleMode(t.out, t.outMode)
}

func (t *terminal) write(s string) { os.Stdout.WriteString(s) }

func (t *terminal) size() (int, int) {
	var info windows.ConsoleScreenBufferInfo
	if windows.GetConsoleScreenBufferInfo(t.out, &info) != nil {
		return 80, 24
	}
	return int(info.Window.Right-info.Window.Left) + 1, int(info.Window.Bottom-info.Window.Top) + 1
}

var escKeys = map[string]string{
	"\x1b[A": "up", "\x1b[B": "down", "\x1b[5~": "pgup", "\x1b[6~": "pgdn",
	"\x1b[H": "home", "\x1b[F": "end", "\x1b[1~": "home", "\x1b[4~": "end",
	"\x1bOP": "f1", "\x1bOQ": "f2", "\x1bOR": "f3", "\x1bOS": "f4",
	"\x1b[15~": "f5", "\x1b[17~": "f6", "\x1b[21~": "f10",
}

// parseKeys turns raw console input into key names ("up", "f2") or the
// typed character itself.
func parseKeys(b []byte) []string {
	var keys []string
	for len(b) > 0 {
		switch {
		case b[0] == 3:
			keys = append(keys, "ctrl-c")
			b = b[1:]
		case b[0] == 0x1b && len(b) == 1:
			keys = append(keys, "esc")
			b = b[1:]
		case b[0] == 0x1b:
			n := seqLen(b)
			if k, ok := escKeys[string(b[:n])]; ok {
				keys = append(keys, k)
			}
			b = b[n:]
		default:
			i := bytes.IndexByte(b, 0x1b)
			if i < 0 {
				i = len(b)
			}
			for _, r := range string(b[:i]) {
				if r == 3 {
					keys = append(keys, "ctrl-c")
				} else {
					keys = append(keys, string(r))
				}
			}
			b = b[i:]
		}
	}
	return keys
}

// seqLen finds where an escape sequence ends: ESC O x, or ESC [ ... final.
func seqLen(b []byte) int {
	if len(b) < 2 {
		return len(b)
	}
	if b[1] == 'O' {
		return min(3, len(b))
	}
	if b[1] != '[' {
		return 1
	}
	for i := 2; i < len(b); i++ {
		if b[i] >= 0x40 && b[i] <= 0x7e {
			return i + 1
		}
	}
	return len(b)
}

func readKeys(out chan<- string) {
	buf := make([]byte, 64)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			close(out)
			return
		}
		for _, k := range parseKeys(buf[:n]) {
			out <- k
		}
	}
}

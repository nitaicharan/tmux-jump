package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// applyStyle parses spec and overwrites *dst with the resulting SGR
// escape. Empty spec is a no-op (keeps the default). A parse error logs
// to stderr and keeps the default — better than dropping the user into
// a styleless render with no hint why.
func applyStyle(role, spec string, dst *string) {
	if spec == "" {
		return
	}
	esc, err := parseStyle(spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "tmux-jump: @jump-color-%s %q: %v — using default\n", role, spec, err)
		return
	}
	*dst = esc
}

// parseStyle converts a tmux-style style spec into an ANSI SGR escape.
// Empty input yields "" with no error so callers can leave a default in
// place. Invalid input yields "" with an error so callers can log and
// fall back to a default — tmux's own style parser is similarly strict.
//
// Recognized tokens (comma-separated, whitespace and case tolerant):
//
//	fg=COLOR, bg=COLOR
//	bold, dim, italic, underline, blink, reverse
//
// COLOR values:
//
//	default | none                 → terminal default (39 fg / 49 bg)
//	black,red,green,yellow,blue,magenta,cyan,white
//	bright<name>                   → 8-color bright variants
//	#RRGGBB                        → 24-bit true color
//	colour0-255 / color0-255       → 256-color palette
//	0-255                          → 256-color palette (bare integer)
//
// Output orders attributes first, then fg, then bg, which round-trips
// the four hardcoded defaults exactly so users can copy them into
// tmux.conf with no visual change.
func parseStyle(spec string) (string, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return "", nil
	}
	var attrs []string
	var fg, bg string
	for _, raw := range strings.Split(spec, ",") {
		tok := strings.ToLower(strings.TrimSpace(raw))
		if tok == "" {
			continue
		}
		if key, val, ok := strings.Cut(tok, "="); ok {
			key = strings.TrimSpace(key)
			val = strings.TrimSpace(val)
			if key == "" || val == "" {
				return "", fmt.Errorf("invalid token %q", raw)
			}
			switch key {
			case "fg":
				code, err := colorCode(val, false)
				if err != nil {
					return "", err
				}
				fg = code
			case "bg":
				code, err := colorCode(val, true)
				if err != nil {
					return "", err
				}
				bg = code
			default:
				return "", fmt.Errorf("unknown key %q", key)
			}
			continue
		}
		code, ok := attrCode(tok)
		if !ok {
			return "", fmt.Errorf("unknown token %q", tok)
		}
		attrs = append(attrs, code)
	}
	parts := attrs
	if fg != "" {
		parts = append(parts, fg)
	}
	if bg != "" {
		parts = append(parts, bg)
	}
	if len(parts) == 0 {
		return "", nil
	}
	return "\x1b[" + strings.Join(parts, ";") + "m", nil
}

func attrCode(s string) (string, bool) {
	switch s {
	case "bold":
		return "1", true
	case "dim":
		return "2", true
	case "italic":
		return "3", true
	case "underline":
		return "4", true
	case "blink":
		return "5", true
	case "reverse":
		return "7", true
	}
	return "", false
}

var namedColors = map[string]int{
	"black": 0, "red": 1, "green": 2, "yellow": 3,
	"blue": 4, "magenta": 5, "cyan": 6, "white": 7,
	"brightblack": 8, "brightred": 9, "brightgreen": 10, "brightyellow": 11,
	"brightblue": 12, "brightmagenta": 13, "brightcyan": 14, "brightwhite": 15,
}

func colorCode(val string, isBg bool) (string, error) {
	if val == "default" || val == "none" {
		if isBg {
			return "49", nil
		}
		return "39", nil
	}
	if strings.HasPrefix(val, "#") {
		if len(val) != 7 {
			return "", fmt.Errorf("hex color must be #RRGGBB: %q", val)
		}
		r, err1 := strconv.ParseUint(val[1:3], 16, 8)
		g, err2 := strconv.ParseUint(val[3:5], 16, 8)
		b, err3 := strconv.ParseUint(val[5:7], 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return "", fmt.Errorf("invalid hex color: %q", val)
		}
		base := 38
		if isBg {
			base = 48
		}
		return fmt.Sprintf("%d;2;%d;%d;%d", base, r, g, b), nil
	}
	if rest, ok := strings.CutPrefix(val, "colour"); ok {
		return parse256(rest, isBg)
	}
	if rest, ok := strings.CutPrefix(val, "color"); ok {
		return parse256(rest, isBg)
	}
	if n, err := strconv.Atoi(val); err == nil {
		if n < 0 || n > 255 {
			return "", fmt.Errorf("color out of range: %d", n)
		}
		return sgr256(n, isBg), nil
	}
	if n, ok := namedColors[val]; ok {
		return namedCode(n, isBg), nil
	}
	return "", fmt.Errorf("unknown color %q", val)
}

func parse256(s string, isBg bool) (string, error) {
	n, err := strconv.Atoi(s)
	if err != nil || n < 0 || n > 255 {
		return "", fmt.Errorf("invalid 256-color: %q", s)
	}
	return sgr256(n, isBg), nil
}

func sgr256(n int, isBg bool) string {
	base := 38
	if isBg {
		base = 48
	}
	return fmt.Sprintf("%d;5;%d", base, n)
}

// namedCode maps 0-15 to legacy SGR slots (30-37 / 40-47 for basic,
// 90-97 / 100-107 for bright) so terminal palette overrides still apply.
func namedCode(n int, isBg bool) string {
	if n < 8 {
		if isBg {
			return strconv.Itoa(40 + n)
		}
		return strconv.Itoa(30 + n)
	}
	if isBg {
		return strconv.Itoa(100 + (n - 8))
	}
	return strconv.Itoa(90 + (n - 8))
}

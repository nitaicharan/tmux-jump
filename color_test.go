package main

import "testing"

func TestParseStyleEmpty(t *testing.T) {
	got, err := parseStyle("")
	if err != nil {
		t.Fatalf("empty spec: unexpected error %v", err)
	}
	if got != "" {
		t.Errorf("empty spec: want \"\", got %q", got)
	}
}

func TestParseStyleSingleAttr(t *testing.T) {
	got, err := parseStyle("bold")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "\x1b[1m" {
		t.Errorf("bold: got %q want \\x1b[1m", got)
	}
}

func TestParseStyleNamedFg(t *testing.T) {
	got, _ := parseStyle("fg=red")
	if got != "\x1b[31m" {
		t.Errorf("fg=red: got %q", got)
	}
}

func TestParseStyleNamedBg(t *testing.T) {
	got, _ := parseStyle("bg=blue")
	if got != "\x1b[44m" {
		t.Errorf("bg=blue: got %q", got)
	}
}

func TestParseStyleBrightNamed(t *testing.T) {
	got, _ := parseStyle("fg=brightwhite,bg=red,bold")
	if got != "\x1b[1;97;41m" {
		t.Errorf("bright/bold mix: got %q want \\x1b[1;97;41m", got)
	}
}

func TestParseStyleReproducesDefaults(t *testing.T) {
	// The parser MUST reproduce the four hardcoded defaults exactly,
	// so users opting in can copy the defaults into tmux.conf and see
	// no visual change. If these break, the migration is not transparent.
	cases := map[string]string{
		"fg=white,dim":                  "\x1b[2;37m",
		"fg=black,bg=brightyellow,bold": "\x1b[1;30;103m",
		"fg=brightwhite,bg=red,bold":    "\x1b[1;97;41m",
		"fg=brightwhite,bg=blue,bold":   "\x1b[1;97;44m",
	}
	for spec, want := range cases {
		got, err := parseStyle(spec)
		if err != nil {
			t.Errorf("parseStyle(%q) error: %v", spec, err)
			continue
		}
		if got != want {
			t.Errorf("parseStyle(%q)=%q want %q", spec, got, want)
		}
	}
}

func TestParseStyleHex(t *testing.T) {
	got, err := parseStyle("fg=#7b7c7e,dim")
	if err != nil {
		t.Fatalf("hex+dim: unexpected error %v", err)
	}
	want := "\x1b[2;38;2;123;124;126m"
	if got != want {
		t.Errorf("hex+dim: got %q want %q", got, want)
	}
}

func TestParseStyleHexFgBg(t *testing.T) {
	got, _ := parseStyle("fg=#0c0c0c,bg=#fdbd27,bold")
	want := "\x1b[1;38;2;12;12;12;48;2;253;189;39m"
	if got != want {
		t.Errorf("got %q want %q", got, want)
	}
}

func TestParseStyleColour256(t *testing.T) {
	got, _ := parseStyle("fg=colour208,bg=color17")
	want := "\x1b[38;5;208;48;5;17m"
	if got != want {
		t.Errorf("256-color: got %q want %q", got, want)
	}
}

func TestParseStyleBareNumber(t *testing.T) {
	got, _ := parseStyle("fg=208")
	if got != "\x1b[38;5;208m" {
		t.Errorf("bare 208: got %q", got)
	}
}

func TestParseStyleDefault(t *testing.T) {
	got, _ := parseStyle("fg=default,bg=default")
	if got != "\x1b[39;49m" {
		t.Errorf("default: got %q", got)
	}
}

func TestParseStyleAllAttrs(t *testing.T) {
	got, _ := parseStyle("bold,dim,italic,underline,blink,reverse")
	if got != "\x1b[1;2;3;4;5;7m" {
		t.Errorf("all attrs: got %q", got)
	}
}

func TestParseStyleWhitespaceAndCase(t *testing.T) {
	got, _ := parseStyle("  Fg=Red , Bold  ")
	if got != "\x1b[1;31m" {
		t.Errorf("case/whitespace tolerance: got %q", got)
	}
}

func TestParseStyleEmptyTokensIgnored(t *testing.T) {
	got, _ := parseStyle("fg=red,,bold,")
	if got != "\x1b[1;31m" {
		t.Errorf("empty tokens: got %q", got)
	}
}

func TestParseStyleInvalid(t *testing.T) {
	cases := []string{
		"bogus",
		"fg=notacolor",
		"fg=#xyz",
		"fg=#12345",    // wrong length
		"fg=300",       // out of range
		"fg=colour300", // out of range
		"fg=",          // missing value
		"=red",         // missing key
	}
	for _, spec := range cases {
		got, err := parseStyle(spec)
		if err == nil {
			t.Errorf("parseStyle(%q) expected error, got %q", spec, got)
		}
	}
}

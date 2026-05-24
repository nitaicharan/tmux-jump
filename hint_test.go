package main

import (
	"strings"
	"testing"
)

func TestDefaultSelectedWithLimit10(t *testing.T) {
	cases := map[int]int{
		0:  0,
		1:  0,
		10: 9,
		11: 0, // beyond limit → 0
	}
	for n, want := range cases {
		if got := defaultSelected(n); got != want {
			t.Errorf("defaultSelected(%d)=%d want %d", n, got, want)
		}
	}
}

// Three "foo" tokens; labels should land past each "foo".
// With query="foo" and default hints "duhetonasi", no hint extends "foo"
// (no "food"/"foou"/… in the buffer), so the first 3 hints — d,u,h — are
// assigned in order.
func TestRenderLabelsAtWordEnd(t *testing.T) {
	rs := rows("foo bar foo baz foo extra")
	ms := findMatches(rs, []rune("foo"))
	if len(ms) != 3 {
		t.Fatalf("setup: want 3 matches, got %d", len(ms))
	}
	for i, m := range ms {
		if m.WordEnd != m.Col+m.Len {
			t.Fatalf("match[%d] WordEnd=%d expected %d (foo has no trailing word chars)", i, m.WordEnd, m.Col+m.Len)
		}
	}
	hints := []rune("duhetonasi")
	labels := assignLabels(rs, []rune("foo"), ms, hints, map[posID]rune{})
	// selected = -1 isolates this test from the selected-highlight path —
	// we only care that labels land at WordEnd, not which match is currently
	// arrow-selected.
	out := render(rs, ms, labels, -1, "foo", 80, 24)

	// Each match emits: ansiMatch + "foo" + ansiReset + ansiHint + <glyph>.
	// This asserts the label glyph sits IMMEDIATELY past the matched word.
	for i, glyph := range []string{"d", "u", "h"} {
		if labels[i] != rune(glyph[0]) {
			t.Fatalf("label[%d] = %q, want %q (default hint pool ordering)", i, labels[i], glyph)
		}
		want := "\x1b[1;30;103mfoo\x1b[0m\x1b[1;97;44m" + glyph
		if !strings.Contains(out, want) {
			t.Errorf("match %d: expected label glyph %q immediately past 'foo' (sequence %q not found) in:\n%q", i, glyph, want, out)
		}
	}
}

// Typing "co" against "configuration extra configuration extra" with hints
// "XY". Neither "coX" nor "coY" extends the query, so both labels survive
// exclusion. The label should land at WordEnd = Col+Len (immediately past
// the typed prefix), overlaying the 'n', NOT at the end of the surrounding
// token. The match band on "co" must stay on.
func TestRenderLabelImmediatelyAfterMatch(t *testing.T) {
	rs := rows("configuration extra configuration extra")
	ms := findMatches(rs, []rune("co"))
	if len(ms) != 2 {
		t.Fatalf("setup: want 2 matches, got %d", len(ms))
	}
	if ms[0].WordEnd != 2 {
		t.Fatalf("setup: first match WordEnd should be 2 (Col+Len), got %d", ms[0].WordEnd)
	}
	hints := []rune("XY")
	labels := assignLabels(rs, []rune("co"), ms, hints, map[posID]rune{})
	if labels[0] != 'X' || labels[1] != 'Y' {
		t.Fatalf("labels = %q, want [X Y] (nothing in buffer extends 'co' with X or Y)", string(labels))
	}
	out := render(rs, ms, labels, -1, "co", 80, 24)
	expected := "\x1b[1;30;103mco\x1b[0m\x1b[1;97;44mX"
	if !strings.Contains(out, expected) {
		t.Errorf("expected label 'X' immediately after the typed match 'co' (looking for %q), got:\n%q", expected, out)
	}
}

// No legacy hint-mode wording in the status line under any branch.
func TestRenderNoLegacyHintModeStatus(t *testing.T) {
	rs := rows("foo bar foo")
	ms := findMatches(rs, []rune("foo"))
	labels := assignLabels(rs, []rune("foo"), ms, []rune("duhetonasi"), map[posID]rune{})
	out := render(rs, ms, labels, 0, "foo", 80, 24)
	if strings.Contains(out, "hint: press") {
		t.Errorf("legacy hint-mode wording leaked into status line:\n%s", out)
	}
}

func TestRenderEmitsDefaultEscapes(t *testing.T) {
	// Locks in that the unconfigured render still emits the historical
	// defaults — leaving every @jump-color-* option unset must be a
	// no-op vs. pre-color-config releases.
	rs := rows("foo bar")
	ms := findMatches(rs, []rune("foo"))
	out := render(rs, ms, nil, -1, "foo", 80, 24)
	for _, esc := range []string{"\x1b[2;37m", "\x1b[1;30;103m"} {
		if !strings.Contains(out, esc) {
			t.Errorf("missing default escape %q in render output", esc)
		}
	}
}

func TestRenderUsesOverriddenEscapes(t *testing.T) {
	saved := ansiMatch
	defer func() { ansiMatch = saved }()
	ansiMatch = "\x1b[SENTINEL"
	rs := rows("foo bar")
	ms := findMatches(rs, []rune("foo"))
	out := render(rs, ms, nil, -1, "foo", 80, 24)
	if !strings.Contains(out, "SENTINEL") {
		t.Errorf("override of ansiMatch did not reach render output")
	}
}

func TestRenderNavigableAtTenMatches(t *testing.T) {
	// 10 matches: "a" in 10 distinct rows
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = "a"
	}
	rs := rows(lines...)
	ms := findMatches(rs, []rune("a"))
	if len(ms) != 10 {
		t.Fatalf("setup: want 10 matches, got %d", len(ms))
	}
	labels := assignLabels(rs, []rune("a"), ms, []rune("duhetonasi"), map[posID]rune{})
	out := render(rs, ms, labels, 0, "a", 80, 24)
	if !strings.Contains(out, "[1/10]") {
		t.Errorf("expected navigable status for 10 matches, got:\n%s", out)
	}
}

// Match runs to EOL: WordEnd == len(row). The EOL clamp must place the
// label at the last column, overlaying the final char of the match
// instead of writing past EOL.
func TestRenderLabelAtEOLOverlaysLastChar(t *testing.T) {
	rs := rows("hello", "hello")
	ms := findMatches(rs, []rune("hello"))
	if len(ms) != 2 {
		t.Fatalf("setup: want 2 matches, got %d", len(ms))
	}
	for i, m := range ms {
		if m.WordEnd != len(rs[m.Row]) {
			t.Fatalf("match[%d]: want WordEnd == row len, got WordEnd=%d row=%d", i, m.WordEnd, len(rs[m.Row]))
		}
	}
	hints := []rune("XY")
	labels := assignLabels(rs, []rune("hello"), ms, hints, map[posID]rune{})
	if labels[0] != 'X' || labels[1] != 'Y' {
		t.Fatalf("labels = %q, want [X Y]", string(labels))
	}
	out := render(rs, ms, labels, -1, "hello", 80, 24)
	expectedFirst := "\x1b[1;30;103mhell\x1b[0m\x1b[1;97;44mX"
	if !strings.Contains(out, expectedFirst) {
		t.Errorf("expected first label 'X' to overlay last char at EOL (looking for %q):\n%q", expectedFirst, out)
	}
	expectedSecond := "\x1b[1;30;103mhell\x1b[0m\x1b[1;97;44mY"
	if !strings.Contains(out, expectedSecond) {
		t.Errorf("expected second label 'Y' to overlay last char at EOL (looking for %q):\n%q", expectedSecond, out)
	}
}

// "aaaa" with query "aa" yields 3 overlapping matches at cols 0,1,2.
func TestRenderLabelsForOverlappingMatches(t *testing.T) {
	rs := rows("aaaa")
	ms := findMatches(rs, []rune("aa"))
	if len(ms) != 3 {
		t.Fatalf("setup: want 3 overlapping matches, got %d", len(ms))
	}
	hints := []rune("XYZ")
	labels := assignLabels(rs, []rune("aa"), ms, hints, map[posID]rune{})
	out := render(rs, ms, labels, 0, "aa", 80, 24)
	if !strings.ContainsAny(out, "XYZ") {
		t.Errorf("expected at least one label glyph from XYZ:\n%s", out)
	}
}

// Two non-overlapping matches in different tokens of the same row.
func TestRenderLabelsForNonOverlappingMatchesKeepBothVisible(t *testing.T) {
	rs := rows("ab xyz ab")
	ms := findMatches(rs, []rune("ab"))
	if len(ms) != 2 {
		t.Fatalf("setup: want 2 non-overlapping matches, got %d: %+v", len(ms), ms)
	}
	hints := []rune("XY")
	labels := assignLabels(rs, []rune("ab"), ms, hints, map[posID]rune{})
	out := render(rs, ms, labels, 0, "ab", 80, 24)
	if !strings.Contains(out, "X") || !strings.Contains(out, "Y") {
		t.Errorf("expected both label glyphs X and Y to be visible (non-overlapping matches):\n%s", out)
	}
}

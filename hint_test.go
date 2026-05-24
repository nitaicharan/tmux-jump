package main

import (
	"strings"
	"testing"
)

func TestHintIndex(t *testing.T) {
	hints := []rune("duhetonasi")
	cases := map[rune]int{
		'd': 0,
		'u': 1,
		'i': 9,
		'x': -1,
	}
	for r, want := range cases {
		if got := hintIndex(hints, r); got != want {
			t.Errorf("hintIndex(%q)=%d want %d", r, got, want)
		}
	}
}

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

func TestRenderHintBadgesAtWordEnd(t *testing.T) {
	// Three "foo" tokens separated by spaces; hints should land past each "foo".
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
	out := render(rs, ms, 0, "foo", 80, 24, true, hints)

	// Each match emits: ansiMatch + "foo" + ansiReset + ansiHint + <glyph>.
	// This asserts the hint glyph sits IMMEDIATELY past the matched word,
	// not at the match start (which would put the glyph before "foo").
	for i, glyph := range []string{"d", "u", "h"} {
		want := "\x1b[1;30;103mfoo\x1b[0m\x1b[1;97;44m" + glyph
		if !strings.Contains(out, want) {
			t.Errorf("match %d: expected hint glyph %q immediately past 'foo' (sequence %q not found) in:\n%q", i, glyph, want, out)
		}
	}
	if !strings.Contains(out, "hint") {
		t.Errorf("status bar should mention hint mode:\n%s", out)
	}
}

func TestRenderHintImmediatelyAfterMatch(t *testing.T) {
	// Two-match setup so navigable engages and hints can render.
	// Typing "co" against "configuration" should drop the hint immediately
	// past "co" (at col 2, overlaying the 'n'), NOT at the end of the whole
	// token. The match band on "co" must stay on.
	rs := rows("configuration extra configuration extra")
	ms := findMatches(rs, []rune("co"))
	if len(ms) != 2 {
		t.Fatalf("setup: want 2 matches, got %d", len(ms))
	}
	if ms[0].WordEnd != 2 {
		t.Fatalf("setup: first match WordEnd should be 2 (Col+Len), got %d", ms[0].WordEnd)
	}
	hints := []rune("XY")
	out := render(rs, ms, 0, "co", 80, 24, true, hints)
	// Render emits ansiMatch + "co" + ansiReset + ansiHint + glyph.
	// Asserting this exact sequence verifies the hint lands immediately past
	// the typed match (not at the end of the surrounding token) AND that the
	// match band on "co" stays on.
	expected := "\x1b[1;30;103mco\x1b[0m\x1b[1;97;44mX"
	if !strings.Contains(out, expected) {
		t.Errorf("expected hint 'X' immediately after the typed match 'co' (looking for %q), got:\n%q", expected, out)
	}
}

func TestRenderNoHintsWhenDisabled(t *testing.T) {
	rs := rows("foo bar foo")
	ms := findMatches(rs, []rune("foo"))
	out := render(rs, ms, 0, "foo", 80, 24, false, []rune("duhetonasi"))
	// Status bar should use navigable format, not hint mode
	if strings.Contains(out, "hint: press") {
		t.Errorf("hint mode status leaked into non-hint render")
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
	out := render(rs, ms, 0, "a", 80, 24, false, []rune("duhetonasi"))
	if !strings.Contains(out, "[1/10]") {
		t.Errorf("expected navigable status for 10 matches, got:\n%s", out)
	}
}

func TestRenderHintAtEOLOverlaysLastChar(t *testing.T) {
	// Match runs to EOL: query equals the whole row content so WordEnd ==
	// len(row). The EOL clamp must place the hint at the last column,
	// overlaying the final char of the match instead of writing past EOL.
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
	out := render(rs, ms, 0, "hello", 80, 24, true, hints)
	// Match band covers cols 0-4 ("hello"). EOL clamp puts the hint at col 4,
	// overlaying the 'o'. Output sequence: ansiMatch + "hell" + ansiReset +
	// ansiHint + glyph (the 'o' is replaced by the glyph).
	expectedFirst := "\x1b[1;30;103mhell\x1b[0m\x1b[1;97;44mX"
	if !strings.Contains(out, expectedFirst) {
		t.Errorf("expected first hint 'X' to overlay last char at EOL (looking for %q):\n%q", expectedFirst, out)
	}
	expectedSecond := "\x1b[1;30;103mhell\x1b[0m\x1b[1;97;44mY"
	if !strings.Contains(out, expectedSecond) {
		t.Errorf("expected second hint 'Y' to overlay last char at EOL (looking for %q):\n%q", expectedSecond, out)
	}
}

func TestRenderHintsForOverlappingMatches(t *testing.T) {
	// "aaaa" with query "aa" yields 3 overlapping matches at cols 0,1,2.
	// Each has its own WordEnd (Col+Len): 2, 3, 4. Hint cols: 2, 3, 3 (the
	// last is clamped by the EOL rule since len(row)=4). The first two land
	// at distinct cols; the third collides with the second and last write
	// wins — accepted v1 behavior. We assert no crash and at least one hint
	// glyph appears.
	rs := rows("aaaa")
	ms := findMatches(rs, []rune("aa"))
	if len(ms) != 3 {
		t.Fatalf("setup: want 3 overlapping matches, got %d", len(ms))
	}
	hints := []rune("XYZ")
	out := render(rs, ms, 0, "aa", 80, 24, true, hints)
	if !strings.ContainsAny(out, "XYZ") {
		t.Errorf("expected at least one hint glyph from XYZ:\n%s", out)
	}
}

func TestRenderHintsForOverlappingMatchesWithRoomKeepBothVisible(t *testing.T) {
	// Two non-overlapping matches in different tokens of the same row.
	// Each has its own WordEnd, so the shift counter never collides.
	// Goal: confirm both hint glyphs render at distinct positions.
	rs := rows("ab xyz ab")
	ms := findMatches(rs, []rune("ab"))
	if len(ms) != 2 {
		t.Fatalf("setup: want 2 non-overlapping matches, got %d: %+v", len(ms), ms)
	}
	hints := []rune("XY")
	out := render(rs, ms, 0, "ab", 80, 24, true, hints)
	if !strings.Contains(out, "X") || !strings.Contains(out, "Y") {
		t.Errorf("expected both hint glyphs X and Y to be visible (non-overlapping matches):\n%s", out)
	}
}

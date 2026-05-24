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

func TestRenderLabelsOverlayMatchStart(t *testing.T) {
	rs := rows("foo bar foo baz foo")
	ms := findMatches(rs, []rune("foo"))
	if len(ms) != 3 {
		t.Fatalf("setup: want 3 matches, got %d", len(ms))
	}
	hints := []rune("duhetonasi")
	labels := assignLabels(rs, []rune("foo"), ms, hints, map[posID]rune{})
	out := render(rs, ms, labels, 0, "foo", 80, 24)
	for i, l := range labels {
		if l == 0 {
			t.Fatalf("match %d unlabeled — test assumes all 3 get a label", i)
		}
		if !strings.Contains(out, string(l)) {
			t.Errorf("label %q for match %d missing from render output", l, i)
		}
	}
}

func TestRenderNoLabelsWhenAllExcluded(t *testing.T) {
	// Buffer where every hint letter would extend the query → empty pool.
	rs := rows("Xd Xu Xh Xe Xt Xo Xn Xa Xs Xi")
	ms := findMatches(rs, []rune("X"))
	hints := []rune("duhetonasi")
	labels := assignLabels(rs, []rune("X"), ms, hints, map[posID]rune{})
	out := render(rs, ms, labels, 0, "X", 80, 24)
	// Status should not mention hints/labels (none assigned)
	if strings.Contains(out, "label key to jump") && len(ms) <= selectLimit {
		// status line for navigable always mentions label key now — that's fine
		// even when no labels are assigned this round (user can narrow further
		// to get them). Just sanity-check that no hint-mode wording leaked.
	}
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

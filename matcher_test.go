package main

import "testing"

func rows(lines ...string) [][]rune {
	out := make([][]rune, len(lines))
	for i, l := range lines {
		out[i] = []rune(l)
	}
	return out
}

func TestFindMatchesEmpty(t *testing.T) {
	if got := findMatches(rows("hello"), []rune("")); got != nil {
		t.Fatalf("empty query: want nil, got %v", got)
	}
}

func TestFindMatchesSubstring(t *testing.T) {
	got := findMatches(rows("foo bar foo"), []rune("foo"))
	if len(got) != 2 {
		t.Fatalf("want 2 matches, got %d: %+v", len(got), got)
	}
	if got[0] != (Match{Row: 0, Col: 0, Len: 3, WordEnd: 3}) {
		t.Errorf("match[0]=%+v", got[0])
	}
	if got[1] != (Match{Row: 0, Col: 8, Len: 3, WordEnd: 11}) {
		t.Errorf("match[1]=%+v", got[1])
	}
}

func TestFindMatchesSmartCaseLower(t *testing.T) {
	got := findMatches(rows("Foo foo FOO"), []rune("foo"))
	if len(got) != 3 {
		t.Fatalf("lowercase query should be case-insensitive; got %d matches: %+v", len(got), got)
	}
}

func TestFindMatchesSmartCaseMixed(t *testing.T) {
	got := findMatches(rows("Foo foo FOO"), []rune("Foo"))
	if len(got) != 1 {
		t.Fatalf("uppercase-containing query should be case-sensitive; got %d: %+v", len(got), got)
	}
	if got[0].Col != 0 {
		t.Errorf("want first Foo at col 0, got %+v", got[0])
	}
}

func TestFindMatchesAcrossRows(t *testing.T) {
	got := findMatches(rows("alpha", "beta", "gamma"), []rune("a"))
	// alpha: col 0, col 4 | beta: col 3 | gamma: col 1, col 4
	if len(got) != 5 {
		t.Fatalf("want 5 matches across rows, got %d: %+v", len(got), got)
	}
}

func TestFindMatchesWordEnd(t *testing.T) {
	cases := []struct {
		name  string
		lines []string
		query string
		want  []Match
	}{
		{
			name:  "WordEnd is just past the typed match (middle of word)",
			lines: []string{"hello configuration world"},
			query: "co",
			// "co" at col 6; WordEnd = Col + Len = 8 (the 'n').
			want: []Match{{Row: 0, Col: 6, Len: 2, WordEnd: 8}},
		},
		{
			name:  "WordEnd is just past the typed match (start of word)",
			lines: []string{"configuration world"},
			query: "co",
			want:  []Match{{Row: 0, Col: 0, Len: 2, WordEnd: 2}},
		},
		{
			name:  "match runs to EOL — WordEnd equals row length",
			lines: []string{"foo bar"},
			query: "bar",
			// "bar" at col 4 spans to EOL; WordEnd = 4 + 3 = 7 = len(row).
			want: []Match{{Row: 0, Col: 4, Len: 3, WordEnd: 7}},
		},
		{
			name:  "single-char match",
			lines: []string{"a b c"},
			query: "a",
			want:  []Match{{Row: 0, Col: 0, Len: 1, WordEnd: 1}},
		},
		{
			name:  "match inside path — hint lands at the typed prefix, not the slash/dot",
			lines: []string{"src/config_test.go end"},
			query: "co",
			// "co" at col 4; WordEnd = 6 (between 'co' and 'n' of config_test.go).
			want: []Match{{Row: 0, Col: 4, Len: 2, WordEnd: 6}},
		},
		{
			name:  "match covers stem before extension dot",
			lines: []string{"test.md test.pdf"},
			query: "test",
			// Each "test" yields WordEnd = Col + 4 (the '.').
			want: []Match{
				{Row: 0, Col: 0, Len: 4, WordEnd: 4},
				{Row: 0, Col: 8, Len: 4, WordEnd: 12},
			},
		},
		{
			name:  "overlapping matches in one token — each gets its own WordEnd",
			lines: []string{"aaaa"},
			query: "aa",
			want: []Match{
				{Row: 0, Col: 0, Len: 2, WordEnd: 2},
				{Row: 0, Col: 1, Len: 2, WordEnd: 3},
				{Row: 0, Col: 2, Len: 2, WordEnd: 4},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := findMatches(rows(tc.lines...), []rune(tc.query))
			if len(got) != len(tc.want) {
				t.Fatalf("got %d matches, want %d: %+v", len(got), len(tc.want), got)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("match[%d]=%+v want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestFindMatchesOverlapping(t *testing.T) {
	// "aaaa" with query "aa" -> positions 0,1,2
	got := findMatches(rows("aaaa"), []rune("aa"))
	if len(got) != 3 {
		t.Fatalf("want 3 overlapping matches, got %d: %+v", len(got), got)
	}
}

func TestFindMatchesShortRow(t *testing.T) {
	got := findMatches(rows("hi"), []rune("hello"))
	if got != nil {
		t.Fatalf("row shorter than query should yield nothing, got %+v", got)
	}
}

func TestHasUpper(t *testing.T) {
	cases := map[string]bool{
		"foo":    false,
		"Foo":    true,
		"FOO":    true,
		"":       false,
		"123":    false,
		"café":   false,
		"CAFÉ":   true,
	}
	for in, want := range cases {
		if got := hasUpper([]rune(in)); got != want {
			t.Errorf("hasUpper(%q)=%v want %v", in, got, want)
		}
	}
}

// --- assignLabels ---

func contains(labels []rune, r rune) bool {
	for _, l := range labels {
		if l == r {
			return true
		}
	}
	return false
}

// Collision case from JIRA-hint-key-collision.md. Six `test*` filenames,
// query "tes", default hints. Every assigned label must be a letter
// that would NOT extend the query — only "t" would (→ "test"), so the
// assigned set must exclude 't'.
func TestAssignLabelsExcludesNarrowingChars(t *testing.T) {
	rs := rows(
		"test1",
		"test2",
		"test.csv",
		"test.md",
		"test.sh",
		"test.txt",
	)
	query := []rune("tes")
	ms := findMatches(rs, query)
	if len(ms) != 6 {
		t.Fatalf("setup: want 6 matches, got %d", len(ms))
	}
	hints := []rune("duhetonasi")
	reuse := map[posID]rune{}
	labels := assignLabels(rs, query, ms, hints, reuse)

	if contains(labels, 't') {
		t.Errorf("'t' would extend query to 'test' (all 6 rows) — must be excluded; got labels %q", string(labels))
	}
	// Every other letter in hints (d, u, h, e, o, n, a, s, i) does NOT
	// extend "tes" against this buffer, so all 6 matches should be labeled.
	for i, l := range labels {
		if l == 0 {
			t.Errorf("match %d should have a label (pool has 9 letters, only 6 matches); labels=%q", i, string(labels))
		}
	}
}

// When every hint letter would extend the query, the pool is empty and
// no labels are assigned. The user must narrow further.
func TestAssignLabelsEmptyPool(t *testing.T) {
	// Construct one row where "X<c>" exists for every c in hints.
	rs := rows("Xd Xu Xh Xe Xt Xo Xn Xa Xs Xi")
	query := []rune("X")
	ms := findMatches(rs, query)
	if len(ms) != 10 {
		t.Fatalf("setup: want 10 matches, got %d", len(ms))
	}
	hints := []rune("duhetonasi")
	reuse := map[posID]rune{}
	labels := assignLabels(rs, query, ms, hints, reuse)
	for i, l := range labels {
		if l != 0 {
			t.Errorf("expected no label at match %d (pool exhausted), got %q", i, l)
		}
	}
}

// Re-use: position-stable label across narrowing keystrokes.
func TestAssignLabelsReuseAcrossNarrow(t *testing.T) {
	rs := rows("config console context")
	hints := []rune("duhetonasi")
	reuse := map[posID]rune{}

	q1 := []rune("co")
	m1 := findMatches(rs, q1)
	if len(m1) != 3 {
		t.Fatalf("setup q1: want 3 matches, got %d (%+v)", len(m1), m1)
	}
	l1 := assignLabels(rs, q1, m1, hints, reuse)

	// Find the match starting at col 7 ("console") and its label.
	var consoleLabel rune
	for i, m := range m1 {
		if m.Col == 7 {
			consoleLabel = l1[i]
		}
	}
	if consoleLabel == 0 {
		t.Fatalf("console match (col 7) should be labeled in q1: labels=%q matches=%+v", string(l1), m1)
	}

	q2 := []rune("con")
	m2 := findMatches(rs, q2)
	if len(m2) != 3 {
		t.Fatalf("setup q2: want 3 matches, got %d", len(m2))
	}
	l2 := assignLabels(rs, q2, m2, hints, reuse)

	for i, m := range m2 {
		if m.Col == 7 && l2[i] != consoleLabel {
			t.Errorf("re-use broken: console (col 7) was %q in q1, now %q in q2", consoleLabel, l2[i])
		}
	}
}

// Backspace: after narrow + widen, labels return to their original values.
func TestAssignLabelsReuseSurvivesBackspace(t *testing.T) {
	rs := rows("config console context")
	hints := []rune("duhetonasi")
	reuse := map[posID]rune{}

	q1 := []rune("co")
	m1 := findMatches(rs, q1)
	l1 := assignLabels(rs, q1, m1, hints, reuse)

	q2 := []rune("con")
	m2 := findMatches(rs, q2)
	_ = assignLabels(rs, q2, m2, hints, reuse)

	// Backspace back to "co": same matches, same labels expected.
	m3 := findMatches(rs, q1)
	l3 := assignLabels(rs, q1, m3, hints, reuse)

	if len(l1) != len(l3) {
		t.Fatalf("label count mismatch: %d vs %d", len(l1), len(l3))
	}
	for i := range m1 {
		if m1[i] != m3[i] {
			t.Fatalf("match order changed between calls; matcher determinism is the test's contract")
		}
		if l1[i] != l3[i] {
			t.Errorf("match %d label hopped %q → %q across narrow+backspace", i, l1[i], l3[i])
		}
	}
}

// Smart-case: exclusion uses the same case rules as findMatches. A
// lowercase query is case-insensitive (so an exclusion letter can be
// matched in either case); an uppercase query is case-sensitive.
func TestAssignLabelsSmartCaseInExclusion(t *testing.T) {
	rs := rows("cot Cox")
	hints := []rune("t")

	// Lowercase query "co" matches both "cot" (col 0) and "Cox" (col 4).
	// For c='t': "cot" insensitive matches "cot" → 't' is excluded.
	qLower := []rune("co")
	mLower := findMatches(rs, qLower)
	if len(mLower) != 2 {
		t.Fatalf("lower-case 'co' should match both words, got %d", len(mLower))
	}
	lLower := assignLabels(rs, qLower, mLower, hints, map[posID]rune{})
	if contains(lLower, 't') {
		t.Errorf("lowercase query: 't' would extend (case-insensitive 'cot') — must be excluded; got %q", string(lLower))
	}

	// Uppercase query "Co" is case-sensitive — matches "Cox" at col 4 only.
	// For c='t': "Cot" case-sensitive does NOT occur in the buffer (only
	// lowercase "cot" and "Cox"), so 't' should NOT be excluded.
	qUpper := []rune("Co")
	mUpper := findMatches(rs, qUpper)
	if len(mUpper) != 1 || mUpper[0].Col != 4 {
		t.Fatalf("uppercase 'Co' should match only 'Cox' at col 4, got %+v", mUpper)
	}
	lUpper := assignLabels(rs, qUpper, mUpper, hints, map[posID]rune{})
	if !contains(lUpper, 't') {
		t.Errorf("uppercase query: 't' should be available (case-sensitive 'Cot' not in buffer); got %q", string(lUpper))
	}
}

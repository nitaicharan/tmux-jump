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

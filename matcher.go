package main

import "unicode"

type Match struct {
	Row, Col, Len int
	WordEnd       int
}

type posID struct{ Row, Col int }

func findMatches(rows [][]rune, query []rune) []Match {
	if len(query) == 0 {
		return nil
	}
	cs := hasUpper(query)
	var out []Match
	for r, row := range rows {
		if len(row) < len(query) {
			continue
		}
		for c := 0; c+len(query) <= len(row); c++ {
			if equalAt(row, c, query, cs) {
				out = append(out, Match{Row: r, Col: c, Len: len(query), WordEnd: c + len(query)})
			}
		}
	}
	return out
}

// hasMatchPrefix reports whether query occurs at least once in rows.
// Short-circuits on first hit. Smart-case rules identical to findMatches.
func hasMatchPrefix(rows [][]rune, query []rune) bool {
	if len(query) == 0 {
		return false
	}
	cs := hasUpper(query)
	for _, row := range rows {
		if len(row) < len(query) {
			continue
		}
		for c := 0; c+len(query) <= len(row); c++ {
			if equalAt(row, c, query, cs) {
				return true
			}
		}
	}
	return false
}

// assignLabels returns one rune per match (0 = unlabeled).
//
// Algorithm (port of flash.nvim's labeler.lua):
//  1. Build a label pool by removing every hint char c for which
//     findMatches(rows, query+c) would be non-empty. The remaining pool
//     is collision-free: pressing one of these chars cannot also be
//     interpreted as narrowing the query.
//  2. First pass — re-use stable labels: for each match, if reuse[pos]
//     names a label still in the pool, take it. Keeps the letter on a
//     given match stable across keystrokes.
//  3. Second pass — fresh assignment: assign remaining pool letters in
//     order to remaining matches. When the pool empties, trailing
//     matches stay unlabeled (the user must narrow further to label them).
//
// reuse is mutated: surviving and fresh assignments are recorded for
// the next call. Stale positions (no longer in matches) linger in the
// map so backspace can bring them back with their old labels.
func assignLabels(rows [][]rune, query []rune, matches []Match, hints []rune, reuse map[posID]rune) []rune {
	out := make([]rune, len(matches))
	if len(matches) == 0 {
		return out
	}

	// Step 1: build the pool by excluding letters that would extend query.
	probe := make([]rune, len(query)+1)
	copy(probe, query)
	poolSet := make(map[rune]bool, len(hints))
	pool := make([]rune, 0, len(hints))
	for _, c := range hints {
		if poolSet[c] {
			continue
		}
		probe[len(query)] = c
		if !hasMatchPrefix(rows, probe) {
			poolSet[c] = true
			pool = append(pool, c)
		}
	}

	// Step 2: first pass — re-use stable labels for known positions.
	used := make(map[rune]bool, len(pool))
	for i, m := range matches {
		id := posID{Row: m.Row, Col: m.Col}
		prev, ok := reuse[id]
		if !ok || used[prev] || !poolSet[prev] {
			continue
		}
		out[i] = prev
		used[prev] = true
	}

	// Step 3: second pass — assign pool head to remaining matches.
	pi := 0
	for i, m := range matches {
		if out[i] != 0 {
			continue
		}
		for pi < len(pool) && used[pool[pi]] {
			pi++
		}
		if pi >= len(pool) {
			break
		}
		label := pool[pi]
		pi++
		out[i] = label
		used[label] = true
		reuse[posID{Row: m.Row, Col: m.Col}] = label
	}

	return out
}

func hasUpper(s []rune) bool {
	for _, r := range s {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

func equalAt(row []rune, c int, query []rune, cs bool) bool {
	for i, q := range query {
		a := row[c+i]
		if cs {
			if a != q {
				return false
			}
		} else {
			if unicode.ToLower(a) != unicode.ToLower(q) {
				return false
			}
		}
	}
	return true
}

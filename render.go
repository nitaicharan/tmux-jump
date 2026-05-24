package main

import (
	"fmt"
	"strings"
)

const (
	ansiReset   = "\x1b[0m"
	ansiStatus  = "\x1b[7m"
	ansiClear   = "\x1b[2J\x1b[H"
	ansiHideCur = "\x1b[?25l"
	ansiShowCur = "\x1b[?25h"
)

var (
	ansiDim      = "\x1b[2;37m"
	ansiMatch    = "\x1b[1;30;103m"
	ansiSelected = "\x1b[1;97;41m"
	ansiHint     = "\x1b[1;97;44m"
)

// cell mark: 0 none, 1 match, 2 selected match, 3 hint badge
//
// labels[i] is the rune assigned to matches[i] by assignLabels (0 = unlabeled).
// Labels render whenever non-zero — there is no mode toggle. Selected-match
// highlight (mark=2) is independent of labels: an arrow-selected match still
// gets the selected color even when labeled.
func render(rows [][]rune, matches []Match, labels []rune, selected int, query string, width, height int) string {
	var b strings.Builder
	b.WriteString(ansiHideCur)
	b.WriteString(ansiClear)

	contentRows := height - 1
	if contentRows < 1 {
		contentRows = 1
	}

	navigable := len(matches) > 1 && len(matches) <= selectLimit

	hi := make([][]byte, len(rows))
	hintGlyph := map[int]map[int]rune{}
	for i, m := range matches {
		if m.Row >= len(hi) {
			continue
		}
		if hi[m.Row] == nil {
			hi[m.Row] = make([]byte, len(rows[m.Row]))
		}
		mark := byte(1)
		if navigable && i == selected {
			mark = 2
		}
		for k := 0; k < m.Len && m.Col+k < len(hi[m.Row]); k++ {
			hi[m.Row][m.Col+k] = mark
		}
		if i < len(labels) && labels[i] != 0 && m.Col < len(hi[m.Row]) {
			hi[m.Row][m.Col] = 3
			if hintGlyph[m.Row] == nil {
				hintGlyph[m.Row] = map[int]rune{}
			}
			hintGlyph[m.Row][m.Col] = labels[i]
		}
	}

	for r := 0; r < contentRows; r++ {
		if r > 0 {
			b.WriteString("\r\n")
		}
		if r >= len(rows) {
			continue
		}
		row := rows[r]
		var mask []byte
		if r < len(hi) {
			mask = hi[r]
		}
		writeRow(&b, row, mask, width, hintGlyph[r])
	}

	b.WriteString("\r\n")
	b.WriteString(ansiStatus)
	var status string
	switch {
	case len(query) == 0:
		status = " jump> (type to narrow; Esc to cancel) "
	case navigable:
		status = fmt.Sprintf(" jump> %s  [%d/%d]  ↑↓ pick · Enter jump · label key to jump ", query, selected+1, len(matches))
	default:
		status = fmt.Sprintf(" jump> %s  (%d matches) ", query, len(matches))
	}
	if len(status) > width {
		status = status[:width]
	}
	b.WriteString(status)
	b.WriteString(strings.Repeat(" ", max(0, width-len(status))))
	b.WriteString(ansiReset)
	return b.String()
}

func writeRow(b *strings.Builder, row []rune, mask []byte, width int, hints map[int]rune) {
	state := byte(0)
	b.WriteString(ansiDim)
	limit := len(row)
	if limit > width {
		limit = width
	}
	for i := 0; i < limit; i++ {
		var m byte
		if i < len(mask) {
			m = mask[i]
		}
		if m != state {
			b.WriteString(ansiReset)
			switch m {
			case 0:
				b.WriteString(ansiDim)
			case 1:
				b.WriteString(ansiMatch)
			case 2:
				b.WriteString(ansiSelected)
			case 3:
				b.WriteString(ansiHint)
			}
			state = m
		}
		if m == 3 {
			if h, ok := hints[i]; ok {
				b.WriteRune(h)
				continue
			}
		}
		b.WriteRune(row[i])
	}
	b.WriteString(ansiReset)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

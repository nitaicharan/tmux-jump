#!/usr/bin/env bash
# tmux-jump wrapper: captures the visible pane, runs the Go TUI in a
# borderless popup sized to the pane, then moves the copy-mode cursor
# to the row/word-end picked by the user (or row/col under JUMP_SELECT=1).
#
# Defensive: every failure path falls back to a no-op cleanly. The
# wrapper never leaves the pane stuck in copy-mode at a half-moved
# position — if we entered copy-mode but didn't complete the jump,
# the cleanup trap cancels copy-mode for us.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN="${JUMP_BINARY:-$SCRIPT_DIR/bin/tmux-jump}"

if ! command -v "$BIN" >/dev/null 2>&1 && [ ! -x "$BIN" ]; then
	tmux run-shell -b "bash $SCRIPT_DIR/install-wizard.sh" 2>/dev/null || true
	exit 0
fi

tmpdir=$(mktemp -d 2>/dev/null) || exit 0
pane=""
entered_copy=0
completed=0

cleanup() {
	rm -rf "$tmpdir" 2>/dev/null || true
	if [ "$entered_copy" = 1 ] && [ "$completed" = 0 ] && [ -n "$pane" ]; then
		tmux send-keys -t "$pane" -X cancel 2>/dev/null || true
	fi
}
trap cleanup EXIT

info=$(tmux display-message -p -F '#{pane_id} #{pane_width} #{pane_height} #{pane_in_mode} #{pane_mode} #{scroll_position}' 2>/dev/null) || exit 0
read -r pane w h in_mode mode scroll_pos <<<"$info"
[ -n "${pane:-}" ] || exit 0

in_copy=0
if [ "${in_mode:-0}" = 1 ] && [ "${mode:-}" = "copy-mode" ]; then
	in_copy=1
fi
case "${scroll_pos:-}" in '' | *[!0-9]*) scroll_pos=0 ;; esac

cap="$tmpdir/cap"
res="$tmpdir/res"

# Use format expansion in -S/-E so tmux evaluates scroll_position
# atomically at capture time. Decoupling the read from the capture (as
# we did before) leaves a race: any new line arriving in the pane
# between the two bumps oy by 1, and our pre-computed -S/-E then
# samples one row off. Empty scroll_position (not in copy-mode)
# evaluates to 0, giving the live-pane range [0, h-1].
tmux capture-pane -p -t "$pane" \
	-S '-#{scroll_position}' \
	-E '#{e|-:#{e|-:#{pane_height},#{scroll_position}},1}' \
	>"$cap" 2>/dev/null || exit 0

if [ -n "${JUMP_DEBUG:-}" ]; then
	{
		echo "info=[$info]"
		echo "pane=$pane w=$w h=$h in_mode=$in_mode mode=[$mode] scroll_pos=$scroll_pos in_copy=$in_copy"
		echo "--- captured ($(wc -l < "$cap") lines) ---"
		cat "$cap"
	} >>"${JUMP_DEBUG}" 2>/dev/null || true
fi

# Read @jump-* options at invocation time so users can tweak colors/
# hints/delay live without re-sourcing the plugin script. Env-var
# overrides (JUMP_*) still win to keep ad-hoc testing simple.
opt() { tmux show-option -gqv "$1" 2>/dev/null || true; }

hints_val="${JUMP_HINTS:-$(opt @jump-hints)}"
hints_arg=""
if [ -n "$hints_val" ]; then
	hints_arg="-hints $(printf %q "$hints_val")"
fi

minlen_val="${JUMP_LABEL_MIN_PATTERN_LENGTH:-$(opt @jump-label-min-pattern-length)}"
minlen_arg=""
if [ -n "$minlen_val" ]; then
	case "$minlen_val" in [0-9]*)
		minlen_arg="-min-pattern-length $(printf %q "$minlen_val")"
	;; esac
fi

color_args=""
for pair in "color-dim:JUMP_COLOR_DIM:@jump-color-dim" \
	"color-match:JUMP_COLOR_MATCH:@jump-color-match" \
	"color-selected:JUMP_COLOR_SELECTED:@jump-color-selected" \
	"color-hint:JUMP_COLOR_HINT:@jump-color-hint"; do
	IFS=: read -r flag var opt_name <<<"$pair"
	val="${!var:-}"
	[ -z "$val" ] && val="$(opt "$opt_name")"
	if [ -n "$val" ]; then
		color_args="$color_args -$flag $(printf %q "$val")"
	fi
done

tmux display-popup -E -B -w "$w" -h "$h" -x P -y P \
	"$BIN -capture $(printf %q "$cap") -out $(printf %q "$res") -w $w -h $h $hints_arg $minlen_arg $color_args" \
	2>/dev/null || exit 0

[ -s "$res" ] || exit 0

IFS=, read -r row col word_end len <"$res" || exit 0
[ -n "${row:-}" ] && [ -n "${col:-}" ] && [ -n "${word_end:-}" ] || exit 0
case "$row" in '' | *[!0-9]*) exit 0 ;; esac
case "$col" in '' | *[!0-9]*) exit 0 ;; esac
case "$word_end" in '' | *[!0-9]*) exit 0 ;; esac
case "${len:-}" in *[!0-9]*) len="" ;; esac

# JUMP_SELECT keeps the cursor at the match start so begin-selection
# + cursor-right Len-1 covers exactly the matched query. Default
# landing is end-of-word for plain jumps.
if [ "${JUMP_SELECT:-0}" = 1 ]; then
	target_col="$col"
else
	target_col="$word_end"
fi

if [ "$in_copy" != 1 ]; then
	tmux copy-mode -t "$pane" 2>/dev/null || exit 0
	entered_copy=1
fi
tmux send-keys -t "$pane" -X top-line 2>/dev/null || exit 0
if [ "$row" -gt 0 ]; then
	tmux send-keys -t "$pane" -N "$row" -X cursor-down 2>/dev/null || exit 0
fi
if [ "$target_col" -gt 0 ]; then
	tmux send-keys -t "$pane" -N "$target_col" -X cursor-right 2>/dev/null || exit 0
fi
if [ "${JUMP_SELECT:-0}" = 1 ] && [ -n "${len:-}" ] && [ "$len" -gt 0 ]; then
	tmux send-keys -t "$pane" -X begin-selection 2>/dev/null || exit 0
	if [ "$len" -gt 1 ]; then
		tmux send-keys -t "$pane" -N "$((len - 1))" -X cursor-right 2>/dev/null || exit 0
	fi
fi
completed=1
exit 0

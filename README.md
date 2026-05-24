# 🐇 tmux-jump

> Incremental-narrowing jump for tmux — type what you see, land on it.

[![pipeline](https://git.j4hangir.com/tmux/tmux-jump/badges/master/pipeline.svg)](https://git.j4hangir.com/tmux/tmux-jump/-/pipelines)
[![release](https://git.j4hangir.com/tmux/tmux-jump/-/badges/release.svg)](https://git.j4hangir.com/tmux/tmux-jump/-/releases)
[![license](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Like [`flash.nvim`](https://github.com/folke/flash.nvim) / [`leap.nvim`](https://github.com/ggandor/leap.nvim), but for the visible tmux pane. Press a key, start typing the word you're looking at, and the moment only one match remains the copy-mode cursor jumps to the **end of that word**. No two-char mnemonics to memorise.

---

## ✨ Features

- 🎯 **Incremental narrow** — type and watch matches shrink, auto-jump on unique match
- 🔍 **Smart-case** — uppercase in query ⇒ case-sensitive, otherwise insensitive
- 🖼️ **In-place overlay** — pane-sized borderless popup dims non-matches and highlights the rest
- 🪶 **Single static binary** — Go, no runtime deps on the user's box
- 🔒 **Fails safe** — any error path yields control back cleanly, never leaves the pane stuck
- 🧵 **Visible pane only** — scoped to what's on screen, nothing funny with scrollback

---

## 📦 Install

### With TPM

```tmux
set -g @plugin 'j4hangir/tmux-jump'
```

Then `prefix + I` to fetch & install.

### Manual

```sh
git clone https://git.j4hangir.com/tmux/tmux-jump ~/.tmux/plugins/tmux-jump
echo "run-shell ~/.tmux/plugins/tmux-jump/tmux-jump.tmux" >> ~/.tmux.conf
tmux source-file ~/.tmux.conf
```

On first launch a wizard prompts you to either **download the prebuilt Linux x86_64 binary** (from GitLab CI) or **build from source** with Go.

### 🗑️ Uninstall

```sh
tmux unbind j                            # or whatever @jump-key you chose
rm -rf ~/.tmux/plugins/tmux-jump         # also removes bin/tmux-jump
sed -i.bak '/tmux-jump\|@jump-/d' ~/.tmux.conf
tmux source ~/.tmux.conf
```

---

## 🕹️ Usage

Press `prefix + j` (default), start typing. That's it.

| Key | Action |
| --- | --- |
| any printable char | append to query (or pick a label if it matches one) |
| `Backspace` | pop last char (labels for surviving positions are preserved) |
| `Enter` | jump to selected match |
| label char | jump to that match (overlaid at the end of the matched word) |
| `↑` `↓` `←` `→` | (≤10 matches) cycle selection |
| `Esc` / `Ctrl-C` / `Ctrl-G` | cancel |

- ✅ Unique match → auto-jump, no `Enter` needed
- ⚡ **Labels render instantly** on every keystroke (flash.nvim-style). Pressing a label is always unambiguous — labels that would extend your query are dropped from the alphabet for that round, so the same key can never mean both "narrow" and "jump"
- 🔔 Zero matches after a keystroke → bell, character rejected

---

## ⚙️ Config

```tmux
set -g @jump-key j                          # default: j  (invoked as prefix + j)
set -g @jump-key-select J                   # optional: jump, then visual-select the match
set -g @jump-hints 'duhetonasi'             # up to 10 label chars
set -g @jump-label-min-pattern-length 0     # query length before labels render (flash: label.min_pattern_length)
set -g @jump-skip-wizard 0                  # 1 = never prompt for auto-update on version mismatch
```

`@jump-key-select` is a second, opt-in binding: same picker, but on landing it enters copy-mode, begins a selection, and extends it across the matched query — ready to yank.

> ⚠️ **Removed in this version:** `@jump-auto-hint-delay`. With flash-style label exclusion the timer is no longer meaningful — labels render synchronously. The plugin emits a one-time `tmux display-message` warning if it sees the option set in your config.

### 🎨 Colors (optional)

The four highlight roles have built-in defaults (unchanged from earlier releases). Leave every option unset for the classic look, or set any of them to override only that role — the rest keep their defaults.

```tmux
# Example overrides — tune to match your theme.
set -g @jump-color-dim      'fg=#7b7c7e,dim'                # non-matching text
set -g @jump-color-match    'fg=#0c0c0c,bg=#fdbd27,bold'    # matched text
set -g @jump-color-selected 'fg=#0c0c0c,bg=#ee5396,bold'    # currently-selected match
set -g @jump-color-hint     'fg=#0c0c0c,bg=#78a9ff,bold'    # label badges
```

Spec grammar (comma-separated, whitespace and case tolerant):

- `fg=COLOR`, `bg=COLOR` — `#RRGGBB`, `colour0-255` / `color0-255`, bare `0-255`, named (`black`…`white`, `brightblack`…`brightwhite`), or `default`
- Attributes: `bold`, `dim`, `italic`, `underline`, `blink`, `reverse`

A malformed spec logs a warning and falls back to the built-in default for that role.

---

## 🔬 How it works

```
prefix+j  →  capture-pane -p  →  display-popup -B  →  Go TUI
                                                       │
                                       type chars  →  narrow
                                                       │
                                              unique match?
                                                       │ yes
                                                       ▼
                                  copy-mode + cursor to (row, word-end)
```

1. **Capture** the visible pane to a temp file (`tmux capture-pane -p`).
2. **Overlay** a borderless `display-popup` sized exactly to the pane — perfect in-place redraw.
3. **Render** the captured text in the Go TUI: matches highlighted, everything else dimmed.
4. **Narrow** on each keystroke (smart-case substring). Unique match → popup closes, pane drops into copy-mode with the cursor at the **end of the matched word** (or at the match start when `@jump-key-select` is in play, so the selection can extend across the typed query).

---

## 🛠️ Development

Requires Go 1.22+ and tmux ≥ 3.2.

```sh
make build   # CGO_ENABLED=0 static binary at bin/tmux-jump
make test    # go test ./...
make clean
```

CI ([`.gitlab-ci.yml`](.gitlab-ci.yml)) runs `go test` + builds a static binary on every push; tags trigger a release with a downloadable asset link.

---

## 🚧 Known limitations (v1)

- ASCII-first column math — CJK / wide chars may misalign the landing column
- Active pane only — no cross-pane targeting
- Requires tmux ≥ 3.2 for borderless pane-sized popups

---

## 🙏 Credits

Inspired by:

- [`schasse/tmux-jump`](https://github.com/schasse/tmux-jump) — original tmux hint-based jumper
- [`fcsonline/tmux-fingers`](https://github.com/fcsonline/tmux-fingers) — the CI + install-wizard pattern lifted here
- [`flash.nvim`](https://github.com/folke/flash.nvim) / [`leap.nvim`](https://github.com/ggandor/leap.nvim) — the incremental-narrowing UX

## 📝 License

MIT — see [LICENSE](LICENSE).

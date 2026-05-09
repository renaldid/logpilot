# logpilot demo — recorded with VHS (https://github.com/charmbracelet/vhs)
# Run: make demo   (from project root)
# Requires: ffmpeg  →  sudo apt install ffmpeg

Output docs/demo.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 700
Set Theme "Catppuccin Mocha"
Set Padding 16
Set PlaybackSpeed 1.0
Set TypingSpeed 80ms

# ── setup ──────────────────────────────────────────────────────────────────
# clear any previous demo data and start fresh log generator
Hide
  Type "rm -rf /tmp/logpilot-demo && mkdir /tmp/logpilot-demo" Enter
  Sleep 200ms
  Type "bash ~/work/logpilot/docs/gen-demo-logs.sh /tmp/logpilot-demo &" Enter
  Sleep 1s
Show

# ── start logpilot ─────────────────────────────────────────────────────────
Type "~/work/logpilot/bin/logpilot --config ~/work/logpilot/docs/demo-config.yaml" Enter
Sleep 2s

# ── watch logs stream in ───────────────────────────────────────────────────
Sleep 3s

# ── fuzzy search ───────────────────────────────────────────────────────────
Type "/"
Sleep 400ms
Type "error"
Sleep 2s

# clear search
Type ""
Sleep 200ms
Escape
Sleep 500ms

# ── regex mode ─────────────────────────────────────────────────────────────
Type "r"
Sleep 300ms
Type "/"
Sleep 200ms
Type "GET|POST"
Sleep 2s
Escape
Sleep 300ms
Type "r"
Sleep 300ms

# ── sidebar: disable a service ─────────────────────────────────────────────
Tab
Sleep 500ms
# cursor starts on "api" — move to "db"
Down
Sleep 300ms
Down
Sleep 300ms
# toggle db off
Space
Sleep 1.5s

# re-enable db
Space
Sleep 800ms
Tab

# ── scroll up to browse history ────────────────────────────────────────────
Type "f"
Sleep 300ms
Up
Sleep 150ms
Up
Sleep 150ms
Up
Sleep 150ms
Up
Sleep 150ms
PageUp
Sleep 1s
PageDown
Sleep 500ms
Type "f"
Sleep 500ms

# ── help overlay ───────────────────────────────────────────────────────────
Type "?"
Sleep 2.5s
Type "?"
Sleep 500ms

# ── quit ───────────────────────────────────────────────────────────────────
Type "q"
Sleep 500ms

# cleanup background generator
Hide
  Type "kill %1 2>/dev/null; rm -rf /tmp/logpilot-demo" Enter
Show

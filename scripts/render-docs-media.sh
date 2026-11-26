#!/usr/bin/env bash
# Regenerates all documentation TUI visuals deterministically:
#   1. Crisp vector .svg terminal cards under static/images/tui/ (via kvx --snapshot | freeze)
#   2. Interactive hero recording static/casts/demo.cast (via scripts/record_demo_cast.py)
#   3. Animated README hero static/images/demo.gif (via agg)
set -euo pipefail

unset NO_COLOR

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

GOBIN="$(go env GOBIN)"
if [ -z "$GOBIN" ]; then
  GOBIN="$(go env GOPATH)/bin"
fi
export PATH="$GOBIN:$HOME/.cargo/bin:$PATH"

KVX_BIN="$ROOT/dist/kvx"
SVG_DIR="$ROOT/static/images/tui"
CAST_FILE="$ROOT/static/casts/demo.cast"
GIF_FILE="$ROOT/static/images/demo.gif"

mkdir -p "$ROOT/dist" "$SVG_DIR" "$(dirname "$CAST_FILE")" "$(dirname "$GIF_FILE")"

echo "==> Building dist/kvx..."
go build -o "$KVX_BIN" .

if ! command -v freeze >/dev/null 2>&1; then
  echo "error: Charmbracelet 'freeze' not found in PATH." >&2
  echo "Install it via: go install github.com/charmbracelet/freeze@latest" >&2
  exit 1
fi

AGG_BIN="${AGG_BIN:-$(command -v agg || true)}"
if [ "${SKIP_GIF:-0}" != "1" ] && [ -z "$AGG_BIN" ]; then
  echo "error: 'agg' (asciinema gif generator) not found in PATH." >&2
  echo "Install it via: cargo install --git https://github.com/asciinema/agg" >&2
  echo "Or pass AGG_BIN=/path/to/agg (or SKIP_GIF=1 to skip GIF generation)." >&2
  exit 1
fi

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT
EMPTY_CFG="$TMP_DIR/empty-config.yaml"
printf "{}\n" > "$EMPTY_CFG"

# Post-process freeze SVG output:
#   - Strip the ~355 KB embedded base64 @font-face blob so each SVG stays ~12-18 KB
#   - Strip dangling clip-path="url(#terminalMask)" attribute emitted by freeze
#   - Use a crisp cross-platform monospace font stack
#   - Add an explicit viewBox, aria-label, and <title> so SVGs scale cleanly in Hugo and GitHub README
optimize_svg() {
  local svg_path="$1"
  local label="$2"
  python3 - "$svg_path" "$label" <<'PY'
import html
import re
import sys
from pathlib import Path

svg_file = Path(sys.argv[1])
label = sys.argv[2]
text = svg_file.read_text(encoding="utf-8")

# Remove <!DOCTYPE ...>, embedded base64 @font-face style block, and dangling clip-path IRI
text = re.sub(r"<!DOCTYPE[^>]*>\s*", "", text, count=1)
text = re.sub(r"<style>\s*@font-face\s*\{.*?\}\s*</style>\s*", "", text, flags=re.DOTALL)
text = text.replace(' clip-path="url(#terminalMask)"', "")

# Replace font-family="JetBrains Mono" with a clean system monospace stack
mono_stack = "ui-monospace, 'JetBrains Mono', SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'DejaVu Sans Mono', monospace"
text = text.replace('font-family="JetBrains Mono"', f'font-family="{mono_stack}"')

# Add viewBox, accessibility attributes, and <title> to the root <svg> element
safe_label = html.escape(label, quote=True)

def patch_root_svg(match: re.Match[str]) -> str:
    w, h = match.group(1), match.group(2)
    return (
        f'<svg width="{w}" height="{h}" viewBox="0 0 {w} {h}" '
        f'role="img" aria-label="{safe_label}" xmlns="http://www.w3.org/2000/svg">'
        f"<title>{safe_label}</title>"
    )

text, count = re.subn(
    r'<svg\s+width="([0-9.]+)"\s+height="([0-9.]+)"\s+xmlns="http://www\.w3\.org/2000/svg">',
    patch_root_svg,
    text,
    count=1,
)
if count != 1:
    raise SystemExit(f"error: expected 1 root <svg> tag in {svg_file}, matched {count}")

svg_file.write_text(text, encoding="utf-8")
PY
}

render_svg_card() {
  local out_name="$1"
  local bg_color="$2"
  local label="$3"
  shift 3

  local out_path="$SVG_DIR/$out_name"
  XDG_CONFIG_HOME="$TMP_DIR" \
  CLICOLOR_FORCE=1 \
  COLORTERM=truecolor \
  TERM=xterm-256color \
    "$KVX_BIN" --config-file "$EMPTY_CFG" "$@" | \
    freeze --language ansi \
      --window \
      --background "$bg_color" \
      --padding "16,20,16,20" \
      --margin "0" \
      --border.radius 8 \
      --border.width 1 \
      --border.color "#1b3d63" \
      --font.size 13 \
      --line-height 1.25 \
      -o "$out_path" >/dev/null

  optimize_svg "$out_path" "$label"
  echo "  wrote static/images/tui/$out_name ($(wc -c < "$out_path" | tr -d ' ') bytes)"
}

echo "==> Rendering static/images/tui/*.svg snapshots..."

# 1. Built-in Theme Gallery (4 themes)
render_svg_card "theme-midnight.svg" "#0c1021" "kvx midnight theme" \
  examples/data/demo_services.json --theme midnight --snapshot --width 88 --height 18 --press "l"

render_svg_card "theme-dark.svg" "#16181d" "kvx dark theme" \
  examples/data/demo_services.json --theme dark --snapshot --width 88 --height 18 --press "l"

render_svg_card "theme-warm.svg" "#261710" "kvx warm theme" \
  examples/data/demo_services.json --theme warm --snapshot --width 88 --height 18 --press "l"

render_svg_card "theme-cool.svg" "#07162c" "kvx cool theme" \
  examples/data/demo_services.json --theme cool --snapshot --width 88 --height 18 --press "l"

# 2. Schema Card-List view (x-kvx-list) & Sectioned Detail view (x-kvx-detail)
render_svg_card "schema-card-list.svg" "#081426" "kvx schema card-list view (x-kvx-list)" \
  examples/data/demo_services.json --schema examples/data/demo_services_schema.json \
  --theme midnight --snapshot --width 92 --height 22 --press "j"

render_svg_card "schema-detail-view.svg" "#081426" "kvx schema sectioned detail view (x-kvx-detail)" \
  examples/data/demo_services.json --schema examples/data/demo_services_schema.json \
  --theme midnight --snapshot --width 92 --height 22 --press "j<Enter>"

render_svg_card "schema-view-toggle-raw.svg" "#081426" "kvx toggling detail view to raw table with v key" \
  examples/data/demo_services.json --schema examples/data/demo_services_schema.json \
  --theme midnight --snapshot --width 92 --height 22 --press "j<Enter>v"

# 3. CEL Expression Mode + Ctrl+Space Function Palette
render_svg_card "cel-function-palette.svg" "#081426" "kvx CEL expression mode and Ctrl+Space function palette" \
  examples/data/demo_services.json \
  --theme midnight --snapshot --width 94 --height 26 --press ":<C-u>_<C-Space><Tab><Tab><Down><Down><Down><Down>"

# 4. Async Status & Waiting Screen (x-kvx-status with masked inline input)
render_svg_card "status-screen.svg" "#081426" "kvx async status screen with hotkey actions and masked inline input" \
  examples/data/status_device_auth.json --schema examples/data/status_device_auth_schema.json \
  --theme midnight --snapshot --width 92 --height 20 --press "secr3t-cb-tok3n"

# 5. JWT / Nested Scalar Decoded view
render_svg_card "jwt-decoded.svg" "#081426" "kvx decoded Bearer JWT header and payload claims" \
  examples/data/demo_services.json \
  --theme midnight --snapshot --width 92 --height 20 --press "jl<Enter><Enter>j"

# Validate rendered SVGs contain expected TUI content and zero error states
python3 - "$SVG_DIR" <<'PY'
import re
import sys
from pathlib import Path

svg_dir = Path(sys.argv[1])
for svg in sorted(svg_dir.glob("*.svg")):
    plain = re.sub(r"<[^>]+>", "", svg.read_text(encoding="utf-8"))
    for bad in ("no matching functions", "Error:", "Path error:"):
        if bad in plain:
            raise SystemExit(f"error: {svg.name} contains unexpected error text {bad!r}")

jwt_plain = re.sub(r"<[^>]+>", "", (svg_dir / "jwt-decoded.svg").read_text(encoding="utf-8"))
for expected in ("header", "payload", "signature"):
    if expected not in jwt_plain:
        raise SystemExit(f"error: jwt-decoded.svg is missing decoded JWT key {expected!r}")

palette_plain = re.sub(r"<[^>]+>", "", (svg_dir / "cel-function-palette.svg").read_text(encoding="utf-8"))
if "filter" not in palette_plain:
    raise SystemExit("error: cel-function-palette.svg is missing list functions (expected 'filter')")
PY

echo "==> Recording interactive asciicast static/casts/demo.cast..."
python3 "$ROOT/scripts/record_demo_cast.py"

if [ "${SKIP_GIF:-0}" != "1" ]; then
  echo "==> Rendering animated hero GIF static/images/demo.gif via $AGG_BIN..."
  "$AGG_BIN" \
    --theme github-dark \
    --font-size 15 \
    --line-height 1.32 \
    --idle-time-limit 2 \
    --last-frame-duration 2.5 \
    --fps-cap 24 \
    --quiet \
    "$CAST_FILE" "$GIF_FILE"
  echo "  wrote static/images/demo.gif ($(wc -c < "$GIF_FILE" | tr -d ' ') bytes)"
else
  echo "==> SKIP_GIF=1 set; skipping static/images/demo.gif"
fi

echo "==> Done."

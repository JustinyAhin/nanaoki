#!/bin/sh
# Render v2-a/b/c.html to v2-a/b/c.png (1200x630) and v2-sheet.png. Needs Google Chrome and sips (macOS).
# Usage: render-v2.sh [a b c sheet]   (default: all)
set -eu
cd "$(dirname "$0")"
CHROME="${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
TMP="$(mktemp -d)"
[ $# -eq 0 ] && set -- a b c sheet

shot() { # shot <html> <out.png> <width> <height> <scale>
  "$CHROME" --headless=new --disable-gpu --hide-scrollbars \
    --force-device-scale-factor="$5" --window-size="$3,$4" \
    --virtual-time-budget=8000 --user-data-dir="$TMP/profile" \
    --screenshot="$2" "file://$PWD/$1" >/dev/null 2>&1
}

for v in "$@"; do
  if [ "$v" = sheet ]; then
    shot v2-sheet.html v2-sheet.png 1000 1670 1
  else
    shot "v2-$v.html" "$TMP/$v@2x.png" 1200 630 2
    sips -z 630 1200 "$TMP/$v@2x.png" --out "v2-$v.png" >/dev/null
  fi
done
rm -rf "$TMP"
ls -l v2-*.png

#!/bin/sh
# Render blog-<slug>.html to ../static/blog/<slug>/card.png (1200x630). Needs Google Chrome and sips (macOS).
# Usage: render-blog.sh <slug>
set -eu
[ $# -eq 1 ] || { echo "usage: render-blog.sh <slug>" >&2; exit 2; }
cd "$(dirname "$0")"
SLUG="$1"
[ -f "blog-$SLUG.html" ] || { echo "no blog-$SLUG.html in $PWD" >&2; exit 1; }
CHROME="${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
TMP="$(mktemp -d)"
OUT="../static/blog/$SLUG"
mkdir -p "$OUT"

"$CHROME" --headless=new --disable-gpu --hide-scrollbars \
  --force-device-scale-factor=2 --window-size=1200,630 \
  --virtual-time-budget=8000 --user-data-dir="$TMP/profile" \
  --screenshot="$TMP/card@2x.png" "file://$PWD/blog-$SLUG.html" >/dev/null 2>&1
sips -z 630 1200 "$TMP/card@2x.png" --out "$OUT/card.png" >/dev/null
rm -rf "$TMP"
ls -l "$OUT/card.png"

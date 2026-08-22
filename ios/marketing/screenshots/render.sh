#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

CHROME="${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
OUT="out"
mkdir -p "$OUT"

# label width height   (iPhone 6.9" is the one set App Store Connect requires)
SIZES=(
  "iphone-6.9 1320 2868"
)

for slide in slides/*.html; do
  name="$(basename "$slide" .html)"
  for size in "${SIZES[@]}"; do
    read -r label w h <<<"$size"
    echo "Rendering $name -> ${name}-${label}.png (${w}x${h})"
    "$CHROME" --headless --disable-gpu --hide-scrollbars \
      "--window-size=${w},${h}" --force-device-scale-factor=1 \
      "--screenshot=${OUT}/${name}-${label}.png" \
      "file://$(pwd)/${slide}" >/dev/null 2>&1
  done
done

echo "Done. Assets in ${OUT}/"

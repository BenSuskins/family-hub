#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

CHROME="${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
OUT="${OUT:-out}"
RAW="raw/generated"
WIDTH=1320
HEIGHT=2868

if [[ ! -x "$CHROME" ]]; then
  echo "Chrome was not found at '$CHROME'. Set CHROME to the Chrome executable path." >&2
  exit 1
fi

captures=(
  "01-home.png"
  "02-recipe-detail.png"
  "03-meals.png"
  "04-recipes-book.png"
  "05-inventory.png"
  "06-calendar.png"
  "07-chores.png"
  "08-household.png"
  "09-shopping.png"
)

missing=()
for capture in "${captures[@]}"; do
  [[ -s "$RAW/$capture" ]] || missing+=("$RAW/$capture")
done
if ((${#missing[@]})); then
  printf 'Missing app screenshots; run ./capture.sh first:\n' >&2
  printf '  %s\n' "${missing[@]}" >&2
  exit 1
fi

mkdir -p "$OUT"
rm -f "$OUT"/*.png
for slide in slides/*.html; do
  name="$(basename "$slide" .html)"
  output="$OUT/$name-iphone-6.9.png"
  echo "Rendering $name -> $output (${WIDTH}x${HEIGHT})"
  "$CHROME" --headless --disable-gpu --hide-scrollbars \
    --run-all-compositor-stages-before-draw --virtual-time-budget=1200 \
    "--window-size=${WIDTH},${HEIGHT}" --force-device-scale-factor=1 \
    "--screenshot=$output" "file://$(pwd)/$slide" >/dev/null 2>&1
  dimensions="$(sips -g pixelWidth -g pixelHeight "$output" | awk '/pixelWidth/ { width=$2 } /pixelHeight/ { height=$2 } END { print width "x" height }')"
  if [[ "$dimensions" != "${WIDTH}x${HEIGHT}" ]]; then
    echo "Unexpected output size for $output: $dimensions" >&2
    exit 1
  fi
done

echo "Rendered ${#captures[@]} app screenshots and one self-hosted slide in $OUT/"

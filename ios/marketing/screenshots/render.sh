#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

if (($# != 1)) || [[ ! "$1" =~ ^[0-9]+(\.[0-9]+)*$ ]]; then
  echo "Usage: ./render.sh <version> (for example: ./render.sh 1.3)" >&2
  exit 2
fi

CHROME="${CHROME:-/Applications/Google Chrome.app/Contents/MacOS/Google Chrome}"
VERSION="$1"
VERSION_DIRECTORY="$VERSION"
RAW_IPHONE="raw/generated"
RAW_IPAD="raw/generated-ipad"
IPHONE_WIDTH=1206
IPHONE_HEIGHT=2622
IPAD_WIDTH=2064
IPAD_HEIGHT=2752

if [[ ! -x "$CHROME" ]]; then
  echo "Chrome was not found at '$CHROME'. Set CHROME to the Chrome executable path." >&2
  exit 1
fi

captures=(
  "01-home"
  "02-recipe-detail"
  "03-meals"
  "04-recipes-book"
  "05-inventory"
  "06-calendar"
  "07-chores"
  "08-household"
  "09-shopping"
  "10-commute"
)

for capture in "${captures[@]}"; do
  if [[ ! -s "$RAW_IPHONE/$capture.png" ]]; then
    echo "Missing iPhone capture '$RAW_IPHONE/$capture.png'; run ./capture.sh first." >&2
    exit 1
  fi
  if [[ ! -s "$RAW_IPAD/$capture.png" ]]; then
    echo "Missing iPad capture '$RAW_IPAD/$capture.png'; run ./capture-ipad.sh first." >&2
    exit 1
  fi
done

TEMP_OUTPUT="$(mktemp -d "${TMPDIR:-/tmp}/family-hub-render.XXXXXX")"
trap 'rm -rf "$TEMP_OUTPUT"' EXIT

for capture in "${captures[@]}"; do
  slide="slides/$capture.html"
  iphone_output="$TEMP_OUTPUT/$capture-iphone-6.3.png"
  echo "Rendering $capture -> $VERSION_DIRECTORY/$(basename "$iphone_output") (${IPHONE_WIDTH}x${IPHONE_HEIGHT})"
  "$CHROME" --headless --disable-gpu --hide-scrollbars \
    --run-all-compositor-stages-before-draw --virtual-time-budget=1200 \
    "--window-size=${IPHONE_WIDTH},${IPHONE_HEIGHT}" --force-device-scale-factor=1 \
    "--screenshot=$iphone_output" "file://$(pwd)/$slide" >/dev/null 2>&1
  dimensions="$(sips -g pixelWidth -g pixelHeight "$iphone_output" | awk '/pixelWidth/ { width=$2 } /pixelHeight/ { height=$2 } END { print width "x" height }')"
  if [[ "$dimensions" != "${IPHONE_WIDTH}x${IPHONE_HEIGHT}" ]]; then
    echo "Unexpected output size for $iphone_output: $dimensions" >&2
    exit 1
  fi

  ipad_html="$TEMP_OUTPUT/$capture-ipad.html"
  ipad_output="$TEMP_OUTPUT/$capture-ipad-13.png"
  python3 render-ipad-slide.py \
    "$slide" slides/ipad-frame.html "$ipad_html" \
    "$RAW_IPAD/$capture.png" assets/ipad-frame.png
  echo "Rendering $capture -> $VERSION_DIRECTORY/$(basename "$ipad_output") (${IPAD_WIDTH}x${IPAD_HEIGHT})"
  "$CHROME" --headless --disable-gpu --hide-scrollbars \
    --run-all-compositor-stages-before-draw --virtual-time-budget=1200 \
    "--window-size=${IPAD_WIDTH},${IPAD_HEIGHT}" --force-device-scale-factor=1 \
    "--screenshot=$ipad_output" "file://$ipad_html" >/dev/null 2>&1
  dimensions="$(sips -g pixelWidth -g pixelHeight "$ipad_output" | awk '/pixelWidth/ { width=$2 } /pixelHeight/ { height=$2 } END { print width "x" height }')"
  if [[ "$dimensions" != "${IPAD_WIDTH}x${IPAD_HEIGHT}" ]]; then
    echo "Unexpected output size for $ipad_output: $dimensions" >&2
    exit 1
  fi
done

mkdir -p "$VERSION_DIRECTORY"
for capture in "${captures[@]}"; do
  iphone_output="$VERSION_DIRECTORY/$capture-iphone-6.3.png"
  ipad_output="$VERSION_DIRECTORY/$capture-ipad-13.png"
  cp "$TEMP_OUTPUT/$capture-iphone-6.3.png" "$iphone_output"
  cp "$TEMP_OUTPUT/$capture-ipad-13.png" "$ipad_output"

  rm -f "$VERSION_DIRECTORY/$capture-iphone-6.9.png"
  echo "Saved $iphone_output and $ipad_output"
done

echo "Updated ten iPhone and ten iPad screenshots in $VERSION_DIRECTORY/ for version $VERSION."

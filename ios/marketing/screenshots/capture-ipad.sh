#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
REPOSITORY="$(cd "$ROOT/../../.." && pwd)"
PROJECT="$REPOSITORY/ios/FamilyHub/FamilyHub.xcodeproj"
SCHEME="${SCHEME:-FamilyHub}"
OUTPUT="${RAW_IPAD:-$ROOT/raw/generated-ipad}"
APP_STORE_ORIENTATION="${APP_STORE_ORIENTATION:-portrait}"
TMP_ROOT="${TMPDIR:-/tmp}"
RESULT_BUNDLE="${RESULT_BUNDLE:-$TMP_ROOT/family-hub-ipad-marketing-$(date +%Y%m%d-%H%M%S).xcresult}"
DERIVED_DATA="${DERIVED_DATA:-$TMP_ROOT/family-hub-ipad-marketing-derived-data}"

if [[ -z "${SIMULATOR_UDID:-}" ]]; then
  if ! SIMULATOR_LIST="$(xcrun simctl list devices available -j 2>/dev/null)"; then
    echo "Could not query iPad simulators. Check CoreSimulatorService or set SIMULATOR_UDID explicitly." >&2
    exit 1
  fi
  SIMULATOR_UDID="$(printf '%s' "$SIMULATOR_LIST" | python3 -c '
import json, sys
try:
    simulator_data = json.load(sys.stdin)
except json.JSONDecodeError:
    raise SystemExit("Could not read simulator data. Check CoreSimulatorService.")
devices = [device for group in simulator_data["devices"].values() for device in group if device["isAvailable"] and "iPad" in device.get("deviceTypeIdentifier", "")]
choices = [device for device in devices if "iPad-Pro-13" in device.get("deviceTypeIdentifier", "") or "iPad-Pro-13" in device.get("name", "")]
if not choices:
    raise SystemExit("No available 13-inch iPad Pro simulator. Create one in Device Hub or install an iOS Simulator runtime.")
print(choices[0]["udid"])
')"
fi

SIMULATOR_INFO="$(xcrun simctl list devices available -j | SIMULATOR_UDID="$SIMULATOR_UDID" python3 -c '
import json, os, sys
simulator_data = json.load(sys.stdin)
devices = [device for group in simulator_data["devices"].values() for device in group]
device = next((device for device in devices if device["udid"] == os.environ["SIMULATOR_UDID"]), None)
if device is None or not ("iPad-Pro-13" in device.get("deviceTypeIdentifier", "") or "iPad-Pro-13" in device.get("name", "")):
    raise SystemExit("SIMULATOR_UDID must identify an available 13-inch iPad Pro simulator.")
print(device["name"])
')"

if [[ -e "$RESULT_BUNDLE" ]]; then
  echo "Result bundle already exists: $RESULT_BUNDLE" >&2
  echo "Set RESULT_BUNDLE to a new path and retry." >&2
  exit 1
fi

if [[ "$APP_STORE_ORIENTATION" != "landscape" && "$APP_STORE_ORIENTATION" != "portrait" ]]; then
  echo "APP_STORE_ORIENTATION must be landscape or portrait." >&2
  exit 1
fi

mkdir -p "$OUTPUT/landscape" "$OUTPUT/portrait" "$OUTPUT/app-store"
echo "Capturing iPad marketing screens on $SIMULATOR_INFO ($SIMULATOR_UDID)"
xcrun simctl boot "$SIMULATOR_UDID" 2>/dev/null || true
xcrun simctl bootstatus "$SIMULATOR_UDID" -b
xcrun simctl status_bar "$SIMULATOR_UDID" override \
  --time "9:41" --batteryState charged --batteryLevel 100 --wifiBars 3 --cellularBars 4
xcrun simctl ui "$SIMULATOR_UDID" appearance light

CLANG_MODULE_CACHE_PATH="$TMP_ROOT/family-hub-ipad-marketing-clang-cache" \
SWIFT_MODULE_CACHE_PATH="$TMP_ROOT/family-hub-ipad-marketing-swift-cache" \
xcodebuild test \
  -project "$PROJECT" \
  -scheme "$SCHEME" \
  -destination "platform=iOS Simulator,id=$SIMULATOR_UDID" \
  -derivedDataPath "$DERIVED_DATA" \
  -resultBundlePath "$RESULT_BUNDLE" \
  -parallel-testing-enabled NO \
  -only-testing:FamilyHubUITests/MarketingScreenshotsUITests/testCaptureMarketingScreensOnIPadLandscape \
  -only-testing:FamilyHubUITests/MarketingScreenshotsUITests/testCaptureMarketingScreensOnIPadPortrait \
  CODE_SIGNING_ALLOWED=NO

EXPORT_DIR="$(mktemp -d "$TMP_ROOT/family-hub-ipad-marketing-export.XXXXXX")"
trap 'rm -rf "$EXPORT_DIR"' EXIT
xcrun xcresulttool export attachments --path "$RESULT_BUNDLE" --output-path "$EXPORT_DIR"

python3 - "$EXPORT_DIR" "$OUTPUT" "$APP_STORE_ORIENTATION" <<'PY'
import json
import shutil
import sys
from pathlib import Path

export_dir = Path(sys.argv[1])
output_dir = Path(sys.argv[2])
app_store_orientation = sys.argv[3]
expected_screens = {
    "01-home", "02-recipe-detail", "03-meals", "04-recipes-book",
    "05-inventory", "06-calendar", "07-chores", "08-household",
    "09-shopping", "10-commute",
}
expected = {
    f"{screen}-ipad-{orientation}": (screen, orientation)
    for screen in expected_screens
    for orientation in ("landscape", "portrait")
}
manifest = json.loads((export_dir / "manifest.json").read_text())
found = {}

for test in manifest:
    for attachment in test.get("attachments", []):
        name = Path(attachment["suggestedHumanReadableName"]).stem
        capture_name = next(
            (candidate for candidate in expected if name == candidate or name.startswith(candidate + "_")),
            None,
        )
        if capture_name:
            found[capture_name] = export_dir / attachment["exportedFileName"]

missing = expected.keys() - found.keys()
if missing:
    raise SystemExit("Missing iPad screenshots: " + ", ".join(sorted(missing)))

for name, source in found.items():
    if not source.is_file() or source.read_bytes()[:8] != b"\x89PNG\r\n\x1a\n":
        raise SystemExit(f"Attachment is missing or is not a PNG: {name}")
    screen, orientation = expected[name]
    destination = output_dir / orientation / f"{screen}.png"
    shutil.copyfile(source, destination)
    import subprocess
    dimensions = subprocess.check_output(["sips", "-g", "pixelWidth", "-g", "pixelHeight", str(destination)], text=True)
    values = [int(line.split(": ")[1]) for line in dimensions.splitlines() if "pixelWidth:" in line or "pixelHeight:" in line]
    expected_dimensions = (2752, 2064) if orientation == "landscape" else (2064, 2752)
    if tuple(values) != expected_dimensions:
        raise SystemExit(f"Unexpected {orientation} screenshot size for {destination}: {values}; expected {expected_dimensions}")
    print(f"Saved {destination}")

for screen in sorted(expected_screens):
    capture_name = f"{screen}-ipad-{app_store_orientation}"
    source = found[capture_name]
    destination = output_dir / "app-store" / f"{screen}.png"
    shutil.copyfile(source, destination)
    print(f"Saved App Store screenshot {destination}")

cards = []
cards.append(f"<h2>App Store upload set ({app_store_orientation.title()}, 10 images)</h2><section>")
for screen in sorted(expected_screens):
    filename = f"app-store/{screen}.png"
    cards.append(
        f'<a href="{filename}"><img loading="lazy" src="{filename}" alt="{screen}">'
        f"<span>{screen}</span></a>"
    )
cards.append("</section>")
for orientation in ("landscape", "portrait"):
    cards.append(f"<h2>{orientation.title()}</h2><section>")
    for screen in sorted(expected_screens):
        filename = f"{orientation}/{screen}.png"
        cards.append(
            f'<a href="{filename}"><img loading="lazy" src="{filename}" alt="{screen}">'
            f"<span>{screen}</span></a>"
        )
    cards.append("</section>")

page = """<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Family Hub iPad screenshots</title>
  <style>
    :root { color-scheme: light dark; font: 16px/1.45 -apple-system, BlinkMacSystemFont, sans-serif; }
    body { margin: 0 auto; max-width: 1800px; padding: 24px; }
    section { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 420px), 1fr)); gap: 20px; }
    a { color: inherit; text-decoration: none; }
    img { display: block; width: 100%; height: auto; border: 1px solid #8886; border-radius: 12px; }
    span { display: block; margin: 8px 0 24px; font-size: .9rem; }
  </style>
</head>
<body>
  <h1>Family Hub iPad screenshots</h1>
  <p>Direct simulator captures from the marketing demo flow. Open an image for full resolution.</p>
  """ + "\n".join(cards) + """
</body>
</html>
"""
(output_dir / "index.html").write_text(page)
PY

echo "Capture complete. Source result bundle: $RESULT_BUNDLE"

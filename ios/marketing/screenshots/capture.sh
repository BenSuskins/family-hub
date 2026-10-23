#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
REPOSITORY="$(cd "$ROOT/../../.." && pwd)"
PROJECT="$REPOSITORY/ios/FamilyHub/FamilyHub.xcodeproj"
SCHEME="${SCHEME:-FamilyHub}"
RAW="$ROOT/raw/generated"
TMP_ROOT="${TMPDIR:-/tmp}"
RESULT_BUNDLE="${RESULT_BUNDLE:-$TMP_ROOT/family-hub-marketing-$(date +%Y%m%d-%H%M%S).xcresult}"
DERIVED_DATA="${DERIVED_DATA:-$TMP_ROOT/family-hub-marketing-derived-data}"

if [[ -z "${SIMULATOR_UDID:-}" ]]; then
  if ! SIMULATOR_LIST="$(xcrun simctl list devices available -j 2>/dev/null)"; then
    echo "Could not query iPhone simulators. Check that CoreSimulatorService is running, or set SIMULATOR_UDID explicitly." >&2
    exit 1
  fi
  SIMULATOR_UDID="$(printf '%s' "$SIMULATOR_LIST" | python3 -c '
import json, sys
try:
    simulator_data = json.load(sys.stdin)
except json.JSONDecodeError:
    raise SystemExit("Could not read simulator data. Check that CoreSimulatorService is running.")
devices = [device for group in simulator_data["devices"].values() for device in group if device["isAvailable"] and "iPhone" in device.get("deviceTypeIdentifier", "")]
preferred = [device for device in devices if "Pro-Max" in device.get("deviceTypeIdentifier", "")]
choices = preferred or devices
if not choices:
    raise SystemExit("No available iPhone simulator. Set SIMULATOR_UDID explicitly or install an iOS Simulator runtime.")
print(choices[0]["udid"])
')"
fi

if [[ -e "$RESULT_BUNDLE" ]]; then
  echo "Result bundle already exists: $RESULT_BUNDLE" >&2
  echo "Set RESULT_BUNDLE to a new path and retry." >&2
  exit 1
fi

mkdir -p "$RAW"
echo "Capturing marketing screens on simulator $SIMULATOR_UDID"
xcrun simctl boot "$SIMULATOR_UDID" 2>/dev/null || true
xcrun simctl bootstatus "$SIMULATOR_UDID" -b
xcrun simctl status_bar "$SIMULATOR_UDID" override \
  --time "9:41" --batteryState charged --batteryLevel 100 --wifiBars 3 --cellularBars 4
xcrun simctl ui "$SIMULATOR_UDID" appearance light

CLANG_MODULE_CACHE_PATH="$TMP_ROOT/family-hub-marketing-clang-cache" \
SWIFT_MODULE_CACHE_PATH="$TMP_ROOT/family-hub-marketing-swift-cache" \
xcodebuild test \
  -project "$PROJECT" \
  -scheme "$SCHEME" \
  -destination "platform=iOS Simulator,id=$SIMULATOR_UDID" \
  -derivedDataPath "$DERIVED_DATA" \
  -resultBundlePath "$RESULT_BUNDLE" \
  -parallel-testing-enabled NO \
  -only-testing:FamilyHubUITests/MarketingScreenshotsUITests/testCaptureMarketingScreens \
  CODE_SIGNING_ALLOWED=NO

EXPORT_DIR="$(mktemp -d "$TMP_ROOT/family-hub-marketing-export.XXXXXX")"
trap 'rm -rf "$EXPORT_DIR"' EXIT
xcrun xcresulttool export attachments --path "$RESULT_BUNDLE" --output-path "$EXPORT_DIR"

python3 - "$EXPORT_DIR" "$RAW" <<'PY'
import json
import shutil
import sys
from pathlib import Path

export_dir = Path(sys.argv[1])
output_dir = Path(sys.argv[2])
manifest_path = export_dir / "manifest.json"
manifest = json.loads(manifest_path.read_text())
expected = {
    "01-home", "02-recipe-detail", "03-meals", "04-recipes-book",
    "05-inventory", "06-calendar", "07-chores", "08-household", "09-shopping", "10-commute",
}
found = {}

for test in manifest:
    for attachment in test.get("attachments", []):
        suggested_name = Path(attachment["suggestedHumanReadableName"]).stem
        name = next((capture for capture in expected if suggested_name == capture or suggested_name.startswith(capture + "_")), None)
        if name:
            found[name] = export_dir / attachment["exportedFileName"]

missing = expected - found.keys()
if missing:
    raise SystemExit("Missing screenshot attachments: " + ", ".join(sorted(missing)))

for name, source in found.items():
    if not source.is_file() or source.read_bytes()[:8] != b"\x89PNG\r\n\x1a\n":
        raise SystemExit(f"Attachment is missing or is not a PNG: {name}")
    shutil.copyfile(source, output_dir / f"{name}.png")
    print(f"Saved {output_dir / (name + '.png')}")
PY

echo "Capture complete. Source result bundle: $RESULT_BUNDLE"

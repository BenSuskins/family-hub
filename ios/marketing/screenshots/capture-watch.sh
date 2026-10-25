#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIRECTORY="$(cd "$(dirname "$0")" && pwd)"
REPOSITORY_ROOT="$(cd "$SCRIPT_DIRECTORY/../../.." && pwd)"
PROJECT="$REPOSITORY_ROOT/ios/FamilyHub/FamilyHub.xcodeproj"
DERIVED_DATA="${DERIVED_DATA:-${TMPDIR:-/tmp}/family-hub-watch-marketing-derived}"
OUTPUT_DIRECTORY="$SCRIPT_DIRECTORY/raw/generated-watch"
DEVICE_TYPE="com.apple.CoreSimulator.SimDeviceType.Apple-Watch-Series-12-46mm"
RUNTIME_TYPE="com.apple.CoreSimulator.SimRuntime.watchOS-27-0"
DEVICE_NAME="Family Hub App Store Watch Series 12"
BUNDLE_IDENTIFIER="uk.co.suskins.FamilyHub.watchkitapp"

mkdir -p "$OUTPUT_DIRECTORY"

DEVICE_UDID="${WATCH_SIMULATOR_UDID:-}"
if [[ -z "$DEVICE_UDID" ]]; then
  DEVICE_UDID="$(xcrun simctl list devices available --json | python3 -c '
import json, sys
devices = json.load(sys.stdin)["devices"]
for runtime, entries in devices.items():
    if runtime.startswith("com.apple.CoreSimulator.SimRuntime.watchOS-27"):
        for device in entries:
            if "Apple Watch Series 12 (46mm)" in device["name"]:
                print(device["udid"])
                raise SystemExit
')"

  if [[ -z "$DEVICE_UDID" ]]; then
    DEVICE_UDID="$(xcrun simctl create "$DEVICE_NAME" "$DEVICE_TYPE" "$RUNTIME_TYPE")"
  fi
fi

DEVICE_DESCRIPTION="$(xcrun simctl list devices --json | python3 -c '
import json, sys
devices = json.load(sys.stdin)["devices"]
for entries in devices.values():
    for device in entries:
        if device["udid"] == sys.argv[1]:
            print(device["name"])
            raise SystemExit
' "$DEVICE_UDID")"
if [[ "$DEVICE_DESCRIPTION" != *"Series 12 (46mm)"* ]]; then
  echo "WATCH_SIMULATOR_UDID must identify an Apple Watch Series 12 (46mm). Found: $DEVICE_DESCRIPTION" >&2
  exit 1
fi

if ! xcrun simctl list devices | rg -q "$DEVICE_UDID.*Booted"; then
  xcrun simctl boot "$DEVICE_UDID"
fi
xcrun simctl bootstatus "$DEVICE_UDID" -b

xcodebuild \
  -quiet \
  -project "$PROJECT" \
  -scheme "FamilyHubWatch Watch App" \
  -destination "id=$DEVICE_UDID" \
  -derivedDataPath "$DERIVED_DATA" \
  build

APP_PATH="$DERIVED_DATA/Build/Products/Debug-watchsimulator/FamilyHubWatch Watch App.app"
if [[ ! -d "$APP_PATH" ]]; then
  echo "Watch app was not built at $APP_PATH" >&2
  exit 1
fi

xcrun simctl install "$DEVICE_UDID" "$APP_PATH"
xcrun simctl launch --terminate-running-process "$DEVICE_UDID" "$BUNDLE_IDENTIFIER" -marketing-screenshots
sleep 3
xcrun simctl io "$DEVICE_UDID" screenshot --type=png "$OUTPUT_DIRECTORY/01-watch-ingredients.png"

xcrun simctl launch --terminate-running-process "$DEVICE_UDID" "$BUNDLE_IDENTIFIER" -marketing-screenshots -watch-marketing-step
sleep 3
xcrun simctl io "$DEVICE_UDID" screenshot --type=png "$OUTPUT_DIRECTORY/02-watch-step.png"

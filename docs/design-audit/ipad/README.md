# Family Hub iPad screenshot audit

This capture set covers the main tabs and representative detail flows in
portrait and landscape. The UI test uses demo data and keeps each screenshot as
an XCTest attachment.

## Capture

Choose an available iPad simulator from `xcrun simctl list devices available`,
then run the iPad UI walkthrough:

```bash
xcodebuild \
  -project ios/FamilyHub/FamilyHub.xcodeproj \
  -scheme FamilyHub \
  -destination 'platform=iOS Simulator,id=<IPAD-SIMULATOR-UDID>' \
  -resultBundlePath /private/tmp/family-hub-ipad-audit.xcresult \
  -only-testing:FamilyHubUITests/DesignUITests \
  test
```

The test captures Today, profile and settings screens, Calendar, Chores, Food,
the meal plan, recipes and cooking, Household, shopping, and inventory forms.
It runs the normal walkthrough in both orientations and includes an
accessibility-text landscape pass.

Export the screenshots into an indexed HTML gallery:

```bash
python3 scripts/build_ipad_audit_gallery.py \
  /private/tmp/family-hub-ipad-audit.xcresult
```

The gallery and PNGs are written to a timestamped folder under
`docs/design-audit/ipad/`.

## Current capture status

No simulator captures are included yet. This machine's Xcode installation has
no `Simulator.app` or iOS simulator runtime, and CoreSimulator currently refuses
connections. The test and exporter are ready to run when an iPad simulator is
available.

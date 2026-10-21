# Complete the remaining Warm Sage screens

21 October 2026. This review covers all nine areas from the remaining-gap review after PR #55.

[Open the native comparison](index.html). [Read the screen checklist](screen-checklist.md). [Read the test evidence](checks.json).

## Changes

| Area | Result |
|---|---|
| Chores | The list starts with the Today entry. Detail groups the current task, repeat rule, and history. Forms put people before the schedule. Repeat opens as a child screen. Done returns to the draft. Save writes the chore. |
| Settings | The profile, this iPhone, family administration, and About use separate groups. Notifications distinguishes actual device permission from the family schedule. Server and family name use full-width commit actions. Home location groups the saved location and capture action. |
| Family messages | The composer has a labelled message, character count, notice, and Send action. Results open separately. Return preserves the draft. Notification help opens Notifications. Existing retry identifiers and accepted/failed counts remain. |
| API tokens | Rows show names and dates. Creation has a full-width action. The one-time result has a selectable value and Copy action. Saved rows open access-management details without exposing the token value. |
| Members and categories | Members show avatars, role badges, and the current user. Role selection stays local until Save and confirmation. Users cannot change their own role. Category rows open Rename and Delete directly. Delete requires confirmation. |
| Meal plan | Weekly rows are compact. Linked meals show the recipe thumbnail, with a book fallback when no image is available. Meals without a link stay text-only. The range stays Friday–Thursday. Today returns to the current week. Recipe search shows metadata. Selection dismisses the keyboard. Typing preserves the existing recipe link until an explicit choice. |
| Weather and events | Weather uses a centred hero and compact hourly rows. Event detail separates date, location, departure, and full details. The Calendar grids, controls, and gestures stay unchanged. |
| Shopping | Each checked item shows its own five-second countdown and Undo. A pending write keeps the item visible. Rename stays inline. Cancel sits beside the field. Done uses the keyboard toolbar. |
| Inventory forms | Count and Level use a segment at standard text sizes. Count has one editable control with stock buttons. Level has precise percentage entry. Thresholds and units remain. Large text uses a native mode menu. |

Demo role changes now survive a refresh. A simulator test exposed the previous Demo behaviour that reset roles after Save. The production API contract does not change.

## Deliberate native adaptations

The app uses native navigation, sheets, keyboards, menus, date pickers, and confirmation popovers. Their size and placement follow iOS. Native titles collapse on scroll. Text rows keep native insets to prevent the rounded row masks from cropping words. Large text stacks content and uses menus where segments do not fit.

The static mockups use a taller phone and fixed sample data. The app shows live dates, complete meal weeks, all 24 weather hours, existing categories, permissions, history, and advanced repeat fields. The app retains the arrival setting in Home location. Today and Calendar do not show the arrival margin.

Inventory keeps saved area colours. The app retains precise percentage entry as the replacement for the duplicate level slider. The iPad meal navigation retains its existing View week and Today actions. This review targets iPhone.

## Evidence boundaries

The comparison and checklist cover the changed screens and states. The [original audit](../README.md) remains the full app inventory. The [PR #54 comparison](../implementation/README.md) and [PR #55 follow-up](../fidelity/README.md) remain snapshots of those commits.

Demo mode supplies the visual data. The message result shows that no Demo iPhones can receive notifications. It does not prove delivery. Token screenshots contain a fixed Demo value. They do not contain credentials.

Simulator checks do not prove physical-device touch accuracy, VoiceOver speech, live sign-in, location capture, notification delivery, or Siri. Existing offline and permission evidence remains separate. This PR changes no server routes or database schema.

## Validation

| Check | Result |
|---|---|
| App unit suite | 142 passed, 42 skipped, no failures. |
| Selected small-iPhone regressions | 25 of 26 passed on the first run. The location test failed because its fast scroll moved the target under the header. |
| Household suite after the test fix | All three passed, including location denial, weather, and large text. All 26 selected checks pass across these runs. |
| Light capture routes | Eight passed on iPhone 17 Pro, including the large-text stock control. |
| Linked recipe follow-up | The meal flow checks the linked status, the recipe action, and link removal. A fresh screenshot shows the book fallback. |
| Dark capture route | Passed on iPhone 17 Pro. The simulator returns to Light appearance. |
| Comparison report | 33 pairs, 84 primary captures, and 56 small-iPhone captures. Local links, image files, and manifests pass validation. |
| Browser check | The screen filter works. The page has no horizontal overflow at widths of 390 and 1440 pixels. |

The [first regression summary](evidence/regression-first-run.json) retains the failed test. The [Household rerun](evidence/household-final.json) records its resolution. These results do not represent one full, passing UI-suite run. Read the [file validation](evidence/report-files.json) and [browser validation](evidence/report-browser.json) with the test summaries.

## Reproduce

Run `CompleteMockupsUITests` with the FamilyHub scheme. Run the relevant regression suites listed in [checks.json](checks.json).

For the optional visual review, copy [CompleteMockupsAuditUITests.swift](../source/CompleteMockupsAuditUITests.swift) into the UI test target. Run each light-screen method with the simulator in Light appearance. Run only `testDarkScreens` in Dark appearance. Restore Light appearance. Remove the copied file after the capture run.

Some simulator runs report a frame-dimension warning during navigation without a source location. The result summaries retain that warning. Earlier failed capture selectors and a zero-test run do not count as passing checks.

Export screenshots with `xcresulttool export attachments`. Run [export_native_captures.py](../source/export_native_captures.py) with the exported folder and `alignment/captures/`. Run [build_alignment_report.py](../source/build_alignment_report.py). The capture manifest records each test, device, and timestamp.

Run [collect_alignment_checks.py](../source/collect_alignment_checks.py) with `run-name=/absolute/path/to/results.xcresult` arguments. It rejects failed and empty passing runs. Use `--diagnostic=run-name=/absolute/path/to/results.xcresult` to retain a separate failed result. Run [validate_alignment_report.py](../source/validate_alignment_report.py) to check local links, manifests, and images.

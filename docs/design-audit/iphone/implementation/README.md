# Warm Sage Cosier: native app implementation

21 October 2026. [Open the screen comparison](index.html).

The iPhone app uses the approved theme and navigation. The comparison covers all 61 audit groups with native simulator captures. The captures include default text, the largest Dynamic Type setting, dark appearance, and display accessibility preferences.

## Applied design

- Cream surfaces, sage actions, clay details, rounded headings, and warm dark colours use shared app styles.
- The iPhone tab bar uses a floating glass surface. Accessibility text sizes use the native tab bar.
- Today shows the next departure, meals, the agenda, and chores, in that order.
- Food and Household use overview screens. Child screens use normal Back navigation.
- Shopping rename stays inline. Each purchase has an independent five-second Undo interval. The row hides after server confirmation.
- Purchased items remain on the server. The app does not delete them when it hides them.
- Shopping does not open a restock sheet. Inventory provides optional stock updates.
- Meal text changes preserve the recipe selection. A recipe result or Use name only changes that selection.
- Recipe forms, ingredient actions, cooking, chores, Inventory, Settings, and access screens use the audit recommendations.
- Cooking actions stack at large text sizes. Empty states stay inside their lists. Inventory labels keep room for full words.

Calendar keeps its grids, controls, gestures, and data behaviour. Its app tab bar uses the shared theme. Departure cards and event details show no arrival-margin text. The setting and departure calculation remain.

The theme also applies to iPad. Its existing navigation and layout remain. Watch, widget, and share-extension sources remain unchanged.

## Local validation

[Read the machine-readable results](checks.json). The evidence folder contains test summaries and selected logs.

| Check | Result |
|---|---|
| App tests | 141 pass; 42 existing contract tests skip |
| Native capture tour | 22 tests pass |
| Chore, meal, shopping, and ingredient regression | 19 tests pass |
| Inventory stock save | Pass; the UI changes the level to 80% |
| Largest-text tour | Pass |
| Largest-text layout regression | Pass |
| Native theme contrast | 22 selected text and surface pairs meet 4.5:1 in light and dark appearance |
| Display preferences | Reduce Motion, Reduce Transparency, and Increase Contrast report enabled in the native test |
| iPad navigation | Pass |
| Shared package | 116 tests; one skip; no failures |
| Shopping HTTP checks | Two local Go packages pass |
| Watch simulator build | Pass |
| Final app and test build | Pass; the temporary capture harness is absent |
| Comparison report | 61 groups; 374 local references; filter and phone layout pass |

The first full UI run has four failing chore and meal checks. Corrected focused runs pass. Screenshot review also finds large-text defects that the original tour does not detect. The new regression test checks their replacements. The evidence retains the earlier failed results. A later whole-suite green run is not claimed.

No server interface or database changes occur. The unrelated Xcode project, Watch scheme, Info.plist, and Xcode Cloud changes remain intact. No commit, push, archive, deployment, or release occurs.

## Evidence limits

- The captures use Demo data. Dates and device heights differ from the approved static mockups.
- iPhone SE and iPhone 17 Pro simulator captures provide phone-size coverage.
- Calendar captures also use its deterministic layout fixtures.
- Native controls retain system behaviour. Their shape can differ from the HTML approximation.
- System-owned sign-in and a completed photo selection remain outside this simulator proof.
- Real notification delivery, physical touch accuracy, VoiceOver speech and focus, and installed-device Siri remain release checks.
- Offline and failed-write behaviour has model tests. A complete forced-offline screen matrix is not captured.
- Workboard tools are unavailable in this session. No Workboard update is posted.

## Repeat the checks

Run the app and UI tests with an available iPhone simulator:

```sh
xcodebuild test -project ios/FamilyHub/FamilyHub.xcodeproj -scheme FamilyHub -destination 'platform=iOS Simulator,id=<iPhone UDID>' CODE_SIGNING_ALLOWED=NO
swift test --package-path ios/FamilyHubKit
```

The normal UI target retains the behaviour and accessibility regression tests. The full capture harness stays in `../source/NativeRedesignAuditUITests.swift`. Copy it into the UI test target for an audit session. Remove that copy after the session. Use a dedicated simulator for its display-preference check.

Export attachments with `xcresulttool`. Use a Python interpreter with Pillow to build the contact sheets:

```sh
python3 docs/design-audit/iphone/source/export_native_captures.py <attachment export> docs/design-audit/iphone/implementation/<profile>
python3 docs/design-audit/iphone/source/build_native_report.py
```

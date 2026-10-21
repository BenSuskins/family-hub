# Warm Sage Cosier: design fidelity follow-up

This folder is an earlier snapshot. [Review the latest remaining-screen alignment](../alignment/README.md).

21 October 2026. This follow-up closes the four largest differences identified after PR #54. It also fixes the floating iPhone tab bar.

[Open the native comparison](index.html). [Read the original audit](../README.md). [Review the PR #54 baseline](../implementation/README.md).

## Changes

| Area | Current behaviour |
|---|---|
| Tab bar | The glass capsule overlays scroll content. Its outer margins have no background. Scroll content keeps enough bottom space for the final row. The bar hides for the keyboard. |
| Welcome | A large family heading, a labelled server field, and prominent Connect and Demo actions replace the grouped form. Errors retain the address. |
| Setup help | Join steps appear before the admin setup instructions. Back to Welcome returns to the entered address. |
| Recipe editor | Ingredients, Method, and Source open as native child screens. The root shows short summaries. Draft edits survive Back or Done. Only Save writes the recipe. |
| Recipes | Search stays visible at the top. A New recipe action also appears after the list. |
| Cooking | Each step has a large heading, a progress bar, and a next-step preview. The final step has explicit final-step text. Buttons and swipes use the same page state. Only the current page appears to accessibility tools. |
| Inventory | Soft icon backgrounds retain the saved area colour. Items use separate cards. The area filter uses a segment at standard text sizes. Each card labels its shopping action. Empty areas show a clear add action. |
| Large text | Inventory uses a menu instead of a cramped segment. Shopping and stock actions stack. Cooking controls stack. The native tab bar remains. |

No Calendar source, server contract, or database changes occur. The app retains native sheets, keyboards, photo selection, navigation, and Back gestures.

## Differences at PR #55

This pass does not claim complete visual parity across all 61 audit groups.

- Chore setup and recurrence retain the form structure described in the earlier gap review.
- Several Settings screens retain inline titles and toolbar commit actions. The mockups use more large headings and full-width actions.
- Family message results and token results retain grouped native layouts.
- Shopping rename and Undo use the approved behaviour. Their exact layout differs, and the five-second hint is absent.
- Inventory retains saved area colours, the toolbar Add item action, and native stock controls.
- Native controls, typography, and adaptive layouts differ from the HTML approximation. Calendar follows the separate decision to keep its current design.

The previous implementation comparison remains a PR #54 snapshot. The screenshots in this folder document this follow-up. The Demo data, dates, and phone height differ from the static proposals.

## Validation

| Check | Result |
|---|---|
| Final follow-up flows | Six tests pass |
| App unit tests | 141 pass; 42 existing contract tests skip |
| Existing navigation, Household, and redesign regressions | Six tests pass |
| Native evidence | 33 captures; 18 focused comparisons |
| Report validation | Local links, PNG integrity, filter, and phone layout pass |

Earlier combined runs retain failures. The corrected final follow-up run passes. A later complete whole-suite green run is not claimed.

[Read the test evidence](checks.json). Test summaries and selected logs are in `evidence/`. The capture manifest records each test, device, and timestamp.

Physical-device VoiceOver, touch accuracy, live sign-in, notification delivery, and installed-device Siri remain release checks. The follow-up does not claim a new complete offline or system-permission capture matrix.

Run `DesignFidelityUITests` with the FamilyHub scheme and an available iPhone simulator. Export attachments with `xcresulttool`. Use `source/export_native_captures.py` to update `fidelity/captures/`, then run `source/build_fidelity_report.py`.

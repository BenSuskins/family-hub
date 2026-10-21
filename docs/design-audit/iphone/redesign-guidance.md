# Revised iPhone design

20 October 2026 · Static proposals · Warm Sage Cosier selected

The redesign uses the two skills requested by the user:

- [Apple Design by Emil Kowalski](https://github.com/emilkowalski/skills/blob/main/skills/apple-design/SKILL.md).
- [UI/UX Pro Max](https://github.com/nextlevelbuilder/ui-ux-pro-max-skill), including its SwiftUI search data and app review checklist.

These are third-party design guides. The audit retains its separate links to Apple’s official iOS 26 guidance.

## Selected theme: Warm Sage Cosier

[Compare three visual directions](style-options.html): Quiet neutral, Warm sage, and Cool indigo. Each covers five key screens and a dark Today sample. The user selects Warm Sage Cosier for the full mockup set. Use warm cream surfaces, sage actions, clay details, and rounded headings. [Review the earlier refinement comparison](cosy-sage.html). The original versions remain available as references. Calendar remains unchanged apart from the agreed removal of arrival-margin text.

## What changes

All current proposed screens use Warm Sage Cosier. The source package contains 68 images, including five superseded Calendar drafts that keep their earlier appearance. The report marks Calendar as Keep current. The simulator captures stay unchanged. The recommendations include the user’s revised priorities.

| Area | Applied design rule | Visible result |
|---|---|---|
| Navigation | Use material to separate controls from content | A restrained floating tab bar and small toolbar controls; solid content groups |
| Typography | Use the system font with size-specific weight, tracking, and line height | Clear titles, readable rows, and distinct supporting text |
| Today | Make the next task clear through order and emphasis | A near-departure card, the meal plan, then chores; meal planning takes priority over chores |
| Stock | Keep a control near the item it affects | Each item has its own edit row, shopping action, and stock controls |
| Forms | Emphasize one commit action | A clear Save control; duplicate Save buttons are removed |
| Cooking | Provide a clear next action and a return route | A full-width Next step button, large instruction text, and a visible Ingredients action |
| Feedback | State the result in words | Stock confirmation, a zero-recipient message result, and errors beside their fields |
| Accessibility | Adapt the layout instead of shrinking the text | Stacked large-text layouts, explicit status text, and theme-specific colours |

Recipes and Inventory remain at the top of their tabs. Meal weeks remain Friday–Thursday. Ingredient checklist and shopping actions remain independent.

## Chosen navigation: overview screens

The user selects option 3, the overview screen. Food shows Meal plan and Recipes navigation rows, with today’s meals below. Household shows Shopping list and Inventory navigation rows, with low-stock items below. Each destination opens with a normal Back action. Remove the inner navigation segments from those destinations. [Compare the earlier alternatives](navigation-options.html). Today departures, Calendar, and event proposals omit arrival-margin text. Show the departure time and travel duration.

## Shopping purchase behaviour

- Show no Purchased section or clear-purchases action.
- On a check, mark the row complete immediately. Keep its original position for five seconds.
- Show Undo beside the checked item. Undo or a second check restores the unchecked row and cancels its removal.
- After five seconds, hide that row. Give each checked item its own removal deadline and Undo action.
- If the purchase fails, restore the item and show an error. Do not report a successful purchase before the server confirms it.
- Keep stock updates optional through Inventory. Do not open a stock form automatically when the user checks an item.

The static states show [the five-second Undo interval](mockups/shopping-pending.png), [the list after removal](mockups/shopping-hidden.png), and [the Undo result](mockups/shopping-undone.png). They do not demonstrate a running timer.

## Inline shopping rename

Rename activates a text field in the existing shopping row and opens the keyboard. Done saves a non-empty name. Cancel restores the original name. Keep the row in place and retain the entered text if the save fails. Put the error beside the field. Do not open a rename screen or sheet.

[View inline rename with the keyboard](mockups/shopping-rename.png). The keyboard is a static approximation; use the system keyboard in the app.

## Calendar decision

Keep the current Calendar design and interactions. Only remove arrival-margin text. Earlier Calendar layout mockups remain in the source package as superseded drafts; the comparison report no longer recommends them.

## Meal name and recipe search

Remove the linked-recipe warning. Typing filters the existing recipe search and preserves the current selection. Selecting another result changes the recipe. Selecting Use name only makes the meal a custom name with no recipe. Save applies the selected choice. Typing alone does not remove a recipe link.

## Visual rules

- Use SwiftUI semantic fonts in the app. The static source approximates a 34-point title, 22-point section title, and 17-point row.
- Use a 20-point phone gutter and a 4-point spacing scale.
- Use solid grouped surfaces. Reserve glass for navigation and transient controls.
- Use sage for actions and muted clay for small decorative details. Use red for destructive actions and overdue status.
- Pair status colours with words or symbols. Use text for the low-stock threshold, including its inclusive boundary.
- Give custom action controls at least a 44-point target. Use native calendar and picker controls.
- Use native SF Symbols in the app. The static source uses a consistent vector approximation.
- Place the tab bar and fixed actions above the home indicator. Add scroll insets so the final item remains reachable.

## Interaction specification for implementation

Static images do not demonstrate these behaviours. Use native SwiftUI controls and transitions when implementing them.

| Interaction | Required behaviour |
|---|---|
| Press | Show the pressed state on touch-down. Commit on release. Permit cancellation when the finger leaves the target. |
| Navigation | Preserve the source and return path. Keep the system back gesture. |
| Sheet | Use a native sheet with standard dismissal. Do not block input while a transition runs. |
| Direct manipulation | Track the finger continuously. Preserve velocity when a gesture ends. |
| Custom motion | Start from the current visible value. Use a non-bouncy spring by default; add bounce only when momentum explains it. |
| Completion | Pair the visible result with a restrained native haptic where useful. Do not imply completion before the action succeeds. |
| Reduce Motion | Replace large movement with a short cross-fade or an immediate state change. Keep the result clear. |
| Reduce Transparency | Use solid navigation surfaces. |
| Increase Contrast | Strengthen text and control boundaries without relying on colour alone. |

## Search decisions

The first UI/UX Pro Max design-system query was `family productivity native minimal`. It returned AI-product and marketing patterns that do not fit this app. A narrower query, `productivity task management`, returned a useful flat mobile style, but still included marketing sections. Those sections, custom fonts, and generic palettes are excluded.

The applied results come from the SwiftUI query `navigation forms dynamic type` and the UX queries `touch target spacing` and `spring reduced motion`. They support native forms, scalable layouts, visible controls, adequate target size, and reduced motion. Platform fit takes priority over a generic style recommendation.

## Verification and limits

The renderer exports all 68 proposals at 402 × 874 points. A separate check tests all proposals at 375 × 812 points for horizontal overflow and navigation overlap. The check measures selected text and surface contrast pairs in both appearances. It also checks the CSS alternatives for reduced transparency and increased contrast.

[Read the static check results](evidence/redesign-checks.json).

These checks validate the static source. They do not prove native Dynamic Type, VoiceOver, physical touch accuracy, motion, landscape behaviour, or production data. The large-text proposals show layout intent. Keep those native checks in the implementation acceptance criteria. iPad stays outside this work.

[Open the revised comparisons](index.html) · [Read the original audit](README.md) · [Read the capture limits](coverage.md)

# Family Hub design

This guide covers the iOS app and paired wall display. The web client serves
only the QR-paired wall display. The source
tokens live in [`AppTheme.swift`](../ios/FamilyHub/FamilyHub/Components/AppTheme.swift)
and [`globals.css`](../web/src/app/globals.css). Update those files and this
guide together when the visual system changes.

## Product structure

- Keep the iOS app's five top-level areas: Today, Calendar, Chores, Food, and
  Household. Keep a separate navigation stack for each tab.
- Keep Food and Household as overviews with direct links to their features.
- Let admins pair, configure, and disconnect wall displays from the iOS app.
- Keep the web client to two states: pairing before approval and the shared
  household display after approval. Do not add member web navigation or kiosk
  settings.
- Use the wall display layout from the [interactive handoff](prototypes/wall-display.html).

## Color palette

Use these light appearance values as the shared Warm Sage base. The iOS app
also defines semantic colors for dark appearance. The web display defines its
dark token set in `globals.css`.

| Role | Light value | Notes |
|------|-------------|-------|
| Paper | `#f7f1e7` | Main page background |
| Surface | `#fffcf6` | Cards and grouped content |
| Ink | `#343c31` | Primary text |
| Secondary text | `#646357` iOS; `#777568` web | Supporting text |
| Accent | `#3c654c` | Primary actions and links |
| Border | `#e6dfd2` | Dividers and outlines |
| Sage | `#e6eddd` | Secondary emphasis and meal planning |
| Clay | `#8b503c` | Chores, warnings, and warm emphasis |
| Gold | `#b88642` web | Focus outline and highlight |
| Destructive | `#9d3c34` iOS | Destructive actions |

The iOS dark tokens are paper `#201f1a`, surface `#2d3028`, ink `#f7f0e2`,
secondary `#c3beaf`, accent `#bad0ab`, border `#494a3e`, sage `#3e4a37`,
clay `#e0baa0`, and destructive `#f0b6a5`. The web dark tokens are paper
`#1e241e`, surface `#292f28`, ink `#eee9dc`, muted `#b1ae9f`, accent
`#a6c49b`, line `#41483d`, sage `#343d32`, clay `#d59a7e`, and gold
`#d7b36f`.

Keep user calendar colors intact. Use a label, icon, or shape with color so
color is never the only status cue. Keep text contrast at WCAG AA levels.

## Typography

- Use the iOS system font and SF Symbols. The large navigation title uses the
  rounded system design and scales with Dynamic Type.
- Use the system rounded sans-serif stack for web text and controls, matching
  the wall-display handoff.
- Keep labels short and readable. Let user-facing text scale or reflow where
  the platform supports it.

## Layout and interaction

- Use warm, quiet surfaces and simple rounded cards. Keep information easy to
  scan.
- Give the wall display larger type and generous spacing for across-room
  reading. Keep its app launcher along the bottom edge.
- Make cards fully clickable only when they represent one action. Give each
  control its own target when a card contains multiple actions.
- Keep interactive targets at least 44 points or pixels in their platform's
  touch layouts. Show visible keyboard focus on the web.
- Support Dynamic Type, VoiceOver, increased contrast, reduced transparency,
  and reduced motion on iOS. Respect reduced motion on the web.
- Show loading, empty, unavailable, offline, saving, and failure states where
  the feature needs them. Do not show unknown data as an empty result.

## iOS screen decisions

### Today

- Show meal planning before chores. Keep agenda actions easy to reach.
- Show commute information after the meal plan during the user's selected
  display ranges.
- Open Settings from the user's profile photo on Today.
- Keep departure cards focused on leave time and travel duration.

### Calendar

- Keep Year, Month, Day, and Week views and their current navigation and
  gestures.
- Keep Week's seven-day timeline. On iPhone, show all-day items in one compact
  row and open the full list in a sheet.
- Preserve user calendar colors. Use sage for meals and clay for chores.

### Food and Household

- Open Food to Meal Plan and Recipes. Keep the meal week Friday through
  Thursday, and show a recipe symbol when a meal links to a recipe.
- Use one meal-name field for text and recipe search. Preserve the recipe link
  when a user selects a recipe.
- Open Household to Shopping List and Inventory. Hide purchased items after
  five seconds and offer Undo during that period.
- Keep inventory restock actions linked to their source shopping item.

### Widgets

- Lead with the next timed event and dinner. Keep chore counts secondary unless
  the user chooses a Chores widget.
- Keep widget actions direct. Report pending and failed saves.
- Use absolute event dates for ordering and skip all-day events when selecting
  “Up next.” Apply WidgetKit privacy redaction to event and meal titles.

## Wall display meal timing

- Show breakfast as the next meal before 11:00, lunch from 11:00, and dinner
  from 16:00 to 18:00 in the household's local time.
- From 20:30, preview tomorrow's plan and mark breakfast as next.
- Label the next slot in text, even when it has no planned meal. Open the meal
  editor at the selected date and meal type when a slot is tapped.

## References

- [Apple Human Interface Guidelines: Tab bars](https://developer.apple.com/design/human-interface-guidelines/tab-bars)
- [Apple Human Interface Guidelines: Materials](https://developer.apple.com/design/human-interface-guidelines/materials)
- [Apple Human Interface Guidelines: Color](https://developer.apple.com/design/human-interface-guidelines/color)
- [Apple Human Interface Guidelines: Buttons](https://developer.apple.com/design/human-interface-guidelines/buttons)
- [Apple Human Interface Guidelines: Accessibility](https://developer.apple.com/design/human-interface-guidelines/accessibility)

# Family Hub iOS design

Use this guide for iPhone app and widget work. Keep the current iPad navigation and Calendar structure unless the task changes them.

See the [iPhone audit](docs/design-audit/iphone/index.html) and the [Calendar and widget proposals](docs/design-audit/iphone/calendar-widgets/index.html) for screen-level examples.

## Product structure

- Keep five top-level areas: Today, Calendar, Chores, Food, and Household.
- Give each tab its own navigation stack. Switching tabs preserves each stack.
- On iPhone, tapping the selected Food or Household tab again returns to that overview.
- Keep Food and Household as overview screens with direct links. Do not add inner segmented tabs.
- On Today, place meal planning before chores. Show departure information when an event needs it.

## Visual design

- Use the Warm Sage Cosier palette from [AppTheme.swift](ios/FamilyHub/FamilyHub/Components/AppTheme.swift).
- Use warm cream for the canvas, sage for primary actions and meal planning, and clay for chores or warnings.
- Keep event-calendar colors supplied by the user's calendars. Pair color with a label or symbol.
- Use system typography, SF Symbols, grouped lists, and native SwiftUI controls.
- Use adaptive colors for light and dark appearances. Check increased contrast and reduced transparency.
- Let the iOS 26 tab bar float above scrolling content. Keep content visible behind it and provide bottom scroll space.
- Use Liquid Glass for system navigation controls. Keep content cards simple and warm.

## Screen decisions

### Today

- Make the meal plan the first main section. Keep chores and agenda actions easy to reach.
- Keep departure cards focused on the leave time and travel duration. Do not show an arrival margin.

### Calendar

- Keep Year, Month, Day, and Week views, their current navigation, and their gestures.
- Keep Week's seven-day timeline. On iPhone, show all-day items in one compact row. Open the full list in a sheet.
- Preserve user calendar colors. Use sage for meals and clay for chores.
- Do not show an arrival margin in calendar details.

### Food

- Open to a clear overview with Meal plan and Recipes links.
- Keep the meal week Friday through Thursday. Show a recipe symbol when a meal links to a recipe.
- Use one meal-name field for text and recipe search. Do not add a separate recipe-search field or a warning that repeats the field's behavior.
- Preserve the recipe link when the user selects a recipe. Treat new free text as an unlinked meal.
- Put one primary “Start cooking” button below the recipe image. Keep Share, Cook on Watch, Edit, and Delete in the top-right actions menu.
- Do not duplicate the primary cooking action in the navigation bar.

### Household

- Open to an overview with Shopping list and Inventory links.
- Keep purchased items out of the list. After a purchase, hide the item after five seconds and offer Undo during that period.
- Rename shopping items inline. Keep inventory restock actions linked to their source item.

### Widgets

- Lead with the next timed event and dinner. Keep chore counts secondary unless the user chooses a Chores widget.
- Keep widget actions separate and direct. Report pending and failed saves. Remove a chore only after the server confirms completion.
- Use absolute event dates for ordering. Skip all-day events when selecting “Up next.”
- Apply WidgetKit privacy redaction to event and meal titles.

## Interaction and access

- Keep interactive areas at least 44 points high or wide.
- Support Dynamic Type. Reflow dense rows and show fewer items at accessibility text sizes.
- Use clear empty, loading, unavailable, pending, and failed states. Do not present unknown data as an empty result.
- Respect Reduce Motion, Reduce Transparency, VoiceOver, and increased contrast.
- Keep color secondary to labels, symbols, and shape. Do not use color as the only status cue.
- Keep navigation and action labels direct. Use the action the control performs, such as “Meal plan” or “Undo.”

## References

- [Apple Human Interface Guidelines: Tab bars](https://developer.apple.com/design/human-interface-guidelines/tab-bars)
- [Apple Human Interface Guidelines: Materials](https://developer.apple.com/design/human-interface-guidelines/materials)
- [Apple Human Interface Guidelines: Color](https://developer.apple.com/design/human-interface-guidelines/color)
- [Apple Human Interface Guidelines: Buttons](https://developer.apple.com/design/human-interface-guidelines/buttons)
- [Apple Human Interface Guidelines: Accessibility](https://developer.apple.com/design/human-interface-guidelines/accessibility)

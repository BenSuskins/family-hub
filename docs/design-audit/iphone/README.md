# Family Hub iPhone design audit

20 October 2026 · Expert review · iPhone only

61 screen/state reviews · 308 simulator captures · 68 static mockups

The app uses a sound native foundation. Most screens use familiar lists, forms, sheets, and system navigation. The largest problems concern priority, discoverability, and clear action labels. A blanket glass redesign does not address those problems.

The most useful changes are: keep the meal plan above chores on Today and bring near departures into view; expose Recipes and Inventory earlier; make Start cooking obvious; clarify chore setup versus completion; and improve notifications and role management.

[Open the visual comparison report](index.html). Each comparison uses real simulator captures and separate, labelled proposal images. [Browse all mockups](mockups/manifest.json).

[Review the latest screen alignment and validation](alignment/index.html). [Review the PR #54 baseline](implementation/index.html).

[Review the new Calendar and widget proposals](calendar-widgets/index.html). These retain the Calendar layouts and apply Warm Sage Cosier. The widget designs start fresh.

Revision 2 uses the requested Apple Design and UI/UX Pro Max skills. [Read the redesign rules and interaction specification](redesign-guidance.md).

## Evidence and limits

- Checkout: `e7d4310501b450e37d3f5af89558f303f0715427`, plus existing local Xcode project, scheme, and Info.plist changes. Those files remain untouched by this audit.
- Runtime: iOS 26.5. Primary device: iPhone 17 Pro. Small device: iPhone SE (3rd generation). Xcode: 27.0.
- Data: built-in demo household. Edits, purchases, token creation, and messages stay in the demo client. No household server data changes.
- Light profile: standard text, light appearance, Increase Contrast off. Dark baseline: dark appearance; other inherited preferences were not recorded before that run.
- The capture register lists each captured state and its test source. The coverage notes list missing states. Source inspection supports those limits; it does not replace a simulator capture.
- This is an expert review, not a user study. It does not prove VoiceOver usability, physical-device touch accuracy, notification delivery, or production behavior.
- Mockups show proposed layouts. HTML/CSS approximates system controls and glass. Use native SwiftUI components during implementation. Static images do not demonstrate motion or accessibility behavior. Mockup content is illustrative demo data; clock and forecast values can differ from the captures.

## Priority

| Priority | Meaning |
|---|---|
| P1 | Changes the user’s ability to find, understand, or complete a main task |
| P2 | Improves clarity, consistency, or recovery |
| Keep | Keep the current native interaction; only a small change is justified |

## Apple iOS 26 assessment

The app already adopts system navigation, a floating tab bar, and native sheets. Keep that base. Apple separates the navigation/control layer from content and advises restraint with custom glass. Opaque content groups are appropriate. More transparency is not a success criterion.

Use consistent destination names, readable content, familiar controls, and clear feedback. Aim for Apple’s 44-point default control size on custom touch controls. Review contrast in both appearances. Keep Dynamic Type, Reduce Motion, Increase Contrast, and Reduce Transparency support.

Sources: [Adopting Liquid Glass](https://developer.apple.com/documentation/technologyoverviews/adopting-liquid-glass), [Accessibility](https://developer.apple.com/design/human-interface-guidelines/accessibility), [Tab bars](https://developer.apple.com/design/human-interface-guidelines/tab-bars), [Build a SwiftUI app with the new design](https://developer.apple.com/videos/play/wwdc2025/323/). Retrieved 20 October 2026. The report paraphrases the guidance.

## Navigation proposal

[Compare the current and proposed navigation maps](navigation.md).

Keep five tabs: Today, Calendar, Chores, Food, Household. Rename Tasks to Chores. Keep task completion on Today and chore rules under Chores. Add direct links between these contexts.

Use overview screens for Food and Household. Show Meal plan and Recipes as navigation rows in Food, followed by today’s meals. Show Shopping list and Inventory as navigation rows in Household, followed by low-stock items. Each destination opens with a normal Back action. Remove the inner navigation segments.

Keep Settings as a separate native sheet. Use the same name for its entry label and destination. Group this iPhone’s preferences separately from family-wide administration.

## Screen register


| Screen | Priority | Captures | Mockup |
|---|---|---|---|
| [Welcome / server entry](#welcome) | P1 | [iphone-light: access-welcome](captures/iphone-light/access-welcome.png) · [iphone-light: access-server-keyboard](captures/iphone-light/access-server-keyboard.png) | [welcome](mockups/welcome.png) |
| [Setup help](#setup-help) | P1 | [iphone-light: access-setup-help](captures/iphone-light/access-setup-help.png) · [iphone-light: access-setup-help-bottom](captures/iphone-light/access-setup-help-bottom.png) | [setup-help](mockups/setup-help.png) |
| [Invalid server address](#server-error) | P2 | [iphone-repairs: settings-server-validation](captures/iphone-repairs/settings-server-validation.png) · [iphone-light: access-invalid-server](captures/iphone-light/access-invalid-server.png) | [server-error](mockups/server-error.png) |
| [Today](#today) | P1 | [iphone-final: today-overview](captures/iphone-final/today-overview.png) · [iphone-final: today-bottom](captures/iphone-final/today-bottom.png) | [today](mockups/today.png) · [today-dark](mockups/today-dark.png) · [today-large](mockups/today-large.png) |
| [Weather detail](#weather) | P2 | [iphone-final: today-weather](captures/iphone-final/today-weather.png) · [iphone-final: today-weather-bottom](captures/iphone-final/today-weather-bottom.png) | [weather](mockups/weather.png) |
| [Event detail](#event) | P1 | [iphone-final: today-event](captures/iphone-final/today-event.png) · [iphone-repairs: today-event](captures/iphone-repairs/today-event.png) | [event](mockups/event.png) |
| [Today chore menu](#chore-actions) | P2 | [iphone-final: today-chore-actions](captures/iphone-final/today-chore-actions.png) · [iphone-repairs: today-chore-actions](captures/iphone-repairs/today-chore-actions.png) | [chore-actions](mockups/chore-actions.png) |
| [Calendar: month and zoom](#calendar-month) | Keep | [iphone-light: calendar-month-details](captures/iphone-light/calendar-month-details.png) · [iphone-light: calendar-month-zoom](captures/iphone-light/calendar-month-zoom.png) | Keep the system screen |
| [Calendar: day and event overlap](#calendar-day) | Keep | [iphone-light: calendar-day-overlap](captures/iphone-light/calendar-day-overlap.png) · [iphone-light: calendar-day](captures/iphone-light/calendar-day.png) | Keep the system screen |
| [Calendar: week](#calendar-week) | Keep | [iphone-light: calendar-week](captures/iphone-light/calendar-week.png) | Keep the system screen |
| [Calendar: year](#calendar-year) | Keep | [iphone-light: calendar-year](captures/iphone-light/calendar-year.png) | Keep the system screen |
| [Calendar: accessibility text](#calendar-large) | Keep | [iphone-small: calendar-accessibility](captures/iphone-small/calendar-accessibility.png) | Keep the system screen |
| [Chore setup list](#chores) | P1 | [iphone-repairs: tasks-list](captures/iphone-repairs/tasks-list.png) · [iphone-light: tasks-list](captures/iphone-light/tasks-list.png) | [chores](mockups/chores.png) |
| [Chore setup detail and history](#chore-detail) | P1 | [iphone-repairs: tasks-detail](captures/iphone-repairs/tasks-detail.png) · [iphone-light: tasks-detail](captures/iphone-light/tasks-detail.png) | [chore-detail](mockups/chore-detail.png) |
| [New chore](#chore-create) | P1 | [iphone-light: tasks-create](captures/iphone-light/tasks-create.png) · [iphone-light: tasks-create-keyboard](captures/iphone-light/tasks-create-keyboard.png) | [chore-create](mockups/chore-create.png) |
| [Edit chore and repeat rule](#chore-edit) | P1 | [iphone-repairs: tasks-edit](captures/iphone-repairs/tasks-edit.png) · [iphone-repairs: tasks-edit-schedule](captures/iphone-repairs/tasks-edit-schedule.png) | [chore-edit](mockups/chore-edit.png) · [chore-repeat](mockups/chore-repeat.png) |
| [Deletion confirmations](#delete-confirmation) | P2 | [iphone-final: household-item-delete-confirmation](captures/iphone-final/household-item-delete-confirmation.png) · [iphone-repairs: tasks-delete-confirmation](captures/iphone-repairs/tasks-delete-confirmation.png) | [delete-confirmation](mockups/delete-confirmation.png) |
| [Food / daily meal plan](#food) | P1 | [iphone-light: food-overview](captures/iphone-light/food-overview.png) · [iphone-small: meal-plan](captures/iphone-small/meal-plan.png) | [food](mockups/food.png) · [meal-plan](mockups/meal-plan.png) |
| [Full week meal plan](#meal-week) | P2 | [iphone-light: food-week](captures/iphone-light/food-week.png) · [iphone-light: food-week-bottom](captures/iphone-light/food-week-bottom.png) | [meal-week](mockups/meal-week.png) |
| [Add meal / recipe selection](#meal-add) | P2 | [iphone-light: food-add-meal](captures/iphone-light/food-add-meal.png) · [iphone-light: food-meal-search](captures/iphone-light/food-meal-search.png) | [meal-add](mockups/meal-add.png) |
| [Edit meal](#meal-edit) | P1 | [iphone-light: food-edit-meal](captures/iphone-light/food-edit-meal.png) · [iphone-small-extra: food-edit-meal](captures/iphone-small-extra/food-edit-meal.png) | [meal-edit](mockups/meal-edit.png) |
| [Recipe list](#recipes) | P2 | [iphone-repairs: food-recipes](captures/iphone-repairs/food-recipes.png) · [iphone-light: food-recipes](captures/iphone-light/food-recipes.png) | [recipes](mockups/recipes.png) |
| [Recipe search: no results](#recipes-empty) | P2 | [iphone-repairs: food-recipes-no-results](captures/iphone-repairs/food-recipes-no-results.png) · [iphone-small-extra: food-recipes-no-results](captures/iphone-small-extra/food-recipes-no-results.png) | [recipes-empty](mockups/recipes-empty.png) |
| [Recipe detail, ingredient actions, and method](#recipe-detail) | P1 | [iphone-light: food-recipe-detail](captures/iphone-light/food-recipe-detail.png) · [iphone-light: food-recipe-ingredients](captures/iphone-light/food-recipe-ingredients.png) | [recipe-detail](mockups/recipe-detail.png) |
| [Recipe action menu](#recipe-actions) | Keep | [iphone-light: food-recipe-actions](captures/iphone-light/food-recipe-actions.png) · [iphone-small-extra: food-recipe-actions](captures/iphone-small-extra/food-recipe-actions.png) | [recipe-detail](mockups/recipe-detail.png) |
| [New recipe](#recipe-create) | P1 | [iphone-light: food-recipe-create](captures/iphone-light/food-recipe-create.png) · [iphone-light: food-recipe-create-details](captures/iphone-light/food-recipe-create-details.png) | [recipe-create](mockups/recipe-create.png) · [recipe-content-edit](mockups/recipe-content-edit.png) |
| [Edit recipe](#recipe-edit) | P1 | [iphone-light: food-recipe-edit](captures/iphone-light/food-recipe-edit.png) · [iphone-light: food-recipe-edit-details](captures/iphone-light/food-recipe-edit-details.png) | [recipe-edit](mockups/recipe-edit.png) · [recipe-content-edit](mockups/recipe-content-edit.png) |
| [Cook mode: ingredients](#cook-ingredients) | P1 | [iphone-repairs: food-cook-ingredients](captures/iphone-repairs/food-cook-ingredients.png) · [iphone-light: food-cook-ingredients](captures/iphone-light/food-cook-ingredients.png) | [cook-ingredients](mockups/cook-ingredients.png) |
| [Cook mode: method step](#cook-step) | P1 | [iphone-repairs: food-cook-step](captures/iphone-repairs/food-cook-step.png) · [iphone-light: food-cook-step](captures/iphone-light/food-cook-step.png) | [cook-step](mockups/cook-step.png) |
| [Cook mode: final step](#cook-finish) | P1 | [iphone-repairs: food-cook-final-step](captures/iphone-repairs/food-cook-final-step.png) · [iphone-light: food-cook-final-step](captures/iphone-light/food-cook-final-step.png) | [cook-finish](mockups/cook-finish.png) |
| [Shopping list and low-stock suggestions](#household) | P1 | [iphone-repairs: household-shopping](captures/iphone-repairs/household-shopping.png) · [iphone-light: household-shopping](captures/iphone-light/household-shopping.png) | [household](mockups/household.png) · [shopping-list](mockups/shopping-list.png) |
| [Shopping item menu](#shopping-actions) | Keep | [iphone-repairs: household-shopping-actions](captures/iphone-repairs/household-shopping-actions.png) · [iphone-light: household-shopping-actions](captures/iphone-light/household-shopping-actions.png) | [household](mockups/household.png) |
| [Rename shopping item](#shopping-rename) | P2 | [iphone-repairs: household-shopping-rename](captures/iphone-repairs/household-shopping-rename.png) · [iphone-light: household-shopping-rename](captures/iphone-light/household-shopping-rename.png) | [shopping-rename](mockups/shopping-rename.png) |
| [Purchase feedback and Undo](#shopping-purchased) | P2 | [iphone-repairs: household-shopping-purchased](captures/iphone-repairs/household-shopping-purchased.png) · [iphone-repairs: household-shopping-restocked](captures/iphone-repairs/household-shopping-restocked.png) | [shopping-pending](mockups/shopping-pending.png) · [shopping-hidden](mockups/shopping-hidden.png) · [shopping-undone](mockups/shopping-undone.png) · [restocked](mockups/restocked.png) |
| [Restock after purchase](#restock) | P2 | [iphone-repairs: household-shopping-restock](captures/iphone-repairs/household-shopping-restock.png) · [iphone-repairs: household-shopping-restock-keyboard](captures/iphone-repairs/household-shopping-restock-keyboard.png) | [restock](mockups/restock.png) |
| [Inventory overview](#inventory) | P1 | [iphone-final: household-inventory](captures/iphone-final/household-inventory.png) · [iphone-repairs: household-inventory](captures/iphone-repairs/household-inventory.png) | [inventory](mockups/inventory.png) |
| [Inventory area](#area) | P1 | [iphone-final: household-area](captures/iphone-final/household-area.png) · [iphone-repairs: household-area](captures/iphone-repairs/household-area.png) | [area](mockups/area.png) |
| [Inventory area menu](#area-actions) | P2 | [iphone-light: household-area-actions](captures/iphone-light/household-area-actions.png) · [iphone-small-extra: household-area-actions](captures/iphone-small-extra/household-area-actions.png) | [area](mockups/area.png) |
| [New inventory area](#area-create) | Keep | [iphone-light: household-area-create](captures/iphone-light/household-area-create.png) · [iphone-small: area-form](captures/iphone-small/area-form.png) | [area-create](mockups/area-create.png) |
| [Edit inventory area](#area-edit) | Keep | [iphone-light: household-area-edit](captures/iphone-light/household-area-edit.png) · [iphone-small-extra: household-area-edit](captures/iphone-small-extra/household-area-edit.png) | [area-edit](mockups/area-edit.png) |
| [New inventory item: count](#item-create) | P2 | [iphone-final: household-item-create-count](captures/iphone-final/household-item-create-count.png) · [iphone-final: household-item-create-stock](captures/iphone-final/household-item-create-stock.png) | [item-create](mockups/item-create.png) |
| [Edit inventory item: level](#item-edit-level) | P2 | [iphone-final: household-item-edit-level](captures/iphone-final/household-item-edit-level.png) · [iphone-repairs: household-item-edit-level](captures/iphone-repairs/household-item-edit-level.png) | [item-edit-level](mockups/item-edit-level.png) |
| [Settings and profile](#settings) | P1 | [iphone-light: settings-profile](captures/iphone-light/settings-profile.png) · [iphone-small: settings](captures/iphone-small/settings.png) | [settings](mockups/settings.png) |
| [System photo picker](#photo-picker) | Keep | [iphone-light: settings-photo-picker](captures/iphone-light/settings-photo-picker.png) · [iphone-small-extra: settings-photo-picker](captures/iphone-small-extra/settings-photo-picker.png) | Keep the system screen |
| [Server configuration](#server-config) | P1 | [iphone-repairs: settings-server-configuration](captures/iphone-repairs/settings-server-configuration.png) · [iphone-small-extra: settings-server-configuration](captures/iphone-small-extra/settings-server-configuration.png) | [server-config](mockups/server-config.png) |
| [Chore reminders and device permission](#reminders) | P1 | [iphone-light: settings-reminders](captures/iphone-light/settings-reminders.png) · [iphone-light: settings-reminders-bottom](captures/iphone-light/settings-reminders-bottom.png) | [reminders](mockups/reminders.png) |
| [Family broadcast: compose](#broadcast) | P2 | [iphone-light: settings-broadcast](captures/iphone-light/settings-broadcast.png) · [iphone-light: settings-broadcast-keyboard](captures/iphone-light/settings-broadcast-keyboard.png) | [broadcast](mockups/broadcast.png) |
| [Family broadcast: result](#broadcast-result) | P1 | [iphone-light: settings-broadcast-result](captures/iphone-light/settings-broadcast-result.png) · [iphone-small-extra: settings-broadcast-result](captures/iphone-small-extra/settings-broadcast-result.png) | [broadcast-result](mockups/broadcast-result.png) |
| [Home information and location](#home-info) | P2 | [iphone-final: settings-home-info](captures/iphone-final/settings-home-info.png) · [iphone-final: settings-home-info-bottom](captures/iphone-final/settings-home-info-bottom.png) | [home-info](mockups/home-info.png) |
| [Family name](#family-name) | Keep | [iphone-final: settings-family-name](captures/iphone-final/settings-family-name.png) · [iphone-final: settings-family-name-keyboard](captures/iphone-final/settings-family-name-keyboard.png) | [family-name](mockups/family-name.png) |
| [Family members and role actions](#members) | P1 | [iphone-final: settings-family-members](captures/iphone-final/settings-family-members.png) · [iphone-final: settings-member-actions](captures/iphone-final/settings-member-actions.png) | [members](mockups/members.png) · [member-role](mockups/member-role.png) |
| [Category list and actions](#categories) | P2 | [iphone-light: settings-categories](captures/iphone-light/settings-categories.png) · [iphone-light: settings-category-actions](captures/iphone-light/settings-category-actions.png) | [categories](mockups/categories.png) |
| [New category](#category-create) | Keep | [iphone-light: settings-category-create](captures/iphone-light/settings-category-create.png) · [iphone-small-extra: settings-category-create](captures/iphone-small-extra/settings-category-create.png) | [category-create](mockups/category-create.png) |
| [Rename category](#category-edit) | Keep | [iphone-light: settings-category-rename](captures/iphone-light/settings-category-rename.png) · [iphone-small-extra: settings-category-rename](captures/iphone-small-extra/settings-category-rename.png) | [category-edit](mockups/category-edit.png) |
| [API token list and empty state](#tokens) | P2 | [iphone-final: settings-tokens-populated](captures/iphone-final/settings-tokens-populated.png) · [iphone-final: settings-tokens](captures/iphone-final/settings-tokens.png) | [tokens](mockups/tokens.png) |
| [Create API token](#token-create) | P2 | [iphone-final: settings-token-create](captures/iphone-final/settings-token-create.png) · [iphone-light: settings-token-create](captures/iphone-light/settings-token-create.png) | [token-create](mockups/token-create.png) |
| [Token created and copied](#token-created) | P2 | [iphone-final: settings-token-created](captures/iphone-final/settings-token-created.png) · [iphone-final: settings-token-copied](captures/iphone-final/settings-token-copied.png) | [token-created](mockups/token-created.png) |
| [Calendar display menu](#calendar-view-menu) | Keep | [iphone-supplement: calendar-view-menu](captures/iphone-supplement/calendar-view-menu.png) | Keep the system screen |
| [Chore recurrence variants](#chore-repeat-options) | P2 | [iphone-supplement: tasks-repeat-menu](captures/iphone-supplement/tasks-repeat-menu.png) · [iphone-supplement: tasks-repeat-monthly](captures/iphone-supplement/tasks-repeat-monthly.png) | [chore-repeat](mockups/chore-repeat.png) · [chore-repeat-monthly](mockups/chore-repeat-monthly.png) · [chore-repeat-custom](mockups/chore-repeat-custom.png) |
| [Edit inventory item: count](#item-edit-count) | P2 | [iphone-final: household-item-edit-count](captures/iphone-final/household-item-edit-count.png) | [item-edit-count](mockups/item-edit-count.png) |
| [Inventory area: empty](#area-empty) | Keep | [iphone-final: household-area-empty](captures/iphone-final/household-area-empty.png) | [area-empty](mockups/area-empty.png) |

<a id="welcome"></a>

## Welcome / server entry

**Goal:** Connect to the family server or try the app.

**User friendly:** The short form is easy to scan. It assumes that a family member knows what a server URL means.

**Interactions:** Connect remains disabled until the field has text. The URL keyboard suits the field.

**Main information and action:** Connect is available, but the help for obtaining an address deserves a clearer position.

**iOS 26:** The native form and controls fit iOS. The first screen needs a clearer household purpose.

**Proposal (P1):** Explain the family benefit. Ask the user to obtain the address from their family admin. Keep Try Demo secondary.

**Evidence:** [iphone-light / access-welcome](captures/iphone-light/access-welcome.png), [iphone-light / access-server-keyboard](captures/iphone-light/access-server-keyboard.png), [iphone-small / login](captures/iphone-small/login.png), [iphone-small-extra / access-welcome](captures/iphone-small-extra/access-welcome.png), [iphone-small-extra / access-server-keyboard](captures/iphone-small-extra/access-server-keyboard.png), [iphone-dark / login](captures/iphone-dark/login.png)

**Mockups:** [welcome](mockups/welcome.png)

<a id="setup-help"></a>

## Setup help

**Goal:** Understand how to join a family or set up a server.

**User friendly:** The help starts with Docker, OIDC, PKCE, and environment variables. This suits a server admin, but burdens family members.

**Interactions:** Done provides a clear exit. The repository link provides full instructions.

**Main information and action:** A family member needs the server address and an account before deployment instructions.

**iOS 26:** The sheet is native. Progressive disclosure better separates common and advanced help.

**Proposal (P1):** Lead with three join steps. Keep server deployment in an admin section and retain the full guide.

**Evidence:** [iphone-light / access-setup-help](captures/iphone-light/access-setup-help.png), [iphone-light / access-setup-help-bottom](captures/iphone-light/access-setup-help-bottom.png), [iphone-small-extra / access-setup-help](captures/iphone-small-extra/access-setup-help.png), [iphone-small-extra / access-setup-help-bottom](captures/iphone-small-extra/access-setup-help-bottom.png)

**Mockups:** [setup-help](mockups/setup-help.png)

<a id="server-error"></a>

## Invalid server address

**Goal:** Correct an invalid connection address.

**User friendly:** The form keeps the address and shows an inline error. This supports recovery.

**Interactions:** The user can edit and retry. Avoid dismissing the keyboard or replacing the whole form.

**Main information and action:** The error needs to identify the required address format next to the field.

**iOS 26:** Inline feedback and native text entry fit the platform.

**Proposal (P2):** Keep the error beside a persistent label. Use a specific correction and retain the entered value.

**Evidence:** [iphone-repairs / settings-server-validation](captures/iphone-repairs/settings-server-validation.png), [iphone-light / access-invalid-server](captures/iphone-light/access-invalid-server.png), [iphone-small-extra / access-invalid-server](captures/iphone-small-extra/access-invalid-server.png), [iphone-small-extra / settings-server-validation](captures/iphone-small-extra/settings-server-validation.png)

**Mockups:** [server-error](mockups/server-error.png)

<a id="today"></a>

## Today

**Goal:** Know what needs attention now and complete household work.

**User friendly:** Agenda, meals, chores, and departures share the page. The current order puts meals before overdue chores and departures last.

**Interactions:** Chore completion, the chore action menu, weather, events, and meals provide direct entry points.

**Main information and action:** The meal plan takes priority over chores, as confirmed by the user. Keep meal actions above the chore list.

**iOS 26:** Native navigation and grouped content suit iOS 26. Content priority needs more work than the materials.

**Proposal (P1):** Put a near departure first, then the meal plan, the agenda, and overdue and today’s chores. Keep weather compact and omit empty departure cards.

**Evidence:** [iphone-final / today-overview](captures/iphone-final/today-overview.png), [iphone-final / today-bottom](captures/iphone-final/today-bottom.png), [iphone-repairs / today-overview](captures/iphone-repairs/today-overview.png), [iphone-repairs / today-bottom](captures/iphone-repairs/today-bottom.png), [iphone-light / today-overview](captures/iphone-light/today-overview.png), [iphone-small / today](captures/iphone-small/today.png), [iphone-small-extra / today-overview](captures/iphone-small-extra/today-overview.png), [iphone-small-extra / today-bottom](captures/iphone-small-extra/today-bottom.png), [iphone-contrast / variant-today](captures/iphone-contrast/variant-today.png), [iphone-dark / today](captures/iphone-dark/today.png)

**Mockups:** [today](mockups/today.png), [today-dark](mockups/today-dark.png), [today-large](mockups/today-large.png)

<a id="weather"></a>

## Weather detail

**Goal:** Read local conditions and the forecast.

**User friendly:** The detail screen keeps weather out of Today. Each hourly row repeats the full date, which slows the scan.

**Interactions:** The header weather control opens the detail screen. Back returns to Today.

**Main information and action:** Current conditions, the forecast, and data freshness should appear in that order.

**iOS 26:** Native navigation and semantic type provide a suitable base.

**Proposal (P2):** Keep the current conditions prominent. Use shorter hourly time labels. Retain all 24 hourly forecasts, weather advice, freshness, and Open-Meteo attribution.

**Evidence:** [iphone-final / today-weather](captures/iphone-final/today-weather.png), [iphone-final / today-weather-bottom](captures/iphone-final/today-weather-bottom.png), [iphone-repairs / today-weather](captures/iphone-repairs/today-weather.png), [iphone-repairs / today-weather-bottom](captures/iphone-repairs/today-weather-bottom.png), [iphone-light / today-weather](captures/iphone-light/today-weather.png), [iphone-light / today-weather-bottom](captures/iphone-light/today-weather-bottom.png), [iphone-small-extra / today-weather](captures/iphone-small-extra/today-weather.png), [iphone-small-extra / today-weather-bottom](captures/iphone-small-extra/today-weather-bottom.png)

**Mockups:** [weather](mockups/weather.png)

<a id="event"></a>

## Event detail

**Goal:** Inspect an event and know when to leave.

**User friendly:** The title, date, location, description, and departure are present. Departure follows the description, which can push it down.

**Interactions:** The event opens from Today or Calendar. The same detail should serve both routes.

**Main information and action:** Group the start time, location, and departure before a long description.

**iOS 26:** The native detail view fits iOS. Better grouping improves the reading order.

**Proposal (P1):** Put date and location together. Put the departure immediately below them. Keep the full description below.

**Evidence:** [iphone-final / today-event](captures/iphone-final/today-event.png), [iphone-repairs / today-event](captures/iphone-repairs/today-event.png), [iphone-light / today-event](captures/iphone-light/today-event.png), [iphone-small-extra / today-event](captures/iphone-small-extra/today-event.png)

**Mockups:** [event](mockups/event.png)

<a id="chore-actions"></a>

## Today chore menu

**Goal:** Skip a particular task without changing its repeat rule.

**User friendly:** A menu keeps secondary actions away from the completion control. The affected chore needs clear context.

**Interactions:** The ellipsis opens the menu. An outside tap dismisses the native popover when no Cancel control appears.

**Main information and action:** Complete remains the direct action. Skip needs a short explanation of its effect.

**iOS 26:** The native menu is appropriate. Do not replace it with a custom floating panel.

**Proposal (P2):** Include the chore name and assignee. Explain Skip. Add a route to the chore setup.

**Evidence:** [iphone-final / today-chore-actions](captures/iphone-final/today-chore-actions.png), [iphone-repairs / today-chore-actions](captures/iphone-repairs/today-chore-actions.png), [iphone-light / today-chore-actions](captures/iphone-light/today-chore-actions.png), [iphone-small-extra / today-chore-actions](captures/iphone-small-extra/today-chore-actions.png)

**Mockups:** [chore-actions](mockups/chore-actions.png)

<a id="calendar-month"></a>

## Calendar: month and zoom

**Goal:** Browse dates and inspect the selected day.

**User friendly:** The grid provides useful month context. Narrow date columns truncate event titles even when the rows are tall.

**Interactions:** Month scroll, pinch zoom, date selection, and Today work in the captured interaction tests.

**Main information and action:** A readable selected-day list helps users inspect events without guessing a truncated title.

**iOS 26:** The system toolbar and floating tab bar fit iOS 26. The content should remain the focus.

**Proposal (Keep):** Keep the current Calendar design and interactions, as selected by the user. Only remove arrival-margin text. Earlier layout proposals are superseded.

**Evidence:** [iphone-light / calendar-month-details](captures/iphone-light/calendar-month-details.png), [iphone-light / calendar-month-zoom](captures/iphone-light/calendar-month-zoom.png), [iphone-light / calendar-month](captures/iphone-light/calendar-month.png), [iphone-small / calendar](captures/iphone-small/calendar.png), [iphone-contrast / variant-calendar](captures/iphone-contrast/variant-calendar.png), [iphone-dark / calendar](captures/iphone-dark/calendar.png)

<a id="calendar-day"></a>

## Calendar: day and event overlap

**Goal:** Inspect the daily schedule and open an event.

**User friendly:** The timeline communicates time and separates overlapping events. Text competes for width in simultaneous blocks.

**Interactions:** Tapping an event opens details. Swipe changes the day. Today restores the current date.

**Main information and action:** The date, all-day items, and full event names need to remain readable.

**iOS 26:** The native toolbar and direct timeline navigation follow the platform. Avoid adding glass behind event content.

**Proposal (Keep):** Keep the current Calendar design and interactions, as selected by the user. Only remove arrival-margin text. Earlier layout proposals are superseded.

**Evidence:** [iphone-light / calendar-day-overlap](captures/iphone-light/calendar-day-overlap.png), [iphone-light / calendar-day](captures/iphone-light/calendar-day.png)

<a id="calendar-week"></a>

## Calendar: week

**Goal:** Compare the schedule across seven days.

**User friendly:** The week capture shows event titles broken into one-letter lines in seven narrow columns. This prevents useful inspection.

**Interactions:** The view menu selects Week. Select a day or event to inspect details.

**Main information and action:** The overview needs a readable path to the full event name.

**iOS 26:** A week overview is appropriate, but the phone width requires an inspection layer.

**Proposal (Keep):** Keep the current Calendar design and interactions, as selected by the user. Only remove arrival-margin text. Earlier layout proposals are superseded.

**Evidence:** [iphone-light / calendar-week](captures/iphone-light/calendar-week.png)

<a id="calendar-year"></a>

## Calendar: year

**Goal:** Move to a month in another part of the year.

**User friendly:** The compact month grids suit broad date navigation. Keep selection obvious.

**Interactions:** Year opens a month. Today restores the current period.

**Main information and action:** The year and month entry points are the main information and actions.

**iOS 26:** The year-to-month hierarchy matches familiar calendar navigation.

**Proposal (Keep):** Keep the current Calendar design and interactions, as selected by the user. Only remove arrival-margin text. Earlier layout proposals are superseded.

**Evidence:** [iphone-light / calendar-year](captures/iphone-light/calendar-year.png)

<a id="calendar-large"></a>

## Calendar: accessibility text

**Goal:** Inspect the schedule with large text.

**User friendly:** The accessibility route uses an agenda and date picker instead of forcing large text into a month grid.

**Interactions:** The date picker and view menu keep date navigation available.

**Main information and action:** Full event labels and the selected date matter more than a dense overview.

**iOS 26:** This is a good adaptive pattern. Preserve it when changing the standard calendar layout.

**Proposal (Keep):** Keep the current Calendar design and interactions, as selected by the user. Only remove arrival-margin text. Earlier layout proposals are superseded.

**Evidence:** [iphone-small / calendar-accessibility](captures/iphone-small/calendar-accessibility.png)

<a id="chores"></a>

## Chore setup list

**Goal:** Create and manage chore rules.

**User friendly:** The screen title says Chores while the tab says Tasks. At maximum text size, Show inactive chores consumes much of the first viewport.

**Interactions:** Rows open setup details. The plus control creates a chore. Inactive chores can be shown.

**Main information and action:** Creation and repeat rules are available. Users also need a direct route to chores due today.

**iOS 26:** The list and toolbar are native. Consistent destination names improve wayfinding.

**Proposal (P1):** Rename the tab Chores. Move Show inactive below the active rules. Keep completion on Today and add a direct Today link.

**Evidence:** [iphone-repairs / tasks-list](captures/iphone-repairs/tasks-list.png), [iphone-light / tasks-list](captures/iphone-light/tasks-list.png), [iphone-small / chores](captures/iphone-small/chores.png), [iphone-small-extra / tasks-list](captures/iphone-small-extra/tasks-list.png), [iphone-contrast / variant-tasks](captures/iphone-contrast/variant-tasks.png), [iphone-dark / chores](captures/iphone-dark/chores.png)

**Mockups:** [chores](mockups/chores.png)

<a id="chore-detail"></a>

## Chore setup detail and history

**Goal:** Inspect the repeat rule, current task, and history.

**User friendly:** The setup, current task, and history are clear groups. “Complete or skip this task on Home” names a destination that is actually Today.

**Interactions:** Edit and delete are below the history. The completion instruction has no direct link.

**Main information and action:** The current task deserves a higher position and a direct route to its action.

**iOS 26:** Native list sections suit the detail. Place the frequent edit action in the toolbar.

**Proposal (P1):** Put the current task first. Replace Home with Today and make the instruction a link. Keep history and deletion.

**Evidence:** [iphone-repairs / tasks-detail](captures/iphone-repairs/tasks-detail.png), [iphone-light / tasks-detail](captures/iphone-light/tasks-detail.png), [iphone-small / chore-detail](captures/iphone-small/chore-detail.png), [iphone-small-extra / tasks-detail](captures/iphone-small-extra/tasks-detail.png), [iphone-dark / chore-detail](captures/iphone-dark/chore-detail.png)

**Mockups:** [chore-detail](mockups/chore-detail.png)

<a id="chore-create"></a>

## New chore

**Goal:** Create a chore and assign its schedule.

**User friendly:** The form includes the required capabilities, but it mixes details, rotating assignees, dates, and advanced repeat options in one long form.

**Interactions:** Save stays in the toolbar. Selecting no assignees means everyone; the existing footer explains this.

**Main information and action:** The name, first due date, and eligible people should lead. Advanced repeat settings can follow.

**iOS 26:** The native form and sheet are suitable. Plain labels and progressive disclosure improve the interaction.

**Proposal (P1):** Use “Due date” and “Due time” labels. Show a repeat summary and keep advanced conditions in a repeat section.

**Evidence:** [iphone-light / tasks-create](captures/iphone-light/tasks-create.png), [iphone-light / tasks-create-keyboard](captures/iphone-light/tasks-create-keyboard.png), [iphone-light / tasks-create-schedule](captures/iphone-light/tasks-create-schedule.png), [iphone-small / chore-form](captures/iphone-small/chore-form.png), [iphone-small-extra / tasks-create](captures/iphone-small-extra/tasks-create.png), [iphone-small-extra / tasks-create-keyboard](captures/iphone-small-extra/tasks-create-keyboard.png), [iphone-small-extra / tasks-create-schedule](captures/iphone-small-extra/tasks-create-schedule.png), [iphone-contrast / variant-chore-form](captures/iphone-contrast/variant-chore-form.png), [iphone-dark / chore-form](captures/iphone-dark/chore-form.png)

**Mockups:** [chore-create](mockups/chore-create.png)

<a id="chore-edit"></a>

## Edit chore and repeat rule

**Goal:** Change a chore rule without changing the outstanding task.

**User friendly:** The existing explanation preserves an important distinction: rule changes affect the next task.

**Interactions:** The form retains current task ownership and dates. The weekday controls are compact for touch.

**Main information and action:** Make the current task constraint and resulting repeat rule easy to read.

**iOS 26:** The native form is appropriate. Aim for 44-point default touch targets on custom weekday controls.

**Proposal (P1):** Keep the current-task explanation. Use full-size weekday controls and a readable repeat summary.

**Evidence:** [iphone-repairs / tasks-edit](captures/iphone-repairs/tasks-edit.png), [iphone-repairs / tasks-edit-schedule](captures/iphone-repairs/tasks-edit-schedule.png), [iphone-repairs / tasks-edit-repeat](captures/iphone-repairs/tasks-edit-repeat.png), [iphone-repairs / tasks-edit-end](captures/iphone-repairs/tasks-edit-end.png), [iphone-light / tasks-edit](captures/iphone-light/tasks-edit.png), [iphone-light / tasks-edit-schedule](captures/iphone-light/tasks-edit-schedule.png), [iphone-light / tasks-edit-repeat](captures/iphone-light/tasks-edit-repeat.png), [iphone-light / tasks-edit-end](captures/iphone-light/tasks-edit-end.png), [iphone-small-extra / tasks-edit](captures/iphone-small-extra/tasks-edit.png), [iphone-small-extra / tasks-edit-schedule](captures/iphone-small-extra/tasks-edit-schedule.png), [iphone-small-extra / tasks-edit-repeat](captures/iphone-small-extra/tasks-edit-repeat.png), [iphone-small-extra / tasks-edit-end](captures/iphone-small-extra/tasks-edit-end.png)

**Mockups:** [chore-edit](mockups/chore-edit.png), [chore-repeat](mockups/chore-repeat.png)

<a id="delete-confirmation"></a>

## Deletion confirmations

**Goal:** Confirm a destructive change with enough context.

**User friendly:** The app uses native confirmation dialogs. Naming the item and the effect reduces mistakes.

**Interactions:** Delete confirms the action. Cancel or an outside tap dismisses the dialog, depending on presentation.

**Main information and action:** The user needs to know exactly what disappears and whether history remains.

**iOS 26:** Native confirmation dialogs fit iOS 26. Preserve their system appearance and roles.

**Proposal (P2):** Name the item in the title. Explain the effect, including area contents or retained chore history.

**Evidence:** [iphone-final / household-item-delete-confirmation](captures/iphone-final/household-item-delete-confirmation.png), [iphone-repairs / tasks-delete-confirmation](captures/iphone-repairs/tasks-delete-confirmation.png), [iphone-repairs / household-item-delete-confirmation](captures/iphone-repairs/household-item-delete-confirmation.png), [iphone-repairs / household-shopping-clear-confirmation](captures/iphone-repairs/household-shopping-clear-confirmation.png), [iphone-light / household-item-delete-confirmation](captures/iphone-light/household-item-delete-confirmation.png), [iphone-light / household-shopping-clear-confirmation](captures/iphone-light/household-shopping-clear-confirmation.png), [iphone-small-final / household-shopping-clear-confirmation](captures/iphone-small-final/household-shopping-clear-confirmation.png), [iphone-small-extra / tasks-delete-confirmation](captures/iphone-small-extra/tasks-delete-confirmation.png), [iphone-small-extra / household-item-delete-confirmation](captures/iphone-small-extra/household-item-delete-confirmation.png), [iphone-small-extra / household-shopping-clear-confirmation](captures/iphone-small-extra/household-shopping-clear-confirmation.png), [iphone-small-extra / household-item-delete-popover-persisted-1](captures/iphone-small-extra/household-item-delete-popover-persisted-1.png), [iphone-small-extra / household-item-delete-popover-persisted-2](captures/iphone-small-extra/household-item-delete-popover-persisted-2.png)

**Mockups:** [delete-confirmation](mockups/delete-confirmation.png)

<a id="food"></a>

## Food / daily meal plan

**Goal:** Read the selected day’s meals and reach recipes.

**User friendly:** The date strip and meal rows are clear. Large empty slots push Recipes down. At maximum text size, the header delays access to the first meal.

**Interactions:** The planned meal row opens Edit. A separate View recipe action opens the recipe.

**Main information and action:** Users often want to read or cook a planned recipe. Opening it should be the main planned-meal action.

**iOS 26:** Native lists and date selection fit iOS. Keep navigation predictable across empty and planned meals.

**Proposal (P1):** Use the chosen Food overview with Meal plan and Recipes navigation rows, followed by today’s meals. Open each destination with a normal Back action.

**Evidence:** [iphone-light / food-overview](captures/iphone-light/food-overview.png), [iphone-small / meal-plan](captures/iphone-small/meal-plan.png), [iphone-small-extra / food-overview](captures/iphone-small-extra/food-overview.png), [iphone-contrast / variant-food](captures/iphone-contrast/variant-food.png), [iphone-dark / meal-plan](captures/iphone-dark/meal-plan.png)

**Mockups:** [food](mockups/food.png), [meal-plan](mockups/meal-plan.png)

<a id="meal-week"></a>

## Full week meal plan

**Goal:** Plan meals across the Friday–Thursday week.

**User friendly:** The week keeps all three meal types for every day. Repeated empty-row helper text makes it long.

**Interactions:** The selected week carries through from Food. Week navigation remains available.

**Main information and action:** Keep the week range and day headings clear while reducing empty-slot height.

**iOS 26:** The native list is suitable. Compact content improves scanning without removing meal slots.

**Proposal (P2):** Keep Friday–Thursday boundaries. Use shorter empty rows and retain all breakfast, lunch, and dinner actions.

**Evidence:** [iphone-light / food-week](captures/iphone-light/food-week.png), [iphone-light / food-week-bottom](captures/iphone-light/food-week-bottom.png), [iphone-small-extra / food-week](captures/iphone-small-extra/food-week.png), [iphone-small-extra / food-week-bottom](captures/iphone-small-extra/food-week-bottom.png)

**Mockups:** [meal-week](mockups/meal-week.png)

<a id="meal-add"></a>

## Add meal / recipe selection

**Goal:** Choose a recipe or enter a custom meal.

**User friendly:** One name field supports two paths. The footer explains Return, but the distinction between text and a linked recipe needs emphasis.

**Interactions:** Typing filters recipes. Choosing a recipe retains its link. Return saves a custom name.

**Main information and action:** Save and the selected recipe need an obvious state.

**iOS 26:** Native sheet controls suit the task. Persistent labels help when the keyboard is open.

**Proposal (P2):** Keep Save visible. State “Choose a recipe or save the meal name.” Show a checkmark for the selected recipe.

**Evidence:** [iphone-light / food-add-meal](captures/iphone-light/food-add-meal.png), [iphone-light / food-meal-search](captures/iphone-light/food-meal-search.png), [iphone-light / food-meal-recipe-selected](captures/iphone-light/food-meal-recipe-selected.png), [iphone-small / meal-form](captures/iphone-small/meal-form.png), [iphone-small-extra / food-add-meal](captures/iphone-small-extra/food-add-meal.png), [iphone-small-extra / food-meal-search](captures/iphone-small-extra/food-meal-search.png), [iphone-small-extra / food-meal-recipe-selected](captures/iphone-small-extra/food-meal-recipe-selected.png), [iphone-dark / meal-form](captures/iphone-dark/meal-form.png)

**Mockups:** [meal-add](mockups/meal-add.png)

<a id="meal-edit"></a>

## Edit meal

**Goal:** Change a planned meal without an unexpected recipe-link change.

**User friendly:** The existing editor supports both custom meals and linked recipes.

**Interactions:** Changing a linked meal name removes the recipe link. That effect needs clear feedback.

**Main information and action:** Show the linked recipe, Save, and Remove meal distinctly.

**iOS 26:** Keep the native editor and recipe search. Make selection explicit instead of adding a warning.

**Proposal (P1):** Remove the link warning. Typing filters recipes without changing the current link. Selecting a result changes the recipe. Use name only explicitly selects a custom meal. Save applies the choice.

**Evidence:** [iphone-light / food-edit-meal](captures/iphone-light/food-edit-meal.png), [iphone-small-extra / food-edit-meal](captures/iphone-small-extra/food-edit-meal.png)

**Mockups:** [meal-edit](mockups/meal-edit.png)

<a id="recipes"></a>

## Recipe list

**Goal:** Find a family recipe.

**User friendly:** The list is short and clear. The two durations have no visible prep/cook labels. Search can be out of view until the list reveals it.

**Interactions:** Search filters by text. Meal type narrows the list. Plus opens the recipe form.

**Main information and action:** Search and the recipe title should lead. Keep creation available without dominating the list.

**iOS 26:** Native searchable lists fit iOS 26. Use the system search presentation in the implementation.

**Proposal (P2):** Keep search accessible from entry. Label prep and cook times. Expose Recipes at the top of Food.

**Evidence:** [iphone-repairs / food-recipes](captures/iphone-repairs/food-recipes.png), [iphone-light / food-recipes](captures/iphone-light/food-recipes.png), [iphone-small / recipes](captures/iphone-small/recipes.png), [iphone-small-extra / food-recipes](captures/iphone-small-extra/food-recipes.png), [iphone-dark / recipes](captures/iphone-dark/recipes.png)

**Mockups:** [recipes](mockups/recipes.png)

<a id="recipes-empty"></a>

## Recipe search: no results

**Goal:** Recover from an unsuccessful search.

**User friendly:** The empty-result view explains that the query has no matches.

**Interactions:** The user can edit the query and change the meal filter.

**Main information and action:** A direct Clear search action reduces the recovery effort.

**iOS 26:** The native empty view is suitable. Keep query context and a specific recovery action.

**Proposal (P2):** Add Clear search and explain that the meal filter can also hide matches.

**Evidence:** [iphone-repairs / food-recipes-no-results](captures/iphone-repairs/food-recipes-no-results.png), [iphone-small-extra / food-recipes-no-results](captures/iphone-small-extra/food-recipes-no-results.png)

**Mockups:** [recipes-empty](mockups/recipes-empty.png)

<a id="recipe-detail"></a>

## Recipe detail, ingredient actions, and method

**Goal:** Read a recipe, prepare ingredients, or start cooking.

**User friendly:** Ingredients and method are readable. Cooking uses a flame-only toolbar button; ingredient check and cart actions are close together.

**Interactions:** Each ingredient has separate check and shopping actions. The actions must remain independent.

**Main information and action:** Start cooking deserves a labelled button near the title and timings.

**iOS 26:** Native content and toolbar controls suit iOS 26. Keep important actions clear without adding glass to ingredient rows.

**Proposal (P1):** Add Start cooking near the title. Keep each ingredient’s quantity, check state, and independent add-to-shopping action.

**Evidence:** [iphone-light / food-recipe-detail](captures/iphone-light/food-recipe-detail.png), [iphone-light / food-recipe-ingredients](captures/iphone-light/food-recipe-ingredients.png), [iphone-light / food-recipe-method](captures/iphone-light/food-recipe-method.png), [iphone-small / recipe-detail](captures/iphone-small/recipe-detail.png), [iphone-small-extra / food-recipe-detail](captures/iphone-small-extra/food-recipe-detail.png), [iphone-small-extra / food-recipe-ingredients](captures/iphone-small-extra/food-recipe-ingredients.png), [iphone-small-extra / food-recipe-method](captures/iphone-small-extra/food-recipe-method.png), [iphone-contrast / variant-recipe](captures/iphone-contrast/variant-recipe.png), [iphone-dark / recipe-detail](captures/iphone-dark/recipe-detail.png)

**Mockups:** [recipe-detail](mockups/recipe-detail.png)

<a id="recipe-actions"></a>

## Recipe action menu

**Goal:** Edit or delete a recipe.

**User friendly:** The ellipsis keeps occasional actions out of the reading flow.

**Interactions:** The menu opens Edit and Delete. Deletion needs a confirmation.

**Main information and action:** Keep Start cooking outside the secondary action menu.

**iOS 26:** The native menu is appropriate. Retain clear labels and destructive roles.

**Proposal (Keep):** Keep the menu. Add a prominent Start cooking button on the detail screen.

**Evidence:** [iphone-light / food-recipe-actions](captures/iphone-light/food-recipe-actions.png), [iphone-small-extra / food-recipe-actions](captures/iphone-small-extra/food-recipe-actions.png)

**Mockups:** [recipe-detail](mockups/recipe-detail.png)

<a id="recipe-create"></a>

## New recipe

**Goal:** Create a recipe with ingredients and steps.

**User friendly:** The large empty photo area precedes the required title. It consumes the most valuable space before the user enters any content.

**Interactions:** The form supports ingredient groups, separate steps, a photo, timings, servings, meal type, and a source.

**Main information and action:** The title and recipe content matter more than an optional photo.

**iOS 26:** The native form is a good base. Required content should lead the hierarchy.

**Proposal (P1):** Put Title first. Reduce Photo to an optional row. Keep every content section and its add/remove actions.

**Evidence:** [iphone-light / food-recipe-create](captures/iphone-light/food-recipe-create.png), [iphone-light / food-recipe-create-details](captures/iphone-light/food-recipe-create-details.png), [iphone-light / food-recipe-create-ingredients](captures/iphone-light/food-recipe-create-ingredients.png), [iphone-light / food-recipe-create-steps](captures/iphone-light/food-recipe-create-steps.png), [iphone-small / recipe-form](captures/iphone-small/recipe-form.png), [iphone-small-extra / food-recipe-create](captures/iphone-small-extra/food-recipe-create.png), [iphone-small-extra / food-recipe-create-details](captures/iphone-small-extra/food-recipe-create-details.png), [iphone-small-extra / food-recipe-create-ingredients](captures/iphone-small-extra/food-recipe-create-ingredients.png), [iphone-small-extra / food-recipe-create-steps](captures/iphone-small-extra/food-recipe-create-steps.png), [iphone-dark / recipe-form](captures/iphone-dark/recipe-form.png)

**Mockups:** [recipe-create](mockups/recipe-create.png), [recipe-content-edit](mockups/recipe-content-edit.png)

<a id="recipe-edit"></a>

## Edit recipe

**Goal:** Edit the complete recipe without losing content.

**User friendly:** The form contains all needed fields. Photo precedes the title. At maximum text size, the ingredient editor shows only part of a line.

**Interactions:** Save applies all sections together. Cancel exits. Existing ingredient groups and steps remain editable.

**Main information and action:** Title, ingredients, and method need the clearest entry points.

**iOS 26:** Keep the native form. A consistent field order across create and edit improves predictability.

**Proposal (P1):** Use the same title-first structure as New recipe. Let ingredient fields expand at large text sizes. Keep the photo, source, groups, and step controls.

**Evidence:** [iphone-light / food-recipe-edit](captures/iphone-light/food-recipe-edit.png), [iphone-light / food-recipe-edit-details](captures/iphone-light/food-recipe-edit-details.png), [iphone-light / food-recipe-edit-ingredients](captures/iphone-light/food-recipe-edit-ingredients.png), [iphone-light / food-recipe-edit-steps](captures/iphone-light/food-recipe-edit-steps.png), [iphone-small-extra / food-recipe-edit](captures/iphone-small-extra/food-recipe-edit.png), [iphone-small-extra / food-recipe-edit-details](captures/iphone-small-extra/food-recipe-edit-details.png), [iphone-small-extra / food-recipe-edit-ingredients](captures/iphone-small-extra/food-recipe-edit-ingredients.png), [iphone-small-extra / food-recipe-edit-steps](captures/iphone-small-extra/food-recipe-edit-steps.png)

**Mockups:** [recipe-edit](mockups/recipe-edit.png), [recipe-content-edit](mockups/recipe-content-edit.png)

<a id="cook-ingredients"></a>

## Cook mode: ingredients

**Goal:** Prepare the ingredients before the first cooking step.

**User friendly:** The checklist is useful. The footer calls this page “1 of 5” while the next page says “Step 1 of 4.”

**Interactions:** Checks provide preparation feedback. The next control is an unlabelled visual chevron.

**Main information and action:** Use Start step 1 and make Ingredients a separate stage from numbered cooking steps.

**iOS 26:** The full-screen focus and awake display suit cooking. Visible labels improve understanding.

**Proposal (P1):** Show Before you start, then Start step 1. Count only cooking steps in the step progress.

**Evidence:** [iphone-repairs / food-cook-ingredients](captures/iphone-repairs/food-cook-ingredients.png), [iphone-light / food-cook-ingredients](captures/iphone-light/food-cook-ingredients.png), [iphone-small / cook-ingredients](captures/iphone-small/cook-ingredients.png), [iphone-small-extra / food-cook-ingredients](captures/iphone-small-extra/food-cook-ingredients.png), [iphone-contrast / variant-cook-ingredients](captures/iphone-contrast/variant-cook-ingredients.png), [iphone-dark / cook-ingredients](captures/iphone-dark/cook-ingredients.png)

**Mockups:** [cook-ingredients](mockups/cook-ingredients.png)

<a id="cook-step"></a>

## Cook mode: method step

**Goal:** Follow one method step with little effort.

**User friendly:** Large step text is useful. The visible “Step 1 of 4” and footer “2 of 5” use different counts.

**Interactions:** Swipe and chevrons move between pages. Previous and Next labels help users with unfamiliar gestures.

**Main information and action:** Keep the current step and a clear next action prominent.

**iOS 26:** The focused native screen is appropriate. Keep Reduce Motion support and interruptible system transitions.

**Proposal (P1):** Use one step count. Add Previous and Next step labels. Keep swipe navigation as an additional path.

**Evidence:** [iphone-repairs / food-cook-step](captures/iphone-repairs/food-cook-step.png), [iphone-light / food-cook-step](captures/iphone-light/food-cook-step.png), [iphone-small / cook-step](captures/iphone-small/cook-step.png), [iphone-small-extra / food-cook-step](captures/iphone-small-extra/food-cook-step.png), [iphone-contrast / variant-cook-step](captures/iphone-contrast/variant-cook-step.png), [iphone-dark / cook-step](captures/iphone-dark/cook-step.png)

**Mockups:** [cook-step](mockups/cook-step.png)

<a id="cook-finish"></a>

## Cook mode: final step

**Goal:** Finish the recipe and return to its details.

**User friendly:** The final Done control is available. In the maximum-text capture, its label wraps inside the footer button and crowds the progress display.

**Interactions:** Done closes cook mode and restores the recipe.

**Main information and action:** Keep the last instruction visible above Finish cooking.

**iOS 26:** A native button and predictable return path fit the platform.

**Proposal (P1):** Use a full-width Finish cooking button and retain Previous step. Keep both labels readable at maximum text size.

**Evidence:** [iphone-repairs / food-cook-final-step](captures/iphone-repairs/food-cook-final-step.png), [iphone-light / food-cook-final-step](captures/iphone-light/food-cook-final-step.png), [iphone-small-extra / food-cook-final-step](captures/iphone-small-extra/food-cook-final-step.png)

**Mockups:** [cook-finish](mockups/cook-finish.png)

<a id="household"></a>

## Shopping list and low-stock suggestions

**Goal:** Add and purchase shopping items.

**User friendly:** The list supports a fast shopping task. Inventory is below the shopping content, so a long list can hide it.

**Interactions:** The check control purchases an item. The item menu supports rename and delete.

**Main information and action:** Adding an item and marking a purchase should remain the fastest actions.

**iOS 26:** The native list and floating tab bar fit iOS 26. Separate shopping from stock actions clearly.

**Proposal (P1):** Use the chosen Household overview with Shopping list and Inventory navigation rows, followed by low-stock items. Keep inline rename and five-second purchase Undo in Shopping list.

**Evidence:** [iphone-repairs / household-shopping](captures/iphone-repairs/household-shopping.png), [iphone-light / household-shopping](captures/iphone-light/household-shopping.png), [iphone-small / shopping](captures/iphone-small/shopping.png), [iphone-small-final / household-shopping](captures/iphone-small-final/household-shopping.png), [iphone-small-extra / household-shopping](captures/iphone-small-extra/household-shopping.png), [iphone-contrast / variant-household](captures/iphone-contrast/variant-household.png), [iphone-dark / shopping](captures/iphone-dark/shopping.png)

**Mockups:** [household](mockups/household.png), [shopping-list](mockups/shopping-list.png)

<a id="shopping-actions"></a>

## Shopping item menu

**Goal:** Rename or delete one shopping item.

**User friendly:** The item menu is compact and appropriate for occasional actions.

**Interactions:** The menu opens a rename sheet or a delete confirmation.

**Main information and action:** Keep purchase outside the menu as the direct check action.

**iOS 26:** The native menu fits the platform.

**Proposal (Keep):** Rename activates an inline field in the existing row. Delete keeps its confirmation. Keep purchase as a separate check action.

**Evidence:** [iphone-repairs / household-shopping-actions](captures/iphone-repairs/household-shopping-actions.png), [iphone-light / household-shopping-actions](captures/iphone-light/household-shopping-actions.png), [iphone-small-final / household-shopping-actions](captures/iphone-small-final/household-shopping-actions.png), [iphone-small-extra / household-shopping-actions](captures/iphone-small-extra/household-shopping-actions.png)

**Mockups:** [household](mockups/household.png)

<a id="shopping-rename"></a>

## Rename shopping item

**Goal:** Correct a shopping item name.

**User friendly:** The current sheet takes the user away from the shopping list for a single-field edit.

**Interactions:** Save applies the name. Cancel returns without changes.

**Main information and action:** Keep the item in its original row. Focus the name field and provide a clear commit and cancel action.

**iOS 26:** Use native text entry and keyboard support within the row.

**Proposal (P2):** Replace the rename sheet with inline text entry. Done saves a non-empty name. Cancel keeps the original name. Keep failed edits in place with an error.

**Evidence:** [iphone-repairs / household-shopping-rename](captures/iphone-repairs/household-shopping-rename.png), [iphone-light / household-shopping-rename](captures/iphone-light/household-shopping-rename.png), [iphone-small-final / household-shopping-rename](captures/iphone-small-final/household-shopping-rename.png), [iphone-small-extra / household-shopping-rename](captures/iphone-small-extra/household-shopping-rename.png)

**Mockups:** [shopping-rename](mockups/shopping-rename.png)

<a id="shopping-purchased"></a>

## Purchase feedback and Undo

**Goal:** Mark an item purchased and correct an accidental check.

**User friendly:** Purchased items remain visible and can be restored. This supports correction.

**Interactions:** Clear purchased items asks for confirmation. A linked purchase can also need a stock update.

**Main information and action:** The user wants checked items hidden after five seconds. Keep Undo visible during that interval.

**iOS 26:** The native list and confirmation are suitable. Text must explain status independently of color.

**Proposal (P2):** Remove the Purchased section and its clear action. Keep the checked row in place with Undo for five seconds, then hide it. Undo restores the unchecked state.

**Evidence:** [iphone-repairs / household-shopping-purchased](captures/iphone-repairs/household-shopping-purchased.png), [iphone-repairs / household-shopping-restocked](captures/iphone-repairs/household-shopping-restocked.png), [iphone-light / household-shopping-purchased](captures/iphone-light/household-shopping-purchased.png), [iphone-small-final / household-shopping-purchased](captures/iphone-small-final/household-shopping-purchased.png), [iphone-small-final / household-shopping-restocked](captures/iphone-small-final/household-shopping-restocked.png), [iphone-small-extra / household-shopping-purchased](captures/iphone-small-extra/household-shopping-purchased.png)

**Mockups:** [shopping-pending](mockups/shopping-pending.png), [shopping-hidden](mockups/shopping-hidden.png), [shopping-undone](mockups/shopping-undone.png), [restocked](mockups/restocked.png)

<a id="restock"></a>

## Restock after purchase

**Goal:** Update inventory after purchasing a linked item.

**User friendly:** The form already shows current stock and offers Not now. The transition from purchase to stock entry needs a short explanation.

**Interactions:** The user enters a quantity or percentage and confirms Restock.

**Main information and action:** Show the current level, the new level, and an explicit defer action.

**iOS 26:** The native sheet is appropriate. Keep the two decisions clear and recoverable.

**Proposal (P2):** Keep stock updates optional through Inventory. Do not interrupt a shopping check with this form. Retain the current stock, a cancel action, and the new-level label.

**Evidence:** [iphone-repairs / household-shopping-restock](captures/iphone-repairs/household-shopping-restock.png), [iphone-repairs / household-shopping-restock-keyboard](captures/iphone-repairs/household-shopping-restock-keyboard.png), [iphone-small-final / household-shopping-restock](captures/iphone-small-final/household-shopping-restock.png), [iphone-small-final / household-shopping-restock-keyboard](captures/iphone-small-final/household-shopping-restock-keyboard.png)

**Mockups:** [restock](mockups/restock.png)

<a id="inventory"></a>

## Inventory overview

**Goal:** Find an area and see stock that needs attention.

**User friendly:** The six-item Running Low list comes before Areas. It pushes the main navigation toward or below the bottom edge.

**Interactions:** Area rows open their contents. Low-stock rows can add one item to Shopping.

**Main information and action:** Area navigation needs a stable, visible position.

**iOS 26:** Native lists work well. Content order should support navigation before a long status list.

**Proposal (P1):** Place Areas first and Running low below. Keep every low-stock item and its shopping action.

**Evidence:** [iphone-final / household-inventory](captures/iphone-final/household-inventory.png), [iphone-repairs / household-inventory](captures/iphone-repairs/household-inventory.png), [iphone-light / household-inventory](captures/iphone-light/household-inventory.png), [iphone-small / inventory](captures/iphone-small/inventory.png), [iphone-small-extra / household-inventory](captures/iphone-small-extra/household-inventory.png), [iphone-dark / inventory](captures/iphone-dark/inventory.png)

**Mockups:** [inventory](mockups/inventory.png)

<a id="area"></a>

## Inventory area

**Goal:** Read stock levels and adjust household items.

**User friendly:** Counts, level bars, low-stock status, and cart controls share each row. Some row-wide edit actions lack a clear visual cue.

**Interactions:** Tap the item name to edit. Plus/minus adjusts count. Cart adds the item to Shopping.

**Main information and action:** Keep quick stock adjustments separate from row navigation and shopping.

**iOS 26:** Native list rows are suitable. Clear control boundaries and target sizes matter more than decoration.

**Proposal (P1):** Use a clear edit affordance, explicit stock values, and distinct shopping controls. Retain All and Running low filters.

**Evidence:** [iphone-final / household-area](captures/iphone-final/household-area.png), [iphone-repairs / household-area](captures/iphone-repairs/household-area.png), [iphone-light / household-area](captures/iphone-light/household-area.png), [iphone-small / area-detail](captures/iphone-small/area-detail.png), [iphone-small-extra / household-area](captures/iphone-small-extra/household-area.png), [iphone-dark / area-detail](captures/iphone-dark/area-detail.png)

**Mockups:** [area](mockups/area.png)

<a id="area-actions"></a>

## Inventory area menu

**Goal:** Edit or delete an inventory area.

**User friendly:** The long-press menu is useful as a shortcut but is hard to discover as the only visible edit route.

**Interactions:** Long press or swipe reveals Edit and Delete. Deletion also removes the area’s items.

**Main information and action:** A visible area action menu helps users find edit and delete.

**iOS 26:** Native context menus suit shortcuts. Important management actions need a discoverable alternative.

**Proposal (P2):** Add a visible area action menu. Keep long press and swipe as shortcuts.

**Evidence:** [iphone-light / household-area-actions](captures/iphone-light/household-area-actions.png), [iphone-small-extra / household-area-actions](captures/iphone-small-extra/household-area-actions.png)

**Mockups:** [area](mockups/area.png)

<a id="area-create"></a>

## New inventory area

**Goal:** Name a storage area and select its icon and colour.

**User friendly:** The compact form and preview explain the result well.

**Interactions:** Name receives keyboard focus. Save stays disabled until the name has text.

**Main information and action:** Name, Save, and the preview are available.

**iOS 26:** This is a suitable native form. Avoid a broad redesign.

**Proposal (Keep):** Keep the form. Add one short description of what an area represents.

**Evidence:** [iphone-light / household-area-create](captures/iphone-light/household-area-create.png), [iphone-small / area-form](captures/iphone-small/area-form.png), [iphone-small-extra / household-area-create](captures/iphone-small-extra/household-area-create.png), [iphone-dark / area-form](captures/iphone-dark/area-form.png)

**Mockups:** [area-create](mockups/area-create.png)

<a id="area-edit"></a>

## Edit inventory area

**Goal:** Change an area name, icon, or colour.

**User friendly:** The form matches area creation and includes a preview.

**Interactions:** Save applies the area changes. Cancel returns to Inventory.

**Main information and action:** Keep the current values and preview visible.

**iOS 26:** The consistent native form follows platform conventions.

**Proposal (Keep):** Retain this form and use the same field order as New area.

**Evidence:** [iphone-light / household-area-edit](captures/iphone-light/household-area-edit.png), [iphone-small-extra / household-area-edit](captures/iphone-small-extra/household-area-edit.png)

**Mockups:** [area-edit](mockups/area-edit.png)

<a id="item-create"></a>

## New inventory item: count

**Goal:** Add an item with a count and low-stock threshold.

**User friendly:** The form is short. Count and Level are domain terms that benefit from a household example.

**Interactions:** The tracking choice changes the stock controls. Save needs a name.

**Main information and action:** Name, quantity, unit, and the meaning of the threshold need clear labels.

**iOS 26:** Native form controls suit this task. A numeric field can complement a stepper for large counts.

**Proposal (P2):** Explain Count and Level. Label the threshold explicitly and retain precise numeric input.

**Evidence:** [iphone-final / household-item-create-count](captures/iphone-final/household-item-create-count.png), [iphone-final / household-item-create-stock](captures/iphone-final/household-item-create-stock.png), [iphone-small / item-form](captures/iphone-small/item-form.png), [iphone-dark / item-form](captures/iphone-dark/item-form.png)

**Mockups:** [item-create](mockups/item-create.png)

<a id="item-edit-level"></a>

## Edit inventory item: level

**Goal:** Update a partly used item and its low-stock threshold.

**User friendly:** The percentage and slider express current stock. The form says “Low below,” but the iOS model includes the threshold value in its low-stock state.

**Interactions:** The slider updates the remaining level. Save applies the form; Delete asks for confirmation.

**Main information and action:** Keep the current percentage and threshold together.

**iOS 26:** Native slider and form controls fit iOS. Add a precise numeric entry route if the slider alone is too coarse.

**Proposal (P2):** Use Remaining level and Alert at or below. Keep the percentage, slider, Save, and delete confirmation.

**Evidence:** [iphone-final / household-item-edit-level](captures/iphone-final/household-item-edit-level.png), [iphone-repairs / household-item-edit-level](captures/iphone-repairs/household-item-edit-level.png), [iphone-light / household-item-edit-level](captures/iphone-light/household-item-edit-level.png), [iphone-small-extra / household-item-edit-level](captures/iphone-small-extra/household-item-edit-level.png)

**Mockups:** [item-edit-level](mockups/item-edit-level.png)

<a id="settings"></a>

## Settings and profile

**Goal:** Manage this iPhone, the profile, and family settings.

**User friendly:** The list groups App and Admin. The avatar label says Open profile, but the destination says Settings. At maximum text size, identity and photo controls delay the settings list.

**Interactions:** Done closes the sheet. Links open settings details. Admin-only entries depend on the current role.

**Main information and action:** Users need a clear distinction between this iPhone and changes that affect the whole family.

**iOS 26:** Native inset-grouped settings and sheets suit iOS 26. Name the entry and destination consistently.

**Proposal (P1):** Use Settings for the entry label. Separate This iPhone and Family administration. Retain every existing settings destination.

**Evidence:** [iphone-light / settings-profile](captures/iphone-light/settings-profile.png), [iphone-small / settings](captures/iphone-small/settings.png), [iphone-small-extra / settings-profile](captures/iphone-small-extra/settings-profile.png), [iphone-contrast / variant-settings](captures/iphone-contrast/variant-settings.png), [iphone-dark / settings](captures/iphone-dark/settings.png)

**Mockups:** [settings](mockups/settings.png)

<a id="photo-picker"></a>

## System photo picker

**Goal:** Choose a profile photo.

**User friendly:** The app uses the system picker. The capture shows its loading state; the settled photo gallery is not reviewed.

**Interactions:** Selection returns an image to the app. Cancel leaves the profile unchanged.

**Main information and action:** Keep photo selection inside the system picker and show upload feedback in Settings.

**iOS 26:** Retain the native Photos picker. Do not reproduce its interface with a custom design.

**Proposal (Keep):** No picker redesign. Preserve upload feedback and retry errors in the profile.

**Evidence:** [iphone-light / settings-photo-picker](captures/iphone-light/settings-photo-picker.png), [iphone-small-extra / settings-photo-picker](captures/iphone-small-extra/settings-photo-picker.png)

<a id="server-config"></a>

## Server configuration

**Goal:** Change the server connection for this iPhone.

**User friendly:** The form is compact. Connect can sign the user out. The captured accessibility hierarchy exposes an unnamed Server URL button when the demo URL is empty.

**Interactions:** Connect validates and discovers server settings before saving. Cancel exits.

**Main information and action:** Explain the sign-out effect before submission and preserve the current configuration on failure.

**iOS 26:** The native form fits iOS. Important consequences need plain text near the action.

**Proposal (P1):** State that connecting to another server signs the user out. Keep the address and error inline.

**Evidence:** [iphone-repairs / settings-server-configuration](captures/iphone-repairs/settings-server-configuration.png), [iphone-small-extra / settings-server-configuration](captures/iphone-small-extra/settings-server-configuration.png)

**Mockups:** [server-config](mockups/server-config.png)

<a id="reminders"></a>

## Chore reminders and device permission

**Goal:** Enable this iPhone’s notifications and set the family reminder schedule.

**User friendly:** The screen combines a local permission with a family-wide schedule. Disabled demo controls need a visible reason.

**Interactions:** The device control opens or changes local permission. Admins edit both shared times and save.

**Main information and action:** Separate the local permission state from the family schedule and show the time zone.

**iOS 26:** Native toggles and time pickers are suitable. Use labels and explanatory text for disabled states.

**Proposal (P1):** Group This iPhone and Family reminder schedule. Explain demo limitations and unavailable server push support.

**Evidence:** [iphone-light / settings-reminders](captures/iphone-light/settings-reminders.png), [iphone-light / settings-reminders-bottom](captures/iphone-light/settings-reminders-bottom.png), [iphone-small-extra / settings-reminders](captures/iphone-small-extra/settings-reminders.png), [iphone-small-extra / settings-reminders-bottom](captures/iphone-small-extra/settings-reminders-bottom.png), [iphone-contrast / variant-reminders](captures/iphone-contrast/variant-reminders.png)

**Mockups:** [reminders](mockups/reminders.png)

<a id="broadcast"></a>

## Family broadcast: compose

**Goal:** Send a short message to family iPhones.

**User friendly:** The form has a message field, character count, and Send. The text already explains lock-screen exposure.

**Interactions:** Send stays disabled for invalid text or unavailable push support. The keyboard has Done.

**Main information and action:** Use a household label such as Family message. Explain that only registered iPhones receive it.

**iOS 26:** The native form works. Keep the action label and delivery scope understandable.

**Proposal (P2):** Rename the screen Family message. Keep the 500-character limit and lock-screen explanation. Preserve the message after a failed send.

**Evidence:** [iphone-light / settings-broadcast](captures/iphone-light/settings-broadcast.png), [iphone-light / settings-broadcast-keyboard](captures/iphone-light/settings-broadcast-keyboard.png), [iphone-small-extra / settings-broadcast](captures/iphone-small-extra/settings-broadcast.png), [iphone-small-extra / settings-broadcast-keyboard](captures/iphone-small-extra/settings-broadcast-keyboard.png)

**Mockups:** [broadcast](mockups/broadcast.png)

<a id="broadcast-result"></a>

## Family broadcast: result

**Goal:** Understand whether the family message reaches any registered devices.

**User friendly:** “Apple accepted: 0” is transport language. At maximum text size, the result sits below the message field and can remain outside the viewport.

**Interactions:** The result remains in the form. Retry must not repeat accepted notifications.

**Main information and action:** Lead with the outcome and next step. Apple acceptance does not prove device display or reading.

**iOS 26:** Clear human feedback is more important than transport detail in a household app.

**Proposal (P1):** Lead with No available iPhones for the zero-device case. Move focus to the result. Retain the message and link to notification setup.

**Evidence:** [iphone-light / settings-broadcast-result](captures/iphone-light/settings-broadcast-result.png), [iphone-small-extra / settings-broadcast-result](captures/iphone-small-extra/settings-broadcast-result.png)

**Mockups:** [broadcast-result](mockups/broadcast-result.png)

<a id="home-info"></a>

## Home information and location

**Goal:** Set the home location and arrival margin.

**User friendly:** Coordinates and the time zone are present. “Home Info” does not explain weather and departures.

**Interactions:** The user captures device location, confirms it, then saves. This avoids silently changing the home location.

**Main information and action:** Explain the purpose and preserve the explicit location confirmation.

**iOS 26:** Native location permission and form controls are suitable. Technical coordinates need plain context.

**Proposal (P2):** Use Home location. Explain weather, departures, and arrival margin. Keep coordinates available and keep confirm-before-save.

**Evidence:** [iphone-final / settings-home-info](captures/iphone-final/settings-home-info.png), [iphone-final / settings-home-info-bottom](captures/iphone-final/settings-home-info-bottom.png), [iphone-repairs / settings-home-info](captures/iphone-repairs/settings-home-info.png), [iphone-repairs / settings-home-info-bottom](captures/iphone-repairs/settings-home-info-bottom.png), [iphone-light / settings-home-info](captures/iphone-light/settings-home-info.png), [iphone-light / settings-home-info-bottom](captures/iphone-light/settings-home-info-bottom.png), [iphone-small / home-info](captures/iphone-small/home-info.png), [iphone-small-final / settings-home-info](captures/iphone-small-final/settings-home-info.png), [iphone-small-final / settings-home-info-bottom](captures/iphone-small-final/settings-home-info-bottom.png), [iphone-small-extra / settings-home-info](captures/iphone-small-extra/settings-home-info.png), [iphone-small-extra / settings-home-info-bottom](captures/iphone-small-extra/settings-home-info-bottom.png), [iphone-dark / home-info](captures/iphone-dark/home-info.png)

**Mockups:** [home-info](mockups/home-info.png)

<a id="family-name"></a>

## Family name

**Goal:** Change the name shown to the family.

**User friendly:** The one-field form and family-wide explanation are sufficient.

**Interactions:** Save applies the name and returns. Empty text disables Save.

**Main information and action:** The field needs a label that remains visible after entry.

**iOS 26:** Keep the native form. No structural redesign is needed.

**Proposal (Keep):** Add a persistent Family name label. Retain the explanation and Save behavior.

**Evidence:** [iphone-final / settings-family-name](captures/iphone-final/settings-family-name.png), [iphone-final / settings-family-name-keyboard](captures/iphone-final/settings-family-name-keyboard.png), [iphone-repairs / settings-family-name](captures/iphone-repairs/settings-family-name.png), [iphone-light / settings-family-name](captures/iphone-light/settings-family-name.png), [iphone-small / family-name](captures/iphone-small/family-name.png), [iphone-small-final / settings-family-name](captures/iphone-small-final/settings-family-name.png), [iphone-small-final / settings-family-name-keyboard](captures/iphone-small-final/settings-family-name-keyboard.png), [iphone-small-extra / settings-family-name](captures/iphone-small-extra/settings-family-name.png), [iphone-dark / family-name](captures/iphone-dark/family-name.png)

**Mockups:** [family-name](mockups/family-name.png)

<a id="members"></a>

## Family members and role actions

**Goal:** See family membership and manage roles.

**User friendly:** Names, emails, and the Admin badge are clear. Promote and Demote only appear after a swipe.

**Interactions:** A swipe changes another member’s role. The current user has no self-demote action.

**Main information and action:** Role changes need a discoverable path and a plain explanation of the permission effect.

**iOS 26:** Native rows are suitable. Swipe-only role management is hard to discover.

**Proposal (P1):** Open role details from the member row. Explain Member and Admin. Confirm a change and retain the self-protection rule.

**Evidence:** [iphone-final / settings-family-members](captures/iphone-final/settings-family-members.png), [iphone-final / settings-member-actions](captures/iphone-final/settings-member-actions.png), [iphone-repairs / settings-family-members](captures/iphone-repairs/settings-family-members.png), [iphone-repairs / settings-member-actions](captures/iphone-repairs/settings-member-actions.png), [iphone-light / settings-family-members](captures/iphone-light/settings-family-members.png), [iphone-light / settings-member-actions](captures/iphone-light/settings-member-actions.png), [iphone-small / members](captures/iphone-small/members.png), [iphone-small-final / settings-family-members](captures/iphone-small-final/settings-family-members.png), [iphone-small-final / settings-member-actions](captures/iphone-small-final/settings-member-actions.png), [iphone-small-extra / settings-family-members](captures/iphone-small-extra/settings-family-members.png), [iphone-small-extra / settings-member-actions](captures/iphone-small-extra/settings-member-actions.png), [iphone-dark / members](captures/iphone-dark/members.png)

**Mockups:** [members](mockups/members.png), [member-role](mockups/member-role.png)

<a id="categories"></a>

## Category list and actions

**Goal:** Create and manage chore categories.

**User friendly:** The list is short and easy to scan. Main Course and Salads look like recipe categories in the demo, which confuses the chore purpose.

**Interactions:** Plus creates a category. Swipe exposes rename and delete.

**Main information and action:** Explain what categories organise and make editing discoverable.

**iOS 26:** Native lists, sheets, and confirmations are suitable.

**Proposal (P2):** Keep the native list. Add a chore-focused explanation and a visible row action. Correct the demo category examples.

**Evidence:** [iphone-light / settings-categories](captures/iphone-light/settings-categories.png), [iphone-light / settings-category-actions](captures/iphone-light/settings-category-actions.png), [iphone-small / categories](captures/iphone-small/categories.png), [iphone-small-extra / settings-categories](captures/iphone-small-extra/settings-categories.png), [iphone-small-extra / settings-category-actions](captures/iphone-small-extra/settings-category-actions.png), [iphone-dark / categories](captures/iphone-dark/categories.png)

**Mockups:** [categories](mockups/categories.png)

<a id="category-create"></a>

## New category

**Goal:** Create a named chore category.

**User friendly:** The short name sheet matches the task.

**Interactions:** Save applies a valid name. Cancel returns without changes.

**Main information and action:** A category name and visible Save are sufficient.

**iOS 26:** Retain the native form and standard sheet controls.

**Proposal (Keep):** Keep the form. Use a persistent Category name label.

**Evidence:** [iphone-light / settings-category-create](captures/iphone-light/settings-category-create.png), [iphone-small-extra / settings-category-create](captures/iphone-small-extra/settings-category-create.png)

**Mockups:** [category-create](mockups/category-create.png)

<a id="category-edit"></a>

## Rename category

**Goal:** Rename an existing category.

**User friendly:** The short sheet suits the task. A family-wide effect explanation can prevent surprises.

**Interactions:** Save applies the name. Delete remains a separate confirmed action.

**Main information and action:** Keep the current name visible and editable.

**iOS 26:** Retain the native sheet and text field.

**Proposal (Keep):** Add a short family-wide effect explanation. Keep deletion separate.

**Evidence:** [iphone-light / settings-category-rename](captures/iphone-light/settings-category-rename.png), [iphone-small-extra / settings-category-rename](captures/iphone-small-extra/settings-category-rename.png)

**Mockups:** [category-edit](mockups/category-edit.png)

<a id="tokens"></a>

## API token list and empty state

**Goal:** Manage credentials used by other apps.

**User friendly:** The empty state offers a plus-based next step but does not explain why a family admin needs a token.

**Interactions:** Plus opens a name sheet. Existing tokens can be deleted.

**Main information and action:** Explain the purpose and the effect of deletion before creation.

**iOS 26:** Native list and empty-state controls are suitable.

**Proposal (P2):** Explain that tokens connect other apps and act like passwords. Keep the token list in family administration.

**Evidence:** [iphone-final / settings-tokens-populated](captures/iphone-final/settings-tokens-populated.png), [iphone-final / settings-tokens](captures/iphone-final/settings-tokens.png), [iphone-light / settings-tokens](captures/iphone-light/settings-tokens.png), [iphone-small / tokens](captures/iphone-small/tokens.png), [iphone-small-extra / settings-tokens](captures/iphone-small-extra/settings-tokens.png), [iphone-dark / tokens](captures/iphone-dark/tokens.png)

**Mockups:** [tokens](mockups/tokens.png)

<a id="token-create"></a>

## Create API token

**Goal:** Create a token with an identifiable name.

**User friendly:** The name-only sheet is concise. It does not explain the one-time reveal before creation.

**Interactions:** Save creates the token and opens its one-time value.

**Main information and action:** Use Create token and explain the next step.

**iOS 26:** Native sheet controls fit the task. Action-specific wording improves feedback.

**Proposal (P2):** Use a clear token name label, Create token, and a one-time reveal notice.

**Evidence:** [iphone-final / settings-token-create](captures/iphone-final/settings-token-create.png), [iphone-light / settings-token-create](captures/iphone-light/settings-token-create.png), [iphone-small / token-form](captures/iphone-small/token-form.png), [iphone-small-extra / settings-token-create](captures/iphone-small-extra/settings-token-create.png), [iphone-dark / token-form](captures/iphone-dark/token-form.png)

**Mockups:** [token-create](mockups/token-create.png)

<a id="token-created"></a>

## Token created and copied

**Goal:** Copy the newly created token before dismissing it.

**User friendly:** The existing screen already warns that the token appears once. This is useful.

**Interactions:** Copy Token copies the value. Done closes the reveal. The capture does not show a distinct copied confirmation.

**Main information and action:** Keep the copy action and one-time warning above other content.

**iOS 26:** Native selectable text and copy controls are suitable. Confirm completion with text.

**Proposal (P2):** Retain the one-time reveal. Change the copy label briefly to Copied and announce the result.

**Evidence:** [iphone-final / settings-token-created](captures/iphone-final/settings-token-created.png), [iphone-final / settings-token-copied](captures/iphone-final/settings-token-copied.png), [iphone-light / settings-token-created](captures/iphone-light/settings-token-created.png), [iphone-light / settings-token-copied](captures/iphone-light/settings-token-copied.png), [iphone-small / token-created](captures/iphone-small/token-created.png), [iphone-small-extra / settings-token-created](captures/iphone-small-extra/settings-token-created.png), [iphone-small-extra / settings-token-copied](captures/iphone-small-extra/settings-token-copied.png), [iphone-dark / token-created](captures/iphone-dark/token-created.png)

**Mockups:** [token-created](mockups/token-created.png)

<a id="calendar-view-menu"></a>

## Calendar display menu

**Goal:** Choose the calendar display.

**User friendly:** The menu contains four supported modes. Its icon does not reveal the active mode before the menu opens.

**Interactions:** The native menu selects Year, Month, Week, or Day.

**Main information and action:** Show the active mode and keep Today available.

**iOS 26:** Keep a native menu or segmented control. Avoid a custom glass panel.

**Proposal (Keep):** Keep the current Calendar design and interactions, as selected by the user. Only remove arrival-margin text. Earlier layout proposals are superseded.

**Evidence:** [iphone-supplement / calendar-view-menu](captures/iphone-supplement/calendar-view-menu.png)

<a id="chore-repeat-options"></a>

## Chore recurrence variants

**Goal:** Set monthly or custom repeat rules and stop conditions.

**User friendly:** These variants add a day of month, interval unit, end date, or task limit. The form needs a readable summary as options change.

**Interactions:** Native pickers and switches reveal the relevant controls. Both stop conditions remain available.

**Main information and action:** Show the resulting schedule before Save.

**iOS 26:** Native form controls are appropriate. Keep custom weekday targets large and label each option plainly.

**Proposal (P2):** Retain all recurrence variants. Add a schedule summary and plain labels.

**Evidence:** [iphone-supplement / tasks-repeat-menu](captures/iphone-supplement/tasks-repeat-menu.png), [iphone-supplement / tasks-repeat-monthly](captures/iphone-supplement/tasks-repeat-monthly.png), [iphone-supplement / tasks-repeat-custom](captures/iphone-supplement/tasks-repeat-custom.png), [iphone-supplement / tasks-repeat-end-date](captures/iphone-supplement/tasks-repeat-end-date.png), [iphone-supplement / tasks-repeat-count](captures/iphone-supplement/tasks-repeat-count.png)

**Mockups:** [chore-repeat](mockups/chore-repeat.png), [chore-repeat-monthly](mockups/chore-repeat-monthly.png), [chore-repeat-custom](mockups/chore-repeat-custom.png)

<a id="item-edit-count"></a>

## Edit inventory item: count

**Goal:** Change an item quantity, unit, or threshold.

**User friendly:** The stock values are explicit. Large quantities need more than repeated stepper taps.

**Interactions:** The stepper changes quantity and threshold. Save applies the values.

**Main information and action:** Keep the quantity and unit together. Offer direct numeric entry.

**iOS 26:** Native numeric entry and steppers fit the platform.

**Proposal (P2):** Keep the stepper and add precise numeric entry. Retain unit and threshold.

**Evidence:** [iphone-final / household-item-edit-count](captures/iphone-final/household-item-edit-count.png)

**Mockups:** [item-edit-count](mockups/item-edit-count.png)

<a id="area-empty"></a>

## Inventory area: empty

**Goal:** Start stock records in a new area.

**User friendly:** The empty area keeps its filter and an Add Item action.

**Interactions:** Add Item opens the same item form used in populated areas.

**Main information and action:** The next action is to add the first item.

**iOS 26:** Keep the native empty explanation and navigation.

**Proposal (Keep):** Use a clear empty-state message and retain Add item. No broader redesign is needed.

**Evidence:** [iphone-final / household-area-empty](captures/iphone-final/household-area-empty.png)

**Mockups:** [area-empty](mockups/area-empty.png)

## Coverage and checks

See [coverage.md](coverage.md) for the capture matrix, test results, remaining gaps, and implementation checks.

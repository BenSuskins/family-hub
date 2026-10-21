# Navigation maps

The maps show iPhone navigation. Settings is a sheet, not a sixth tab.

## Current

```mermaid
flowchart TD
    App[Family Hub] --> Today
    App --> Calendar
    App --> Tasks
    App --> Food
    App --> Household
    Today --> Event[Event details]
    Today --> Weather
    Today --> Complete[Complete or skip a chore]
    Today --> Meal[Meal or recipe]
    Calendar --> Event
    Calendar --> Modes[Year / Month / Week / Day]
    Tasks --> Rules[Chore rules]
    Rules --> Rule[Setup detail / history / edit]
    Food --> Plan[Daily meal plan]
    Plan --> Week[Full week]
    Plan --> Recipes[Recipes below meals]
    Recipes --> Recipe[Recipe detail / cook / edit]
    Household --> Shopping[Shopping list]
    Shopping --> Inventory[Inventory entry below shopping]
    Inventory --> Areas[Areas / items / stock]
    Today --> Settings
    Food --> Settings
    Household --> Settings
    Settings --> Phone[Profile / reminders / server]
    Settings --> Admin[Family / home / members / categories / tokens / broadcast]
```

## Proposed

```mermaid
flowchart TD
    App[Family Hub] --> Today
    App --> Calendar
    App --> Chores
    App --> Food
    App --> Household
    Today --> Urgent[Next departure]
    Today --> Meals[Meal plan · before chores]
    Today --> TodayChores[Overdue chores / today chores]
    Today --> Weather
    Urgent --> Event[Event details]
    TodayChores --> Complete[Complete or skip]
    Calendar --> Modes[Year / Month / Week / Day]
    Modes --> Agenda[Readable selected-day events]
    Agenda --> Event
    Chores --> Rules[Chore rules / detail / history / edit]
    Rules --> TodayLink[View current task on Today]
    Food[Food overview] --> Plan[Meal plan]
    Food --> Recipes[Recipes]
    Plan --> Week[Friday–Thursday week]
    Plan --> Recipe[Linked recipe]
    Recipes --> Recipe
    Recipe --> Cook[Prepare / numbered steps / finish]
    Recipe --> Edit[Edit / share / delete / Watch where supported]
    Household[Household overview] --> Shopping[Shopping list]
    Household --> Inventory[Inventory]
    Shopping --> Undo[Check / Undo for five seconds / hide]
    Inventory --> Stock[Optional stock update]
    Inventory --> Areas[Areas / items / quick stock / add to Shopping]
    App -. Settings sheet .-> Settings
    Settings --> Phone[This iPhone]
    Settings --> Admin[Family administration]
```

## Navigation rules

- Use the chosen overview screens for Food and Household. Replace inner navigation segments with rows and normal Back actions.

- Keep five top-level tabs. Use Chores for the tab and the screen title.
- Keep completion on Today. Link the current task to Today from its rule detail.
- Keep Meal plan and Recipes as destinations within Food.
- Keep Shopping and Inventory as destinations within Household.
- Preserve the selected meal date, week, and in-tab context when users open details and return.
- Open a linked recipe from the main planned-meal row. Keep Edit meal available through a visible secondary action.
- Keep all existing settings and admin-only access rules.
- Use native back navigation and native sheets. Preserve tab visibility for ordinary detail navigation.
- Keep share, source links, photo selection, optional Watch actions, and destructive confirmations in their existing contexts.

# Siri and Shortcuts

Family Hub exposes ten actions on iPhone. Sign in to the app first. The phone
must reach the configured Family Hub server. Siri can run the actions while
the phone is locked, subject to the phone's Siri settings and system access.

## Actions

Say “Hey Siri” before these trigger phrases:

| Phrase | Action |
| --- | --- |
| “What's for dinner in Family Hub?” | Read a meal. Breakfast and lunch also work. |
| “Plan a meal in Family Hub.” | Set a meal name, meal type, and date. |
| “What are my chores in Family Hub?” | Read your due and overdue chores. |
| “Complete a chore in Family Hub.” | Select one of your due or overdue chores. |
| “Complete my chores in Family Hub.” | Complete your current due and overdue chores. |
| “Add to my shopping list in Family Hub.” | Add an item, with an inventory link when a name matches. |
| “Read inventory in Family Hub.” | Select an item or storage area. |
| “Read low stock in Family Hub.” | Read low items across the household or in an area. |
| “Update stock in Family Hub.” | Set a count or percentage, or add or subtract a count. |
| “Add an inventory item in Family Hub.” | Create an item in an existing area. |

These are registered trigger phrases. Spoken recognition needs a test on a
physical iPhone. Siri asks for missing values. Longer requests, such as “What's
for dinner next Tuesday in Family Hub?”, depend on Siri's parameter resolution.
Each action also appears in the Shortcuts app with editable inputs.

## Dates and changes

Meal dates use calendar days. An omitted date means today in the household
timezone. Siri accepts a supplied date, including a date that Siri resolves
from “next Tuesday.” Meal responses state the resolved date and year.

A meal change replaces the meal in that date and meal slot. A unique matching
recipe becomes a link. Duplicate recipe matches require a choice. An unmatched
name becomes a meal without a recipe link.

“My chores” means incomplete chores assigned to the signed-in account, due
today or overdue. Future and undated chores stay outside this set. Bulk
completion uses the original set and does not complete new recurring chores.
Siri reports partial failures and uncertain results.

Clear changes do not ask for app confirmation. Siri still asks for missing
values or a choice between duplicate names. iOS can require its own prompts.

Inventory supports whole-item counts and whole percentages. An item that is
not tracked differs from an item with zero stock. Count decreases stop at
zero. New items require an existing area, measurement, initial stock, unit for
count items, and low-stock threshold. Duplicate names in the same area require
an update through the Update Stock action.

Long lists speak the count and first five items. The Shortcuts result contains
the complete list. These actions do not create inventory areas, move items, or
delete items.

## Release checks

Release the server with the stock endpoint before the iOS version that uses it.
The app does not fall back to a full-item update on older servers.

The build and automated tests do not prove spoken recognition or locked-phone
behavior. Check the following on a physical iPhone before release:

- Run the registered phrases with the app closed.
- Ask for breakfast, lunch, and dinner on another date.
- Ask “What's for dinner next Tuesday in Family Hub?”
- Say “Set dinner for next Tuesday in Family Hub.” Supply the meal name when asked.
- Check the resolved date in the response and the meal in the app.
- Complete one chore and all personal chores. Check recurring chores.
- Read stock by item and area. Check duplicate item names.
- Change a count and a percentage. Create an inventory item.
- Repeat the checks with the phone locked.
- Check signed-out and unreachable-server responses.

## Contract test

Run the Swift client against a temporary Go API and database from `server/`:

```sh
RUN_SWIFT_STOCK_CONTRACT=1 go test ./internal/handlers -run TestInventoryStockSwiftContract -count=1 -timeout=4m
```

This test requires Swift and local network access. It checks count changes,
percentage changes, metadata preservation, tracking-mode errors, and the absence
of automatic retries for relative writes.

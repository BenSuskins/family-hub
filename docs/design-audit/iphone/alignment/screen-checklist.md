# Remaining screen checklist

21 October 2026. Read this with the [native comparison](index.html).

The table maps each changed screen to the approved mockup and fresh simulator evidence. It records functional and layout alignment, not identical pixels.

| Screen | Approved mockup | Native evidence | Review |
|---|---|---|---|
| Chores | [chores](../mockups/chores.png) | [chores-list](captures/chores-list.png) | The Today entry appears before the repeat rules. Chore completion stays on Today. |
| Chore detail | [chore-detail](../mockups/chore-detail.png) | [chore-detail](captures/chore-detail.png) | The current task, repeat rule, history, and Edit action have distinct positions. |
| New chore | [chore-create](../mockups/chore-create.png) | [chore-create](captures/chore-create.png) | Name, description, people, and schedule follow the approved order. |
| Edit chore | [chore-edit](../mockups/chore-edit.png) | [chore-edit](captures/chore-edit.png) | The form preserves the current task, rotation, category, and advanced schedule fields. |
| Weekly repeat rule | [chore-repeat](../mockups/chore-repeat.png) | [chore-repeat-weekly](captures/chore-repeat-weekly.png) | Repeat opens as a native child screen. Done returns to the draft. Save writes the chore. |
| Monthly repeat rule | [chore-repeat-monthly](../mockups/chore-repeat-monthly.png) | [chore-repeat-monthly](captures/chore-repeat-monthly.png) | The native day picker preserves the monthly rule. |
| Custom repeat rule | [chore-repeat-custom](../mockups/chore-repeat-custom.png) | [chore-repeat-custom](captures/chore-repeat-custom.png) | Interval, unit, completion timing, and stop rules remain available. |
| Settings | [settings](../mockups/settings.png) | [settings-top](captures/settings-top.png) | The profile, this iPhone, family administration, and About form separate groups. |
| Notifications | [reminders](../mockups/reminders.png) | [notifications-top](captures/notifications-top.png) | The page distinguishes device permission from the family reminder schedule. Demo permission status is real simulator state. |
| Family reminder schedule | [reminders](../mockups/reminders.png) | [notifications-schedule](captures/notifications-schedule.png) | Both reminder times and the time zone remain visible after a short scroll on a small iPhone. |
| Server | [server-config](../mockups/server-config.png) | [server](captures/server.png) | The server form has a labelled address, a clear sign-out notice, and a full-width Connect action. |
| Family name | [family-name](../mockups/family-name.png) | [family-name](captures/family-name.png) | A full-width Save family name action commits the name. |
| Home location | [home-info](../mockups/home-info.png) | [home-location](captures/home-location.png) | The saved location, capture action, and arrival setting remain available. Calendar and Today do not show the margin. |
| Family message | [broadcast](../mockups/broadcast.png) | [family-message-ready](captures/family-message-ready.png) | The composer has one message field, a character count, a permission notice, and a full-width Send action. |
| Message result | [broadcast-result](../mockups/broadcast-result.png) | [family-message-result](captures/family-message-result.png) | The result opens on a separate screen. Back to message preserves the draft. Notification help opens Notifications. |
| API tokens | [tokens](../mockups/tokens.png) | [tokens-populated](captures/tokens-populated.png) | Token rows show names and dates. Opening a row shows its access-management actions. |
| New token | [token-create](../mockups/token-create.png) | [token-create-ready](captures/token-create-ready.png) | The form uses a name, a one-time-value explanation, and a full-width Create action. |
| Created token | [token-created](../mockups/token-created.png) | [token-created](captures/token-created.png) | The one-time value is selectable. Copy is full width. The list never exposes the value after dismissal. |
| Family members | [members](../mockups/members.png) | [family-members](captures/family-members.png) | People have avatars or soft person icons. Admin badges and the current user remain clear. |
| Member role | [member-role](../mockups/member-role.png) | [member-role](captures/member-role.png) | Member and Admin are draft selections. Save asks for confirmation. The app protects the current user’s role. |
| Categories | [categories](../mockups/categories.png) | [categories](captures/categories.png) | Each category opens Rename and Delete directly. Swipe actions remain available. |
| New category | [category-create](../mockups/category-create.png) | [category-create](captures/category-create.png) | The native sheet has a labelled name, Cancel, and Save. |
| Edit category | [category-edit](../mockups/category-edit.png) | [category-edit](captures/category-edit.png) | The sheet groups the name and family impact. Delete asks for confirmation with the category name. |
| Weekly meal plan | [meal-week](../mockups/meal-week.png) | [meal-week-linked-recipe](captures/meal-week-linked-recipe.png) | The Friday–Thursday range has compact meal rows and a Today action. Linked meals show a recipe thumbnail with a book fallback. Meals without a link stay text-only. Other weeks use the truthful Meal plan title. |
| Meal editor | [meal-edit](../mockups/meal-edit.png) | [meal-recipe-selected](captures/meal-recipe-selected.png) | Recipe search keeps its metadata and selection. An explicit Use name only action follows the results. |
| Recipe search in a meal | [meal-add](../mockups/meal-add.png) | [meal-search-keyboard](captures/meal-search-keyboard.png) | Typing filters recipes. Selecting a recipe dismisses the keyboard. Typing alone preserves the recipe link. |
| Home weather | [weather](../mockups/weather.png) | [weather](captures/weather.png) | The weather card centres the condition, temperature, high, low, and advice. Hourly data and provider attribution remain available. |
| Event detail | [event](../mockups/event.png) | [event-details](captures/event-details.png) | Date, location, departure, and full details use separate cards. The Calendar grids and gestures do not change. |
| Inline shopping rename | [shopping-rename](../mockups/shopping-rename.png) | [shopping-inline-rename](captures/shopping-inline-rename.png) | Cancel stays beside the name. Done uses the native keyboard toolbar. Errors keep the draft. |
| Shopping Undo hint | [shopping-pending](../mockups/shopping-pending.png) | [shopping-undo](captures/shopping-undo.png) | Each checked item shows its own countdown and Undo. It hides after five seconds and a confirmed server write. |
| Count stock | [item-edit-count](../mockups/item-edit-count.png) | [inventory-count](captures/inventory-count.png) | Count and Level use a segment at standard text sizes. One control combines the editable count and stock buttons. |
| Level stock | [item-edit-level](../mockups/item-edit-level.png) | [inventory-level](captures/inventory-level.png) | Precise percentage entry and the low-stock threshold remain editable. The compact threshold label stays readable on small screens. |
| New inventory item | [item-create](../mockups/item-create.png) | [inventory-create-keyboard](captures/inventory-create-keyboard.png) | The new-item form retains name, mode, threshold, unit, and validation. The keyboard remains native. |

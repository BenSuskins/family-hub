"""Build the static comparison report from the audit register and captures."""
import html
import json
import re
from pathlib import Path

root = Path(__file__).resolve().parents[1]
entries = []

def screen(key, title, match, goal, usability, interaction, primary, apple, change, priority='P2', mockups=None):
    entries.append(dict(key=key,title=title,match=match,goal=goal,usability=usability,interaction=interaction,primary=primary,apple=apple,change=change,priority=priority,mockups=mockups if mockups is not None else [key]))

screen('welcome','Welcome / server entry',['login','access-welcome','access-server-keyboard'],
 'Connect to the family server or try the app.',
 'The short form is easy to scan. It assumes that a family member knows what a server URL means.',
 'Connect remains disabled until the field has text. The URL keyboard suits the field.',
 'Connect is available, but the help for obtaining an address deserves a clearer position.',
 'The native form and controls fit iOS. The first screen needs a clearer household purpose.',
 'Explain the family benefit. Ask the user to obtain the address from their family admin. Keep Try Demo secondary.', 'P1')
screen('setup-help','Setup help',['access-setup-help','access-setup-help-bottom'],
 'Understand how to join a family or set up a server.',
 'The help starts with Docker, OIDC, PKCE, and environment variables. This suits a server admin, but burdens family members.',
 'Done provides a clear exit. The repository link provides full instructions.',
 'A family member needs the server address and an account before deployment instructions.',
 'The sheet is native. Progressive disclosure better separates common and advanced help.',
 'Lead with three join steps. Keep server deployment in an admin section and retain the full guide.', 'P1')
screen('server-error','Invalid server address',['access-invalid-server','settings-server-validation'],
 'Correct an invalid connection address.',
 'The form keeps the address and shows an inline error. This supports recovery.',
 'The user can edit and retry. Avoid dismissing the keyboard or replacing the whole form.',
 'The error needs to identify the required address format next to the field.',
 'Inline feedback and native text entry fit the platform.',
 'Keep the error beside a persistent label. Use a specific correction and retain the entered value.')
screen('today','Today',['today','today-overview','today-bottom','home-chores'],
 'Know what needs attention now and complete household work.',
 'Agenda, meals, chores, and departures share the page. The current order puts meals before overdue chores and departures last.',
 'Chore completion, the chore action menu, weather, events, and meals provide direct entry points.',
 'The meal plan takes priority over chores, as confirmed by the user. Keep meal actions above the chore list.',
 'Native navigation and grouped content suit iOS 26. Content priority needs more work than the materials.',
 'Put a near departure first, then the meal plan, the agenda, and overdue and today’s chores. Keep weather compact and omit empty departure cards.', 'P1', ['today','today-dark','today-large'])
screen('weather','Weather detail',['today-weather','today-weather-bottom'],
 'Read local conditions and the forecast.',
 'The detail screen keeps weather out of Today. Each hourly row repeats the full date, which slows the scan.',
 'The header weather control opens the detail screen. Back returns to Today.',
 'Current conditions, the forecast, and data freshness should appear in that order.',
 'Native navigation and semantic type provide a suitable base.',
 'Keep the current conditions prominent. Use shorter hourly time labels. Retain all 24 hourly forecasts, weather advice, freshness, and Open-Meteo attribution.')
screen('event','Event detail',['today-event'],
 'Inspect an event and know when to leave.',
 'The title, date, location, description, and departure are present. Departure follows the description, which can push it down.',
 'The event opens from Today or Calendar. The same detail should serve both routes.',
 'Group the start time, location, and departure before a long description.',
 'The native detail view fits iOS. Better grouping improves the reading order.',
 'Put date and location together. Put the departure immediately below them. Keep the full description below.', 'P1')
screen('chore-actions','Today chore menu',['today-chore-actions'],
 'Skip a particular task without changing its repeat rule.',
 'A menu keeps secondary actions away from the completion control. The affected chore needs clear context.',
 'The ellipsis opens the menu. An outside tap dismisses the native popover when no Cancel control appears.',
 'Complete remains the direct action. Skip needs a short explanation of its effect.',
 'The native menu is appropriate. Do not replace it with a custom floating panel.',
 'Include the chore name and assignee. Explain Skip. Add a route to the chore setup.', 'P2')
screen('calendar-month','Calendar: month and zoom',['calendar','calendar-month-details','calendar-month-zoom','calendar-month','calendar-Month'],
 'Browse dates and inspect the selected day.',
 'The grid provides useful month context. Narrow date columns truncate event titles even when the rows are tall.',
 'Month scroll, pinch zoom, date selection, and Today work in the captured interaction tests.',
 'A readable selected-day list helps users inspect events without guessing a truncated title.',
 'The system toolbar and floating tab bar fit iOS 26. The content should remain the focus.',
 'Retain the month grid and pinch zoom. Add a selected-day agenda. Label the four supported display modes.', 'P1')
screen('calendar-day','Calendar: day and event overlap',['calendar-day-overlap','calendar-day','calendar-Day'],
 'Inspect the daily schedule and open an event.',
 'The timeline communicates time and separates overlapping events. Text competes for width in simultaneous blocks.',
 'Tapping an event opens details. Swipe changes the day. Today restores the current date.',
 'The date, all-day items, and full event names need to remain readable.',
 'The native toolbar and direct timeline navigation follow the platform. Avoid adding glass behind event content.',
 'Keep the timeline. Separate all-day items clearly and provide readable event details.', 'P2')
screen('calendar-week','Calendar: week',['calendar-week','calendar-Week'],
 'Compare the schedule across seven days.',
 'The week capture shows event titles broken into one-letter lines in seven narrow columns. This prevents useful inspection.',
 'The view menu selects Week. Select a day or event to inspect details.',
 'The overview needs a readable path to the full event name.',
 'A week overview is appropriate, but the phone width requires an inspection layer.',
 'Keep the week overview and pair it with the selected day’s full event list.', 'P1')
screen('calendar-year','Calendar: year',['calendar-year','calendar-Year'],
 'Move to a month in another part of the year.',
 'The compact month grids suit broad date navigation. Keep selection obvious.',
 'Year opens a month. Today restores the current period.',
 'The year and month entry points are the main information and actions.',
 'The year-to-month hierarchy matches familiar calendar navigation.',
 'Retain year browsing. Make the current month clear and keep the display selector visible.', 'P2')
screen('calendar-large','Calendar: accessibility text',['calendar-accessibility'],
 'Inspect the schedule with large text.',
 'The accessibility route uses an agenda and date picker instead of forcing large text into a month grid.',
 'The date picker and view menu keep date navigation available.',
 'Full event labels and the selected date matter more than a dense overview.',
 'This is a good adaptive pattern. Preserve it when changing the standard calendar layout.',
 'Retain the readable list and date picker. Keep all view choices reachable.', 'Keep')
screen('chores','Chore setup list',['chores','tasks-list','chore-setup-list'],
 'Create and manage chore rules.',
 'The screen title says Chores while the tab says Tasks. At maximum text size, Show inactive chores consumes much of the first viewport.',
 'Rows open setup details. The plus control creates a chore. Inactive chores can be shown.',
 'Creation and repeat rules are available. Users also need a direct route to chores due today.',
 'The list and toolbar are native. Consistent destination names improve wayfinding.',
 'Rename the tab Chores. Move Show inactive below the active rules. Keep completion on Today and add a direct Today link.', 'P1')
screen('chore-detail','Chore setup detail and history',['chore-detail','tasks-detail','chore-setup-detail'],
 'Inspect the repeat rule, current task, and history.',
 'The setup, current task, and history are clear groups. “Complete or skip this task on Home” names a destination that is actually Today.',
 'Edit and delete are below the history. The completion instruction has no direct link.',
 'The current task deserves a higher position and a direct route to its action.',
 'Native list sections suit the detail. Place the frequent edit action in the toolbar.',
 'Put the current task first. Replace Home with Today and make the instruction a link. Keep history and deletion.', 'P1')
screen('chore-create','New chore',['chore-form','tasks-create','tasks-create-keyboard','tasks-create-schedule'],
 'Create a chore and assign its schedule.',
 'The form includes the required capabilities, but it mixes details, rotating assignees, dates, and advanced repeat options in one long form.',
 'Save stays in the toolbar. Selecting no assignees means everyone; the existing footer explains this.',
 'The name, first due date, and eligible people should lead. Advanced repeat settings can follow.',
 'The native form and sheet are suitable. Plain labels and progressive disclosure improve the interaction.',
 'Use “Due date” and “Due time” labels. Show a repeat summary and keep advanced conditions in a repeat section.', 'P1')
screen('chore-edit','Edit chore and repeat rule',['tasks-edit','tasks-edit-schedule','tasks-edit-repeat','tasks-edit-end'],
 'Change a chore rule without changing the outstanding task.',
 'The existing explanation preserves an important distinction: rule changes affect the next task.',
 'The form retains current task ownership and dates. The weekday controls are compact for touch.',
 'Make the current task constraint and resulting repeat rule easy to read.',
 'The native form is appropriate. Aim for 44-point default touch targets on custom weekday controls.',
 'Keep the current-task explanation. Use full-size weekday controls and a readable repeat summary.', 'P1', ['chore-edit','chore-repeat'])
screen('delete-confirmation','Deletion confirmations',['tasks-delete-confirmation','household-item-delete-confirmation','household-shopping-clear-confirmation','household-item-delete-popover-persisted-1','household-item-delete-popover-persisted-2'],
 'Confirm a destructive change with enough context.',
 'The app uses native confirmation dialogs. Naming the item and the effect reduces mistakes.',
 'Delete confirms the action. Cancel or an outside tap dismisses the dialog, depending on presentation.',
 'The user needs to know exactly what disappears and whether history remains.',
 'Native confirmation dialogs fit iOS 26. Preserve their system appearance and roles.',
 'Name the item in the title. Explain the effect, including area contents or retained chore history.', 'P2')
screen('food','Food / daily meal plan',['meal-plan','food-overview'],
 'Read the selected day’s meals and reach recipes.',
 'The date strip and meal rows are clear. Large empty slots push Recipes down. At maximum text size, the header delays access to the first meal.',
 'The planned meal row opens Edit. A separate View recipe action opens the recipe.',
 'Users often want to read or cook a planned recipe. Opening it should be the main planned-meal action.',
 'Native lists and date selection fit iOS. Keep navigation predictable across empty and planned meals.',
 'Use the chosen Food overview with Meal plan and Recipes navigation rows, followed by today’s meals. Open each destination with a normal Back action.', 'P1', ['food','meal-plan'])
screen('meal-week','Full week meal plan',['food-week','food-week-bottom'],
 'Plan meals across the Friday–Thursday week.',
 'The week keeps all three meal types for every day. Repeated empty-row helper text makes it long.',
 'The selected week carries through from Food. Week navigation remains available.',
 'Keep the week range and day headings clear while reducing empty-slot height.',
 'The native list is suitable. Compact content improves scanning without removing meal slots.',
 'Keep Friday–Thursday boundaries. Use shorter empty rows and retain all breakfast, lunch, and dinner actions.')
screen('meal-add','Add meal / recipe selection',['meal-form','food-add-meal','food-meal-search','food-meal-recipe-selected'],
 'Choose a recipe or enter a custom meal.',
 'One name field supports two paths. The footer explains Return, but the distinction between text and a linked recipe needs emphasis.',
 'Typing filters recipes. Choosing a recipe retains its link. Return saves a custom name.',
 'Save and the selected recipe need an obvious state.',
 'Native sheet controls suit the task. Persistent labels help when the keyboard is open.',
 'Keep Save visible. State “Choose a recipe or save the meal name.” Show a checkmark for the selected recipe.')
screen('meal-edit','Edit meal',['food-edit-meal'],
 'Change a planned meal without an unexpected recipe-link change.',
 'The existing editor supports both custom meals and linked recipes.',
 'Changing a linked meal name removes the recipe link. That effect needs clear feedback.',
 'Show the linked recipe, Save, and Remove meal distinctly.',
 'Keep the native editor and recipe search. Make selection explicit instead of adding a warning.',
 'Remove the link warning. Typing filters recipes without changing the current link. Selecting a result changes the recipe. Use name only explicitly selects a custom meal. Save applies the choice.', 'P1')
screen('recipes','Recipe list',['recipes','food-recipes'],
 'Find a family recipe.',
 'The list is short and clear. The two durations have no visible prep/cook labels. Search can be out of view until the list reveals it.',
 'Search filters by text. Meal type narrows the list. Plus opens the recipe form.',
 'Search and the recipe title should lead. Keep creation available without dominating the list.',
 'Native searchable lists fit iOS 26. Use the system search presentation in the implementation.',
 'Keep search accessible from entry. Label prep and cook times. Expose Recipes at the top of Food.')
screen('recipes-empty','Recipe search: no results',['food-recipes-no-results'],
 'Recover from an unsuccessful search.',
 'The empty-result view explains that the query has no matches.',
 'The user can edit the query and change the meal filter.',
 'A direct Clear search action reduces the recovery effort.',
 'The native empty view is suitable. Keep query context and a specific recovery action.',
 'Add Clear search and explain that the meal filter can also hide matches.')
screen('recipe-detail','Recipe detail, ingredient actions, and method',['recipe-detail','food-recipe-detail','food-recipe-ingredients','food-recipe-method'],
 'Read a recipe, prepare ingredients, or start cooking.',
 'Ingredients and method are readable. Cooking uses a flame-only toolbar button; ingredient check and cart actions are close together.',
 'Each ingredient has separate check and shopping actions. The actions must remain independent.',
 'Start cooking deserves a labelled button near the title and timings.',
 'Native content and toolbar controls suit iOS 26. Keep important actions clear without adding glass to ingredient rows.',
 'Add Start cooking near the title. Keep each ingredient’s quantity, check state, and independent add-to-shopping action.', 'P1')
screen('recipe-actions','Recipe action menu',['food-recipe-actions'],
 'Edit or delete a recipe.',
 'The ellipsis keeps occasional actions out of the reading flow.',
 'The menu opens Edit and Delete. Deletion needs a confirmation.',
 'Keep Start cooking outside the secondary action menu.',
 'The native menu is appropriate. Retain clear labels and destructive roles.',
 'Keep the menu. Add a prominent Start cooking button on the detail screen.', 'Keep', ['recipe-detail'])
screen('recipe-create','New recipe',['recipe-form','food-recipe-create','food-recipe-create-details','food-recipe-create-ingredients','food-recipe-create-steps'],
 'Create a recipe with ingredients and steps.',
 'The large empty photo area precedes the required title. It consumes the most valuable space before the user enters any content.',
 'The form supports ingredient groups, separate steps, a photo, timings, servings, meal type, and a source.',
 'The title and recipe content matter more than an optional photo.',
 'The native form is a good base. Required content should lead the hierarchy.',
 'Put Title first. Reduce Photo to an optional row. Keep every content section and its add/remove actions.', 'P1', ['recipe-create','recipe-content-edit'])
screen('recipe-edit','Edit recipe',['food-recipe-edit','food-recipe-edit-details','food-recipe-edit-ingredients','food-recipe-edit-steps'],
 'Edit the complete recipe without losing content.',
 'The form contains all needed fields. Photo precedes the title. At maximum text size, the ingredient editor shows only part of a line.',
 'Save applies all sections together. Cancel exits. Existing ingredient groups and steps remain editable.',
 'Title, ingredients, and method need the clearest entry points.',
 'Keep the native form. A consistent field order across create and edit improves predictability.',
 'Use the same title-first structure as New recipe. Let ingredient fields expand at large text sizes. Keep the photo, source, groups, and step controls.', 'P1', ['recipe-edit','recipe-content-edit'])
screen('cook-ingredients','Cook mode: ingredients',['cook-ingredients','food-cook-ingredients'],
 'Prepare the ingredients before the first cooking step.',
 'The checklist is useful. The footer calls this page “1 of 5” while the next page says “Step 1 of 4.”',
 'Checks provide preparation feedback. The next control is an unlabelled visual chevron.',
 'Use Start step 1 and make Ingredients a separate stage from numbered cooking steps.',
 'The full-screen focus and awake display suit cooking. Visible labels improve understanding.',
 'Show Before you start, then Start step 1. Count only cooking steps in the step progress.', 'P1')
screen('cook-step','Cook mode: method step',['cook-step','food-cook-step'],
 'Follow one method step with little effort.',
 'Large step text is useful. The visible “Step 1 of 4” and footer “2 of 5” use different counts.',
 'Swipe and chevrons move between pages. Previous and Next labels help users with unfamiliar gestures.',
 'Keep the current step and a clear next action prominent.',
 'The focused native screen is appropriate. Keep Reduce Motion support and interruptible system transitions.',
 'Use one step count. Add Previous and Next step labels. Keep swipe navigation as an additional path.', 'P1')
screen('cook-finish','Cook mode: final step',['food-cook-final-step'],
 'Finish the recipe and return to its details.',
 'The final Done control is available. In the maximum-text capture, its label wraps inside the footer button and crowds the progress display.',
 'Done closes cook mode and restores the recipe.',
 'Keep the last instruction visible above Finish cooking.',
 'A native button and predictable return path fit the platform.',
 'Use a full-width Finish cooking button and retain Previous step. Keep both labels readable at maximum text size.', 'P1')
screen('household','Shopping list and low-stock suggestions',['shopping','household-shopping'],
 'Add and purchase shopping items.',
 'The list supports a fast shopping task. Inventory is below the shopping content, so a long list can hide it.',
 'The check control purchases an item. The item menu supports rename and delete.',
 'Adding an item and marking a purchase should remain the fastest actions.',
 'The native list and floating tab bar fit iOS 26. Separate shopping from stock actions clearly.',
 'Use the chosen Household overview with Shopping list and Inventory navigation rows, followed by low-stock items. Keep inline rename and five-second purchase Undo in Shopping list.', 'P1', ['household','shopping-list'])
screen('shopping-actions','Shopping item menu',['household-shopping-actions'],
 'Rename or delete one shopping item.',
 'The item menu is compact and appropriate for occasional actions.',
 'The menu opens a rename sheet or a delete confirmation.',
 'Keep purchase outside the menu as the direct check action.',
 'The native menu fits the platform.',
 'Rename activates an inline field in the existing row. Delete keeps its confirmation. Keep purchase as a separate check action.', 'Keep', ['household'])
screen('shopping-rename','Rename shopping item',['household-shopping-rename'],
 'Correct a shopping item name.',
 'The current sheet takes the user away from the shopping list for a single-field edit.',
 'Save applies the name. Cancel returns without changes.',
 'Keep the item in its original row. Focus the name field and provide a clear commit and cancel action.',
 'Use native text entry and keyboard support within the row.',
 'Replace the rename sheet with inline text entry. Done saves a non-empty name. Cancel keeps the original name. Keep failed edits in place with an error.', 'P2')
screen('shopping-purchased','Purchase feedback and Undo',['household-shopping-purchased','household-shopping-restocked'],
 'Mark an item purchased and correct an accidental check.',
 'Purchased items remain visible and can be restored. This supports correction.',
 'Clear purchased items asks for confirmation. A linked purchase can also need a stock update.',
 'The user wants checked items hidden after five seconds. Keep Undo visible during that interval.',
 'The native list and confirmation are suitable. Text must explain status independently of color.',
 'Remove the Purchased section and its clear action. Keep the checked row in place with Undo for five seconds, then hide it. Undo restores the unchecked state.', 'P2', ['shopping-pending','shopping-hidden','shopping-undone','restocked'])
screen('restock','Restock after purchase',['household-shopping-restock','household-shopping-restock-keyboard'],
 'Update inventory after purchasing a linked item.',
 'The form already shows current stock and offers Not now. The transition from purchase to stock entry needs a short explanation.',
 'The user enters a quantity or percentage and confirms Restock.',
 'Show the current level, the new level, and an explicit defer action.',
 'The native sheet is appropriate. Keep the two decisions clear and recoverable.',
 'Keep stock updates optional through Inventory. Do not interrupt a shopping check with this form. Retain the current stock, a cancel action, and the new-level label.', 'P2')
screen('inventory','Inventory overview',['inventory','household-inventory'],
 'Find an area and see stock that needs attention.',
 'The six-item Running Low list comes before Areas. It pushes the main navigation toward or below the bottom edge.',
 'Area rows open their contents. Low-stock rows can add one item to Shopping.',
 'Area navigation needs a stable, visible position.',
 'Native lists work well. Content order should support navigation before a long status list.',
 'Place Areas first and Running low below. Keep every low-stock item and its shopping action.', 'P1')
screen('area','Inventory area',['area-detail','household-area'],
 'Read stock levels and adjust household items.',
 'Counts, level bars, low-stock status, and cart controls share each row. Some row-wide edit actions lack a clear visual cue.',
 'Tap the item name to edit. Plus/minus adjusts count. Cart adds the item to Shopping.',
 'Keep quick stock adjustments separate from row navigation and shopping.',
 'Native list rows are suitable. Clear control boundaries and target sizes matter more than decoration.',
 'Use a clear edit affordance, explicit stock values, and distinct shopping controls. Retain All and Running low filters.', 'P1')
screen('area-actions','Inventory area menu',['household-area-actions'],
 'Edit or delete an inventory area.',
 'The long-press menu is useful as a shortcut but is hard to discover as the only visible edit route.',
 'Long press or swipe reveals Edit and Delete. Deletion also removes the area’s items.',
 'A visible area action menu helps users find edit and delete.',
 'Native context menus suit shortcuts. Important management actions need a discoverable alternative.',
 'Add a visible area action menu. Keep long press and swipe as shortcuts.', 'P2', ['area'])
screen('area-create','New inventory area',['area-form','household-area-create'],
 'Name a storage area and select its icon and colour.',
 'The compact form and preview explain the result well.',
 'Name receives keyboard focus. Save stays disabled until the name has text.',
 'Name, Save, and the preview are available.',
 'This is a suitable native form. Avoid a broad redesign.',
 'Keep the form. Add one short description of what an area represents.', 'Keep')
screen('area-edit','Edit inventory area',['household-area-edit'],
 'Change an area name, icon, or colour.',
 'The form matches area creation and includes a preview.',
 'Save applies the area changes. Cancel returns to Inventory.',
 'Keep the current values and preview visible.',
 'The consistent native form follows platform conventions.',
 'Retain this form and use the same field order as New area.', 'Keep')
screen('item-create','New inventory item: count',['item-form','household-item-create-count','household-item-create-stock'],
 'Add an item with a count and low-stock threshold.',
 'The form is short. Count and Level are domain terms that benefit from a household example.',
 'The tracking choice changes the stock controls. Save needs a name.',
 'Name, quantity, unit, and the meaning of the threshold need clear labels.',
 'Native form controls suit this task. A numeric field can complement a stepper for large counts.',
 'Explain Count and Level. Label the threshold explicitly and retain precise numeric input.', 'P2')
screen('item-edit-level','Edit inventory item: level',['household-item-edit-level'],
 'Update a partly used item and its low-stock threshold.',
 'The percentage and slider express current stock. The form says “Low below,” but the iOS model includes the threshold value in its low-stock state.',
 'The slider updates the remaining level. Save applies the form; Delete asks for confirmation.',
 'Keep the current percentage and threshold together.',
 'Native slider and form controls fit iOS. Add a precise numeric entry route if the slider alone is too coarse.',
 'Use Remaining level and Alert at or below. Keep the percentage, slider, Save, and delete confirmation.')
screen('settings','Settings and profile',['settings','settings-profile'],
 'Manage this iPhone, the profile, and family settings.',
 'The list groups App and Admin. The avatar label says Open profile, but the destination says Settings. At maximum text size, identity and photo controls delay the settings list.',
 'Done closes the sheet. Links open settings details. Admin-only entries depend on the current role.',
 'Users need a clear distinction between this iPhone and changes that affect the whole family.',
 'Native inset-grouped settings and sheets suit iOS 26. Name the entry and destination consistently.',
 'Use Settings for the entry label. Separate This iPhone and Family administration. Retain every existing settings destination.', 'P1')
screen('photo-picker','System photo picker',['settings-photo-picker'],
 'Choose a profile photo.',
 'The app uses the system picker. The capture shows its loading state; the settled photo gallery is not reviewed.',
 'Selection returns an image to the app. Cancel leaves the profile unchanged.',
 'Keep photo selection inside the system picker and show upload feedback in Settings.',
 'Retain the native Photos picker. Do not reproduce its interface with a custom design.',
 'No picker redesign. Preserve upload feedback and retry errors in the profile.', 'Keep', [])
screen('server-config','Server configuration',['settings-server-configuration'],
 'Change the server connection for this iPhone.',
 'The form is compact. Connect can sign the user out. The captured accessibility hierarchy exposes an unnamed Server URL button when the demo URL is empty.',
 'Connect validates and discovers server settings before saving. Cancel exits.',
 'Explain the sign-out effect before submission and preserve the current configuration on failure.',
 'The native form fits iOS. Important consequences need plain text near the action.',
 'State that connecting to another server signs the user out. Keep the address and error inline.', 'P1')
screen('reminders','Chore reminders and device permission',['settings-reminders','settings-reminders-bottom'],
 'Enable this iPhone’s notifications and set the family reminder schedule.',
 'The screen combines a local permission with a family-wide schedule. Disabled demo controls need a visible reason.',
 'The device control opens or changes local permission. Admins edit both shared times and save.',
 'Separate the local permission state from the family schedule and show the time zone.',
 'Native toggles and time pickers are suitable. Use labels and explanatory text for disabled states.',
 'Group This iPhone and Family reminder schedule. Explain demo limitations and unavailable server push support.', 'P1')
screen('broadcast','Family broadcast: compose',['settings-broadcast','settings-broadcast-keyboard'],
 'Send a short message to family iPhones.',
 'The form has a message field, character count, and Send. The text already explains lock-screen exposure.',
 'Send stays disabled for invalid text or unavailable push support. The keyboard has Done.',
 'Use a household label such as Family message. Explain that only registered iPhones receive it.',
 'The native form works. Keep the action label and delivery scope understandable.',
 'Rename the screen Family message. Keep the 500-character limit and lock-screen explanation. Preserve the message after a failed send.', 'P2')
screen('broadcast-result','Family broadcast: result',['settings-broadcast-result'],
 'Understand whether the family message reaches any registered devices.',
 '“Apple accepted: 0” is transport language. At maximum text size, the result sits below the message field and can remain outside the viewport.',
 'The result remains in the form. Retry must not repeat accepted notifications.',
 'Lead with the outcome and next step. Apple acceptance does not prove device display or reading.',
 'Clear human feedback is more important than transport detail in a household app.',
 'Lead with No available iPhones for the zero-device case. Move focus to the result. Retain the message and link to notification setup.', 'P1')
screen('home-info','Home information and location',['home-info','settings-home-info','settings-home-info-bottom'],
 'Set the home location and arrival margin.',
 'Coordinates and the time zone are present. “Home Info” does not explain weather and departures.',
 'The user captures device location, confirms it, then saves. This avoids silently changing the home location.',
 'Explain the purpose and preserve the explicit location confirmation.',
 'Native location permission and form controls are suitable. Technical coordinates need plain context.',
 'Use Home location. Explain weather, departures, and arrival margin. Keep coordinates available and keep confirm-before-save.')
screen('family-name','Family name',['family-name','settings-family-name','settings-family-name-keyboard'],
 'Change the name shown to the family.',
 'The one-field form and family-wide explanation are sufficient.',
 'Save applies the name and returns. Empty text disables Save.',
 'The field needs a label that remains visible after entry.',
 'Keep the native form. No structural redesign is needed.',
 'Add a persistent Family name label. Retain the explanation and Save behavior.', 'Keep')
screen('members','Family members and role actions',['members','settings-family-members','settings-member-actions'],
 'See family membership and manage roles.',
 'Names, emails, and the Admin badge are clear. Promote and Demote only appear after a swipe.',
 'A swipe changes another member’s role. The current user has no self-demote action.',
 'Role changes need a discoverable path and a plain explanation of the permission effect.',
 'Native rows are suitable. Swipe-only role management is hard to discover.',
 'Open role details from the member row. Explain Member and Admin. Confirm a change and retain the self-protection rule.', 'P1', ['members','member-role'])
screen('categories','Category list and actions',['categories','settings-categories','settings-category-actions'],
 'Create and manage chore categories.',
 'The list is short and easy to scan. Main Course and Salads look like recipe categories in the demo, which confuses the chore purpose.',
 'Plus creates a category. Swipe exposes rename and delete.',
 'Explain what categories organise and make editing discoverable.',
 'Native lists, sheets, and confirmations are suitable.',
 'Keep the native list. Add a chore-focused explanation and a visible row action. Correct the demo category examples.', 'P2')
screen('category-create','New category',['settings-category-create'],
 'Create a named chore category.',
 'The short name sheet matches the task.',
 'Save applies a valid name. Cancel returns without changes.',
 'A category name and visible Save are sufficient.',
 'Retain the native form and standard sheet controls.',
 'Keep the form. Use a persistent Category name label.', 'Keep')
screen('category-edit','Rename category',['settings-category-rename'],
 'Rename an existing category.',
 'The short sheet suits the task. A family-wide effect explanation can prevent surprises.',
 'Save applies the name. Delete remains a separate confirmed action.',
 'Keep the current name visible and editable.',
 'Retain the native sheet and text field.',
 'Add a short family-wide effect explanation. Keep deletion separate.', 'Keep')
screen('tokens','API token list and empty state',['settings-tokens-populated','tokens','settings-tokens'],
 'Manage credentials used by other apps.',
 'The empty state offers a plus-based next step but does not explain why a family admin needs a token.',
 'Plus opens a name sheet. Existing tokens can be deleted.',
 'Explain the purpose and the effect of deletion before creation.',
 'Native list and empty-state controls are suitable.',
 'Explain that tokens connect other apps and act like passwords. Keep the token list in family administration.')
screen('token-create','Create API token',['token-form','settings-token-create'],
 'Create a token with an identifiable name.',
 'The name-only sheet is concise. It does not explain the one-time reveal before creation.',
 'Save creates the token and opens its one-time value.',
 'Use Create token and explain the next step.',
 'Native sheet controls fit the task. Action-specific wording improves feedback.',
 'Use a clear token name label, Create token, and a one-time reveal notice.')
screen('token-created','Token created and copied',['token-created','settings-token-created','settings-token-copied'],
 'Copy the newly created token before dismissing it.',
 'The existing screen already warns that the token appears once. This is useful.',
 'Copy Token copies the value. Done closes the reveal. The capture does not show a distinct copied confirmation.',
 'Keep the copy action and one-time warning above other content.',
 'Native selectable text and copy controls are suitable. Confirm completion with text.',
 'Retain the one-time reveal. Change the copy label briefly to Copied and announce the result.', 'P2')

screen('calendar-view-menu','Calendar display menu',['calendar-view-menu'],
 'Choose the calendar display.',
 'The menu contains four supported modes. Its icon does not reveal the active mode before the menu opens.',
 'The native menu selects Year, Month, Week, or Day.',
 'Show the active mode and keep Today available.',
 'Keep a native menu or segmented control. Avoid a custom glass panel.',
 'Label the current mode. Preserve all four choices.', 'P2', ['calendar-month'])
screen('chore-repeat-options','Chore recurrence variants',['tasks-repeat-menu','tasks-repeat-monthly','tasks-repeat-custom','tasks-repeat-end-date','tasks-repeat-count'],
 'Set monthly or custom repeat rules and stop conditions.',
 'These variants add a day of month, interval unit, end date, or task limit. The form needs a readable summary as options change.',
 'Native pickers and switches reveal the relevant controls. Both stop conditions remain available.',
 'Show the resulting schedule before Save.',
 'Native form controls are appropriate. Keep custom weekday targets large and label each option plainly.',
 'Retain all recurrence variants. Add a schedule summary and plain labels.', 'P2', ['chore-repeat','chore-repeat-monthly','chore-repeat-custom'])
screen('item-edit-count','Edit inventory item: count',['household-item-edit-count'],
 'Change an item quantity, unit, or threshold.',
 'The stock values are explicit. Large quantities need more than repeated stepper taps.',
 'The stepper changes quantity and threshold. Save applies the values.',
 'Keep the quantity and unit together. Offer direct numeric entry.',
 'Native numeric entry and steppers fit the platform.',
 'Keep the stepper and add precise numeric entry. Retain unit and threshold.', 'P2')
screen('area-empty','Inventory area: empty',['household-area-empty'],
 'Start stock records in a new area.',
 'The empty area keeps its filter and an Add Item action.',
 'Add Item opens the same item form used in populated areas.',
 'The next action is to add the first item.',
 'Keep the native empty explanation and navigation.',
 'Use a clear empty-state message and retain Add item. No broader redesign is needed.', 'Keep')

for entry in entries:
    if entry['key'].startswith('calendar-'):
        entry['priority'] = 'Keep'
        entry['change'] = 'Keep the current Calendar design and interactions, as selected by the user. Only remove arrival-margin text. Earlier layout proposals are superseded.'
        entry['mockups'] = []

profiles=['iphone-final','iphone-repairs','iphone-light','iphone-supplement','iphone-small','iphone-small-final','iphone-small-extra','iphone-contrast','iphone-dark']
all_captures=[]
for profile in profiles:
    for path in sorted((root/'captures'/profile).glob('*.png')):
        all_captures.append(dict(profile=profile,name=path.stem,path=str(path.relative_to(root))))
variant_map={'today':['variant-today'],'calendar-month':['variant-calendar'],'chores':['variant-tasks'],'food':['variant-food'],'household':['variant-household'],'settings':['variant-settings'],'reminders':['variant-reminders'],'chore-create':['variant-chore-form'],'recipe-detail':['variant-recipe'],'cook-ingredients':['variant-cook-ingredients'],'cook-step':['variant-cook-step']}
for entry in entries:
    entry['match'] += variant_map.get(entry['key'], [])
    entry['captures']=[capture for capture in all_captures if capture['name'] in [name.lower() for name in entry['match']]]
    entry['captures'].sort(key=lambda capture: (profiles.index(capture['profile']), entry['match'].index(capture['name']) if capture['name'] in entry['match'] else 999))
    entry['mockups']=[key for key in entry['mockups'] if (root/'mockups'/f'{key}.png').exists()]
(root/'audit-register.json').write_text(json.dumps(entries,indent=2,ensure_ascii=False)+'\n')

intro='''# Family Hub iPhone design audit

20 October 2026 · Expert review · iPhone only

The app uses a sound native foundation. Most screens use familiar lists, forms, sheets, and system navigation. The largest problems concern priority, discoverability, and clear action labels. A blanket glass redesign does not address those problems.

The most useful changes are: keep the meal plan above chores on Today and bring near departures into view; expose Recipes and Inventory earlier; make Start cooking obvious; clarify chore setup versus completion; and improve notifications and role management.

[Open the visual comparison report](index.html). Each comparison uses real simulator captures and separate, labelled proposal images. [Browse all mockups](mockups/manifest.json).

[Review the native implementation and validation](implementation/index.html).

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

'''
intro=intro.replace('20 October 2026 · Expert review · iPhone only', f'20 October 2026 · Expert review · iPhone only\n\n{len(entries)} screen/state reviews · {len(all_captures)} simulator captures · {len(list((root / "mockups").glob("*.png")))} static mockups')
md=[intro,'| Screen | Priority | Captures | Mockup |\n|---|---|---|---|']
for e in entries:
    captures=' · '.join(f'[{c["profile"]}: {c["name"]}]({c["path"]})' for c in e['captures'][:2]) or 'See coverage notes'
    mocks=' · '.join(f'[{m}](mockups/{m}.png)' for m in e['mockups']) or 'Keep the system screen'
    md.append(f'| [{e["title"]}](#{e["key"]}) | {e["priority"]} | {captures} | {mocks} |')
for e in entries:
    md.extend([f'\n<a id="{e["key"]}"></a>\n\n## {e["title"]}\n',f'**Goal:** {e["goal"]}',f'\n**User friendly:** {e["usability"]}',f'\n**Interactions:** {e["interaction"]}',f'\n**Main information and action:** {e["primary"]}',f'\n**iOS 26:** {e["apple"]}',f'\n**Proposal ({e["priority"]}):** {e["change"]}'])
    if e['captures']:md.append('\n**Evidence:** '+', '.join(f'[{c["profile"]} / {c["name"]}]({c["path"]})' for c in e['captures']))
    if e['mockups']:md.append('\n**Mockups:** '+', '.join(f'[{m}](mockups/{m}.png)' for m in e['mockups']))
md.append('\n## Coverage and checks\n\nSee [coverage.md](coverage.md) for the capture matrix, test results, remaining gaps, and implementation checks.\n')
(root/'README.md').write_text('\n'.join(md).rstrip()+'\n')

css='''*{box-sizing:border-box}html{scroll-behavior:smooth}body{margin:0;background:#f7f1e7;color:#343c31;font:16px/1.5 -apple-system,BlinkMacSystemFont,Arial,sans-serif}header{background:#334b3b;color:white;padding:55px max(28px,calc((100vw - 1180px)/2));}header h1{font-size:42px;line-height:1.1;letter-spacing:-1px;margin:12px 0}header p{max-width:800px;color:#e6eddd}a{color:#3c654c}header a{color:#d6e5bd}main{max-width:1180px;margin:auto;padding:30px 24px}.intro{max-width:900px}.tags{font-size:12px;font-weight:650;letter-spacing:.8px;text-transform:uppercase}.index{display:grid;grid-template-columns:repeat(3,1fr);gap:7px;margin:25px 0 50px}.index a{font-size:14px;text-decoration:none}.screen{background:#fffcf6;border:1px solid #e6dfd2;border-radius:24px;padding:28px;margin:24px 0 42px;scroll-margin-top:20px}.screen h2{font-size:28px;margin:0 0 6px;letter-spacing:-.6px}.priority{font-size:12px;letter-spacing:.7px;font-weight:700;color:#8b503c}.goal{color:#555563}.comparison{display:grid;grid-template-columns:1fr 1fr;gap:28px;background:#eee7db;border-radius:16px;padding:25px;margin-top:22px}.comparison figure{margin:0;text-align:center}.comparison img{aspect-ratio:402/874;width:100%;max-width:300px;height:auto;max-height:680px;object-fit:contain;vertical-align:top;border-radius:8px;box-shadow:0 3px 14px #0001}.comparison figcaption{font-size:12px;font-weight:650;letter-spacing:.6px;margin:0 0 14px;text-transform:uppercase}.comparison .missing{height:250px;display:grid;place-items:center;color:#666}.notes{display:grid;grid-template-columns:1fr 1fr;gap:10px 26px;margin-top:22px}.notes p{margin:8px 0}.notes strong{display:block;font-size:13px;color:#656571;margin-bottom:3px}.links{font-size:13px;margin-top:18px;overflow-wrap:anywhere}.variants{display:flex;flex-wrap:wrap;gap:16px;margin-top:20px}.variants img{aspect-ratio:402/874;width:140px;height:auto;border:1px solid #ddd;border-radius:6px}.variants figure{margin:0}.variants figcaption{font-size:12px;max-width:140px}.foot{font-size:13px;color:#62626c;margin:30px 0} @media(max-width:700px){header h1{font-size:32px}.index{grid-template-columns:1fr 1fr}.screen{padding:18px}.comparison{gap:12px;padding:12px}.notes{grid-template-columns:1fr}.comparison img{border-radius:4px}}@media print{.index{display:none}.screen{break-inside:avoid}.comparison img{max-height:380px}.screen{margin:12px 0}header{padding:25px}.notes{font-size:12px}}'''
escape=html.escape
parts=[f'<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Family Hub · iPhone design audit</title><style>{css}</style><header><div class="tags">Family Hub · iPhone only · 20 October 2026</div><h1>A clearer day.<br>A more useful Family Hub.</h1><p>Real simulator screenshots, a screen-by-screen audit, and static proposals. The main changes improve priority, discoverability, and feedback.</p><a href="implementation/index.html">Native implementation · 21 October</a> · <a href="fidelity/index.html">Design fidelity follow-up</a> · <a href="README.md">Full Markdown report</a> · <a href="coverage.md">Evidence and coverage</a> · <a href="redesign-guidance.md">Redesign rules · Warm Sage Cosier</a> · <a href="navigation-options.html">Food and Household alternatives</a> · <a href="style-options.html">Compare visual styles</a> · <a href="cosy-sage.html">Cosier Warm Sage</a></header><main><div class="intro"><p><strong>Selected theme: Warm Sage Cosier.</strong> Proposed screens now use the warmer cream, sage, and clay palette. Calendar stays marked Keep current. All 68 source mockups follow the requested <a href="https://github.com/emilkowalski/skills/blob/main/skills/apple-design/SKILL.md">Apple Design</a> and <a href="https://github.com/nextlevelbuilder/ui-ux-pro-max-skill">UI/UX Pro Max</a> skills. <a href="redesign-guidance.md">Read the design rules and motion specification.</a></p><h2>The native foundation works. The task hierarchy needs attention.</h2><p>Keep native controls, five tabs, and all existing features. Keep the meal plan above chores and show near departures. Expose Recipes and Inventory. Make cooking and shared settings easier to understand.</p><p>Each “Current” image is a real simulator capture. Each “Proposal” is a separate static mockup. The images do not prove motion, VoiceOver behavior, or production data.</p></div><nav class="index">']
for e in entries:parts.append(f'<a href="#{e["key"]}">{escape(e["title"])}</a>')
parts.append('</nav>')
for e in entries:
    before=e['captures'][0] if e['captures'] else None
    after=e['mockups'][0] if e['mockups'] else None
    parts.append(f'<section class="screen" id="{e["key"]}"><span class="priority">{e["priority"]}</span><h2>{escape(e["title"])}</h2><p class="goal">{escape(e["goal"])}</p><div class="comparison"><figure><figcaption>Current · {escape(before["profile"]) if before else "Capture gap"}</figcaption>')
    parts.append(f'<a href="{before["path"]}"><img width="402" height="874" loading="lazy" src="{before["path"]}" alt="Current {escape(e["title"])}"></a>' if before else '<div class="missing">No simulator capture. See coverage notes.</div>')
    parts.append('</figure><figure><figcaption>Revised proposal · Static mockup</figcaption>')
    parts.append(f'<a href="mockups/{after}.png"><img width="402" height="874" loading="lazy" src="mockups/{after}.png" alt="Proposed {escape(e["title"])}"></a>' if after else '<div class="missing">Keep the native system screen.</div>')
    parts.append('</figure></div><div class="notes">')
    for label,field_name in [('User friendly','usability'),('Interactions','interaction'),('Main information and action','primary'),('Apple iOS 26','apple')]:parts.append(f'<p><strong>{label}</strong>{escape(e[field_name])}</p>')
    parts.append(f'</div><p><b>Proposed change:</b> {escape(e["change"])}</p>')
    if len(e['mockups'])>1:
        parts.append('<div class="variants">')
        for m in e['mockups'][1:]:parts.append(f'<figure><a href="mockups/{m}.png"><img width="402" height="874" loading="lazy" src="mockups/{m}.png" alt="Proposal {m}"></a><figcaption>{m}</figcaption></figure>')
        parts.append('</div>')
    parts.append('<div class="links"><b>All captures:</b> '+(' · '.join(f'<a href="{c["path"]}">{c["profile"]} / {c["name"]}</a>' for c in e['captures']) or 'See coverage notes')+'</div></section>')
parts.append('<p class="foot">Apple references: <a href="https://developer.apple.com/documentation/technologyoverviews/adopting-liquid-glass">Adopting Liquid Glass</a> · <a href="https://developer.apple.com/design/human-interface-guidelines/accessibility">Accessibility</a> · <a href="https://developer.apple.com/design/human-interface-guidelines/tab-bars">Tab bars</a>. Expert review, not a user study. No product code changes.</p></main></html>')
(root/'index.html').write_text(''.join(parts))
covered={c['path'] for e in entries for c in e['captures']}
unmapped=[c for c in all_captures if c['path'] not in covered]
(root/'unmapped-captures.json').write_text(json.dumps(unmapped,indent=2)+'\n')
print(f'{len(entries)} screen reviews; {len(all_captures)} captures; {len(unmapped)} unmapped captures')

"""Build the remaining-screen comparison from native iPhone captures."""
import html
import json
from pathlib import Path

root = Path(__file__).resolve().parents[1] / 'alignment'
rows = [
    ('Chores', 'chores-list', 'chores', 'The Today entry appears before the repeat rules. Chore completion stays on Today.'),
    ('Chore detail', 'chore-detail', 'chore-detail', 'The current task, repeat rule, history, and Edit action have distinct positions.'),
    ('New chore', 'chore-create', 'chore-create', 'Name, description, people, and schedule follow the approved order.'),
    ('Edit chore', 'chore-edit', 'chore-edit', 'The form preserves the current task, rotation, category, and advanced schedule fields.'),
    ('Weekly repeat rule', 'chore-repeat-weekly', 'chore-repeat', 'Repeat opens as a native child screen. Done returns to the draft. Save writes the chore.'),
    ('Monthly repeat rule', 'chore-repeat-monthly', 'chore-repeat-monthly', 'The native day picker preserves the monthly rule.'),
    ('Custom repeat rule', 'chore-repeat-custom', 'chore-repeat-custom', 'Interval, unit, completion timing, and stop rules remain available.'),
    ('Settings', 'settings-top', 'settings', 'The profile, this iPhone, family administration, and About form separate groups.'),
    ('Notifications', 'notifications-top', 'reminders', 'The page distinguishes device permission from the family reminder schedule. Demo permission status is real simulator state.'),
    ('Family reminder schedule', 'notifications-schedule', 'reminders', 'Both reminder times and the time zone remain visible after a short scroll on a small iPhone.'),
    ('Server', 'server', 'server-config', 'The server form has a labelled address, a clear sign-out notice, and a full-width Connect action.'),
    ('Family name', 'family-name', 'family-name', 'A full-width Save family name action commits the name.'),
    ('Home location', 'home-location', 'home-info', 'The saved location, capture action, and arrival setting remain available. Calendar and Today do not show the margin.'),
    ('Family message', 'family-message-ready', 'broadcast', 'The composer has one message field, a character count, a permission notice, and a full-width Send action.'),
    ('Message result', 'family-message-result', 'broadcast-result', 'The result opens on a separate screen. Back to message preserves the draft. Notification help opens Notifications.'),
    ('API tokens', 'tokens-populated', 'tokens', 'Token rows show names and dates. Opening a row shows its access-management actions.'),
    ('New token', 'token-create-ready', 'token-create', 'The form uses a name, a one-time-value explanation, and a full-width Create action.'),
    ('Created token', 'token-created', 'token-created', 'The one-time value is selectable. Copy is full width. The list never exposes the value after dismissal.'),
    ('Family members', 'family-members', 'members', 'People have avatars or soft person icons. Admin badges and the current user remain clear.'),
    ('Member role', 'member-role', 'member-role', 'Member and Admin are draft selections. Save asks for confirmation. The app protects the current user’s role.'),
    ('Categories', 'categories', 'categories', 'Each category opens Rename and Delete directly. Swipe actions remain available.'),
    ('New category', 'category-create', 'category-create', 'The native sheet has a labelled name, Cancel, and Save.'),
    ('Edit category', 'category-edit', 'category-edit', 'The sheet groups the name and family impact. Delete asks for confirmation with the category name.'),
    ('Weekly meal plan', 'meal-week-linked-recipe', 'meal-week', 'The Friday–Thursday range has compact meal rows and a Today action. Linked meals show a recipe thumbnail with a book fallback. Meals without a link stay text-only. Other weeks use the truthful Meal plan title.'),
    ('Meal editor', 'meal-recipe-selected', 'meal-edit', 'Recipe search keeps its metadata and selection. An explicit Use name only action follows the results.'),
    ('Recipe search in a meal', 'meal-search-keyboard', 'meal-add', 'Typing filters recipes. Selecting a recipe dismisses the keyboard. Typing alone preserves the recipe link.'),
    ('Home weather', 'weather', 'weather', 'The weather card centres the condition, temperature, high, low, and advice. Hourly data and provider attribution remain available.'),
    ('Event detail', 'event-details', 'event', 'Date, location, departure, and full details use separate cards. The Calendar grids and gestures do not change.'),
    ('Inline shopping rename', 'shopping-inline-rename', 'shopping-rename', 'Cancel stays beside the name. Done uses the native keyboard toolbar. Errors keep the draft.'),
    ('Shopping Undo hint', 'shopping-undo', 'shopping-pending', 'Each checked item shows its own countdown and Undo. It hides after five seconds and a confirmed server write.'),
    ('Count stock', 'inventory-count', 'item-edit-count', 'Count and Level use a segment at standard text sizes. One control combines the editable count and stock buttons.'),
    ('Level stock', 'inventory-level', 'item-edit-level', 'Precise percentage entry and the low-stock threshold remain editable. The compact threshold label stays readable on small screens.'),
    ('New inventory item', 'inventory-create-keyboard', 'item-create', 'The new-item form retains name, mode, threshold, unit, and validation. The keyboard remains native.'),
]
root.mkdir(parents=True, exist_ok=True)
manifest = json.loads((root / 'captures/manifest.json').read_text())
by_name = {record['screen']: record for record in manifest}
style = '''*{box-sizing:border-box}body{margin:0;background:#f7f1e7;color:#343c31;font:17px/1.5 system-ui}main{max-width:1250px;margin:auto;padding:24px}a{color:#3c654c}header{max-width:880px}h1{font-size:clamp(30px,5vw,48px);line-height:1.1}article{border-top:1px solid #d9d4c7;margin-top:36px;padding-top:16px}.pair,.gallery{display:flex;gap:20px;flex-wrap:wrap}figure{margin:0;max-width:100%;width:300px}img{width:100%;height:auto;border-radius:20px}figcaption{margin:8px 0;overflow-wrap:anywhere}.note{background:#fffcf6;padding:20px;border-radius:24px}input{width:100%;max-width:500px;padding:14px;font:inherit;border:1px solid #646357;border-radius:14px}footer{margin-top:40px}button{font:inherit}.gallery figure{width:230px}'''
parts = [f'''<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Family Hub · remaining mockup alignment</title><style>{style}</style><main><header><a href="../index.html">Original audit</a> · <a href="../fidelity/index.html">Previous follow-up</a><h1>The remaining Warm Sage screens</h1><p>21 October 2026 · Native iPhone simulator review</p><p>All nine areas from the remaining-gap review appear below. Each pair shows the approved mockup and the app.</p><p class="note">Native controls, text size, dates, phone height, and system chrome differ from the static proposals. Calendar retains its approved native design. This report documents the changed screens; the earlier audit remains the full app inventory.</p><p><a href="README.md">Changes and evidence boundaries</a> · <a href="screen-checklist.md">Screen checklist</a> · <a href="checks.json">Test evidence</a> · <a href="captures/manifest.json">Capture manifest</a></p><label for="filter">Find a screen or state</label><br><input id="filter" type="search" placeholder="For example: role, repeat, keyboard, dark"></header>''']
checklist = ['# Remaining screen checklist', '', '21 October 2026. Read this with the [native comparison](index.html).', '', 'The table maps each changed screen to the approved mockup and fresh simulator evidence. It records functional and layout alignment, not identical pixels.', '', '| Screen | Approved mockup | Native evidence | Review |', '|---|---|---|---|']
used = set()
for title, capture, mockup, note in rows:
    for path in [root / 'captures' / f'{capture}.png', root.parent / 'mockups' / f'{mockup}.png']:
        if not path.exists(): raise FileNotFoundError(path)
    used.add(capture)
    parts.append(f'<article data-search="{html.escape((title + " " + note).lower())}"><h2>{html.escape(title)}</h2><p>{html.escape(note)}</p><div class="pair"><figure><figcaption>Approved mockup</figcaption><a href="../mockups/{mockup}.png"><img loading="lazy" src="../mockups/{mockup}.png" alt="{html.escape(title)} mockup"></a></figure><figure><figcaption>Native app</figcaption><a href="captures/{capture}.png"><img loading="lazy" src="captures/{capture}.png" alt="{html.escape(title)} in the app"></a></figure></div></article>')
    checklist.append(f'| {title} | [{mockup}](../mockups/{mockup}.png) | [{capture}](captures/{capture}.png) | {note} |')
parts.append('<h2>More states and accessible layouts</h2><p>The following captures include keyboards, confirmations, stop rules, large text, and dark appearance.</p><div class="gallery">')
for name in sorted(by_name.keys() - used):
    parts.append(f'<figure data-search="{html.escape(name.replace("-", " "))}"><figcaption>{html.escape(name)}</figcaption><a href="captures/{name}.png"><img loading="lazy" src="captures/{name}.png" alt="{html.escape(name)}"></a></figure>')
parts.append('</div>')
small_manifest = root / 'small-captures/manifest.json'
small_records = json.loads(small_manifest.read_text()) if small_manifest.exists() else []
if small_records:
    parts.append('<h2>Small iPhone checks</h2><p>Fresh iPhone SE captures show the keyboard, form labels, and interaction regressions.</p><div class="gallery">')
    for record in sorted(small_records, key=lambda item: item['screen']):
        name = record['screen']
        parts.append(f'<figure data-search="small iphone se {html.escape(name.replace("-", " "))}"><figcaption>iPhone SE · {html.escape(name)}</figcaption><a href="small-captures/{name}.png"><img loading="lazy" src="small-captures/{name}.png" alt="iPhone SE {html.escape(name)}"></a></figure>')
    parts.append('</div>')
parts.append('<footer><p>Review the checklist for deliberate native adaptations and the limits of simulator evidence.</p></footer></main><script>document.getElementById("filter").addEventListener("input",event=>{for(const item of document.querySelectorAll("[data-search]"))item.hidden=!item.dataset.search.includes(event.target.value.toLowerCase());});</script></html>')
(root / 'index.html').write_text(''.join(parts))
(root / 'screen-checklist.md').write_text('\n'.join(checklist) + '\n')
print(f'{len(rows)} comparisons; {len(manifest)} primary captures; {len(small_records)} small-iPhone captures')

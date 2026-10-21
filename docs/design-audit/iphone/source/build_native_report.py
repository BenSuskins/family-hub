"""Build the native implementation comparison from exported simulator captures."""
import html
import json
from pathlib import Path

root = Path(__file__).resolve().parents[1]
output = root / 'implementation'
entries = json.loads((root / 'audit-register.json').read_text())
profiles = ['default', 'stock', 'calendar', 'large-fixed', 'large', 'dark', 'solid']
records = []
for profile in profiles:
    manifest = output / profile / 'manifest.json'
    if manifest.exists():
        for record in json.loads(manifest.read_text()):
            records.append({**record, 'profile': profile, 'path': f'{profile}/{record["file"]}'})

aliases = {
    'household': ['household-overview', 'household-shopping'],
    'shopping-purchased': ['household-shopping-pending', 'household-shopping-undone', 'household-shopping-hidden'],
    'restock': ['inventory-stock-updated', 'household-item-edit-level'],
    'members': ['settings-family-members', 'settings-member-role'],
    'calendar-day': ['calendar-day', 'calendar-day-top', 'calendar-day-overlap'],
    'calendar-week': ['calendar-week', 'calendar-week-overlap'],
    'calendar-year': ['calendar-year', 'calendar-year-top'],
    'calendar-large': ['calendar-accessibility', 'calendar-large-text'],
    'area': ['inventory-stock-updated', 'household-area'],
    'tokens': ['settings-tokens', 'tokens'],
    'cook-step': ['large-cook-actions', 'food-cook-step', 'cook-step'],
    'inventory': ['large-inventory-rows', 'household-inventory', 'inventory'],
}

def candidates(entry):
    names = aliases.get(entry['key'], []) + entry['match']
    matches = [record for name in names for record in records if record['screen'] == name]
    # A later focused capture replaces an earlier state with the same name.
    preferred = {'stock': 0, 'default': 1, 'calendar': 2, 'large-fixed': 3, 'large': 4, 'dark': 5, 'solid': 6}
    matches.sort(key=lambda record: preferred[record['profile']])
    unique = {}
    for record in matches:
        unique.setdefault((record['screen'], record['profile']), record)
    return list(unique.values())

coverage = []
for entry in entries:
    matches = candidates(entry)
    coverage.append({'key': entry['key'], 'title': entry['title'], 'calendar': entry['key'].startswith('calendar-'),
                     'captures': [record['path'] for record in matches], 'mockups': entry['mockups']})
(output / 'coverage.json').write_text(json.dumps(coverage, indent=2) + '\n')
covered = sum(bool(entry['captures']) for entry in coverage)

def figure(path, label):
    return f'<figure><figcaption>{html.escape(label)}</figcaption><a href="{path}"><img loading="lazy" src="{path}" alt="{html.escape(label)}"></a></figure>'

parts = [f'''<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Family Hub · Native Warm Sage Cosier</title><style>
*{{box-sizing:border-box}}body{{margin:0;background:#f7f1e7;color:#343c31;font:17px/1.5 -apple-system,BlinkMacSystemFont,sans-serif}}main{{max-width:1220px;margin:auto;padding:32px 20px}}h1,h2{{font-family:ui-rounded,-apple-system,sans-serif;line-height:1.15}}h1{{font-size:40px}}h2{{font-size:25px}}a{{color:#3c654c}}header,p{{max-width:850px}}.note{{padding:18px;border:1px solid #e6dfd2;border-radius:18px;background:#fffcf6}}.controls{{position:sticky;top:0;padding:12px 0;background:#f7f1e7;z-index:1}}input{{font:inherit;padding:12px;border:1px solid #b0ae9c;border-radius:12px;width:100%;max-width:460px;background:#fffcf6;color:#343c31}}article{{border-top:1px solid #d9d2c5;margin-top:32px;padding-top:20px}}.comparison{{display:flex;flex-wrap:wrap;gap:24px;align-items:flex-start}}figure{{margin:0;width:min(240px,100%)}}figcaption{{font-size:14px;margin-bottom:8px;color:#646357}}img{{display:block;width:100%;height:auto;border:1px solid #e6dfd2;border-radius:18px}}.badge{{font-size:14px;color:#646357}}[hidden]{{display:none!important}}table{{border-collapse:collapse;max-width:100%}}td,th{{padding:10px;border-bottom:1px solid #d9d2c5;text-align:left}}@media(max-width:540px){{h1{{font-size:32px}}.comparison{{gap:12px}}figure{{width:calc(50% - 6px)}}main{{padding:20px 14px}}}}
</style><main><header><p><a href="../fidelity/index.html">Later design fidelity changes and floating tab bar</a></p><a href="../index.html">← Original audit and approved mockups</a><h1>Warm Sage Cosier in the app · PR #54 baseline.</h1><p>21 October 2026 · Native iPhone screenshots · {covered} of {len(entries)} audit groups have a native capture.</p><p>The iPhone app uses the approved theme and overview screens. Today places meals before the agenda and chores. Calendar keeps its current grids, controls, and gestures.</p><p class="note">These are simulator captures of SwiftUI screens. The approved mockups remain beside them. Data, dates, phone height, and native system controls can differ. Add and edit forms, inline rename, five-second Undo, stock controls, and cooking use real app interactions.</p><p><a href="README.md">Implementation and validation notes</a> · <a href="checks.json">Test evidence</a> · <a href="coverage.json">Screen coverage</a></p></header><div class="controls"><label for="filter">Find a screen</label><br><input id="filter" type="search" placeholder="For example: meal, shopping, settings"></div>''']
for entry, item in zip(entries, coverage):
    matches = candidates(entry)
    state = 'Calendar kept' if item['calendar'] else ('Captured' if matches else 'Capture gap')
    parts.append(f'<article data-search="{html.escape((entry["title"]+" "+entry["key"]).lower())}"><span class="badge">{state}</span><h2>{html.escape(entry["title"])}</h2><p>{html.escape(entry["change"])}</p><div class="comparison">')
    if entry['mockups']:
        parts.append(figure('../mockups/' + entry['mockups'][0] + '.png', 'Approved mockup'))
    if not matches:
        parts.append('<p>No current capture for this group. See the coverage notes.</p>')
    else:
        selected = matches[:2]
        for profile in ['large-fixed', 'large', 'solid']:
            variant = next((record for record in matches if record['profile'] == profile), None)
            if variant and variant not in selected: selected.append(variant)
        for record in selected[:4]:
            parts.append(figure(record['path'], 'Native · ' + record['profile'] + ' · ' + record['screen']))
    parts.append('</div></article>')
parts.append('''<footer><p class="note">System-owned sign-in, photo selection, notification delivery, and physical-device VoiceOver remain release checks. The legacy Purchased view and automatic restock sheet are retired. Their replacements appear in the purchase feedback and Inventory comparisons.</p></footer></main><script>document.getElementById('filter').addEventListener('input',event=>{const query=event.target.value.toLowerCase();for(const article of document.querySelectorAll('article'))article.hidden=!article.dataset.search.includes(query);});</script></html>''')
(output / 'index.html').write_text('\n'.join(parts))
print(f'{covered}/{len(entries)} audit groups, {len(records)} native captures')

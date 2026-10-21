"""Build the focused native comparison from simulator test captures."""
import html
import json
from pathlib import Path

root = Path(__file__).resolve().parents[1] / 'fidelity'
rows = [
    ('Welcome', 'welcome', 'welcome'),
    ('Join your family', 'join-family', 'setup-help'),
    ('Invalid address', 'welcome-invalid-address', 'server-error'),
    ('Recipe editor summary', 'recipe-editor-summary', 'recipe-edit'),
    ('Ingredient draft', 'recipe-ingredients-draft', 'recipe-content-edit'),
    ('Cooking progress', 'cooking-step-progress', 'cook-step'),
    ('Final cooking step', 'cooking-last-step', 'cook-finish'),
    ('Inventory icons', 'inventory-soft-icons', 'inventory'),
    ('Inventory item actions', 'inventory-area-actions', 'area'),
    ('Low-stock filter', 'inventory-low-filter', 'area'),
    ('Empty Inventory area', 'inventory-empty-area', 'area-empty'),
    ('Today with floating tabs', 'today-floating-tabs', 'today'),
    ('Final row above the tab bar', 'recipes-floating-tabs-last-row', 'recipes'),
    ('Keyboard hides the tab bar', 'shopping-keyboard-no-tabs', 'shopping-rename'),
    ('Large-text method editor', 'large-method-editor', None),
    ('Large-text cooking progress', 'large-cooking-progress', None),
    ('Large-text Inventory actions', 'large-inventory-actions', None),
    ('Large-text shopping action', 'large-inventory-shopping-action', None),
]
style = '''body{margin:0;background:#f7f1e7;color:#343c31;font:17px/1.5 system-ui}main{max-width:1250px;margin:auto;padding:24px}a{color:#3c654c}header{max-width:850px}h1{font-size:clamp(30px,5vw,48px);line-height:1.1}article{border-top:1px solid #e6dfd2;margin-top:36px;padding-top:16px}.pair{display:flex;gap:20px;flex-wrap:wrap}figure{margin:0;max-width:100%;width:300px}img{width:100%;height:auto;border-radius:20px}figcaption{margin-bottom:8px}.note{background:#fffcf6;padding:20px;border-radius:24px}input{box-sizing:border-box;width:100%;max-width:500px;padding:14px;font:inherit;border:1px solid #646357;border-radius:14px}footer{margin-top:40px}'''
parts = [f'<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Family Hub · design fidelity</title><style>{style}</style><main><header><a href="../index.html">Original audit</a><h1>Closer to Warm Sage Cosier</h1><p>21 October 2026 · Native iPhone simulator captures</p><p>Onboarding, recipe editing, cooking, Inventory, and the floating tab bar.</p><p class="note">The glass tab bar overlays content. Only its capsule has a material background. The comparison covers the changed screens. It does not claim visual parity for every app screen.</p><p><a href="README.md">Changes and remaining differences</a> · <a href="checks.json">Test evidence</a> · <a href="captures/manifest.json">Capture manifest</a></p><label for="filter">Find a screen</label><br><input id="filter" type="search" placeholder="For example: cooking, Inventory, tabs"></header>']
for title, capture, mockup in rows:
    path = root / 'captures' / (capture + '.png')
    if not path.exists():
        raise FileNotFoundError(path)
    parts.append(f'<article data-search="{html.escape(title.lower())}"><h2>{html.escape(title)}</h2><div class="pair">')
    if mockup:
        parts.append(f'<figure><figcaption>Approved mockup</figcaption><a href="../mockups/{mockup}.png"><img loading="lazy" src="../mockups/{mockup}.png" alt="{html.escape(title)} mockup"></a></figure>')
    parts.append(f'<figure><figcaption>Native app</figcaption><a href="captures/{capture}.png"><img loading="lazy" src="captures/{capture}.png" alt="{html.escape(title)} in the app"></a></figure></div></article>')
parts.append('<footer><p>Native controls and accessible layouts adapt to the device. Physical-device checks remain separate.</p></footer></main><script>document.getElementById("filter").addEventListener("input",event=>{for(const article of document.querySelectorAll("article"))article.hidden=!article.dataset.search.includes(event.target.value.toLowerCase());});</script></html>')
(root / 'index.html').write_text(''.join(parts))
print(f'{len(rows)} screen comparisons')

"""Build a static comparison page for the three visual directions."""
from pathlib import Path
from html import escape

root = Path(__file__).resolve().parents[1]
styles = [
    ('sage', 'Warm Sage · First version', 'The first sage palette, with cool cream surfaces and green emphasis.', '#f5f3ed', '#2c624c'),
    ('cosy', 'Warm Sage · Cosier refinement', 'Oat and cream surfaces, soft rounded headings, warm clay details, and sage actions.', '#f7f1e7', '#3c654c'),
]
screens = [('today','Today'),('food','Food overview'),('household','Household overview'),('shopping-list','Shopping list'),('recipe-detail','Recipe detail'),('today-dark','Today · Dark appearance')]
parts = ['''<!doctype html><html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>Family Hub · Cosier Warm Sage</title><style>
*{box-sizing:border-box}html{scroll-behavior:smooth}body{margin:0;background:#f2f2f4;color:#202124;font:16px/1.5 -apple-system,BlinkMacSystemFont,sans-serif}a{color:#245997}main{max-width:1040px;padding:40px 24px;margin:auto}h1{font-size:44px;line-height:1.1;letter-spacing:-1px;margin:22px 0 16px}h2{font-size:26px;letter-spacing:-.4px}p{max-width:850px;color:#5b5e65}.directions,.comparison{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:18px}.direction{background:white;border:1px solid #dedee4;border-radius:18px;padding:18px}.direction p{font-size:14px;margin-bottom:0}.swatch{display:inline-block;width:22px;height:22px;border-radius:50%;border:1px solid #0002;margin-right:6px;vertical-align:middle}.toolbar{position:sticky;top:0;z-index:2;padding:14px 0;background:#f2f2f4f5;backdrop-filter:blur(12px);display:flex;align-items:center;gap:10px;flex-wrap:wrap}.toolbar label{font-size:14px}select{font:inherit;border:1px solid #b8bbc3;border-radius:10px;padding:10px 32px 10px 12px;background:white;color:#202124;min-height:44px}.jump{font-size:14px;display:flex;gap:16px;flex-wrap:wrap;margin:18px 0}section{margin:36px 0 50px;scroll-margin-top:90px}figure{margin:0;text-align:center}figcaption{font-size:14px;font-weight:600;margin-bottom:12px}.comparison img{width:100%;height:auto;aspect-ratio:402/874;border-radius:12px;border:1px solid #d4d5db;box-shadow:0 5px 15px #00000006;vertical-align:top}body.filtered .comparison{grid-template-columns:repeat(2,minmax(0,340px));justify-content:center}figure[hidden]{display:none}.note{font-size:14px}footer{border-top:1px solid #d8d9df;padding-top:20px;margin-top:40px} @media(max-width:850px){.directions{grid-template-columns:1fr 1fr}.comparison{gap:10px;grid-template-columns:1fr 1fr}h1{font-size:36px}main{padding:28px 16px}body.filtered .comparison{grid-template-columns:1fr 1fr}} @media(prefers-reduced-motion:reduce){html{scroll-behavior:auto}}
</style><main><a href="index.html">← Back to the audit</a><h1>A cosier Warm Sage.</h1><p>The selected Warm Sage direction has a warmer, softer refinement. Compare the first version with the new treatment across the same five screens and dark Today.</p><p class="note">The agreed content and navigation stay fixed: overview screens, meals before chores, inline shopping rename, five-second Undo, and no early-arrival text. Calendar keeps its current design. These images do not change the app.</p><div class="directions">''']
for key,title,description,background,accent in styles:
    parts.append(f'<div class="direction"><span class="swatch" style="background:{background}"></span><span class="swatch" style="background:{accent}"></span><h3>{title}</h3><p>{description}</p></div>')
parts.append('''</div><div class="toolbar"><label for="direction">Compare:</label><select id="direction"><option value="all">Both versions</option><option value="cosy">First version and refinement</option></select><span class="note">Select any image for full size.</span></div><nav class="jump">''')
for key,title in screens:
    parts.append(f'<a href="#{key}">{title}</a>')
parts.append('</nav>')
for key,title in screens:
    parts.append(f'<section id="{key}"><h2>{title}</h2><div class="comparison">')
    for style,label,*_ in styles:
        source=f'style-options/sage-{key}.png' if style=='sage' else f'cosy-sage/cosy-{key}.png'
        parts.append(f'<figure data-style="{style}"><figcaption>{label}</figcaption><a href="{source}"><img loading="lazy" width="402" height="874" src="{source}" alt="{escape(title)} in {escape(label)}"></a></figure>')
    parts.append('</div></section>')
parts.append('''<footer><p>The refinement uses cream for the main canvas, warm clay for small details, and sage for actions. Rounded headings and soft icon backgrounds add warmth without changing the tasks.</p><p class="note">System typography and native interaction patterns remain the implementation target. Both versions include a dark Today sample. Native accessibility and full appearance checks remain implementation work.</p><a href="cosy-sage/README.md">Design notes and static checks</a></footer></main><script>document.getElementById('direction').addEventListener('change',event=>{const value=event.target.value;document.body.classList.toggle('filtered',value!=='all');for(const figure of document.querySelectorAll('figure[data-style]'))figure.hidden=value!=='all'&&figure.dataset.style!=='sage'&&figure.dataset.style!==value;});</script></html>''')
(root/'cosy-sage.html').write_text(''.join(parts))

"""Export named simulator evidence and build contact sheets."""
import json
import re
import shutil
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

root = Path(__file__).resolve().parents[1]
export = Path(sys.argv[1])
profile = sys.argv[2]
destination = root / 'captures' / profile
destination.mkdir(parents=True, exist_ok=True)
manifest = json.loads((export / 'manifest.json').read_text())
records = []
for test in manifest:
    for attachment in test['attachments']:
        original = attachment['exportedFileName']
        if not original.endswith('.png'):
            continue
        name = attachment['suggestedHumanReadableName'].split('_0_')[0]
        if not re.fullmatch(r'[A-Za-z0-9 -]+', name):
            continue
        if name.startswith(('Screenshot', 'Failure', 'Automatic')):
            continue
        name = name.lower().replace(' ', '-')
        target = destination / (name + '.png')
        shutil.copyfile(export / original, target)
        records.append({'screen': name, 'path': str(target.relative_to(root)), 'test': test.get('testName', test.get('testIdentifier', '')), 'device': attachment['deviceName'], 'timestamp': attachment['timestamp']})
manifest_path = root / 'captures' / (profile + '-manifest.json')
previous = json.loads(manifest_path.read_text()) if manifest_path.exists() else []
merged = {record['screen']: record for record in previous}
merged.update({record['screen']: record for record in records})
manifest_path.write_text(json.dumps(list(merged.values()), indent=2) + '\n')
images = sorted(destination.glob('*.png'))
font = ImageFont.truetype('/System/Library/Fonts/Supplemental/Arial.ttf', 15)
for offset in range(0, len(images), 6):
    subset = images[offset:offset + 6]
    sheet = Image.new('RGB', (1020, 1540), '#e8e9ed')
    draw = ImageDraw.Draw(sheet)
    for index, path in enumerate(subset):
        screenshot = Image.open(path).convert('RGB')
        screenshot.thumbnail((320, 710))
        left = index % 3 * 340 + (340 - screenshot.width) // 2
        top = index // 3 * 770 + 34
        sheet.paste(screenshot, (left, top))
        draw.text((index % 3 * 340 + 8, index // 3 * 770 + 10), path.stem, font=font, fill='#111111')
    sheet.save(root / 'captures' / f'{profile}-contact-{offset // 6 + 1:02}.jpg', quality=90)
print(f'{profile}: {len(records)} captures, {(len(images) + 5) // 6} contact sheets')

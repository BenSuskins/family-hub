"""Build contact sheets for visual review of the static proposals."""
import sys
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont
root = Path(__file__).resolve().parents[1]
profile = sys.argv[1] if len(sys.argv) > 1 else 'mockups'
directory = root / 'mockups' if profile == 'mockups' else root / 'captures' / profile
paths = sorted(directory.glob('*.png'))
font = ImageFont.truetype('/System/Library/Fonts/Supplemental/Arial.ttf', 15)
for offset in range(0, len(paths), 6):
    sheet = Image.new('RGB', (1020, 1540), '#e8e9ed')
    draw = ImageDraw.Draw(sheet)
    for index, path in enumerate(paths[offset:offset + 6]):
        image = Image.open(path).convert('RGB')
        image.thumbnail((320, 710))
        left = index % 3 * 340 + 10
        top = index // 3 * 770 + 34
        sheet.paste(image, (left, top))
        draw.text((left, top - 24), path.stem, font=font, fill='#111111')
    target = directory / f'contact-{offset // 6 + 1:02}.jpg' if profile == 'mockups' else root / 'captures' / f'{profile}-contact-{offset // 6 + 1:02}.jpg'
    sheet.save(target, quality=90)
print(f'{profile}: {len(paths)} images on {(len(paths) + 5) // 6} contact sheets')

"""Check local links, register coverage, and PNG integrity."""
import json
from html.parser import HTMLParser
from pathlib import Path
from urllib.parse import unquote
from PIL import Image

root = Path(__file__).resolve().parents[1]
class Links(HTMLParser):
    def __init__(self):
        super().__init__()
        self.links = []
        self.identifiers = set()
    def handle_starttag(self, tag, attributes):
        attributes = dict(attributes)
        if 'id' in attributes:
            self.identifiers.add(attributes['id'])
        for name in ('href', 'src'):
            if name in attributes:
                self.links.append(attributes[name])
parser = Links()
parser.feed((root / 'index.html').read_text())
errors = []
for link in parser.links:
    if link.startswith(('http:', 'https:')):
        continue
    if link.startswith('#'):
        if link[1:] not in parser.identifiers:
            errors.append('Missing anchor: ' + link)
    elif not (root / unquote(link.split('#')[0])).is_file():
        errors.append('Missing file: ' + link)
entries = json.loads((root / 'audit-register.json').read_text())
for entry in entries:
    if not entry['captures']:
        errors.append('Missing screen evidence: ' + entry['key'])
unmapped = json.loads((root / 'unmapped-captures.json').read_text())
if unmapped:
    errors.append(f'{len(unmapped)} captures lack an audit entry')
images = list((root / 'captures').glob('*/*.png')) + list((root / 'mockups').glob('*.png'))
for path in images:
    with Image.open(path) as image:
        image.verify()
mockups = json.loads((root / 'mockups' / 'manifest.json').read_text())
if any(item['overflow'] for item in mockups):
    errors.append('A mockup has horizontal overflow')
if errors:
    raise SystemExit('\n'.join(errors))
print(f'Passed: {len(entries)} screen reviews, {len(images)} PNG files, local links, audit coverage, and horizontal overflow checks')

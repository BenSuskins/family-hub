"""Check the comparison links, capture manifests, and image files."""
from html.parser import HTMLParser
import json
from pathlib import Path
import re
from urllib.parse import unquote, urlsplit
from PIL import Image

root = Path(__file__).resolve().parents[1] / 'alignment'
class Links(HTMLParser):
    def __init__(self):
        super().__init__()
        self.links = []
    def handle_starttag(self, tag, attributes):
        self.links.extend(value for name, value in attributes if name in ['href', 'src'])

parser = Links()
parser.feed((root / 'index.html').read_text())
links = [(root, link) for link in parser.links]
for path in root.glob('*.md'):
    links.extend((path.parent, link) for link in re.findall(r'\]\(([^)]+)\)', path.read_text()))
for parent, link in links:
    parsed = urlsplit(link)
    if parsed.scheme or not parsed.path:
        continue
    path = parent / unquote(parsed.path)
    if not path.is_file():
        raise SystemExit('Missing local link: ' + str(path))
counts = {}
for folder in ['captures', 'small-captures']:
    directory = root / folder
    records = json.loads((directory / 'manifest.json').read_text())
    files = {path.name for path in directory.glob('*.png')}
    if files != {record['file'] for record in records}:
        raise SystemExit('Capture manifest does not match ' + folder)
    counts[folder] = len(files)
images = list(root.rglob('*.png')) + list(root.rglob('*.jpg'))
for path in images:
    with Image.open(path) as image:
        image.verify()
print(json.dumps({'localLinks': 'passed', 'captureManifests': counts, 'verifiedImages': len(images)}))

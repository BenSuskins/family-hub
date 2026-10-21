import json,re,shutil,sys
from pathlib import Path
from PIL import Image,ImageDraw,ImageFont
export=Path(sys.argv[1]);destination=Path(sys.argv[2]);destination.mkdir(parents=True,exist_ok=True)
records=[]
for test in json.load(open(export/'manifest.json')):
 for attachment in test['attachments']:
  file=attachment['exportedFileName'];name=attachment['suggestedHumanReadableName'].split('_0_')[0]
  if not file.endswith('.png') or not re.fullmatch('[A-Za-z0-9 -]+',name) or name.startswith(('Screenshot','Failure','Automatic')):continue
  name=name.lower().replace(' ','-');shutil.copyfile(export/file,destination/(name+'.png'))
  records.append({'screen':name,'file':name+'.png','test':test.get('testName',test.get('testIdentifier','')),'device':attachment['deviceName'],'timestamp':attachment['timestamp']})
manifest_path=destination/'manifest.json'
previous=json.loads(manifest_path.read_text()) if manifest_path.exists() else []
records=list({record['screen']:record for record in previous+records}.values())
(destination/'manifest.json').write_text(json.dumps(records,indent=2)+'\n')
images=sorted(destination.glob('*.png'));font=ImageFont.truetype('/System/Library/Fonts/Supplemental/Arial.ttf',15)
for offset in range(0,len(images),6):
 sheet=Image.new('RGB',(1020,1540),'#e6dfd2');draw=ImageDraw.Draw(sheet)
 for index,path in enumerate(images[offset:offset+6]):
  screenshot=Image.open(path).convert('RGB');screenshot.thumbnail((320,710));left=index%3*340+(340-screenshot.width)//2;top=index//3*770+34;sheet.paste(screenshot,(left,top));draw.text((index%3*340+8,index//3*770+10),path.stem,font=font,fill='#343c31')
 sheet.save(destination/f'contact-{offset//6+1:02}.jpg',quality=90)
print(len(images),'captures')

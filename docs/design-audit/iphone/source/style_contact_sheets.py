"""Create one review sheet for each visual direction."""
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont
root=Path(__file__).resolve().parents[1]/'style-options'
font=ImageFont.truetype('/System/Library/Fonts/Supplemental/Arial.ttf',16)
keys=['today','food','household','shopping-list','recipe-detail','today-dark']
for style in ['neutral','sage','indigo']:
    sheet=Image.new('RGB',(1020,1540),'#e8e9ed')
    draw=ImageDraw.Draw(sheet)
    for index,key in enumerate(keys):
        picture=Image.open(root/f'{style}-{key}.png').convert('RGB')
        picture.thumbnail((320,710))
        left=index%3*340+10
        top=index//3*770+34
        sheet.paste(picture,(left,top))
        draw.text((left,top-24),f'{style} · {key}',font=font,fill='#202124')
    sheet.save(root/f'contact-{style}.jpg',quality=92)

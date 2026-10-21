"""Make compact overview boards from the rendered proposal images."""
from pathlib import Path
from PIL import Image, ImageDraw, ImageFont

root = Path(__file__).resolve().parents[1]
font_path = '/System/Library/Fonts/Supplemental/Arial.ttf'
def font(size): return ImageFont.truetype(font_path, size)
def paste(board, name, left, top, width):
    picture = Image.open(root / 'images' / (name + '.png')).convert('RGBA')
    picture = picture.resize((width, round(picture.height * width / picture.width)), Image.Resampling.LANCZOS)
    board.paste(picture, (left, top), picture)

calendar = Image.new('RGB', (1800, 1130), '#eae4d8')
draw = ImageDraw.Draw(calendar)
draw.text((52, 34), 'Calendar · Warm Sage Cosier', font=font(38), fill='#343c31')
draw.text((52, 89), 'The same Year, Month, Week, and Day views.', font=font(22), fill='#646357')
for index, (name, label) in enumerate([('calendar-year','Year'),('calendar-month','Month'),('calendar-week','Week'),('calendar-overlap','Day')]):
    left = 52 + index * 430
    draw.text((left, 144), label, font=font(23), fill='#3c654c')
    paste(calendar, name, left, 188, 402)
draw.text((52, 1088), 'Static iPhone proposals · Existing layouts and interactions · System controls are illustrative', font=font(18), fill='#646357')
calendar.save(root / 'calendar-overview.png')

widgets = Image.new('RGB', (1500, 1120), '#eae4d8')
draw = ImageDraw.Draw(widgets)
draw.text((52, 34), 'Widgets · Warm Sage Cosier', font=font(38), fill='#343c31')
draw.text((52, 89), 'Meals and the next event, with chores as an optional focus.', font=font(22), fill='#646357')
draw.text((52, 150), 'Today · medium', font=font(22), fill='#3c654c')
paste(widgets, 'widget-today-medium', 52, 192, 620)
draw.text((52, 510), 'Tonight · small', font=font(22), fill='#3c654c')
paste(widgets, 'widget-tonight', 52, 552, 285)
draw.text((382, 510), 'Up next · small', font=font(22), fill='#3c654c')
paste(widgets, 'widget-next', 382, 552, 285)
draw.text((52, 885), 'Chores · medium', font=font(22), fill='#3c654c')
paste(widgets, 'widget-chores-medium', 52, 927, 364)
draw.text((770, 150), 'Today · large', font=font(22), fill='#3c654c')
paste(widgets, 'widget-today-large', 770, 192, 620)
draw.text((770, 885), 'Lock Screen · next event', font=font(22), fill='#3c654c')
paste(widgets, 'widget-rectangular', 770, 927, 270)
draw.text((770, 1081), 'Static proposals · WidgetKit controls native materials', font=font(18), fill='#646357')
widgets.save(root / 'widget-overview.png')
print('Saved the Calendar and widget overview boards.')

const symbols = {
  back:'m14 5-7 7 7 7', right:'m9 5 7 7-7 7', home:'m3 10 9-7 9 7v11h-6v-7H9v7H3Z',
  calendar:'M4 5h16v16H4ZM4 10h16M8 3v4m8-4v4', chores:'m3 6 2 2 3-4m3 2h10M3 13l2 2 3-4m3 2h10M3 20l2 2 3-4m3 2h10',
  food:'M5 3v7m4-7v7M3 3v5c0 4 8 4 8 0V3M7 11v10M19 3c-4 4-4 9 0 9V3Zm0 9v9', household:'M4 3h16v18H4ZM4 11h16m-9-7v6m0 3v6',
  view:'M3 4h18v16H3ZM3 9h18', day:'M5 3h14v18H5Z', week:'M3 4h18v16H3ZM9 4v16M15 4v16',
  check:'m5 12 4 4L20 5', circle:'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18m-4 9 3 3 5-6',
  clock:'M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18m0 4v5l4 2', pin:'M12 22s7-8 7-13a7 7 0 0 0-14 0c0 5 7 13 7 13Zm0-16a3 3 0 1 0 0 6 3 3 0 0 0 0-6',
  wifi:'M3 9c5-5 13-5 18 0M6 12c3-3 9-3 12 0m-9 3c2-2 4-2 6 0m-3 3h.1', signal:'M3 19v-4m5 4v-8m5 8V7m5 12V3',
  offline:'m3 3 18 18M3 9c3-3 6-4 9-4m4 1c2 0 4 2 5 3M6 12c2-2 4-3 7-2m-4 5c2-2 4-2 6 0m-3 3h.1',
  person:'M12 3a4 4 0 1 0 0 8 4 4 0 0 0 0-8M4 21v-3c0-6 16-6 16 0v3',
  book:'M12 5C8 2 4 3 2 4v16c4-2 7-1 10 1m0-16c4-3 8-2 10-1v16c-4-2-7-1-10 1Zm0 0v16',
  lock:'M6 10h12v11H6ZM8 10V7a4 4 0 0 1 8 0v3', camera:'M3 7h5l2-3h4l2 3h5v14H3Zm9 3a4 4 0 1 0 0 8 4 4 0 0 0 0-8',
  torch:'M7 3h10v5l-3 4v9h-4v-9L7 8ZM7 6h10', message:'M3 4h18v13H9l-6 4Z',
  photo:'M3 4h18v16H3Zm0 12 5-5 5 5 3-3 5 5M16 7h.1', phone:'M6 3 3 6c0 7 8 15 15 15l3-3-5-4-2 2-6-6 2-2Z',
};
const icon = (name, size=22) => `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="${symbols[name] || symbols.calendar}"/></svg>`;
const status = () => `<div class="status"><span>9:41</span><div class="status-right">${icon('signal',16)}${icon('wifi',17)}<span class="battery"></span></div></div><div class="island"></div>`;
const tabs = () => `<div class="tabbar">${[['Today','home'],['Calendar','calendar'],['Chores','chores'],['Food','food'],['Household','household']].map(([title,symbol])=>`<div class="tab ${title==='Calendar'?'on':''}">${icon(symbol)}<span>${title}</span></div>`).join('')}</div><div class="home-indicator"></div>`;
function toolbar(mode, event=false) {
  const back = event ? '' : mode==='Year' ? '' : mode==='Month' ? '2026' : 'Oct';
  return `<div class="navbar ${back?'':'no-back'}"><div class="nav-left">${event?`<button class="glass-control">${icon('back',20)}<span>Back</span></button>`:back?`<button class="glass-control">${icon('back',20)}<span>${back}</span></button>`:''}</div><div class="title">${event?'Event':mode==='Year'||mode==='Month'?'Calendar':'21 October'}</div><div class="nav-right">${event?'':`<div class="glass-control trailing"><button>Today</button><button class="view-icon" aria-label="Calendar view">${icon('view',23)}</button></div>`}</div></div>`;
}
const phone = (content, mode='Month', options={}) => `<div class="phone theme ${options.dark?'dark':''} ${options.small?'small-phone':''}" data-screen="${options.id||''}">${status()}${toolbar(mode, options.event)}<div class="calendar-body ${options.week?'week':''}">${content}</div>${tabs()}${options.menu?modeMenu():''}</div>`;
const weekdays = () => '<div class="weekdays">'+['M','T','W','T','F','S','S'].map(day=>`<span>${day}</span>`).join('')+'</div>';
const monthNames = ['January','February','March','April','May','June','July','August','September','October','November','December'];
function monthDates(month) {
  const first = new Date(Date.UTC(2026,month,1));
  const padding = (first.getUTCDay()+6)%7;
  const count = new Date(Date.UTC(2026,month+1,0)).getUTCDate();
  return [...Array(padding).fill(null),...Array.from({length:count},(_,index)=>index+1)];
}
function miniMonth(month) {
  return `<section class="mini-month"><h2>${monthNames[month]}</h2><div class="mini-grid">${monthDates(month).map(day=>`<span class="${month===9&&day===21?'today-mini':''}">${day||''}</span>`).join('')}</div></section>`;
}
function year() {return `<h1 class="year-title">2026</h1><div class="mini-year">${monthNames.map((_,index)=>miniMonth(index)).join('')}</div><h1 class="year-title next-year">2027</h1>`;}
const monthItems = {
  3:[['Family walk','']], 5:[['School run','blue']], 8:[['Swimming','blue']], 12:[['School run','blue']],
  14:[['Dentist','purple']], 16:[['Family trip','']], 19:[['School run','blue'],['Do the dishes','chore','circle'],['Pasta bake','','food']],
  20:[['Swimming','blue'],['Oatmeal','','food']],
  21:[['Family trip',''],['School run','blue'],['Dentist','purple'],['Lunch with Sam','chore'],['Do the dishes','chore','circle'],['Chicken stir fry','','food']],
  22:[['Coffee','blue'],['Vacuum','chore','circle']], 24:[['Grandma’s','purple']], 28:[['Swimming','blue']], 30:[['Pizza night','','food']],
};
function month(density='normal') {
  const limit = density==='compact'?1:density==='expanded'?9:3;
  let days = monthDates(9);
  if (density==='expanded') days=Array.from({length:21},(_,index)=>index+12).map(day=>day<=31?day:null);
  return `${weekdays()}${density==='expanded'?'':'<h1 class="month-title">October</h1>'}<div class="month-grid ${density==='normal'?'':density}">${days.map(day=>{
    if(!day) return '<div class="month-cell blank"></div>';
    const items=monthItems[day]||[];
    return `<div class="month-cell"><span class="date-num ${day===21?'today-date':''}">${day}</span>${items.slice(0,limit).map(([text,color,symbol])=>`<div class="cal-chip ${color}">${symbol?icon(symbol,9):''}${text}</div>`).join('')}${items.length>limit?`<div class="more-items">+${items.length-limit}</div>`:''}</div>`;
  }).join('')}</div>${density==='compact'?'<h1 class="month-title">November</h1><div class="month-grid compact">'+monthDates(10).slice(0,21).map(day=>`<div class="month-cell ${day?'':'blank'}"><span class="date-num">${day||''}</span></div>`).join('')+'</div>':''}`;
}
function dateStrip(week=false) {
  return `<div class="date-strip">${week?'<span class="axis-spacer"></span>':''}${['M','T','W','T','F','S','S'].map((day,index)=>`<div class="date-choice ${index===2?'selected':''}"><small>${day}</small><b>${index+19}</b></div>`).join('')}</div>`;
}
function untimed(week=false, empty=false) {
  const items=empty?'':`<div class="untimed">Family trip</div><div class="untimed chore">${icon('circle',13)}Do the dishes</div><div class="untimed">${icon('food',13)}Breakfast</div><div class="untimed">${icon('food',13)}Lunch</div>`;
  return `<div class="all-day" style="${empty?'height:36px':''}"><div class="axis-label">All day</div><div class="all-day-items">${week?Array.from({length:7},(_,index)=>`<div class="all-day-column">${index===2?items:index===5?'<div class="untimed">Family walk</div>':''}</div>`).join(''):`<div class="all-day-column">${items}</div>`}</div></div>`;
}
function block(title, color, top, left, width, height, details='') {
  return `<div class="event-block ${color}" style="top:${top}px;left:${left};width:${width};height:${height}px"><strong>${title}</strong>${details}</div>`;
}
function timeline({week=false,overlap=false,empty=false,error=false,small=false}={}) {
  const banner=error?'<div class="error-bar"><span>The hub is unavailable.</span><button>Retry</button></div>':'';
  const lines=Array.from({length:13},(_,index)=>`<div class="hour"><time>${index+8}:00</time><div class="hour-line"></div></div>`).join('');
  let blocks='';
  if(!empty) {
    if(week) {
      blocks=block('School run','',64,'calc(48px + (100% - 48px) / 7 * 2 + 1px)','calc((100% - 48px) / 7 / 2 - 3px)',48)
        +block('Dentist','purple',96,'calc(48px + (100% - 48px) / 7 * 2.5 + 1px)','calc((100% - 48px) / 7 / 2 - 3px)',96)
        +block('Lunch with Sam','orange',256,'calc(48px + (100% - 48px) / 7 * 2 + 1px)','calc((100% - 48px) / 7 - 3px)',64)
        +block('Coffee','',128,'calc(48px + (100% - 48px) / 7 * 3 + 1px)','calc((100% - 48px) / 7 - 3px)',64);
    } else if(overlap) {
      blocks=block('School run','',64,'49px','calc((100% - 48px) / 2 - 3px)',48,'<p>9:00</p><p>Oakwood School</p>')
        +block('Dentist','purple',96,'calc(48px + (100% - 48px) / 2 + 1px)','calc((100% - 48px) / 2 - 3px)',96,'<p>9:30</p><p>High Street Dental</p>');
    } else blocks=block('School run','',64,'49px','calc(100% - 51px)',48,'<p>9:00 · Oakwood School</p>');
    if(!week) blocks+=block('Lunch with Sam','orange',256,'49px','calc(100% - 51px)',64,'<p>12:00</p><p>High Street Café</p>');
  }
  return `${banner}${dateStrip(week)}${untimed(week,empty)}<div class="timeline">${lines}${week?'<div class="week-cols">'+Array(7).fill('<span></span>').join('')+'</div>':''}${blocks}${empty?'<p class="empty-timeline">No timed events</p>':''}<div class="now-line" style="top:108px;${week?'left:calc(48px + (100% - 48px) / 7 * 2);width:calc((100% - 48px) / 7);right:auto;':''}"></div></div>`;
}
function modeMenu() {
  return `<div class="menu-dim"></div><div class="mode-menu">${[['Year','calendar'],['Month','calendar'],['Day','day'],['Week','week']].map(([name,symbol])=>`<div class="menu-row ${name==='Month'?'selected':''}">${icon(symbol,20)}<span>${name}</span>${name==='Month'?`<span class="tick">${icon('check',18)}</span>`:''}</div>`).join('')}</div>`;
}
function accessible(week=false) {
  if(week) return `<div class="agenda"><div class="date-label">Date</div><div class="native-date">21 Oct 2026 ${icon('calendar',23)}</div><h1>Monday 19 October</h1><div class="agenda-item"><div>School run<small>9:00</small><small>Oakwood School</small></div><span class="chevron">${icon('right',18)}</span></div><div class="agenda-item chore"><div>${icon('circle',23)} Do the dishes<small>Chore</small></div></div><div class="agenda-item meal"><div>${icon('food',23)} Pasta bake<small>Dinner</small></div></div><h1>Tuesday 20 October</h1><div class="agenda-item"><div>Swimming lesson<small>16:30</small><small>Leisure centre</small></div><span class="chevron">${icon('right',18)}</span></div><h1>Wednesday 21 October</h1></div>`;
  return `<div class="agenda"><div class="date-label">Date</div><div class="native-date">21 Oct 2026 ${icon('calendar',23)}</div><h1>Wednesday 21 October</h1><div class="agenda-item"><div>Family trip<small>All day</small></div><span class="chevron">${icon('right',18)}</span></div><div class="agenda-item"><div>School run<small>9:00</small><small>Oakwood School</small></div><span class="chevron">${icon('right',18)}</span></div><div class="agenda-item purple"><div>Dentist<small>9:30</small><small>High Street Dental</small></div><span class="chevron">${icon('right',18)}</span></div><div class="agenda-item chore"><div>${icon('circle',23)} Do the dishes<small>Chore</small></div></div><div class="agenda-item meal"><div>${icon('food',23)} Chicken stir fry<small>Dinner</small></div></div>${week?'<h1>Thursday 22 October</h1>':'<p class="date-label" style="color:var(--accent);margin-top:20px">Open day</p>'}</div>`;
}
function eventDetails() {
  return `<div class="event-details"><h1>School run</h1><div class="calendar-source"><span class="source-dot"></span>Family calendar</div><div class="detail-card"><div class="detail-icon">${icon('calendar')}</div><div>Wednesday 21 October<small>9:00–9:45</small></div></div><div class="detail-card"><div class="detail-icon">${icon('home')}</div><div>Oakwood School<small>View location</small></div><span class="chevron">${icon('right',18)}</span></div><div class="detail-card" style="background:var(--warm)"><div class="detail-icon">${icon('clock')}</div><div>Leave at 8:40<small>20 min drive</small></div></div><h2>Details</h2><div class="detail-card notes">Drop off at the main entrance. Bring the book bag and swimming kit.</div></div>`;
}
const mealSymbol = () => `<span class="meal-symbol">${icon('food')}</span>`;
const widgetHead = (title='Today',date='Wed 21 Oct')=>`<div class="widget-head"><span class="widget-title">${title}</span><time>${date}</time></div>`;
const widgetFooter = text=>`<div class="widget-foot">${text}</div>`;
function todayMedium({cached=false,long=false}={}) {
  return `${widgetHead()}<div class="two-columns"><div><div class="eyebrow">Next up</div><div class="time">12:00</div><h2>${long?'Lunch with Grandma and Grandad':'Lunch with Sam'}</h2></div><div><div class="eyebrow meal-label">Tonight</div><h2>${long?'Roasted vegetables with rice and lemon':'Chicken stir fry'}</h2></div></div>${widgetFooter(cached?`${icon('clock',12)} Saved at 8:20`:`${icon('chores',12)} 2 chores due`)}`;
}
function chore(title,person,overdue=false) {return `<div class="widget-chore"><button aria-label="Complete ${title}"><span class="check-circle"></span></button><div><strong>${title}</strong><small class="${overdue?'overdue':''}">${person}${overdue?' · Overdue':''}</small></div></div>`;}
function todayLarge() {
  return `${widgetHead()}<div class="dinner-panel">${mealSymbol()}<div><div class="eyebrow">Tonight</div><h2>Chicken stir fry</h2></div></div><div class="widget-events"><div class="widget-event"><time>12:00</time><span class="event-marker"></span><div><strong>Lunch with Sam</strong><small>High Street Café</small></div></div><div class="widget-event"><time>16:30</time><span class="event-marker"></span><div><strong>Swimming lesson</strong><small>Leisure centre</small></div></div></div><div class="widget-section"><span>Chores</span><span>2 due</span></div>${chore('Do the dishes','Sam')}${chore('Vacuum living room','Alex',true)}${widgetFooter('Tap a chore’s circle to complete it')}`;
}
function tonight(){return `<div class="widget-head"><span class="widget-title">Tonight</span>${mealSymbol()}</div><h2>Chicken<br>stir fry</h2><div class="slot">Dinner · Today</div>`;}
function nextUp(){return `${widgetHead('Next up','')}<h2>12:00</h2><div class="next-title">Lunch with Sam</div><div class="next-location">High Street Café</div>`;}
function meals(){return `${widgetHead('Meal plan')}<div class="meal-list"><div class="meal-line"><small>Breakfast</small><span>Oatmeal</span></div><div class="meal-line"><small>Lunch</small><span>Pasta salad</span></div><div class="meal-line emphasis"><small>Dinner</small><span>Chicken stir fry</span></div></div>`;}
function chores(small=false){return small?`${widgetHead('Chores','') }<div class="count">2</div><div class="count-label">chores due</div><div class="due-label">1 overdue · Alex & Sam</div>`:`${widgetHead('Chores','2 due')}${chore('Do the dishes','Sam')}${chore('Vacuum living room','Alex',true)}${widgetFooter('Today · 21 October')}`;}
function problem(kind) {
  const states={empty:['calendar','Nothing scheduled','Plan a meal or check the full week.'],unavailable:['offline','Can’t reach your hub','Open Family Hub to retry.'],signedout:['lock','Sign in to Family Hub','Open the app to connect your family.']};
  const [symbol,title,copy]=states[kind];
  return `${widgetHead()}<div class="state-copy"><div class="state-icon">${icon(symbol,23)}</div><div><h2>${title}</h2><p>${copy}</p></div></div>`;
}
const widget = (content,classes='',theme='') => `<div class="widget theme ${classes} ${theme}">${content}</div>`;
function accessory(kind) {
  if(kind==='inline') return `<div class="accessory inline">${icon('food',15)}<span>Tonight · Chicken stir fry</span></div>`;
  if(kind==='circular') return `<div class="accessory circular">${icon('chores',15)}<strong>2</strong><small>chores due</small></div>`;
  return `<div class="accessory rectangular"><span class="accessory-mark"></span><div><strong>12:00 Lunch with Sam</strong><p>High Street Café</p></div></div>`;
}
function homeContext() {
  const apps=[['Calendar','calendar'],['Family Hub','home'],['Photos','photo'],['Messages','message']];
  const app = (title,symbol)=>`<div class="app-slot"><div class="app-icon ${title==='Family Hub'?'family':''}">${icon(symbol,30)}</div>${title}</div>`;
  return `<div class="home-phone theme">${status()}<div class="home-widgets">${widget(todayMedium(),'today-medium')}<div class="widget-caption">Family Hub</div><div class="home-two"><div>${widget(tonight(),'small tonight')}<div class="widget-caption">Family Hub</div></div><div>${widget(chores(true),'small chore-small')}<div class="widget-caption">Family Hub</div></div></div><div class="home-apps">${apps.map(([title,symbol])=>app(title,symbol)).join('')}</div></div><div class="page-dots"><span></span><span></span></div><div class="dock">${['phone','message','camera','book'].map(symbol=>`<div class="app-icon">${icon(symbol,30)}</div>`).join('')}</div><div class="home-indicator"></div></div>`;
}
function lockContext(privateMode=false) {
  return `<div class="lock-phone">${status()}<div class="lock-inline">${privateMode?'<div class="lock-date" style="margin-top:0">Wednesday 21 October</div>':accessory('inline')}</div><div class="lock-time">9:41</div><div class="lock-accessories">${privateMode?'<div class="accessory rectangular"><span class="accessory-mark"></span><div><strong>Unlock to see your day</strong><p>Family Hub</p></div></div>':accessory('rectangular')}${accessory('circular')}</div><div class="lock-controls"><span class="lock-control">${icon('torch')}</span><span class="lock-control">${icon('camera')}</span></div><div class="home-indicator"></div></div>`;
}
const proposals=[];
function add(id,title,group,description,render,notes='') {proposals.push({id,title,group,description,render,notes});}
add('calendar-year','Year','Calendar','The same twelve-month overview. Sage replaces the red accent. Month positions and tap targets stay the same.',()=>phone(year(),'Year'));
add('calendar-month','Month','Calendar','The existing scrolling month grid at its default density. Calendar colours remain distinct.',()=>phone(month()));
add('calendar-month-compact','Month · zoomed out','Calendar','The existing minimum zoom. A label and the overflow count remain in each busy date.',()=>phone(month('compact')));
add('calendar-month-expanded','Month · zoomed in','Calendar','The existing expanded grid, scrolled to the middle of October. This is a zoom state, not a new mode.',()=>phone(month('expanded')));
add('calendar-week','Week','Calendar','The same seven columns, all-day area, time axis, and event overlaps. Narrow columns remain narrow.',()=>phone(timeline({week:true}),'Week',{week:true}));
add('calendar-day','Day','Calendar','The same week strip and single-day timeline. The background, text, and selected date use Warm Sage.',()=>phone(timeline(),'Day'));
add('calendar-overlap','Day · overlapping events','Calendar','Events keep their start time, duration, and side-by-side placement. Calendar colours retain their meaning.',()=>phone(timeline({overlap:true}),'Day'));
add('calendar-menu','View menu','Calendar','Keep the existing Year, Month, Day, and Week menu. There is no new tab or filter.',()=>phone(month(),'Month',{menu:true}));
add('calendar-event','Event detail','Calendar','Carry forward the merged event detail design. Keep the date, location, departure, and full details. No arrival margin.',()=>phone(eventDetails(),'Day',{event:true}));
add('calendar-empty','Day · no items','Calendar','Retain the date controls and timeline when the day has no items.',()=>phone(timeline({empty:true}),'Day'));
add('calendar-error','Calendar · load error','Calendar','Keep the existing error strip and Retry action above the saved timeline. Do not present a failure as an empty day.',()=>phone(timeline({error:true,overlap:true}),'Day'));
add('calendar-month-large','Month · large text','Calendar','Keep the existing accessible date picker and agenda fallback. Use the same content colours and clear row spacing.',()=>phone(accessible()));
add('calendar-week-large','Week · large text','Calendar','Keep the existing accessible agenda layout. Dates and items continue vertically through the selected week.',()=>phone(accessible(true),'Week'));
add('calendar-month-dark','Month · dark','Calendar','Warm charcoal, soft cream text, and light sage. Calendar colours retain separate accents.',()=>phone(month(),'Month',{dark:true}));
add('calendar-day-dark','Day · dark','Calendar','The same overlap layout and time axis in the dark palette.',()=>phone(timeline({overlap:true}),'Day',{dark:true}));
add('calendar-small','Day · small iPhone','Calendar','The same controls on a 375-point iPhone. More of the timeline remains below the visible area.',()=>phone(timeline({overlap:true}),'Day',{small:true}));
add('widget-today-medium','Today · medium','Widgets','The next timed event and tonight’s meal share the top level. The chore count stays secondary.',()=>widget(todayMedium(),'today-medium'),'Tap the event area to open Calendar. Tap the meal area to open Meals. Tap the count or background to open Today.');
add('widget-today-large','Today · large','Widgets','Dinner, the next two events, and two chore actions. Meals stay above chores.',()=>widget(todayLarge(),'large today-large'),'Chore circles complete tasks. Chore labels open Today. Event rows open Calendar. The dinner panel opens Meals.');
add('widget-tonight','Tonight · small','Widgets','One answer: what is for dinner? A plain meal symbol works without a recipe photo.',()=>widget(tonight(),'small tonight'),'Tap anywhere to open Meals. This proposal does not require a recipe thumbnail or a new image field.');
add('widget-meals','Meal plan · medium','Widgets','Today’s three meals. Dinner has emphasis, without hiding breakfast or lunch.',()=>widget(meals(),'meals'),'Tap a meal row to open Meals. The current URL opens the Meals destination; it does not select an individual row.');
add('widget-next','Up next · small','Widgets','The next timed event has a large start time, a clear title, and a location.',()=>widget(nextUp(),'small next-up'),'Tap anywhere to open Calendar. This is a start time, not a departure time.');
add('widget-chores-small','Chores · small','Widgets','A count and an overdue label. Keep detailed completion actions in larger widgets.',()=>widget(chores(true),'small chore-small'),'Tap anywhere to open Today. The overdue label uses text as well as colour.');
add('widget-chores-medium','Chores · medium','Widgets','Two clear chore rows with separate completion circles and names.',()=>widget(chores(),'chore-medium'),'Tap a circle to complete the task. Tap the row label to open Today. Retain the existing completion error handling.');
add('widget-today-dark','Today · dark','Appearances','Use the app’s dark tokens on a solid widget surface.',()=>widget(todayMedium(),'today-medium','dark'));
add('widget-today-tinted','Today · tinted','Appearances','A two-tone approximation of a user-selected sage tint. Labels and hierarchy carry the meaning.',()=>widget(todayMedium(),'today-medium','tinted'),'WidgetKit owns the final tint. Preserve the accent and primary groups.');
add('widget-today-clear','Today · clear','Appearances','A glass approximation on a wallpaper sample. Keep readable content without extra glass panels inside it.',()=>`<div class="widget-mount">${widget(todayMedium(),'today-medium','clear')}</div>`,'WidgetKit owns the material, highlights, and vibrancy. This static image is not a native rendering.');
add('widget-solid','Today · reduced transparency','Appearances','Keep the same content on a solid surface when translucency is reduced.',()=>widget(todayMedium(),'today-medium'));
add('widget-large-text','Today · large text','Widget states','Reduce the content to dinner and a summary. Keep readable type instead of shrinking all the rows.',()=>widget(`${widgetHead()}<div class="eyebrow" style="margin-top:13px">Tonight</div><h2>Chicken stir fry</h2>${widgetFooter('2 events · 2 chores due')}`,'large-text-widget'));
add('widget-long-title','Today · long names','Widget states','Allow a third title line. Omit location text before shrinking the title.',()=>widget(todayMedium({long:true}),'long-title'));
add('widget-empty','Today · empty','Widget states','Use a calm empty state. Open the app for the full week or meal plan.',()=>widget(problem('empty'),'problem'));
add('widget-cached','Today · saved data','Widget states','Keep the saved meal and event information. State its age. Do not imply a live countdown.',()=>widget(todayMedium({cached:true}),'cached'),'Disable completion actions in a cached interactive layout until a fresh response arrives.');
add('widget-unavailable','Today · unavailable','Widget states','When no saved data exists, show a connection state instead of zero events.',()=>widget(problem('unavailable'),'problem'));
add('widget-signed-out','Today · signed out','Widget states','Use one direct instruction to restore the widget.',()=>widget(problem('signedout'),'problem'));
add('widget-no-dinner','Today · dinner not planned','Widget states','Keep the next event. Use the meal area to show that dinner needs a plan.',()=>widget(todayMedium().replace('Chicken stir fry','Not planned').replace('meal-label','meal-label'),'today-medium'),'Tap the meal area to open Meals. An empty meal slot is not an empty day.');
add('widget-small-phone','Today · small iPhone size','Widget states','The medium family at 321 × 148 points, for a 375 × 667-point iPhone.',()=>widget(todayMedium(),'small-widget-phone'),'Use fewer lines and keep the footer separate. Native text metrics still need a device check.');
add('widget-chore-saving','Chores · save pending','Widget states','Keep the chore visible while its completion request is pending.',()=>widget(chores().replace('aria-label="Complete Do the dishes"','disabled aria-label="Saving Do the dishes"').replace('<span class="check-circle"></span>',icon('clock',22)).replace('Today · 21 October','Saving…'),'chore-medium'),'Proposed transient presentation. The existing intent must confirm the write before the task disappears.');
add('widget-chore-error','Chores · save failed','Widget states','Keep the unchecked task and show a recovery message.',()=>widget(chores().replace('Today · 21 October','Couldn’t save. Open the app to retry.'),'chore-medium chore-error'),'New persistent error presentation requires state shared between the intent and the timeline provider.');
add('widget-inline','Lock Screen · inline','Lock Screen','A meal name and a familiar symbol. No custom background.',()=>`<div class="accessory-mount">${accessory('inline')}</div>`,'Tap opens Meals. The system renders the accessory in a monochrome style.');
add('widget-rectangular','Lock Screen · rectangular','Lock Screen','The next event and location in two compact lines.',()=>`<div class="accessory-mount">${accessory('rectangular')}</div>`,'Tap opens Calendar. Do not add a departure estimate without a data contract.');
add('widget-circular','Lock Screen · circular','Lock Screen','A chore count with a text label. There is no progress ring that could imply a completion ratio.',()=>`<div class="accessory-mount">${accessory('circular')}</div>`,'Tap opens Today.');
add('widget-home-context','Home Screen · context','Context','The widget sizes on a 430-point iPhone. The wallpaper and app icons are schematic context.',homeContext);
add('widget-lock-context','Lock Screen · context','Context','Meal, next event, and chore count in the accessory positions. The wallpaper is an original CSS illustration.',()=>lockContext());
add('widget-lock-private','Lock Screen · private titles','Context','A proposed privacy option hides meal and event titles while keeping a useful task count.',()=>lockContext(true),'New option: hide titles on the Lock Screen. Configure it through a WidgetKit intent. Unlock behaviour follows the system privacy setting.');
window.proposals=proposals;
const selected=new URLSearchParams(location.search).get('screen')||'calendar-month';
const proposal=proposals.find(item=>item.id===selected);
if(!proposal) throw new Error('Unknown proposal: '+selected);
document.getElementById('artboard').innerHTML=proposal.render();
document.title='Family Hub · '+proposal.title;

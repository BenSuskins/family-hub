const weekEvents = [
  {title:'School run',day:2,start:9,end:9.75,color:'',location:'Oakwood School',column:0,columns:2},
  {title:'Dentist',day:2,start:9.5,end:11,color:'purple',location:'High Street Dental',column:1,columns:2},
  {title:'Lunch with Sam',day:2,start:12,end:13,color:'orange',location:'High Street Café',column:0,columns:1},
  {title:'Swimming lesson',day:2,start:16.5,end:17.5,color:'',location:'Leisure centre',column:0,columns:1},
  {title:'Coffee',day:3,start:10,end:11,color:'',location:'Market café',column:0,columns:1},
];
const weekEntries=[];
const weekIconDown=()=>`<span class="down">${icon('back',12)}</span>`;
function optionStrip(three=false,future=false) {
  const dates=future?[26,27,28,29,30,31,1]:[19,20,21,22,23,24,25];
  const weekdays=three?['TUE','WED','THU']:['M','T','W','T','F','S','S'];
  const shown=three?dates.slice(1,4):dates;
  return `<div class="date-strip">${three?'':'<span class="axis-spacer"></span>'}${shown.map((date,index)=>`<button class="date-choice ${date===(future?28:21)?'selected':''}" aria-label="${date} ${future&&date===1?'November':'October'}"><small>${weekdays[index]}</small><b>${date}</b></button>`).join('')}</div>`;
}
function compactAllDay(three=false,future=false) {
  const count=three?3:7;
  const selected=three?1:2;
  return `<div class="compact-all-day"><span class="compact-label">All day</span><div class="compact-cells">${Array.from({length:count},(_,index)=>{
    const selectedDate=index===selected;
    const weekendWalk=!future&&!three&&index===5;
    const hasItems=selectedDate||weekendWalk;
    const date=weekendWalk?24:future?28:21;
    const items=weekendWalk||future?1:5;
    const title=weekendWalk?'Walk':future?'Dinner':three?'Family trip':'Trip';
    return hasItems?`<button class="compact-cell ${selectedDate?'active':''} has-items" aria-label="${items} all-day items on ${date} October"><span>${title}</span>${items===1?'':`<span class="count">+4 ${weekIconDown()}</span>`}</button>`:'<div class="compact-cell"></div>';
  }).join('')}</div></div>`;
}
function fullAllDay() {
  return untimed(true).replace(`<div class="untimed">${icon('food',13)}Lunch</div>`,`<div class="untimed">${icon('food',13)}Lunch</div><div class="untimed">${icon('food',13)}Dinner</div>`);
}
function optionTimeline({three=false,future=false}={}) {
  const columns=three?3:7;
  const offset=three?1:0;
  const start=future?16:8;
  const events=future?[{title:'Swimming lesson',day:2,start:17.5,end:18.5,color:'',column:0,columns:1}]:weekEvents;
  const hourLines=Array.from({length:Math.min(12,25-start)},(_,index)=>`<div class="hour"><time>${index+start}:00</time><div class="hour-line"></div></div>`).join('');
  const eventBlocks=events.map(event=>{
    const day=event.day-offset;
    const left=`calc(48px + (100% - 48px) / ${columns} * ${day+event.column/event.columns} + 1px)`;
    const width=`calc((100% - 48px) / ${columns} / ${event.columns} - 3px)`;
    return block(event.title,event.color,(event.start-start)*64,left,width,Math.max(44,(event.end-event.start)*64),'');
  }).join('');
  const todayColumn=three?1:2;
  const now=future?'':`<div class="now-line" style="top:108px;left:calc(48px + (100% - 48px) / ${columns} * ${todayColumn});width:calc((100% - 48px) / ${columns});right:auto"></div>`;
  return `<div class="week-canvas"><div class="timeline">${hourLines}<div class="timeline-columns">${Array.from({length:columns},(_,index)=>`<span class="${index===todayColumn?'selected-column':''}"></span>`).join('')}</div>${eventBlocks}${now}</div></div>`;
}
function selectedAgenda() {
  return `<section class="agenda-pane"><header class="pane-header"><div><strong>Wed 21 October</strong><small>4 timed events</small></div><button class="open-day">Open day ${icon('right',13)}</button></header><div class="pane-list">${weekEvents.filter(event=>event.day===2).map(event=>{
    const hour=Math.floor(event.start);
    const minute=event.start%1?'30':'00';
    const endHour=Math.floor(event.end);
    const endMinute=event.end%1===.75?'45':event.end%1===.5?'30':'00';
    return `<div class="pane-row ${event.color}"><div class="pane-time">${hour}:${minute}<br>${endHour}:${endMinute}</div><span class="pane-marker"></span><div class="pane-copy"><strong>${event.title}</strong><small>${event.location}</small></div>${icon('right',14)}</div>`;
  }).join('')}</div></section>`;
}
function weekPhone({compact=true,agenda=false,three=false,dark=false,small=false,future=false}={}) {
  const grid=optionStrip(three,future)+(compact?compactAllDay(three,future):fullAllDay())+optionTimeline({three,future});
  let result=phone(agenda?`<div class="short-week">${grid}</div>${selectedAgenda()}`:grid,'Week',{week:true,dark,small});
  result=result.replace('phone theme','phone week-exploration theme');
  if(agenda) result=result.replace('week-exploration theme','week-exploration agenda-mode theme');
  if(three) result=result.replace('week-exploration theme','week-exploration three-day-mode theme').replace(`<button class="view-icon" aria-label="Calendar view">${icon('view',23)}</button>`,`<button class="density-control" aria-label="Calendar view, 3 days">3 days ${weekIconDown()}</button>`);
  if(future) result=result.replace('week-exploration theme','week-exploration future-week theme').replace('21 October','28 October');
  return result;
}
function expandedAllDay() {
  const rows=[['Family trip','All day','calendar',''],['Do the dishes','Chore','circle','chore'],['Oatmeal','Breakfast','food',''],['Pasta salad','Lunch','food',''],['Chicken stir fry','Dinner','food','']];
  return weekPhone().replace('</div><div class="home-indicator"></div></div>',`</div><div class="home-indicator"></div><div class="sheet-dim"></div><section class="all-day-sheet"><div class="sheet-grip"></div><header class="sheet-header"><div><strong>All-day items</strong><small>Wednesday 21 October · 5 items</small></div><button>Done</button></header>${rows.map(([title,type,symbol,className])=>`<div class="all-day-row ${className}"><div class="type-icon">${icon(symbol,21)}</div><div><strong>${title}</strong><small>${type}</small></div></div>`).join('')}<div class="sheet-indicator"></div></section></div>`);
}
function dayDrillDown() {
  return phone(timeline({overlap:true}),'Day').replace('<span>Oct</span>','<span>Week</span>');
}
function weekMenu() {
  return weekPhone({agenda:true}).replace('</div><div class="home-indicator"></div></div>',`</div><div class="home-indicator"></div><div class="menu-dim"></div><section class="week-layout-menu"><div class="menu-heading">Week layout</div><div class="menu-row">${icon('week',20)}<span>7 days</span></div><div class="menu-row selected">${icon('view',20)}<span>7 days + agenda</span><span class="tick">${icon('check',18)}</span></div><div class="menu-row">${icon('week',20)}<span>3 days</span></div></section></div>`);
}
function addWeek(id,title,group,description,notes,render) {weekEntries.push({id,title,group,description,notes,render});}
addWeek('week-baseline','Existing layout','Compare','The current seven-column layout in the selected theme. The all-day area uses 160 points.','Static layout reference with the same sample events as the alternatives. Narrow event labels are part of the existing layout.',()=>weekPhone({compact:false}));
addWeek('week-compact','A · Compact seven days','Compare','Keep the full week. Collapse all-day items into one 44-point row.','Selected. Gain 116 points for timed events. Tap Trip +4 to inspect all five items. Event names remain narrow.',()=>weekPhone());
addWeek('week-agenda','B · Week with an agenda','Compare','Keep seven days above a readable list for the selected day.','Alternative. Tap a date to select its column and agenda. Tap an event to open its detail. Open day gives the full timeline. Each pane scrolls independently.',()=>weekPhone({agenda:true}));
addWeek('week-three-days','C · Three-day zoom','Compare','Give each day more width. Keep the same time axis and overlap placement.','Choose 3 days from the existing view menu. Swipe horizontally to move through dates. Tap a date to open Day. This option shows less of the week.',()=>weekPhone({three:true}));
addWeek('week-all-day-expanded','Inspect all-day items','Interactions','Trip +4 opens the selected date’s full list in a sheet.','Keep the event, chore, and all three meals. Done returns to the same week position. This sheet is a proposed interaction.',expandedAllDay);
addWeek('week-open-day','Open the selected day','Interactions','Open day shows the existing Day timeline. Back returns to Week.','For option B, tapping a date selects the agenda. Open day is the separate drill-down action. For A and C, tapping a date can open Day directly.',dayDrillDown);
addWeek('week-layout-menu','Choose a week layout','Interactions','Use a submenu in the existing Calendar view menu.','Keep Year, Month, Day, and Week. Add week layout choices inside that menu. Do not add an inner tab bar. This menu shows the alternatives together for review.',weekMenu);
addWeek('week-smart-start','Start near the first event','Interactions','A future week opens near its first timed event. The sample starts at 16:00 for a 17:30 event.','For this week, start near now. For another week, use its first daytime event. Keep manual scroll positions. Do not reset the viewport when data refreshes.',()=>weekPhone({future:true}));
addWeek('week-compact-dark','Compact all-day · dark','Adaptations','The selected layout with the approved dark tokens.','Keep the source calendar colours distinct. All-day items remain available through the compact row.',()=>weekPhone({dark:true}));
addWeek('week-compact-small','Compact all-day · small iPhone','Adaptations','The selected layout on a 375 × 667-point iPhone.','Keep the full seven-day timeline. The floating tab bar stays over content. More hours remain available through vertical scroll.',()=>weekPhone({small:true}));
addWeek('week-large-text','Week · large text','Adaptations','Use the existing date picker and full-width agenda at accessibility text sizes.','Remove the split layout at accessibility sizes. Keep readable labels and the existing week agenda fallback.',()=>phone(accessible(true),'Week').replace('phone theme','phone week-exploration large-week theme'));
window.weekProposals=weekEntries;
const selectedWeek=new URLSearchParams(location.search).get('variant')||'week-compact';
const selectedWeekEntry=weekEntries.find(entry=>entry.id===selectedWeek);
if(!selectedWeekEntry) throw new Error('Unknown week proposal: '+selectedWeek);
document.getElementById('artboard').innerHTML=selectedWeekEntry.render();
document.title='Family Hub · '+selectedWeekEntry.title;

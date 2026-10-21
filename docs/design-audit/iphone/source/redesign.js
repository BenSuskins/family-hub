const departure = `<section class="departure"><div class="departure-label">${icon('clock',18)} NEXT DEPARTURE</div><h2>Leave in 15 minutes</h2><p class="sub">Family dinner · 18:30<br>20 min drive</p><div class="departure-link">View event ${icon('arrow',16).replace('<svg','<svg style="transform:rotate(180deg)"')}</div></section>`;
const todayMeals = `<div class="split"><h2>Meal plan</h2><span class="section-link">View week ›</span></div><div class="today-meals">${group(row('Breakfast','Not planned','+',icon('food')),row('Pasta Salad','Lunch · 22 min'),row('Dinner','Not planned','+',icon('food')))}</div>`;
const todayChores = `<div class="split"><h2>Chores</h2><span class="tag">1 overdue</span></div>${group(choreRow('Vacuum living room','Alex','Due yesterday'),choreRow('Do the dishes','Sam','Due today'))}`;
const revisedToday = `<div class="split"><h1>Today</h1><span class="section-link">${icon('sun',20)} 16°</span></div><p class="sub today-date">Tuesday, 20 October</p>${departure}${todayMeals}${todayChores}`;
for (const key of ['today','today-dark','chore-actions']) {
  screens[key].body=revisedToday;
  screens[key].note='Keep a near departure visible. Put the meal plan before chores. Retain direct meal and chore actions.';
}
screens['today-large'].body=`<h1>Today</h1><p class="sub today-date">Tuesday, 20 October</p>${departure}${todayMeals}<h2>Overdue chores</h2>${group(choreRow('Vacuum living room','Alex','Due yesterday'))}<h2>Today’s chores</h2>${group(choreRow('Do the dishes','Sam','Due today'))}`;
screens['today-large'].note='Keep the meal plan before chores at large text sizes. Stack content and allow vertical scroll.';

const stockItem = (name,detail,quantity) => `<section class="group stock-group">${row(name,detail,'›')}<div class="stock-actions"><span class="section-link">${icon('cart',18)} Add to Shopping</span>${quantity===null?'<span class="tag">Low · 15%</span>':`<span class="stepper"><span>−</span><b>${quantity}</b><span>+</span></span>`}</div></section>`;
screens.area.body=`<h1>Laundry cupboard</h1><div class="segments"><span class="on">All items</span><span>Running low · 1</span></div>${stockItem('Washing tablets','45 pods · Alert at or below 20',45)}${stockItem('Fabric softener','15% remaining · Alert at or below 20%',null)}${stockItem('Dryer sheets','28 sheets · Alert at or below 20',28)}${button('Add item',true)}<p class="foot">Select an item to edit it. More items continue below.</p>`;
screens.area.note='Keep each item and its controls together. Separate stock adjustment, item edit, and Add to Shopping targets.';

screens['cook-step'].bottom=`<div class="sticky"><div class="split"><span class="control">‹ Ingredients</span><span class="sub">1 of 4</span></div>${button('Next step ›')}</div>`;
screens['cook-step'].note='Keep the current instruction large. Give Next step a full-width target. Preserve Ingredients as the return action.';

// Each form has one emphasized commit action.
screens['shopping-rename'].body=screens['shopping-rename'].body.replace(button('Save name'),'');
screens['member-role'].body=screens['member-role'].body.replace(button('Save role'),'');
screens['token-create'].action='';
screens['server-config'].action='';
screens['chore-create'].note+=' Keep Save as the only emphasized commit action.';

nav = function(title,back,action) {
  const modal=['Cancel','Close','Done','Not now'].includes(back);
  const trailing=action===undefined?(!back?icon('gear'):''):action;
  const commit=typeof trailing==='string' && /^(Save|Done|Create|<span style="color:#81818a">Save)/.test(trailing);
  const disabled=trailing.includes && trailing.includes('#81818a');
  return `<div class="navbar"><span class="control back">${back?(modal?'':icon('arrow',20))+back:''}</span>${back?`<strong>${title}</strong>`:''}<span class="control ${trailing?'bubble':''} ${commit?'commit':''} ${disabled?'disabled':''}">${disabled?'Save':trailing}</span></div>`;
};
screens['server-error'].action='';
screens['delete-confirmation'].action='';
screens['setup-help'].back='Close';
screens['setup-help'].action='';
screens['calendar-day'].body=screens['calendar-day'].body.replace('Travel 20 min · Arrival margin 10 min','Travel 20 min');
screens['calendar-day'].note+=' Show the departure time and travel duration. Omit arrival-margin text from Calendar.';

screens.event.body=screens.event.body.replace('20 min drive + 10 min arrival margin','20 min drive');

const shoppingMore = `<span class="target">${icon('more')}</span>`;
const shoppingLow = `<div class="split"><h2>Running low</h2><span class="blue">Inventory ›</span></div>${group(row('Fabric softener','15% remaining','On list'),row('Olive oil','1 bottle',add))}`;
const shoppingHeader = `<h1>Household</h1><div class="segments"><span class="on">Shopping</span><span>Inventory</span></div>`;
const shoppingEntry = `<div class="input blue">${icon('plus',18)} Add an item</div>`;
const activeShoppingRows = row('Milk','',shoppingMore,checked())+row('Fabric softener','Laundry cupboard',shoppingMore,checked());
screens.household.body=`${shoppingHeader}<div class="split"><h2>Shopping</h2><span class="sub">2 to buy</span></div>${shoppingEntry}${group(activeShoppingRows)}${shoppingLow}`;
screens.household.note='Show only items still to buy. After a check, keep the item in place for five seconds with Undo, then hide it.';
const pendingShoppingRow = row('<span class="completed-name">Milk</span>','Hides in 5s',`<span class="undo-action">Undo</span>`,checked(true));
put('shopping-pending','Undo a purchase',`${shoppingHeader}<div class="split"><h2>Shopping</h2><span class="sub">1 to buy</span></div>${shoppingEntry}${group(pendingShoppingRow,row('Fabric softener','Laundry cupboard',shoppingMore,checked()))}${shoppingLow}`,{tabs:'Household',note:'Immediate checked state. Keep the row in its original position for five seconds. Undo or uncheck restores it and cancels removal.'});
put('shopping-hidden','Shopping after five seconds',`${shoppingHeader}<div class="split"><h2>Shopping</h2><span class="sub">1 to buy</span></div>${shoppingEntry}${group(row('Fabric softener','Laundry cupboard',shoppingMore,checked()))}${shoppingLow}`,{tabs:'Household',note:'After five seconds, hide Milk. Do not create a Purchased section or a clear-purchases action.'});
put('shopping-undone','Purchase undone',`${shoppingHeader}<div class="notice">Milk returned to the shopping list.</div><div class="split"><h2>Shopping</h2><span class="sub">2 to buy</span></div>${shoppingEntry}${group(activeShoppingRows)}${shoppingLow}`,{tabs:'Household',note:'Undo keeps Milk unchecked in its original position. Cancel its pending removal and confirm the result.'});
screens.restock.note='Optional stock update for an inventory-linked purchase. Keep this form available through Inventory. Checking a shopping item does not open it automatically.';
screens.restock.body=screens.restock.body.replace('Purchased. Update the stock level now, or select Not now.','Update the stock level, or select Not now.');
screens.restock.back='Cancel';
screens.restock.body=screens.restock.body.replace('or select Not now.','or select Cancel.');
screens.restocked.body=`${shoppingHeader}<div class="notice">Fabric softener stock is now 80%.</div><div class="split"><h2>Shopping</h2><span class="sub">1 to buy</span></div>${shoppingEntry}${group(row('Milk','',shoppingMore,checked()))}${group(row('View inventory','Review the updated stock.'))}`;
screens.restocked.note='Confirm the stock update without a Purchased section. Keep the shopping list limited to items still to buy.';

const inlineShoppingName = `<div class="row inline-rename-row">${checked()}<div class="grow"><div class="inline-field-label">Item name</div><div class="inline-name">Oat milk<span class="text-caret"></span></div></div><span class="inline-cancel">Cancel</span></div>`;
screens['shopping-rename']={title:'Rename shopping item inline',tabs:'Household',body:`${shoppingHeader}<div class="split"><h2>Shopping</h2><span class="sub">2 to buy</span></div>${shoppingEntry}${group(inlineShoppingName,row('Fabric softener','Laundry cupboard',shoppingMore,checked()))}`,bottom:`<div class="mock-keyboard"><div class="keyboard-bar"><span>Rename item</span><span class="blue">Done</span></div>${['QWERTYUIOP','ASDFGHJKL','ZXCVBNM'].map(letters=>`<div class="keyboard-row">${Array.from(letters).map(letter=>`<span>${letter}</span>`).join('')}</div>`).join('')}<div class="keyboard-row keyboard-bottom"><span>123</span><span class="space-key">space</span><span class="keyboard-done">Done</span></div></div>`,note:'Rename replaces the item name with an inline field and focuses the keyboard. Done saves a non-empty name. Cancel restores the original name. Keep the row in place; do not open another screen.'};

const foodOverviewBody = `<h1>Food</h1><p class="sub">Plan the week. Find a recipe.</p>${group(row('Meal plan','Tuesday, 20 October','›',icon('calendar')),row('Recipes','Browse your family recipes','›',icon('book')))}<h2>Today’s meals</h2>${group(row('Breakfast','Not planned','+'),row('Pasta Salad','Lunch · 22 min'),row('Dinner','Not planned','+'))}`;
const householdOverviewBody = `<h1>Household</h1><p class="sub">The shopping list and your stock.</p>${group(row('Shopping list','2 items to buy','›',icon('cart')),row('Inventory','Browse areas and update stock','›',icon('household')))}<h2>Running low</h2>${group(row('Fabric softener','Laundry cupboard · 15%',add),row('Olive oil','Kitchen pantry · 1 bottle',add))}`;
const destinationBody = (body,title) => body.replace(/<h1>.*?<\/h1>/,`<h1>${title}</h1>`).replace(/<div class="segments">.*?<\/div>/,'');
put('meal-plan','Meal plan',destinationBody(screens.food.body,'Meal plan'),{tabs:'Food',back:'Food',action:'',note:'Open from the Food overview. Keep the Friday–Thursday week and return to Food with Back.'});
put('shopping-list','Shopping list',destinationBody(screens.household.body,'Shopping list').replace('<div class="split"><h2>Shopping</h2><span class="sub">2 to buy</span></div>','<p class="sub">2 items to buy</p>'),{tabs:'Household',back:'Household',action:'',note:'Open from the Household overview. Keep inline rename and the five-second Undo window. Show no Purchased section.'});
screens.food.body=foodOverviewBody;
screens.food.action='';
screens.food.note='Chosen overview design. Show Meal plan and Recipes as navigation rows, with today’s meals below.';
screens.household.body=householdOverviewBody;
screens.household.action='';
screens.household.note='Chosen overview design. Show Shopping list and Inventory as navigation rows, with low-stock items below.';
for (const [key,title,parent] of [['recipes','Recipes','Food'],['inventory','Inventory','Household'],['restocked','Shopping list','Household'],['shopping-pending','Shopping list','Household'],['shopping-hidden','Shopping list','Household'],['shopping-undone','Shopping list','Household'],['shopping-rename','Shopping list','Household']]) {
  screens[key].body=destinationBody(screens[key].body,title);
  screens[key].back=parent;
  if (key.startsWith('shopping-')||key==='restocked') screens[key].action='';
}
screens['meal-week'].back='Meal plan';
screens.inventory.note='Open from the Household overview. Put Areas before Running low. Back returns to Household.';
screens.recipes.note='Open from the Food overview. Keep search, meal type, and labelled recipe times. Back returns to Food.';

screens['meal-edit'].title='Edit meal';
screens['meal-edit'].body=`${field('Meal name','Chicken')}${heading('Matching recipes')}${group(row('Chicken Stir Fry','Dinner · 35 min · 4 servings','✓',icon('food')),row('Use “Chicken” only','No recipe',''))}${group(row('<span class="red">Remove meal</span>','',''))}`;
screens['meal-edit'].note='Typing filters recipes and preserves the existing recipe selection. Select a result to change the recipe, or choose Use name only for a custom meal. Save applies that choice. Remove the link warning.';
screens['meal-add'].body=`${field('Meal name','Chicken')}${heading('Matching recipes')}${group(row('Chicken Stir Fry','Dinner · 35 min · 4 servings','›',icon('food')),row('Use “Chicken” only','No recipe',''))}`;
screens['meal-add'].note='Use the existing recipe search. Select a recipe or explicitly choose the typed name only. Keep Save visible.';

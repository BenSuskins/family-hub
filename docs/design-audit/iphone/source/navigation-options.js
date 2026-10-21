const navigationOption = new URLSearchParams(location.search).get('navigation');
if (navigationOption) {
  const screenKey = new URLSearchParams(location.search).get('screen');
  const food = screenKey === 'food';
  const screen = screens[screenKey];
  const section = food ? 'Food' : 'Household';
  const primary = food ? 'Meal plan' : 'Shopping list';
  const secondary = food ? 'Recipes' : 'Inventory';
  let original = screens[food?'meal-plan':'shopping-list'].body.replace(/<h1>.*?<\/h1>/,'').replace(/<div class="segments">.*?<\/div>/,'');
  if (navigationOption === 'direct') {
    if (!food) original=original.replace('<div class="split"><h2>Shopping</h2><span class="sub">2 to buy</span></div>', '<p class="sub">2 items to buy</p>').replace('<span class="blue">Inventory ›</span>','');
    screen.body=`<h1>${primary}</h1>${original}`;
    screen.action=secondary;
    screen.back='';
    screen.note=`${section} opens ${primary.toLowerCase()}. ${secondary} stays visible in the toolbar and opens as a detail screen. The system back action returns to ${primary.toLowerCase()}.`;
  }
  if (navigationOption === 'menu') {
    screen.body=`<h1>${primary} <span class="title-disclosure">⌄</span></h1>${original}`;
    screen.action='';
    screen.back='';
    screen.bottom=`<div class="shade"></div><div class="destination-menu">${group(row(primary,'','✓'),row(secondary,'',''))}</div>`;
    screen.note=`Select the title to change destinations within ${section}. This image shows the open menu. It saves space but hides ${secondary}.`;
  }
  if (navigationOption === 'overview') {
    screen.action='';
    screen.back='';
    screen.body=food
      ? `<h1>Food</h1><p class="sub">Plan the week. Find a recipe.</p>${group(row('Meal plan','Tuesday, 20 October','›',icon('calendar')),row('Recipes','Browse your family recipes','›',icon('book')))}<h2>Today’s meals</h2>${group(row('Breakfast','Not planned','+'),row('Pasta Salad','Lunch · 22 min'),row('Dinner','Not planned','+'))}`
      : `<h1>Household</h1><p class="sub">The shopping list and your stock.</p>${group(row('Shopping list','2 items to buy','›',icon('cart')),row('Inventory','Browse areas and update stock','›',icon('household')))}<h2>Running low</h2>${group(row('Fabric softener','Laundry cupboard · 15%',add),row('Olive oil','Kitchen pantry · 1 bottle',add))}`;
    screen.note=`Show both destinations as normal navigation rows. This gives them equal visibility, but adds a step before the main task.`;
  }
}

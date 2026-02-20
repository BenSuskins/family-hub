# UI Redesign — Targeted Visual Refinement Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the "cheap" feeling of the current UI with a calm, structured, quietly premium aesthetic using indigo accent, card shadows, sidebar depth, and refined typography — touching only the visual layer, no structural changes.

**Architecture:** Pure Tailwind class swaps and small structural changes inside `.templ` files. No logic, no new routes, no JS changes. Every task ends with `make templ && go build ./... && make css` to confirm nothing is broken.

**Tech Stack:** Go templ templates, Tailwind CSS v3, `make templ` / `make css` / `go build`

---

## Reference: What Changes Everywhere

| Old class | New class |
|-----------|-----------|
| `bg-blue-*` (accent) | `bg-indigo-*` |
| `text-blue-*` (accent) | `text-indigo-*` |
| `border-blue-*` (accent) | `border-indigo-*` |
| `hover:bg-blue-*` | `hover:bg-indigo-*` |
| `focus:border-blue-500 focus:ring-blue-500` | `focus:border-indigo-500 focus:ring-indigo-500` |
| `border border-stone-200` (cards) | `ring-1 ring-stone-100 shadow-sm` |
| `text-2xl font-bold text-stone-900` (page h1) | `text-xl font-semibold text-stone-800` |

---

## Task 1: Global CSS — Font + Focus Ring

**Files:**
- Modify: `static/css/input.css`

**Step 1: Apply the changes**

Replace the entire file with:

```css
@import url('https://fonts.googleapis.com/css2?family=Inter:wght@300;400;500;600;700&display=swap');

@tailwind base;
@tailwind components;
@tailwind utilities;

@layer base {
    body {
        font-family: 'Inter', sans-serif;
        font-feature-settings: 'cv11', 'ss01';
    }

    input[type="text"],
    input[type="date"],
    input[type="time"],
    input[type="datetime-local"],
    input[type="email"],
    input[type="password"],
    input[type="url"],
    input[type="number"],
    textarea,
    select {
        @apply block w-full rounded-xl border-stone-200 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 sm:text-sm;
    }
}
```

**Step 2: Verify**

```bash
make templ && go build ./... && make css
```
Expected: no errors.

**Step 3: Commit**

```bash
git add static/css/input.css
git commit -m "style: apply Inter font and switch focus ring to indigo"
```

---

## Task 2: Sidebar & Layout (`layouts/base.templ`)

**Files:**
- Modify: `templates/layouts/base.templ`

**Step 1: Update `navLinkClass` function**

Old:
```go
func navLinkClass(currentPath, linkPath string) string {
	if currentPath == linkPath {
		return "flex items-center gap-3 px-3 py-2 rounded-xl text-sm font-medium bg-blue-50 text-blue-600"
	}
	return "flex items-center gap-3 px-3 py-2 rounded-xl text-sm font-medium text-stone-600 hover:bg-stone-100 hover:text-stone-900"
}
```

New:
```go
func navLinkClass(currentPath, linkPath string) string {
	if currentPath == linkPath {
		return "flex items-center gap-3 px-3 py-2 rounded-xl text-sm font-medium bg-indigo-50 text-indigo-700"
	}
	return "flex items-center gap-3 px-3 py-2 rounded-xl text-sm font-medium text-stone-600 hover:bg-stone-100/80 hover:text-stone-900"
}
```

**Step 2: Update the mobile top bar**

Old:
```templ
<div class="lg:hidden sticky top-0 z-40 flex items-center justify-between bg-white border-b border-stone-200 px-4 h-14">
```

New:
```templ
<div class="lg:hidden sticky top-0 z-40 flex items-center justify-between bg-white border-b border-stone-100 shadow-sm px-4 h-14">
```

**Step 3: Update the sidebar element**

Old:
```templ
<aside id="sidebar" class="fixed inset-y-0 left-0 z-50 w-64 bg-white border-r border-stone-200 transform -translate-x-full lg:translate-x-0 transition-transform duration-200 ease-in-out flex flex-col">
```

New:
```templ
<aside id="sidebar" class="fixed inset-y-0 left-0 z-50 w-64 bg-white border-r border-stone-100 shadow-[1px_0_12px_0_rgba(0,0,0,0.06)] transform -translate-x-full lg:translate-x-0 transition-transform duration-200 ease-in-out flex flex-col">
```

**Step 4: Update the sidebar logo**

Old:
```templ
<div class="flex items-center h-14 px-5 border-b border-stone-200 shrink-0">
	<a href="/" class="text-lg font-bold text-stone-900">{ GetFamilyName(ctx) }</a>
</div>
```

New:
```templ
<div class="flex items-center h-14 px-5 border-b border-stone-100 shrink-0">
	<a href="/" class="text-lg font-semibold text-stone-800">{ GetFamilyName(ctx) }</a>
</div>
```

**Step 5: Update the user section at the bottom**

Old:
```templ
<div class="border-t border-stone-200 px-4 py-3 shrink-0">
```

New:
```templ
<div class="border-t border-stone-100 bg-stone-50 px-4 py-3 shrink-0">
```

**Step 6: Verify and commit**

```bash
make templ && go build ./... && make css
git add templates/layouts/base.templ
git commit -m "style: sidebar shadow, indigo nav active state, refined logo and user section"
```

---

## Task 3: Shared Components — StatCard, PageHeader, Avatar

**Files:**
- Modify: `templates/components/stat_card.templ`
- Modify: `templates/components/page_header.templ`
- Modify: `templates/components/avatar.templ`

### `stat_card.templ`

Replace the entire file:

```go
package components

import "fmt"

type StatCardProps struct {
	Label    string
	Value    int
	SubLabel string
	SubColor string
}

func statSubColor(color string) string {
	if color != "" {
		return color
	}
	return "text-stone-400"
}

templ StatCard(props StatCardProps) {
	<div class="bg-white ring-1 ring-stone-100 shadow-sm rounded-xl overflow-hidden">
		<div class="flex">
			<div class="w-1 bg-indigo-400 flex-shrink-0"></div>
			<div class="p-5 flex-1">
				<div class="flex items-start justify-between">
					<div>
						<p class="text-xs font-medium text-stone-400 uppercase tracking-wide">{ props.Label }</p>
						<p class="text-2xl font-bold text-stone-800 mt-1">{ fmt.Sprintf("%d", props.Value) }</p>
						if props.SubLabel != "" {
							<p class={ "text-xs mt-1 " + statSubColor(props.SubColor) }>{ props.SubLabel }</p>
						}
					</div>
					<div class="text-stone-300 mt-0.5">
						{ children... }
					</div>
				</div>
			</div>
		</div>
	</div>
}
```

### `page_header.templ`

Replace the entire file:

```go
package components

templ PageHeader(title string) {
	<div class="flex justify-between items-start mb-8">
		<div>
			<h1 class="text-xl font-semibold text-stone-800">{ title }</h1>
		</div>
	</div>
}

templ PageHeaderWithAction(title string) {
	<div class="flex flex-col sm:flex-row sm:justify-between sm:items-start gap-4 mb-8">
		<div>
			<h1 class="text-xl font-semibold text-stone-800">{ title }</h1>
		</div>
		<div>
			{ children... }
		</div>
	</div>
}
```

### `avatar.templ`

Change the fallback avatar background from `bg-blue-600` to `bg-indigo-600`:

Old:
```templ
<div class={ "rounded-full flex items-center justify-center bg-blue-600 text-white font-medium " + sizeClass }>
```

New:
```templ
<div class={ "rounded-full flex items-center justify-center bg-indigo-600 text-white font-medium " + sizeClass }>
```

**Verify and commit:**

```bash
make templ && go build ./... && make css
git add templates/components/stat_card.templ templates/components/page_header.templ templates/components/avatar.templ
git commit -m "style: stat card left-border accent, downscale page headers, indigo avatar"
```

---

## Task 4: Dashboard Page (`pages/dashboard.templ`)

**Files:**
- Modify: `templates/pages/dashboard.templ`

**Step 1: Update the two content cards (Chores Due Today, Upcoming Events)**

Both widget cards use `class="bg-white border border-stone-200 rounded-xl p-6"`.

Change both to:
```templ
class="bg-white ring-1 ring-stone-100 shadow-sm rounded-xl p-6"
```

There are 3 cards total in the widget grid: Chores Due Today, Upcoming Events, Today's Meals.

**Step 2: Update card header link hover color**

Three occurrences of `hover:text-blue-600`:
```templ
class="text-lg font-semibold text-stone-900 hover:text-blue-600 transition-colors"
```

Change all three to:
```templ
class="text-lg font-semibold text-stone-800 hover:text-indigo-600 transition-colors"
```

**Step 3: Update `leaderboardPeriodClass` function**

Old:
```go
func leaderboardPeriodClass(activePeriod, thisPeriod string) string {
	if activePeriod == thisPeriod {
		return "px-3 py-1 text-xs font-medium rounded-md bg-blue-50 text-blue-600"
	}
	return "px-3 py-1 text-xs font-medium rounded-md text-stone-500 hover:text-stone-900"
}
```

New:
```go
func leaderboardPeriodClass(activePeriod, thisPeriod string) string {
	if activePeriod == thisPeriod {
		return "px-3 py-1 text-xs font-medium rounded-md bg-indigo-50 text-indigo-700"
	}
	return "px-3 py-1 text-xs font-medium rounded-md text-stone-500 hover:text-stone-900"
}
```

**Step 4: Update the leaderboard card itself**

Old:
```templ
<div id="leaderboard" class="bg-white border border-stone-200 rounded-xl p-6">
```

New:
```templ
<div id="leaderboard" class="bg-white ring-1 ring-stone-100 shadow-sm rounded-xl p-6">
```

**Step 5: Update icon colors passed to StatCard**

The three stat card icon calls pass `text-stone-600`. Since StatCard now wraps them in `text-stone-300`, update the icon color to let the wrapper provide the color (change to no color, or `text-current`). Actually, the icon gets its color from the parent's `text-stone-300` class if no color class is set on the icon — but these icons have explicit color classes. Change all three to use a lighter color to match the new subtle icon placement:

Old (three occurrences in stat cards):
```templ
@components.IconClipboardList("h-5 w-5 text-stone-600")
@components.IconCalendarDays("h-5 w-5 text-stone-600")
@components.IconFire("h-5 w-5 text-stone-600")
```

New:
```templ
@components.IconClipboardList("h-5 w-5")
@components.IconCalendarDays("h-5 w-5")
@components.IconFire("h-5 w-5")
```

(The parent `div class="text-stone-300"` in StatCard now provides the color.)

**Step 6: Verify and commit**

```bash
make templ && go build ./... && make css
git add templates/pages/dashboard.templ
git commit -m "style: dashboard card shadows, indigo accents, refined stat card icons"
```

---

## Task 5: Chores Page (`pages/chores.templ`)

**Files:**
- Modify: `templates/pages/chores.templ`

**Step 1: Update primary button**

Old:
```templ
<a href="/chores/new" class="inline-flex items-center gap-1.5 bg-blue-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-blue-700">
```

New:
```templ
<a href="/chores/new" class="inline-flex items-center gap-1.5 bg-indigo-600 text-white px-4 py-2 rounded-xl shadow-sm text-sm font-medium hover:bg-indigo-700">
```

**Step 2: Replace the tab container and `choreTabClass` function**

The tab container div:

Old:
```templ
<div class="flex space-x-1 bg-stone-100 p-1 rounded-lg w-fit" id="chore-tabs">
```

New:
```templ
<div class="flex border-b border-stone-100" id="chore-tabs">
```

Old `choreTabClass` function:
```go
func choreTabClass(activeTab, thisTab string) string {
	if activeTab == thisTab {
		return "px-4 py-2 text-sm font-medium rounded-md bg-blue-50 text-blue-600"
	}
	return "px-4 py-2 text-sm font-medium rounded-md text-stone-500 hover:text-stone-900"
}
```

New:
```go
func choreTabClass(activeTab, thisTab string) string {
	if activeTab == thisTab {
		return "px-4 py-3 text-sm font-medium text-stone-900 border-b-2 border-indigo-500 -mb-px"
	}
	return "px-4 py-3 text-sm font-medium text-stone-400 hover:text-stone-700"
}
```

**Step 3: Update the inline JavaScript `switchTab` class strings**

In the `<script>` block inside `ChoreList`:

Old:
```javascript
btn.className = btnText === tabLabels[tab] ?
    'px-4 py-2 text-sm font-medium rounded-md bg-blue-50 text-blue-600' :
    'px-4 py-2 text-sm font-medium rounded-md text-stone-500 hover:text-stone-900';
```

New:
```javascript
btn.className = btnText === tabLabels[tab] ?
    'px-4 py-3 text-sm font-medium text-stone-900 border-b-2 border-indigo-500 -mb-px' :
    'px-4 py-3 text-sm font-medium text-stone-400 hover:text-stone-700';
```

**Step 4: Update filter pill functions**

Old `choreStatusPillClass` active branch:
```go
return "px-3 py-1 rounded-full text-sm font-medium bg-blue-50 text-blue-600"
```
Old inactive:
```go
return "px-3 py-1 rounded-full text-sm font-medium text-stone-600 hover:bg-stone-100"
```

New:
```go
// active
return "px-3 py-1 rounded-full text-sm font-medium bg-indigo-50 text-indigo-700 ring-1 ring-indigo-100"
// inactive
return "px-3 py-1 rounded-full text-sm font-medium text-stone-500 hover:text-stone-700 hover:bg-stone-50"
```

Apply the same pattern to `choreUserPillClass` and `choreCategoryPillClass` (same old/new classes, three functions total).

**Step 5: Update content cards**

`ChoreTableContent` empty state:
```templ
// Old
class="bg-white border border-stone-200 rounded-xl p-8 text-center text-stone-500"
// New
class="bg-white ring-1 ring-stone-100 shadow-sm rounded-xl p-8 text-center text-stone-500"
```

`ChoreHistoryContent` empty state: same swap.

`ChoreRow` card:
```templ
// Old
class="bg-white border border-stone-200 rounded-xl p-4 md:rounded-none md:border-x-0 md:border-t-0 md:border-b md:grid md:grid-cols-5 md:gap-4 md:items-center md:px-4 md:py-3"
// New
class="bg-white ring-1 ring-stone-100 shadow-sm rounded-xl p-4 md:shadow-none md:ring-0 md:border-b md:border-stone-100 md:rounded-none md:grid md:grid-cols-5 md:gap-4 md:items-center md:px-4 md:py-3"
```

`ChoreHistoryContent` rows (the div inside the loop):
```templ
// Old
class="bg-white border border-stone-200 rounded-xl p-4 md:rounded-none md:border-x-0 md:border-t-0 md:border-b md:grid md:grid-cols-4 md:gap-4 md:items-center md:px-4 md:py-3"
// New
class="bg-white ring-1 ring-stone-100 shadow-sm rounded-xl p-4 md:shadow-none md:ring-0 md:border-b md:border-stone-100 md:rounded-none md:grid md:grid-cols-4 md:gap-4 md:items-center md:px-4 md:py-3"
```

**Step 6: Update form card and form buttons**

Form card:
```templ
// Old
class="space-y-6 bg-white border border-stone-200 rounded-xl p-6"
// New
class="space-y-6 bg-white ring-1 ring-stone-100 shadow-sm rounded-xl p-6"
```

Form h1:
```templ
// Old
<h1 class="text-2xl font-bold text-stone-900 mb-6">
// New
<h1 class="text-xl font-semibold text-stone-800 mb-6">
```

Submit button:
```templ
// Old
class="bg-blue-600 py-2 px-4 border border-transparent rounded-xl shadow-sm text-sm font-medium text-white hover:bg-blue-700"
// New
class="bg-indigo-600 py-2 px-4 border border-transparent rounded-xl shadow-sm text-sm font-medium text-white hover:bg-indigo-700"
```

Checkboxes (two occurrences — eligible assignees, recur_on_complete, and recurrence days):
```templ
// Old
class="h-4 w-4 text-blue-600 focus:ring-blue-500 border-stone-300 rounded"
// New
class="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-stone-300 rounded"
```

**Step 7: Verify and commit**

```bash
make templ && go build ./... && make css
git add templates/pages/chores.templ
git commit -m "style: chores page - indigo pills, underline tabs, card shadows, indigo buttons"
```

---

## Task 6: Events Page (`pages/events.templ`)

**Files:**
- Modify: `templates/pages/events.templ`

**Step 1: Primary button**

Old: `bg-blue-600 text-white ... hover:bg-blue-700`
New: `bg-indigo-600 text-white ... shadow-sm hover:bg-indigo-700`

**Step 2: Category badge in event list rows**

Old:
```templ
<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700">
```
New:
```templ
<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-indigo-50 text-indigo-700">
```

**Step 3: Update `eventCategoryPillClass` function**

Old active: `bg-blue-50 text-blue-600`
New active: `bg-indigo-50 text-indigo-700 ring-1 ring-indigo-100`
Old inactive: `text-stone-600 hover:bg-stone-100`
New inactive: `text-stone-500 hover:text-stone-700 hover:bg-stone-50`

**Step 4: Event list rows — card style**

The empty-state card:
```templ
// Old
class="bg-white border border-stone-200 rounded-xl p-8 text-center text-stone-500"
// New
class="bg-white ring-1 ring-stone-100 shadow-sm rounded-xl p-8 text-center text-stone-500"
```

The event row divs (the ones inside the loop):
```templ
// Old
class={ "bg-white border border-stone-200 rounded-xl p-4 md:rounded-none md:border-x-0 md:border-t-0 md:border-b md:grid md:gap-4 md:items-center md:px-4 md:py-3 " + eventGridCols(props.User) }
// New
class={ "bg-white ring-1 ring-stone-100 shadow-sm rounded-xl p-4 md:shadow-none md:ring-0 md:border-b md:border-stone-100 md:rounded-none md:grid md:gap-4 md:items-center md:px-4 md:py-3 " + eventGridCols(props.User) }
```

**Step 5: Form card and h1**

Form card: `border border-stone-200` → `ring-1 ring-stone-100 shadow-sm`
Form h1: `text-2xl font-bold text-stone-900` → `text-xl font-semibold text-stone-800`
Submit button: `bg-blue-600 hover:bg-blue-700` → `bg-indigo-600 hover:bg-indigo-700`
Checkbox: `text-blue-600 focus:ring-blue-500` → `text-indigo-600 focus:ring-indigo-500`

**Step 6: Verify and commit**

```bash
make templ && go build ./... && make css
git add templates/pages/events.templ
git commit -m "style: events page - indigo pills and buttons, card shadows"
```

---

## Task 7: Meals Page (`pages/meals.templ`)

**Files:**
- Modify: `templates/pages/meals.templ`

**Step 1: Day cards**

Old:
```templ
class={ "bg-white border border-stone-200 rounded-xl overflow-hidden " + mealTodayBorder(day) }
```
New:
```templ
class={ "bg-white ring-1 ring-stone-100 shadow-sm rounded-xl overflow-hidden " + mealTodayBorder(day) }
```

**Step 2: `mealTodayBorder` function**

Old: returns `"border-blue-300"`
New: returns `"ring-indigo-300"` (note: `ring-1` is on the parent, so use `ring-indigo-300` to override the ring color via Tailwind's ring utilities)

Actually, since the card uses `ring-1 ring-stone-100`, the "today" state should override the ring color. Change the function to return a class that overrides the ring color:

```go
func mealTodayBorder(day time.Time) string {
	today := time.Now()
	if day.Year() == today.Year() && day.Month() == today.Month() && day.Day() == today.Day() {
		return "ring-indigo-300"
	}
	return ""
}
```

**Step 3: `mealDayHeaderTextClass` function**

Old: `"text-blue-600"` → New: `"text-indigo-600"`

**Step 4: Recipe links in `mealRowContent`**

Old: `class="text-blue-600 hover:text-blue-800 underline"`
New: `class="text-indigo-600 hover:text-indigo-800 underline"`

**Step 5: Add meal button**

Old:
```templ
class="inline-flex items-center gap-0.5 text-xs text-blue-600 hover:text-blue-800 px-2 py-1 rounded hover:bg-blue-50"
```
New:
```templ
class="inline-flex items-center gap-0.5 text-xs text-indigo-600 hover:text-indigo-800 px-2 py-1 rounded hover:bg-indigo-50"
```

**Step 6: Save button in `MealCellEdit`**

Old: `class="bg-blue-600 text-white px-3 py-1.5 rounded-lg text-xs font-medium hover:bg-blue-700"`
New: `class="bg-indigo-600 text-white px-3 py-1.5 rounded-lg text-xs font-medium hover:bg-indigo-700"`

**Step 7: Week navigation links (Prev/Next/This Week)**

Old: `class="inline-flex items-center gap-1 text-stone-600 hover:text-stone-900 px-3 py-1 rounded-xl border border-stone-200"`
New: `class="inline-flex items-center gap-1 text-stone-600 hover:text-stone-900 px-3 py-1 rounded-xl ring-1 ring-stone-200"`

(Three links in the week nav: Prev, Next, This Week)

**Step 8: Recipe link in dashboard `TodayMeals`**

In `dashboard.templ` (not meals.templ), there is also:
```templ
<a href={ ... } class="text-amber-700 hover:text-amber-900 underline">
```
This is already using amber (semantic color) — leave it as-is.

**Step 9: Verify and commit**

```bash
make templ && go build ./... && make css
git add templates/pages/meals.templ
git commit -m "style: meals page - indigo today highlight, card shadows, indigo links and buttons"
```

---

## Task 8: Recipes Page (`pages/recipes.templ`)

**Files:**
- Modify: `templates/pages/recipes.templ`

**Step 1: Primary buttons**

All `bg-blue-600 hover:bg-blue-700` → `bg-indigo-600 hover:bg-indigo-700`
Add `shadow-sm` to primary buttons that don't already have it.

There are three:
- "New Recipe" button in list
- "Edit" button in detail view
- Submit button in form

**Step 2: Recipe list cards**

Old:
```templ
class="block bg-white border border-stone-200 rounded-xl p-5 hover:border-stone-300 hover:shadow-card transition-all"
```
New:
```templ
class="block bg-white ring-1 ring-stone-100 shadow-sm rounded-xl p-5 hover:shadow-md transition-all"
```

**Step 3: Category badges (two occurrences)**

Old: `bg-blue-50 text-blue-700`
New: `bg-indigo-50 text-indigo-700`

Locations: inside recipe list card, and in recipe detail view.

**Step 4: Source link**

Old: `text-blue-600 hover:text-blue-800`
New: `text-indigo-600 hover:text-indigo-800`

Also the link icon: `@components.IconLink("h-4 w-4 text-blue-600")` → `@components.IconLink("h-4 w-4 text-indigo-600")`

**Step 5: Content cards in recipe detail**

Both the Ingredients and Instructions cards:
```templ
// Old
class="bg-white border border-stone-200 rounded-xl p-6"
// New
class="bg-white ring-1 ring-stone-100 shadow-sm rounded-xl p-6"
```

**Step 6: Recipe detail h1 and form h1**

Old: `text-2xl font-bold text-stone-900`
New: `text-xl font-semibold text-stone-800`

**Step 7: Form card**

Old: `class="space-y-6 bg-white border border-stone-200 rounded-xl p-6"`
New: `class="space-y-6 bg-white ring-1 ring-stone-100 shadow-sm rounded-xl p-6"`

**Step 8: Ingredient group sub-card**

Old: `class="border border-stone-200 rounded-xl p-4 bg-stone-50 relative"`
New: `class="ring-1 ring-stone-100 rounded-xl p-4 bg-stone-50 relative"` (no shadow — it's nested)

**Step 9: Verify and commit**

```bash
make templ && go build ./... && make css
git add templates/pages/recipes.templ
git commit -m "style: recipes page - card shadows, indigo buttons and badges, refined headers"
```

---

## Task 9: Calendar Page (`pages/calendar.templ`)

This file has the most blue references — mostly in Go helper functions.

**Files:**
- Modify: `templates/pages/calendar.templ`

**Step 1: Share Calendar button**

Old: `bg-blue-600 text-white ... hover:bg-blue-700`
New: `bg-indigo-600 text-white ... shadow-sm hover:bg-indigo-700`

**Step 2: View toggle (`viewToggleClass` function)**

Old active: `"bg-blue-600 text-white border-blue-600"`
New active: `"bg-indigo-600 text-white border-indigo-600"`

**Step 3: Today link**

Old:
```templ
class="text-blue-600 hover:text-blue-800 px-3 py-1 rounded-xl border border-blue-200 text-sm font-medium"
```
New:
```templ
class="text-indigo-600 hover:text-indigo-800 px-3 py-1 rounded-xl border border-indigo-200 text-sm font-medium"
```

**Step 4: Navigation links (Prev/Next)**

Old: `class="inline-flex items-center gap-1 text-stone-600 hover:text-stone-900 px-3 py-1 rounded-xl border border-stone-200"`
New: `class="inline-flex items-center gap-1 text-stone-600 hover:text-stone-900 px-3 py-1 rounded-xl ring-1 ring-stone-200"`

**Step 5: Calendar view cards (month, week, day, year)**

All view containers use `class="bg-white border border-stone-200 rounded-xl overflow-hidden"`:
- `monthView` outer div
- `weekView` outer div
- `dayView` outer div

Change to: `class="bg-white ring-1 ring-stone-100 shadow-sm rounded-xl overflow-hidden"`

Year view mini-calendars:
```templ
// Old
class="bg-white border border-stone-200 rounded-xl overflow-hidden"
// New
class="bg-white ring-1 ring-stone-100 shadow-sm rounded-xl overflow-hidden"
```

Year month header hover:
```templ
// Old (in yearView)
class="block px-3 py-2 text-sm font-medium text-stone-900 bg-stone-50 hover:bg-blue-50 border-b border-stone-200 text-center"
// New
class="block px-3 py-2 text-sm font-medium text-stone-900 bg-stone-50 hover:bg-indigo-50 border-b border-stone-100 text-center"
```

**Step 6: Event chips (multiple locations)**

All event chip divs across month, week-allday, week-timed, day-allday, day-timed views:
```templ
// Old
class="text-xs bg-blue-50 text-blue-700 rounded px-1 py-0.5 mb-0.5 truncate cursor-pointer hover:bg-blue-100"
// New
class="text-xs bg-indigo-50 text-indigo-700 rounded px-1 py-0.5 mb-0.5 truncate cursor-pointer hover:bg-indigo-100"
```

There are 5 occurrences (month view, week all-day, week timed, day all-day, day timed). The larger day-view versions use `text-sm` instead of `text-xs` — apply the same color swap.

**Step 7: CalendarShareModal copy button**

Old: `bg-blue-600 text-white ... hover:bg-blue-700`
New: `bg-indigo-600 text-white ... hover:bg-indigo-700`

**Step 8: Go helper functions**

`todayClass`:
```go
// Old
return "text-blue-600 font-bold"
// New
return "text-indigo-600 font-bold"
```

`yearDayClass` (5 occurrences of blue):
- `"bg-blue-600 text-white font-bold"` → `"bg-indigo-600 text-white font-bold"`
- `"bg-blue-100 text-blue-700 font-bold"` → `"bg-indigo-100 text-indigo-700 font-bold"`
- `"bg-blue-50 text-blue-700 hover:bg-blue-100"` → `"bg-indigo-50 text-indigo-700 hover:bg-indigo-100"`

`yearDayClassWithMeals`: same three blue → indigo swaps.

`weekDayHeaderClass`:
- `"bg-blue-50"` → `"bg-indigo-50"`

`weekDayCellClass`:
- `"bg-blue-50"` → `"bg-indigo-50"`

`dayTimeCellClass`:
- `"bg-blue-50"` → `"bg-indigo-50"`

**Step 9: Category badges in modals**

In `EventDetailFragment` and `ChoreDetailFragment`:
```templ
// Old
<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700">
// New
<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-indigo-50 text-indigo-700">
```

**Step 10: Verify and commit**

```bash
make templ && go build ./... && make css
git add templates/pages/calendar.templ
git commit -m "style: calendar - indigo event chips, today highlight, view toggle, card shadows"
```

---

## Task 10: Admin Page (`pages/admin.templ`)

**Files:**
- Modify: `templates/pages/admin.templ`

**Step 1: Page h1**

Old: `<h1 class="text-2xl font-bold text-stone-900">Admin</h1>`
New: `<h1 class="text-xl font-semibold text-stone-800">Admin</h1>`

**Step 2: All primary buttons (4 occurrences)**

All `bg-blue-600 text-white ... hover:bg-blue-700` → `bg-indigo-600 text-white ... hover:bg-indigo-700`

Locations: Save (family name), Add (category), Create Token, and the input fields that have explicit inline focus classes.

**Step 3: Explicit focus ring overrides in admin inputs**

There are two inputs with explicit `focus:border-blue-500 focus:ring-blue-500` inline class attributes (not covered by the base CSS layer because they have their own class strings):

Line ~60:
```templ
class="flex-1 rounded-xl border-stone-200 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
```

Line ~142:
```templ
class="flex-1 rounded-xl border-stone-200 shadow-sm focus:border-blue-500 focus:ring-blue-500 sm:text-sm"
```

Change both to `focus:border-indigo-500 focus:ring-indigo-500`.

**Step 4: Admin role badge**

Old: `class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700"`
New: `class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-indigo-50 text-indigo-700"`

**Step 5: Content cards**

All `class="bg-white border border-stone-200 rounded-xl p-6"` and `class="bg-white border border-stone-200 rounded-xl overflow-hidden"`:
→ `ring-1 ring-stone-100 shadow-sm` replacing `border border-stone-200`

Four card containers in admin page.

**Step 6: Verify and commit**

```bash
make templ && go build ./... && make css
git add templates/pages/admin.templ
git commit -m "style: admin - indigo buttons, badges, card shadows, refined header"
```

---

## Final Verification

After all tasks complete, do a final sweep to confirm no `blue-` accent references remain:

```bash
grep -rn "blue-" templates/ | grep -v "// " | grep -v "_templ.go"
```

Any remaining `blue-` references should only be in:
- `sky-*` (meal type badge — intentional, different semantic color)
- Non-accent uses that are intentionally kept (there should be none)

Then do a final build:

```bash
make templ && make css && go build ./...
```

Visual inspection checklist:
- [ ] Sidebar has visible shadow depth on desktop
- [ ] Active nav item is indigo, not blue
- [ ] Stat cards show left indigo strip with subtle right icon
- [ ] Cards have soft shadow instead of harsh border
- [ ] All buttons are indigo
- [ ] Calendar event chips are indigo
- [ ] Today highlight on calendar is indigo
- [ ] Filter pills use indigo when active
- [ ] Chore tabs use underline style
- [ ] Form focus rings are indigo

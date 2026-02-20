# UI Redesign — Targeted Visual Refinement

**Date:** 2026-02-20
**Approach:** A — Targeted refinement (visual layer only, no structural changes)

## Goal

Replace the "cheap" feeling of the current UI with a calm, structured, quietly premium aesthetic: soft neutral backgrounds, restrained indigo accent, generous whitespace, subtle shadows, and clear typographic hierarchy.

## Design Decisions

### Accent color
- Replace all `blue-` references with `indigo-` equivalents
- Primary accent: `indigo-600` (buttons, active nav, form focus)
- Light accent background: `indigo-50` with `text-indigo-700`
- Accent strip on stat cards: `border-indigo-400`

### Section 1 — Global base styles (`static/css/input.css`)
- Set `font-family: 'Inter', sans-serif` on `body` in `@layer base`
- Add `font-feature-settings: 'cv11', 'ss01'` for refined Inter numerals
- Change form focus ring from `blue-500` → `indigo-500`

### Section 2 — Sidebar (`templates/layouts/base.templ`)
- Add right-side shadow: `shadow-[1px_0_12px_0_rgba(0,0,0,0.06)]`
- Lighten sidebar border: `border-stone-200` → `border-stone-100`
- Nav active state: `bg-indigo-50 text-indigo-700` (was `bg-blue-50 text-blue-600`)
- Nav inactive hover: `hover:bg-stone-100/80`
- Logo: `font-semibold` (was `font-bold`), `text-stone-800`
- User section: add `bg-stone-50` tint to distinguish from nav
- Mobile top bar: `border-stone-200` → `border-stone-100`, same shadow treatment

### Section 3 — Cards & stat cards
**Content cards** (dashboard widgets, chore rows, etc.):
- Replace `border border-stone-200` with `ring-1 ring-stone-100 shadow-sm`
- Card header text: `text-stone-800` (was `text-stone-900`)

**Stat cards** (`templates/components/stat_card.templ`):
- Remove `bg-blue-50 rounded-lg` icon container
- Add `border-l-4 border-indigo-400` accent strip on the card left edge
- Adjust padding: card gets `pl-5` for content after the strip
- Stat label: `text-xs font-medium text-stone-400 uppercase tracking-wide`
- Stat value: `text-stone-800` (was `text-stone-900`)

### Section 4 — Buttons, badges, tabs, filter pills
**Primary buttons:**
- `bg-blue-600 hover:bg-blue-700` → `bg-indigo-600 hover:bg-indigo-700`
- Add `shadow-sm`

**Status badges:** keep pill style; overdue shifts to `text-red-600` (less saturated)

**Tab switchers** (leaderboard period, chore tabs):
- Remove `bg-stone-100 p-1 rounded-lg` filled container
- Use border-bottom indicator: inactive = `text-stone-400`, active = `text-stone-900 border-b-2 border-indigo-500`

**Filter pills** (status, user, category):
- Inactive: `text-stone-500 hover:text-stone-700 hover:bg-stone-50`
- Active: `bg-indigo-50 text-indigo-700 ring-1 ring-indigo-100`

### Section 5 — Typography & page headers
- `PageHeader`/`PageHeaderWithAction`: `text-xl font-semibold text-stone-800` (was `text-2xl font-bold text-stone-900`)
- Body primary text: `text-stone-700`
- Secondary/meta text: `text-stone-400`
- Form labels: `text-stone-600` (was `text-stone-700`)

## Files to change
1. `static/css/input.css` — base font and focus ring
2. `templates/layouts/base.templ` — sidebar, mobile bar, nav states
3. `templates/components/stat_card.templ` — left-border accent, icon removal
4. `templates/components/page_header.templ` — typography scale
5. `templates/pages/dashboard.templ` — card classes, button/badge/tab classes
6. `templates/pages/chores.templ` — card classes, button/badge/tab/pill classes
7. `templates/pages/events.templ` — card and button classes
8. `templates/pages/meals.templ` — card and button classes
9. `templates/pages/recipes.templ` — card and button classes
10. `templates/pages/calendar.templ` — card and button classes
11. `templates/pages/admin.templ` — card and button classes

## Out of scope
- No layout or structural changes
- No new components
- No JS changes
- No dark mode

# Calendar Week options

21 October 2026. [Open the gallery](index.html). [Open the comparison board](overview.png).

These static iPhone proposals explore the week-view suggestions. They go beyond the earlier colour-only Calendar proposal. They change no app code.

## Compare

| Option | Benefit | Tradeoff |
|---|---|---|
| A: compact seven days | Recover 116 points by reducing the all-day area from 160 to 44 points | Event names still occupy narrow columns |
| B: week with an agenda | Keep the full week and read full event names and locations below | Less of the timed grid is visible; each pane needs its own scroll position |
| C: three-day zoom | Give events wider columns without losing their time and duration | Show less of the week; users move horizontally to see other days |

**Selected: A · Compact all-day.** Keep the full seven-day timeline and reduce the all-day area to one row. Tap its count for the complete list. B and C remain alternatives for reference.

The baseline is a static representation of the current layout in Warm Sage Cosier. The four comparison images use the same sample events and dates. They are not fresh simulator screenshots.

## Interaction rules

The selected change is the compact all-day area and its expanded list. Day drill-down, the layout submenu, and smart initial time remain separate ideas for review. They are not implied implementation requirements.

- All three alternatives use a compact all-day row. The selected sample has one all-day event, one chore, and three meals. “Trip +4” accounts for all five items.
- Tap the all-day count to inspect the full list. Done returns to the same week position. The sheet is a new proposed interaction.
- In B, tap a weekday to select its column and agenda. Tap an event row to open its detail. Use Open day for the full timeline.
- In A and C, a weekday tap can open Day directly. These are alternative tap behaviours, not two simultaneous actions.
- Back from Day returns to the same date, week layout, and scroll position.
- Keep the existing view menu. Put the week-layout choices in a submenu. Do not add an inner tab bar.
- In C, show three consecutive days. The initial sample centres Today. Horizontal swipes move through dates.
- Start this week near now. Start another week near its first timed event. Retain manual scroll positions when data refreshes.
- If a week has no timed events, use a stable default time. Keep all-day items visible.
- At accessibility text sizes, use the existing full-width week agenda. Do not force the split layout to fit.

The future-week sample shows 28 October. The first event starts at 17:30. The viewport starts at 16:00. It has no current-time marker because Today is outside that week.

The selected-day list shows four timed events: School run, Dentist, Lunch with Sam, and Swimming lesson. The top grid may show only part of that day’s hours. The list makes the remaining events available without a long timeline scroll.

In B, grid labels stay on one line instead of wrapping into individual letters. The agenda supplies the full names and locations. The grid retains event times, duration, and overlap positions.

All calendar source colours retain their meaning. The current-time marker stays in Today’s column. Meals stay in the all-day list. The designs show no arrival margin.

## Review package

There are eleven images: the baseline, three main alternatives, four interaction states, and three adaptations. The adaptations cover the selected compact layout in dark appearance and on a small iPhone, plus the existing large-text fallback.

[The manifest](manifest.json) records the canvas dimensions. [Validation](validation.json) covers the files and static layouts. Phone chrome, menu materials, sheet presentation, and text metrics are illustrative. Native motion, scrolling, VoiceOver, and touch still need implementation checks.

The editable source reuses [the Calendar theme and drawing helpers](../calendar-widgets/source/mockups.js). Change [week.js](source/week.js) and [week.css](source/week.css) for these alternatives. Keep the earlier Calendar and widget proposals intact.

Rebuild with `node source/render.cjs`, then `python3 source/build_gallery.py`. Run `node source/validate.cjs` for the static checks. The Python script uses Pillow for the comparison board. The Node scripts accept `CHROME_PATH` and `PLAYWRIGHT_MODULE` overrides.

# App Store screenshot compositor

Renders App Store screenshots as HTML/CSS templates through headless Chrome.
Each slide shows a benefit headline plus a device frame. If a real capture
exists in `raw/`, it fills the frame; otherwise a hand-built mock screen with
curated demo data stands in.

## Render

```bash
./render.sh
```

Output lands in `out/` at 1320×2868 (iPhone 6.9", the one set App Store
Connect requires). Override the Chrome path with `CHROME=/path/to/chrome`.

## Use real app captures

Drop simulator captures into `raw/` using these exact names:

| File | Slide |
|------|------------------|
| `01-home.png` | Home dashboard |
| `02-chores.png` | Chores + leaderboard |
| `03-meals.png` | Meal plan week |
| `04-recipes.png` | Recipe grid |
| `05-calendar.png` | Unified calendar |
| `06-selfhosted.png` | (no capture — typographic slide) |

Slides reference `../raw/<name>.png`; when present, the image covers the mock
screen (`object-fit: cover`). Capture at any 6.9"-aspect resolution
(1320×2868 ideal): Simulator ⌘S, or `xcrun simctl io booted screenshot`.

### Capture checklist

The old listing's core problem was empty states. Before capturing:

- Fill every meal slot for today and the planned week.
- Add 4+ chores across several family members, some completed.
- No "Overdue", no grey AD/SD initials — use colourful avatars and real names.
- Import 2–3 recipes so the grid has covers.
- Subscribe to one iCal feed so the calendar shows all three source types.
- Set Simulator status bar: `xcrun simctl status_bar booted override --time "9:41" --batteryLevel 100 --wifiBars 3`

## Editing design/copy

- Tokens, frame geometry, chips, mock-screen styles: `lib/slide.css`.
- Headlines, eyebrows, chip text, accent colour: each `slides/*.html`
  (`--accent` on `<body>` picks the tint; `<mark>` gets the highlight swipe).
- Chip positions are global (`chip-a` right, `chip-b` left); override inline
  per slide when a mock needs it (see `05-calendar.html`).

## TODO

- **iPad set**: the app targets iPad (`TARGETED_DEVICE_FAMILY = 1,2`), so
  App Store Connect requires a 13" iPad set (2064×2752). Slides are
  fixed-pixel layouts; build iPad variants before shipping an iPad update.

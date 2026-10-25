# App Store screenshot pipeline

Capture genuine Family Hub screens from iPhone, iPad, and Apple Watch simulators, then place them in the App Store templates.
The templates follow [`DESIGN.md`](../../../DESIGN.md) and the app's Warm Sage Cosier palette.

## Requirements

- macOS with Xcode and an available iPhone Simulator runtime
- watchOS 27 Simulator runtime for Apple Watch Series 12
- Python 3 (included with current macOS/Xcode installations)
- Google Chrome

## Generate and render

From this directory, capture each device and render a versioned set:

```bash
./capture.sh
./capture-ipad.sh
./capture-watch.sh
./render.sh 1.3
```

`capture.sh` selects an available iPhone Pro simulator when possible, which
matches the 6.3-inch screenshot target. To use a specific simulator, set its UDID:

```bash
SIMULATOR_UDID=<simulator-udid> ./capture.sh
```

Run `./capture-ipad.sh` for an iPad screenshot set. It selects an available
13-inch iPad Pro simulator and captures ten portrait screens at 2064 × 2752
px. Set `SIMULATOR_UDID` to choose a specific 13-inch iPad Pro. Open
`raw/generated-ipad/index.html` to browse the ten images.

The capture script runs only `MarketingScreenshotsUITests/testCaptureMarketingScreens` in the `FamilyHub` scheme.
It launches the app in a screenshot-only demo fixture, sets the status bar to 9:41, and exports the UI test attachments.
It requires no Family Hub server or account.

The iPad capture script runs the same marketing demo flow through the iPad UI
in portrait. It writes the test result bundle under `$TMPDIR` and exports
exactly ten screenshots into `raw/generated-ipad/` for App Store Connect.

`capture-watch.sh` creates or selects a 46mm Apple Watch Series 12 simulator
on watchOS 27 and captures the ingredient checklist and a timed recipe step.
It seeds a local marketing recipe through launch arguments; it does not need a
paired phone, server, or account. The screenshots are rendered into two
Series 12 marketing slides at 416 × 496 px.

`render.sh <version>` writes ten iPhone App Store screenshots at 1206 × 2622
px and ten portrait iPad marketing slides at 2064 × 2752 px into the named
version folder. Each iPad slide reuses its matching iPhone headline, copy, and
benefits, then places the iPad capture inside `assets/ipad-frame.png`. The two
Apple Watch slides use the same warm palette and a compact watch case layout.
The self-hosting promo remains a marketing reference outside the iPhone and
iPad ten-image upload sets. The iPad workflow creates portrait only because
App Store Connect needs one orientation set for this listing.

Set `CHROME` to override the default Chrome executable path.
Set `RESULT_BUNDLE` to keep the Xcode result bundle at a chosen path. By default, the bundle and derived data live under `$TMPDIR`.

## Outputs

- `raw/generated/`: named full-resolution simulator captures, created by `capture.sh`
- `raw/generated-ipad/`: ten full-screen iPad portrait captures and an HTML gallery (ignored by Git)
- `raw/generated-watch/`: two Apple Watch Series 12 captures (ignored by Git)
- `1.3/`: ten framed iPhone images, ten framed iPad images, two Apple Watch marketing images, and the version 1.3 release copy

Create or refresh a versioned image set with `./render.sh 1.3` after capturing all three devices. The renderer replaces the generated app screenshots for each device and preserves the release copy and self-hosting promo.

Existing JPEGs in `raw/` are retained as older reference captures; the current templates use only `raw/generated/`.

## Capture set

| File | Screen |
|------|--------|
| `01-home.png` | Today overview |
| `02-recipe-detail.png` | Recipe details and cooking action |
| `03-meals.png` | Weekly meal plan |
| `04-recipes-book.png` | Recipe collection |
| `05-inventory.png` | Inventory overview |
| `06-calendar.png` | Family calendar |
| `07-chores.png` | Chore list |
| `08-household.png` | Household overview |
| `09-shopping.png` | Shared shopping list |
| `10-commute.png` | Sample Southend Central → London Liverpool Street train board |

The simulator flow records deterministic demo content. It does not change the normal demo data unless the app is launched with `-marketing-screenshots`.

## Template design

- `lib/slide.css` holds the shared cream, sage and clay palette, system typography, phone frame and benefit labels.
- `assets/iphone-frame.png` and `assets/ipad-frame.png` supply the transparent device bezels.
- `slides/*.html` holds the iPhone marketing copy; the iPad and Watch templates provide device-specific layouts.
- `render.sh`, `render-ipad-slide.py`, and `render-watch-slide.py` check each output size before updating a version folder.

Update `DESIGN.md` first when the app's visual direction changes, then align these templates with its tokens and product language.

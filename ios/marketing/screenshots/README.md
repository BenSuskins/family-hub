# App Store screenshot pipeline

Capture genuine Family Hub screens from an iPhone simulator, then place them in the App Store templates.
The templates follow [`DESIGN.md`](../../../DESIGN.md) and the app's Warm Sage Cosier palette.

## Requirements

- macOS with Xcode and an available iPhone Simulator runtime
- Python 3 (included with current macOS/Xcode installations)
- Google Chrome

## Generate and render

From this directory, run:

```bash
./capture.sh
./render.sh
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

`render.sh` creates ten iPhone App Store screenshots at 1206 × 2622 px, an
accepted size for 6.3-inch displays. The text-led self-hosting promo slide
remains a marketing reference in `slides/`, but is excluded from the upload
set. The iPad upload set contains ten 2064 × 2752 px portrait screenshots.
App Store Connect needs one orientation set for this listing, so the workflow
does not create a duplicate landscape set.

Set `CHROME` to override the default Chrome executable path. Set `OUT` to change the rendered-image directory.
Set `RESULT_BUNDLE` to keep the Xcode result bundle at a chosen path. By default, the bundle and derived data live under `$TMPDIR`.

## Outputs

- `raw/generated/`: named full-resolution simulator captures, created by `capture.sh`
- `raw/generated-ipad/`: ten 2064 × 2752 iPad portrait screenshots and an HTML gallery (ignored by Git)
- `out/`: ten temporary 1206×2622 iPhone App Store PNG images, created by `render.sh` and ignored by Git
- `1.3/`: versioned App Store images and release copy committed for version 1.3

Create a versioned image set with `OUT=1.3 ./render.sh` after capturing the current app screens. Keep `raw/generated/` and `out/` as working outputs; keep the reviewed release set in its numbered folder.

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
- `assets/iphone-frame.png` supplies the transparent graphite iPhone bezel used by the screenshot slides.
- `slides/*.html` holds per-screen headlines, copy and screenshot mappings.
- `render.sh` checks for every required capture and fails before rendering if any are missing.

Update `DESIGN.md` first when the app's visual direction changes, then align these templates with its tokens and product language.

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

`capture.sh` selects an available iPhone Pro Max simulator when possible. To use a specific simulator, set its UDID:

```bash
SIMULATOR_UDID=<simulator-udid> ./capture.sh
```

The capture script runs only `MarketingScreenshotsUITests/testCaptureMarketingScreens` in the `FamilyHub` scheme.
It launches the app in a screenshot-only demo fixture, sets the status bar to 9:41, and exports the UI test attachments.
It requires no Family Hub server or account.

Set `CHROME` to override the default Chrome executable path. Set `OUT` to change the rendered-image directory.
Set `RESULT_BUNDLE` to keep the Xcode result bundle at a chosen path. By default, the bundle and derived data live under `$TMPDIR`.

## Outputs

- `raw/generated/`: named full-resolution simulator captures, created by `capture.sh`
- `out/`: temporary 1320×2868 PNG images, created by `render.sh` and ignored by Git
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
| `11-selfhosted-iphone-6.9.png` | Text-led self-hosted feature slide; no app capture |

The simulator flow records deterministic demo content. It does not change the normal demo data unless the app is launched with `-marketing-screenshots`.

## Template design

- `lib/slide.css` holds the shared cream, sage and clay palette, system typography, phone frame and benefit labels.
- `slides/*.html` holds per-screen headlines, copy and screenshot mappings.
- `render.sh` checks for every required capture and fails before rendering if any are missing.

Update `DESIGN.md` first when the app's visual direction changes, then align these templates with its tokens and product language.

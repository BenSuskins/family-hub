# Recipe Improvements Design

Date: 2026-02-23

## Summary

Three improvements to the recipes section:
1. Recipe images (uploaded, stored as base64 in SQLite)
2. Structured steps + cook mode (dedicated `/recipes/:id/cook` page)
3. Fixed recipe meal-type categories (Breakfast / Lunch / Dinner / Side / Dessert)

## Data Model

### New type

```go
type RecipeMealType string
const (
    RecipeMealTypeBreakfast RecipeMealType = "breakfast"
    RecipeMealTypeLunch     RecipeMealType = "lunch"
    RecipeMealTypeDinner    RecipeMealType = "dinner"
    RecipeMealTypeSide      RecipeMealType = "side"
    RecipeMealTypeDessert   RecipeMealType = "dessert"
)
```

### Recipe model additions

- `MealType *RecipeMealType` — nullable fixed enum
- `Steps []string` — JSON array, replaces free-text instructions going forward
- `Instructions string` — kept as legacy fallback (read-only after migration)
- `HasImage bool` — computed from `image_data != ''`, not a stored field

### Migration 012

```sql
ALTER TABLE recipes ADD COLUMN meal_type TEXT;
ALTER TABLE recipes ADD COLUMN steps TEXT NOT NULL DEFAULT '[]';
ALTER TABLE recipes ADD COLUMN image_data TEXT NOT NULL DEFAULT '';
```

The existing `instructions` column is retained unchanged. No data is lost.

### Backward compatibility

- Detail page renders `Steps` as a numbered list when non-empty, otherwise falls back to `Instructions` plain text.
- Edit form pre-populates the step editor by splitting `Instructions` on newlines when `Steps` is empty.
- Once a recipe is saved via the new form, `Steps` is populated and `Instructions` is ignored.

## Repository

### Interface additions to `RecipeRepository`

```go
FindImageData(ctx context.Context, id string) (string, error)
UpdateImage(ctx context.Context, id, imageData string) error
ClearImage(ctx context.Context, id string) error
```

### Query changes

- `FindAll`: scans `meal_type` and `image_data != ''` as `HasImage`. Does not scan `steps`, `instructions`, or image blobs (keeps list queries lean).
- `FindByID`: scans `meal_type`, `steps`, `instructions`, and `HasImage`. Does not scan `image_data` blob.
- `Create` / `Update`: write `meal_type` and `steps`. Do not touch `instructions`.

## Routes

```
GET  /recipes/:id/image        → serve image bytes (Content-Type from stored data URI)
POST /recipes/:id/image        → upload image (multipart, max 2 MB)
POST /recipes/:id/image/delete → remove image
GET  /recipes/:id/cook         → cook mode page
GET  /recipes/step             → HTMX fragment: new step textarea field
```

## Templates & UI

### Recipe List

- Cards show an image thumbnail at the top via `/recipes/ID/image` when `HasImage`.
- Meal type badge rendered in amber (distinct from indigo chore category badges).

### Recipe Detail

- Image displayed full-width below the title when `HasImage`.
- Steps rendered as a numbered list. Falls back to legacy plain-text `Instructions` when `Steps` is empty.
- "Cook Mode" button in the header actions row alongside Edit / Delete.
- Image upload / remove widget below the main card (mirrors profile avatar UX).

### Recipe Form

- Meal type `<select>` with fixed options (None / Breakfast / Lunch / Dinner / Side / Dessert).
- Image upload widget: file input, current image preview, remove button.
- Step editor: ordered list of textareas, each with a Remove button. HTMX appends new steps via `GET /recipes/step?index=N`. Pre-populated by splitting `Instructions` on newlines when editing a legacy recipe with empty `Steps`.

### Cook Mode (`/recipes/:id/cook`)

- Minimal focused layout.
- Header: recipe title + "Step N of M".
- Large text block showing the current step.
- Prev / Next navigation buttons.
- Collapsible ingredient list for reference.
- "Exit Cook Mode" link back to the detail page.
- Step navigation driven by JavaScript (all steps rendered in DOM, JS shows/hides active step).

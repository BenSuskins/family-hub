# iOS Recipe CRUD + Backend API

## Problem

Recipe management is only available through the web UI. The iOS app can view recipes but cannot create, edit, or delete them. Image upload is also broken (silently fails with no error). Adding full CRUD to iOS requires new backend API endpoints since only read endpoints exist.

## Backend API Endpoints

Add three new endpoints to `api.go`, registered under the token-auth route group in `server.go`.

### POST /api/recipes — Create

Accepts JSON body:

```json
{
  "title": "Pasta Carbonara",
  "steps": ["Boil pasta", "Fry pancetta", "Mix eggs and cheese", "Combine"],
  "ingredients": [{"name": "Main", "items": ["400g spaghetti", "200g pancetta"]}],
  "mealType": "dinner",
  "servings": 4,
  "prepTime": "15 min",
  "cookTime": "20 min",
  "sourceURL": "https://example.com/carbonara",
  "imageData": "data:image/jpeg;base64,/9j/4AAQ..."
}
```

- `title` is required; all other fields are optional
- `imageData` when present is a base64 data URI, stored directly in `image_data` column
- Enforces 2MB image size limit (same as web)
- Sets `CreatedByUserID` from the authenticated user context
- Returns the created recipe as JSON (201)

### PUT /api/recipes/{id} — Update

Same JSON body as create. Semantics:

- Omitted fields are set to nil/empty (full replacement, not patch)
- `imageData` omitted = image unchanged; empty string = clear image; present = replace image
- Returns updated recipe as JSON (200)

### DELETE /api/recipes/{id} — Delete

- Clears recipe references from `meal_plans` table first (same as web handler)
- Returns 204 No Content

## iOS Model Updates

Expand `Recipe.swift` to include `mealType` and `sourceURL` fields (currently missing from the model but returned by the API).

Add `RecipeRequest` encodable struct for create/update payloads with all editable fields plus optional `imageData`.

## iOS API Client

Add to `APIClientProtocol` and `APIClient`:

- `createRecipe(_ request: RecipeRequest) async throws -> Recipe`
- `updateRecipe(id: String, _ request: RecipeRequest) async throws -> Recipe`
- `deleteRecipe(id: String) async throws`

The `post` and `delete` helpers in `APIClient` already support these patterns. Need to add a `put` helper for the update endpoint.

## iOS Views

### RecipeFormView (new file)

A modal form sheet for creating and editing recipes. Shared for both operations — receives an optional `Recipe` for edit mode.

Fields:
- Title (text, required)
- Meal type (picker: breakfast/lunch/dinner/side/dessert/none)
- Servings (number stepper)
- Prep time, cook time (text)
- Source URL (text)
- Ingredients (grouped, dynamic add/remove for groups and items)
- Steps (ordered list, dynamic add/remove)
- Photo (PhotosPicker, shows current image preview in edit mode)

Save button calls create or update depending on mode. On success, dismisses and refreshes the recipe list.

### RecipesView changes

- Add "+" button in navigation bar to present RecipeFormView as a sheet

### RecipeDetailView changes

- Add edit button (toolbar) to present RecipeFormView in edit mode
- Add delete with confirmation alert
- On delete, pop navigation back to list

## RecipesViewModel Updates

Add methods following the `MealsViewModel` pattern:

- `createRecipe(_ request: RecipeRequest) async -> Bool`
- `updateRecipe(id: String, _ request: RecipeRequest) async -> Bool`
- `deleteRecipe(id: String) async -> Bool`

Each calls the API client, refreshes the list on success, returns success/failure bool.

## Testing

### Backend

- Handler tests for `CreateRecipe`, `UpdateRecipe`, `DeleteRecipe` in `api_test.go`
- Test cases: happy path, validation (missing title), image handling, not found on update/delete

### iOS

- `FakeAPIClient` updated with new methods and configurable results

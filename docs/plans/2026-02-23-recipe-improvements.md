# Recipe Improvements Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add recipe images, structured steps with a dedicated cook mode page, and fixed meal-type categories (Breakfast / Lunch / Dinner / Side / Dessert) to the recipes feature.

**Architecture:** New SQLite columns (`meal_type`, `steps`, `image_data`) are added via migration 012. Steps replace the free-text `instructions` field in the UI; `instructions` is preserved in the DB as a legacy fallback. Images are stored as base64 data URIs in SQLite, matching the existing avatar pattern. Cook mode is a separate server-rendered page at `/recipes/:id/cook`.

**Tech Stack:** Go, chi, templ, HTMX, SQLite (modernc.org/sqlite), Tailwind CSS.

---

## Task 1: Write migration 012

**Files:**
- Create: `internal/database/migrations/012_recipe_improvements.up.sql`

**Step 1: Write the migration**

```sql
ALTER TABLE recipes ADD COLUMN meal_type TEXT;
ALTER TABLE recipes ADD COLUMN steps TEXT NOT NULL DEFAULT '[]';
ALTER TABLE recipes ADD COLUMN image_data TEXT NOT NULL DEFAULT '';
```

**Step 2: Verify migration runs with the test suite**

Run: `make test`
Expected: all tests pass (migration is applied automatically by `testutil.NewTestDatabase`)

**Step 3: Commit**

```bash
git add internal/database/migrations/012_recipe_improvements.up.sql
git commit -m "feat: add meal_type, steps, image_data columns to recipes"
```

---

## Task 2: Update Recipe model

**Files:**
- Modify: `internal/models/models.go`

**Step 1: Add `RecipeMealType` type after `MealType` constants (~line 144)**

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

**Step 2: Update the `Recipe` struct**

Replace the existing `Recipe` struct with:

```go
type Recipe struct {
	ID           string
	Title        string
	Instructions string // legacy read-only; prefer Steps
	Steps        []string
	Ingredients  []IngredientGroup
	MealType     *RecipeMealType
	Servings     *int
	PrepTime     *string
	CookTime     *string
	SourceURL    *string
	CategoryID   *string
	HasImage     bool // computed: image_data != ''
	CreatedByUserID string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
```

**Step 3: Verify the project still compiles**

Run: `go build ./...`
Expected: no errors (existing code uses `Instructions` which is still present)

**Step 4: Commit**

```bash
git add internal/models/models.go
git commit -m "feat: add MealType, Steps, HasImage to Recipe model"
```

---

## Task 3: Update repository — new fields

**Files:**
- Modify: `internal/repository/recipes.go`
- Modify: `internal/repository/recipes_test.go`

**Step 1: Write failing tests for new fields**

Add to `internal/repository/recipes_test.go`:

```go
func TestRecipeRepository_MealTypeAndSteps(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	recipeRepo := repository.NewRecipeRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	mealType := models.RecipeMealTypeDinner
	created, err := recipeRepo.Create(ctx, models.Recipe{
		Title:           "Roast Chicken",
		Steps:           []string{"Preheat oven to 200C.", "Season the chicken.", "Roast for 90 minutes."},
		MealType:        &mealType,
		Ingredients:     []models.IngredientGroup{{Name: "Main", Items: []string{"1 whole chicken"}}},
		CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating recipe: %v", err)
	}

	found, err := recipeRepo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("finding recipe: %v", err)
	}
	if found.MealType == nil || *found.MealType != models.RecipeMealTypeDinner {
		t.Errorf("expected meal type dinner, got %v", found.MealType)
	}
	if len(found.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(found.Steps))
	}
	if found.Steps[0] != "Preheat oven to 200C." {
		t.Errorf("unexpected first step: %s", found.Steps[0])
	}
}

func TestRecipeRepository_FindAll_IncludesMealTypeAndHasImage(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	recipeRepo := repository.NewRecipeRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	mealType := models.RecipeMealTypeBreakfast
	recipeRepo.Create(ctx, models.Recipe{
		Title: "Pancakes", MealType: &mealType,
		Steps: []string{"Mix batter.", "Cook on griddle."},
		Ingredients: []models.IngredientGroup{}, CreatedByUserID: user.ID,
	})

	recipes, err := recipeRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("finding recipes: %v", err)
	}
	if len(recipes) == 0 {
		t.Fatal("expected at least 1 recipe")
	}
	if recipes[0].MealType == nil || *recipes[0].MealType != models.RecipeMealTypeBreakfast {
		t.Errorf("expected breakfast, got %v", recipes[0].MealType)
	}
	if recipes[0].HasImage {
		t.Error("expected HasImage false for recipe without image")
	}
}

func TestRecipeRepository_Update_PreservesInstructionsUpdatesSteps(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	recipeRepo := repository.NewRecipeRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	created, _ := recipeRepo.Create(ctx, models.Recipe{
		Title: "Legacy Recipe", Instructions: "Old free text instructions.",
		Ingredients: []models.IngredientGroup{}, CreatedByUserID: user.ID,
	})

	created.Steps = []string{"Step one.", "Step two."}
	mealType := models.RecipeMealTypeLunch
	created.MealType = &mealType

	if err := recipeRepo.Update(ctx, created); err != nil {
		t.Fatalf("updating: %v", err)
	}

	found, _ := recipeRepo.FindByID(ctx, created.ID)
	if found.Instructions != "Old free text instructions." {
		t.Errorf("expected legacy instructions preserved, got: %s", found.Instructions)
	}
	if len(found.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(found.Steps))
	}
	if found.MealType == nil || *found.MealType != models.RecipeMealTypeLunch {
		t.Errorf("expected lunch, got %v", found.MealType)
	}
}
```

**Step 2: Run tests to confirm they fail**

Run: `make test`
Expected: failures about `MealType`, `Steps`, `HasImage` not being populated.

**Step 3: Update `FindAll` query and scan**

Replace the `FindAll` method in `internal/repository/recipes.go`:

```go
func (repository *SQLiteRecipeRepository) FindAll(ctx context.Context) ([]models.Recipe, error) {
	rows, err := repository.database.QueryContext(ctx,
		`SELECT id, title, ingredients, servings, prep_time, cook_time,
			source_url, category_id, meal_type,
			CASE WHEN image_data != '' THEN 1 ELSE 0 END,
			created_by_user_id, created_at, updated_at
		FROM recipes ORDER BY title ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("finding recipes: %w", err)
	}
	defer rows.Close()

	var recipes []models.Recipe
	for rows.Next() {
		var recipe models.Recipe
		var ingredientsJSON string
		var mealTypeRaw *string
		var hasImageInt int
		if err := rows.Scan(
			&recipe.ID, &recipe.Title, &ingredientsJSON,
			&recipe.Servings, &recipe.PrepTime, &recipe.CookTime,
			&recipe.SourceURL, &recipe.CategoryID, &mealTypeRaw, &hasImageInt,
			&recipe.CreatedByUserID, &recipe.CreatedAt, &recipe.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning recipe: %w", err)
		}
		if err := json.Unmarshal([]byte(ingredientsJSON), &recipe.Ingredients); err != nil {
			return nil, fmt.Errorf("unmarshalling ingredients: %w", err)
		}
		if mealTypeRaw != nil {
			mt := models.RecipeMealType(*mealTypeRaw)
			recipe.MealType = &mt
		}
		recipe.HasImage = hasImageInt != 0
		recipes = append(recipes, recipe)
	}
	return recipes, rows.Err()
}
```

**Step 4: Update `FindByID` query and scan**

Replace the `FindByID` method:

```go
func (repository *SQLiteRecipeRepository) FindByID(ctx context.Context, id string) (models.Recipe, error) {
	var recipe models.Recipe
	var ingredientsJSON string
	var stepsJSON string
	var mealTypeRaw *string
	var hasImageInt int
	err := repository.database.QueryRowContext(ctx,
		`SELECT id, title, ingredients, instructions, steps, servings, prep_time, cook_time,
			source_url, category_id, meal_type,
			CASE WHEN image_data != '' THEN 1 ELSE 0 END,
			created_by_user_id, created_at, updated_at
		FROM recipes WHERE id = ?`, id,
	).Scan(
		&recipe.ID, &recipe.Title, &ingredientsJSON, &recipe.Instructions, &stepsJSON,
		&recipe.Servings, &recipe.PrepTime, &recipe.CookTime,
		&recipe.SourceURL, &recipe.CategoryID, &mealTypeRaw, &hasImageInt,
		&recipe.CreatedByUserID, &recipe.CreatedAt, &recipe.UpdatedAt,
	)
	if err != nil {
		return models.Recipe{}, fmt.Errorf("finding recipe by id: %w", err)
	}
	if err := json.Unmarshal([]byte(ingredientsJSON), &recipe.Ingredients); err != nil {
		return models.Recipe{}, fmt.Errorf("unmarshalling ingredients: %w", err)
	}
	if err := json.Unmarshal([]byte(stepsJSON), &recipe.Steps); err != nil {
		return models.Recipe{}, fmt.Errorf("unmarshalling steps: %w", err)
	}
	if mealTypeRaw != nil {
		mt := models.RecipeMealType(*mealTypeRaw)
		recipe.MealType = &mt
	}
	recipe.HasImage = hasImageInt != 0
	return recipe, nil
}
```

**Step 5: Update `Create` to persist steps and meal_type**

Replace the `Create` method:

```go
func (repository *SQLiteRecipeRepository) Create(ctx context.Context, recipe models.Recipe) (models.Recipe, error) {
	if recipe.ID == "" {
		recipe.ID = uuid.New().String()
	}
	now := time.Now()
	recipe.CreatedAt = now
	recipe.UpdatedAt = now

	if recipe.Ingredients == nil {
		recipe.Ingredients = []models.IngredientGroup{}
	}
	if recipe.Steps == nil {
		recipe.Steps = []string{}
	}

	ingredientsJSON, err := json.Marshal(recipe.Ingredients)
	if err != nil {
		return models.Recipe{}, fmt.Errorf("marshalling ingredients: %w", err)
	}
	stepsJSON, err := json.Marshal(recipe.Steps)
	if err != nil {
		return models.Recipe{}, fmt.Errorf("marshalling steps: %w", err)
	}

	var mealTypeStr *string
	if recipe.MealType != nil {
		s := string(*recipe.MealType)
		mealTypeStr = &s
	}

	_, err = repository.database.ExecContext(ctx,
		`INSERT INTO recipes (id, title, ingredients, instructions, steps, servings, prep_time, cook_time,
			source_url, category_id, meal_type, created_by_user_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		recipe.ID, recipe.Title, string(ingredientsJSON), recipe.Instructions, string(stepsJSON),
		recipe.Servings, recipe.PrepTime, recipe.CookTime,
		recipe.SourceURL, recipe.CategoryID, mealTypeStr,
		recipe.CreatedByUserID, recipe.CreatedAt, recipe.UpdatedAt,
	)
	if err != nil {
		return models.Recipe{}, fmt.Errorf("creating recipe: %w", err)
	}
	return recipe, nil
}
```

**Step 6: Update `Update` to persist steps and meal_type (not instructions)**

Replace the `Update` method:

```go
func (repository *SQLiteRecipeRepository) Update(ctx context.Context, recipe models.Recipe) error {
	recipe.UpdatedAt = time.Now()

	if recipe.Ingredients == nil {
		recipe.Ingredients = []models.IngredientGroup{}
	}
	if recipe.Steps == nil {
		recipe.Steps = []string{}
	}

	ingredientsJSON, err := json.Marshal(recipe.Ingredients)
	if err != nil {
		return fmt.Errorf("marshalling ingredients: %w", err)
	}
	stepsJSON, err := json.Marshal(recipe.Steps)
	if err != nil {
		return fmt.Errorf("marshalling steps: %w", err)
	}

	var mealTypeStr *string
	if recipe.MealType != nil {
		s := string(*recipe.MealType)
		mealTypeStr = &s
	}

	_, err = repository.database.ExecContext(ctx,
		`UPDATE recipes SET title = ?, ingredients = ?, steps = ?, servings = ?,
			prep_time = ?, cook_time = ?, source_url = ?, category_id = ?, meal_type = ?, updated_at = ?
		WHERE id = ?`,
		recipe.Title, string(ingredientsJSON), string(stepsJSON), recipe.Servings,
		recipe.PrepTime, recipe.CookTime, recipe.SourceURL, recipe.CategoryID,
		mealTypeStr, recipe.UpdatedAt, recipe.ID,
	)
	if err != nil {
		return fmt.Errorf("updating recipe: %w", err)
	}
	return nil
}
```

**Step 7: Run tests**

Run: `make test`
Expected: all tests pass, including the three new tests.

**Step 8: Commit**

```bash
git add internal/repository/recipes.go internal/repository/recipes_test.go
git commit -m "feat: update recipe repository to persist and return meal_type, steps, HasImage"
```

---

## Task 4: Update repository — image methods

**Files:**
- Modify: `internal/repository/recipes.go`
- Modify: `internal/repository/recipes_test.go`

**Step 1: Write failing tests**

Add to `internal/repository/recipes_test.go`:

```go
func TestRecipeRepository_ImageMethods(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	recipeRepo := repository.NewRecipeRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)
	created, _ := recipeRepo.Create(ctx, models.Recipe{
		Title: "Photo Recipe", Ingredients: []models.IngredientGroup{}, CreatedByUserID: user.ID,
	})

	// no image initially
	data, err := recipeRepo.FindImageData(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindImageData: %v", err)
	}
	if data != "" {
		t.Errorf("expected empty image data, got %q", data)
	}

	found, _ := recipeRepo.FindByID(ctx, created.ID)
	if found.HasImage {
		t.Error("expected HasImage false before upload")
	}

	// upload image
	fakeDataURI := "data:image/png;base64,abc123"
	if err := recipeRepo.UpdateImage(ctx, created.ID, fakeDataURI); err != nil {
		t.Fatalf("UpdateImage: %v", err)
	}

	data, err = recipeRepo.FindImageData(ctx, created.ID)
	if err != nil {
		t.Fatalf("FindImageData after upload: %v", err)
	}
	if data != fakeDataURI {
		t.Errorf("expected %q, got %q", fakeDataURI, data)
	}

	found, _ = recipeRepo.FindByID(ctx, created.ID)
	if !found.HasImage {
		t.Error("expected HasImage true after upload")
	}

	// remove image
	if err := recipeRepo.ClearImage(ctx, created.ID); err != nil {
		t.Fatalf("ClearImage: %v", err)
	}

	data, _ = recipeRepo.FindImageData(ctx, created.ID)
	if data != "" {
		t.Errorf("expected empty after clear, got %q", data)
	}
}
```

**Step 2: Run to confirm failure**

Run: `make test`
Expected: compile error — `FindImageData`, `UpdateImage`, `ClearImage` undefined.

**Step 3: Add methods to the interface**

In `internal/repository/recipes.go`, update `RecipeRepository` interface:

```go
type RecipeRepository interface {
	FindByID(ctx context.Context, id string) (models.Recipe, error)
	FindAll(ctx context.Context) ([]models.Recipe, error)
	Create(ctx context.Context, recipe models.Recipe) (models.Recipe, error)
	Update(ctx context.Context, recipe models.Recipe) error
	Delete(ctx context.Context, id string) error
	FindImageData(ctx context.Context, id string) (string, error)
	UpdateImage(ctx context.Context, id string, imageData string) error
	ClearImage(ctx context.Context, id string) error
}
```

**Step 4: Implement the three methods on `SQLiteRecipeRepository`**

Add after the `Delete` method:

```go
func (repository *SQLiteRecipeRepository) FindImageData(ctx context.Context, id string) (string, error) {
	var imageData string
	err := repository.database.QueryRowContext(ctx,
		`SELECT image_data FROM recipes WHERE id = ?`, id,
	).Scan(&imageData)
	if err != nil {
		return "", fmt.Errorf("finding image data: %w", err)
	}
	return imageData, nil
}

func (repository *SQLiteRecipeRepository) UpdateImage(ctx context.Context, id string, imageData string) error {
	_, err := repository.database.ExecContext(ctx,
		`UPDATE recipes SET image_data = ?, updated_at = ? WHERE id = ?`,
		imageData, time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("updating recipe image: %w", err)
	}
	return nil
}

func (repository *SQLiteRecipeRepository) ClearImage(ctx context.Context, id string) error {
	_, err := repository.database.ExecContext(ctx,
		`UPDATE recipes SET image_data = '', updated_at = ? WHERE id = ?`,
		time.Now(), id,
	)
	if err != nil {
		return fmt.Errorf("clearing recipe image: %w", err)
	}
	return nil
}
```

**Step 5: Run tests**

Run: `make test`
Expected: all tests pass.

**Step 6: Commit**

```bash
git add internal/repository/recipes.go internal/repository/recipes_test.go
git commit -m "feat: add image methods to RecipeRepository"
```

---

## Task 5: Update handlers — form parsing and new fragment

**Files:**
- Modify: `internal/handlers/recipes.go`

**Step 1: Add `parseMealType` and `parseSteps` helpers**

Add at the bottom of `internal/handlers/recipes.go`, alongside `parseIngredientGroups`:

```go
func parseMealType(value string) *models.RecipeMealType {
	switch models.RecipeMealType(value) {
	case models.RecipeMealTypeBreakfast, models.RecipeMealTypeLunch,
		models.RecipeMealTypeDinner, models.RecipeMealTypeSide, models.RecipeMealTypeDessert:
		mt := models.RecipeMealType(value)
		return &mt
	}
	return nil
}

func parseSteps(r *http.Request) []string {
	var steps []string
	for i := 0; ; i++ {
		step := strings.TrimSpace(r.FormValue(fmt.Sprintf("step_%d", i)))
		if step == "" {
			break
		}
		steps = append(steps, step)
	}
	return steps
}

func splitIntoSteps(text string) []string {
	var steps []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			steps = append(steps, line)
		}
	}
	return steps
}
```

**Step 2: Update `Create` handler**

Replace the recipe construction block in `Create` (around lines 109–131):

```go
recipe := models.Recipe{
	Title:           r.FormValue("title"),
	Steps:           parseSteps(r),
	Ingredients:     parseIngredientGroups(r),
	MealType:        parseMealType(r.FormValue("meal_type")),
	CreatedByUserID: user.ID,
}
```

Remove the `Instructions` assignment — do not set `recipe.Instructions` from the form.

**Step 3: Update `Update` handler**

Replace the field assignment block in `Update` (around lines 183–213):

```go
recipe.Title = r.FormValue("title")
recipe.Steps = parseSteps(r)
recipe.Ingredients = parseIngredientGroups(r)
recipe.MealType = parseMealType(r.FormValue("meal_type"))
// Instructions intentionally not updated — preserved from DB
```

Remove the existing `recipe.Instructions = r.FormValue("instructions")` line.

**Step 4: Update `EditForm` handler to pre-populate steps from legacy instructions**

In `EditForm`, after fetching `recipe`, add:

```go
if len(recipe.Steps) == 0 && recipe.Instructions != "" {
	recipe.Steps = splitIntoSteps(recipe.Instructions)
}
```

**Step 5: Add `Step` fragment handler**

```go
func (handler *RecipeHandler) Step(w http.ResponseWriter, r *http.Request) {
	indexStr := r.URL.Query().Get("index")
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		index = 0
	}
	component := pages.RecipeStepField(index, "")
	component.Render(r.Context(), w)
}
```

**Step 6: Verify compilation**

Run: `go build ./...`
Expected: no errors (templates not yet updated — ignore missing templ symbols until Task 9).

**Step 7: Commit**

```bash
git add internal/handlers/recipes.go
git commit -m "feat: update recipe handler to parse meal_type and steps"
```

---

## Task 6: Add image handlers

**Files:**
- Modify: `internal/handlers/recipes.go`

**Step 1: Add image handler methods**

Add after the `Delete` handler:

```go
const maxRecipeImageBytes = 2 * 1024 * 1024 // 2 MB

func (handler *RecipeHandler) ServeImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipeID := chi.URLParam(r, "id")

	imageData, err := handler.recipeRepo.FindImageData(ctx, recipeID)
	if err != nil || imageData == "" {
		http.NotFound(w, r)
		return
	}

	withoutPrefix, ok := strings.CutPrefix(imageData, "data:")
	if !ok {
		http.NotFound(w, r)
		return
	}
	parts := strings.SplitN(withoutPrefix, ";base64,", 2)
	if len(parts) != 2 {
		http.NotFound(w, r)
		return
	}

	imageBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		slog.Error("decoding recipe image", "error", err)
		http.Error(w, "Corrupted image data", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", parts[0])
	w.WriteHeader(http.StatusOK)
	w.Write(imageBytes)
}

func (handler *RecipeHandler) UploadImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipeID := chi.URLParam(r, "id")

	if _, err := handler.recipeRepo.FindByID(ctx, recipeID); err != nil {
		http.NotFound(w, r)
		return
	}

	if err := r.ParseMultipartForm(maxRecipeImageBytes + 1024); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Missing image file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	imageBytes, err := io.ReadAll(io.LimitReader(file, maxRecipeImageBytes+1))
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	if len(imageBytes) > maxRecipeImageBytes {
		http.Error(w, "Image exceeds 2 MB limit", http.StatusBadRequest)
		return
	}

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(imageBytes)
	}

	dataURI := "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(imageBytes)

	if err := handler.recipeRepo.UpdateImage(ctx, recipeID, dataURI); err != nil {
		slog.Error("updating recipe image", "error", err)
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/recipes/%s", recipeID), http.StatusFound)
}

func (handler *RecipeHandler) RemoveImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipeID := chi.URLParam(r, "id")

	if err := handler.recipeRepo.ClearImage(ctx, recipeID); err != nil {
		slog.Error("clearing recipe image", "error", err)
		http.Error(w, "Failed to remove image", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/recipes/%s", recipeID), http.StatusFound)
}
```

**Step 2: Add required imports**

Ensure these are in the import block: `"encoding/base64"`, `"io"`.

**Step 3: Compile check**

Run: `go build ./...`
Expected: no errors.

**Step 4: Commit**

```bash
git add internal/handlers/recipes.go
git commit -m "feat: add recipe image upload, serve, and remove handlers"
```

---

## Task 7: Add cook mode handler

**Files:**
- Modify: `internal/handlers/recipes.go`

**Step 1: Add `RecipeCookProps` type to the pages package**

This props type will be added to the template file in Task 9. For now, add a placeholder type to note its shape (skip if it causes compilation issues — finalize in Task 9).

**Step 2: Add `CookMode` handler**

Add after `RemoveImage`:

```go
func (handler *RecipeHandler) CookMode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)
	recipeID := chi.URLParam(r, "id")

	recipe, err := handler.recipeRepo.FindByID(ctx, recipeID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	steps := recipe.Steps
	if len(steps) == 0 && recipe.Instructions != "" {
		steps = splitIntoSteps(recipe.Instructions)
	}

	component := pages.RecipeCook(pages.RecipeCookProps{
		User:   user,
		Recipe: recipe,
		Steps:  steps,
	})
	component.Render(ctx, w)
}
```

**Step 3: Compile check (will succeed after templates are added in Task 10)**

Run: `go build ./...`
Expected: errors about `RecipeCook` undefined — that's fine. Fix in Task 10.

**Step 4: Commit after templates are working (defer commit to Task 10)**

---

## Task 8: Wire new routes

**Files:**
- Modify: `internal/server/server.go`

**Step 1: Add routes in the authenticated group, around the existing recipe routes**

Replace the existing recipe routes block (~lines 115–122):

```go
r.Get("/recipes", recipeHandler.List)
r.Get("/recipes/new", recipeHandler.CreateForm)
r.Get("/recipes/ingredient-group", recipeHandler.IngredientGroup)
r.Get("/recipes/step", recipeHandler.Step)
r.Get("/recipes/{id}", recipeHandler.Detail)
r.Get("/recipes/{id}/image", recipeHandler.ServeImage)
r.Get("/recipes/{id}/cook", recipeHandler.CookMode)
r.Post("/recipes", recipeHandler.Create)
r.Post("/recipes/{id}/image", recipeHandler.UploadImage)
r.Post("/recipes/{id}/image/delete", recipeHandler.RemoveImage)
r.Get("/recipes/{id}/edit", recipeHandler.EditForm)
r.Post("/recipes/{id}", recipeHandler.Update)
r.Post("/recipes/{id}/delete", recipeHandler.Delete)
```

Note: static-path routes (`/recipes/new`, `/recipes/ingredient-group`, `/recipes/step`) must appear before the wildcard `{id}` routes.

**Step 2: Compile check**

Run: `go build ./...`
Expected: errors about missing templ symbols until templates are added — that's fine.

**Step 3: Commit after templates compile (defer to end of Task 11)**

---

## Task 9: Update recipe list and detail templates

**Files:**
- Modify: `templates/pages/recipes.templ`

**Step 1: Add helper functions for meal type**

Add at the bottom of `recipes.templ`:

```go
func recipeMealTypeLabel(mt models.RecipeMealType) string {
	switch mt {
	case models.RecipeMealTypeBreakfast:
		return "Breakfast"
	case models.RecipeMealTypeLunch:
		return "Lunch"
	case models.RecipeMealTypeDinner:
		return "Dinner"
	case models.RecipeMealTypeSide:
		return "Side"
	case models.RecipeMealTypeDessert:
		return "Dessert"
	}
	return ""
}
```

**Step 2: Update `RecipeList` template — cards with image thumbnail and meal type badge**

Replace the card `<a>` block inside the grid:

```templ
<a href={ templ.SafeURL(fmt.Sprintf("/recipes/%s", recipe.ID)) } class="block bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl overflow-hidden hover:shadow-md dark:hover:ring-slate-600 transition-all duration-200">
    if recipe.HasImage {
        <img src={ templ.SafeURL(fmt.Sprintf("/recipes/%s/image", recipe.ID)) } alt={ recipe.Title } class="w-full h-40 object-cover"/>
    } else {
        <div class="w-full h-40 bg-zinc-100 dark:bg-slate-700 flex items-center justify-center">
            <svg class="h-10 w-10 text-zinc-300 dark:text-slate-600" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M12 6.042A8.967 8.967 0 006 3.75c-1.052 0-2.062.18-3 .512v14.25A8.987 8.987 0 016 18c2.305 0 4.408.867 6 2.292m0-14.25a8.966 8.966 0 016-2.292c1.052 0 2.062.18 3 .512v14.25A8.987 8.987 0 0018 18a8.967 8.967 0 00-6 2.292m0-14.25v14.25"/></svg>
        </div>
    }
    <div class="p-5">
        <h3 class="text-lg font-medium text-stone-900 dark:text-slate-100 mb-2">{ recipe.Title }</h3>
        <div class="flex flex-wrap gap-2 items-center text-sm text-stone-500 dark:text-slate-400">
            if recipe.MealType != nil {
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-50 dark:bg-amber-500/15 text-amber-700 dark:text-amber-400">
                    { recipeMealTypeLabel(*recipe.MealType) }
                </span>
            }
            if recipe.CategoryID != nil {
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-indigo-50 dark:bg-indigo-500/15 text-indigo-700 dark:text-indigo-400">
                    { recipeCategory(props.CategoryMap, *recipe.CategoryID) }
                </span>
            }
            if recipe.Servings != nil {
                <span>{ fmt.Sprintf("%d servings", *recipe.Servings) }</span>
            }
            if recipe.PrepTime != nil {
                <span>Prep: { *recipe.PrepTime }</span>
            }
            if recipe.CookTime != nil {
                <span>Cook: { *recipe.CookTime }</span>
            }
        </div>
    </div>
</a>
```

**Step 3: Update `RecipeDetail` template**

Replace the `RecipeDetail` templ body:

```templ
templ RecipeDetail(props RecipeDetailProps) {
	@layouts.Base(props.Recipe.Title, props.User, "/recipes") {
		<div class="max-w-3xl mx-auto space-y-6">
			<!-- Header -->
			<div class="flex justify-between items-start">
				<div>
					<h1 class="text-xl font-semibold text-stone-800 dark:text-slate-100">{ props.Recipe.Title }</h1>
					<div class="flex flex-wrap gap-2 mt-2">
						if props.Recipe.MealType != nil {
							<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-50 dark:bg-amber-500/15 text-amber-700 dark:text-amber-400">
								{ recipeMealTypeLabel(*props.Recipe.MealType) }
							</span>
						}
						if props.CategoryName != "" {
							<span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-indigo-50 dark:bg-indigo-500/15 text-indigo-700 dark:text-indigo-400">
								{ props.CategoryName }
							</span>
						}
					</div>
				</div>
				<div class="flex space-x-2">
					<a href={ templ.SafeURL(fmt.Sprintf("/recipes/%s/cook", props.Recipe.ID)) } class="inline-flex items-center gap-1.5 bg-emerald-600 text-white px-3 py-1.5 rounded-xl shadow-sm text-sm font-medium hover:bg-emerald-500 transition-colors duration-150 hover:-translate-y-px active:translate-y-0">
						Cook Mode
					</a>
					<a href={ templ.SafeURL(fmt.Sprintf("/recipes/%s/edit", props.Recipe.ID)) } class="inline-flex items-center gap-1.5 bg-indigo-600 text-white px-3 py-1.5 rounded-xl shadow-sm text-sm font-medium hover:bg-indigo-500 transition-colors duration-150 hover:-translate-y-px active:translate-y-0">
						@components.IconPencil("h-4 w-4")
						Edit
					</a>
					<form method="POST" action={ templ.SafeURL(fmt.Sprintf("/recipes/%s/delete", props.Recipe.ID)) } class="inline">
						<button type="submit" onclick="return confirm('Delete this recipe?')" class="inline-flex items-center gap-1.5 bg-red-600 text-white px-3 py-1.5 rounded-xl text-sm font-medium hover:bg-red-500 transition-colors duration-150">
							@components.IconTrash("h-4 w-4")
							Delete
						</button>
					</form>
				</div>
			</div>

			<!-- Recipe image -->
			if props.Recipe.HasImage {
				<img src={ templ.SafeURL(fmt.Sprintf("/recipes/%s/image", props.Recipe.ID)) } alt={ props.Recipe.Title } class="w-full rounded-xl object-cover max-h-72"/>
			}

			<!-- Metadata -->
			<div class="flex flex-wrap gap-4 text-sm text-stone-600 dark:text-slate-400">
				if props.Recipe.Servings != nil {
					<div class="flex items-center gap-1">
						<span class="font-medium dark:text-slate-300">Servings:</span>
						<span>{ fmt.Sprintf("%d", *props.Recipe.Servings) }</span>
					</div>
				}
				if props.Recipe.PrepTime != nil {
					<div class="flex items-center gap-1">
						<span class="font-medium dark:text-slate-300">Prep:</span>
						<span>{ *props.Recipe.PrepTime }</span>
					</div>
				}
				if props.Recipe.CookTime != nil {
					<div class="flex items-center gap-1">
						<span class="font-medium dark:text-slate-300">Cook:</span>
						<span>{ *props.Recipe.CookTime }</span>
					</div>
				}
				if props.Recipe.SourceURL != nil {
					<div class="flex items-center gap-1">
						@components.IconLink("h-4 w-4 text-indigo-600 dark:text-indigo-400")
						<a href={ templ.SafeURL(*props.Recipe.SourceURL) } target="_blank" rel="noopener noreferrer" class="text-indigo-600 dark:text-indigo-400 hover:text-indigo-800 dark:hover:text-indigo-300 underline transition-colors duration-150">Source</a>
					</div>
				}
			</div>

			<!-- Ingredients -->
			if len(props.Recipe.Ingredients) > 0 {
				<div class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-6">
					<h2 class="text-lg font-medium text-stone-900 dark:text-slate-100 mb-4">Ingredients</h2>
					for _, group := range props.Recipe.Ingredients {
						if group.Name != "" && group.Name != "Main" {
							<h3 class="text-sm font-medium text-stone-700 dark:text-slate-300 mt-3 mb-1">{ group.Name }</h3>
						}
						<ul class="list-disc list-inside space-y-1 text-sm text-stone-700 dark:text-slate-300">
							for _, item := range group.Items {
								<li>{ item }</li>
							}
						</ul>
					}
				</div>
			}

			<!-- Instructions / Steps -->
			if len(props.Recipe.Steps) > 0 {
				<div class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-6">
					<h2 class="text-lg font-medium text-stone-900 dark:text-slate-100 mb-4">Instructions</h2>
					<ol class="space-y-4">
						for i, step := range props.Recipe.Steps {
							<li class="flex gap-4">
								<span class="flex-shrink-0 w-7 h-7 rounded-full bg-indigo-100 dark:bg-indigo-500/20 text-indigo-700 dark:text-indigo-400 text-sm font-semibold flex items-center justify-center">{ strconv.Itoa(i + 1) }</span>
								<span class="text-sm text-stone-700 dark:text-slate-300 pt-1">{ step }</span>
							</li>
						}
					</ol>
				</div>
			} else if props.Recipe.Instructions != "" {
				<div class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-6">
					<h2 class="text-lg font-medium text-stone-900 dark:text-slate-100 mb-4">Instructions</h2>
					<div class="text-sm text-stone-700 dark:text-slate-300 whitespace-pre-wrap">{ props.Recipe.Instructions }</div>
				</div>
			}

			<!-- Image upload / remove -->
			<div class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-6">
				<h2 class="text-base font-medium text-stone-900 dark:text-slate-100 mb-4">Recipe Photo</h2>
				if props.Recipe.HasImage {
					<div class="flex items-center gap-4">
						<img src={ templ.SafeURL(fmt.Sprintf("/recipes/%s/image", props.Recipe.ID)) } alt="Current photo" class="h-20 w-20 rounded-lg object-cover"/>
						<div class="space-y-2">
							<form method="POST" action={ templ.SafeURL(fmt.Sprintf("/recipes/%s/image", props.Recipe.ID)) } enctype="multipart/form-data" class="flex items-center gap-2">
								<input type="file" name="image" accept="image/*" class="text-sm text-stone-600 dark:text-slate-400"/>
								<button type="submit" class="text-sm text-indigo-600 dark:text-indigo-400 hover:underline font-medium">Replace</button>
							</form>
							<form method="POST" action={ templ.SafeURL(fmt.Sprintf("/recipes/%s/image/delete", props.Recipe.ID)) }>
								<button type="submit" class="text-sm text-red-600 dark:text-red-400 hover:underline font-medium">Remove photo</button>
							</form>
						</div>
					</div>
				} else {
					<form method="POST" action={ templ.SafeURL(fmt.Sprintf("/recipes/%s/image", props.Recipe.ID)) } enctype="multipart/form-data" class="flex items-center gap-3">
						<input type="file" name="image" accept="image/*" class="text-sm text-stone-600 dark:text-slate-400"/>
						<button type="submit" class="bg-indigo-600 text-white px-3 py-1.5 rounded-xl text-sm font-medium hover:bg-indigo-500 transition-colors duration-150">Upload Photo</button>
					</form>
				}
			</div>

			<div class="pt-2">
				<a href="/recipes" class="inline-flex items-center gap-1 text-stone-600 dark:text-slate-400 hover:text-stone-900 dark:hover:text-slate-100 text-sm transition-colors duration-150">
					@components.IconChevronLeft("h-4 w-4")
					Back to recipes
				</a>
			</div>
		</div>
	}
}
```

**Step 4: Run templ generate**

Run: `make templ`
Expected: `*_templ.go` files regenerated with no errors.

**Step 5: Run tests**

Run: `make test`
Expected: all tests pass.

**Step 6: Commit**

```bash
git add templates/pages/recipes.templ templates/pages/recipes_templ.go
git commit -m "feat: update recipe list and detail templates with images, meal types, and structured steps"
```

---

## Task 10: Update recipe form template and add step fragment

**Files:**
- Modify: `templates/pages/recipes.templ`

**Step 1: Add `RecipeStepField` templ component**

Add after `IngredientGroupFields`:

```templ
templ RecipeStepField(index int, value string) {
	<div class="flex gap-2 items-start" id={ fmt.Sprintf("recipe-step-%d", index) }>
		<span class="flex-shrink-0 w-7 h-7 rounded-full bg-zinc-100 dark:bg-slate-700 text-stone-500 dark:text-slate-400 text-sm font-semibold flex items-center justify-center mt-1">{ strconv.Itoa(index + 1) }</span>
		<textarea
			name={ fmt.Sprintf("step_%d", index) }
			rows="2"
			placeholder="Describe this step..."
			class="flex-1"
		>{ value }</textarea>
		if index > 0 {
			<button
				type="button"
				onclick="this.closest('[id^=recipe-step-]').remove()"
				class="mt-1 text-stone-400 dark:text-slate-500 hover:text-red-600 dark:hover:text-red-400 text-sm transition-colors duration-150 flex-shrink-0"
			>Remove</button>
		}
	</div>
}
```

**Step 2: Update `RecipeForm` template**

Replace the full `RecipeForm` templ:

```templ
templ RecipeForm(props RecipeFormProps) {
	@layouts.Base(recipeFormTitle(props.IsEdit), props.User, "/recipes") {
		<div class="max-w-2xl mx-auto">
			<h1 class="text-xl font-semibold text-stone-800 dark:text-slate-100 mb-6">{ recipeFormTitle(props.IsEdit) }</h1>

			<form
				if props.IsEdit && props.Recipe != nil {
					action={ templ.SafeURL(fmt.Sprintf("/recipes/%s", props.Recipe.ID)) }
				} else {
					action="/recipes"
				}
				method="POST"
				class="space-y-6 bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-6"
			>
				<div>
					<label for="title" class="block text-sm font-medium text-stone-700 dark:text-slate-300">Title</label>
					<input
						type="text"
						id="title"
						name="title"
						required
						if props.Recipe != nil {
							value={ props.Recipe.Title }
						}
					/>
				</div>

				<div>
					<label for="meal_type" class="block text-sm font-medium text-stone-700 dark:text-slate-300">Meal Type</label>
					<select id="meal_type" name="meal_type">
						<option value="">None</option>
						for _, option := range recipeMealTypeOptions() {
							<option
								value={ option.value }
								if props.Recipe != nil && props.Recipe.MealType != nil && string(*props.Recipe.MealType) == option.value {
									selected
								}
							>{ option.label }</option>
						}
					</select>
				</div>

				<div>
					<label for="category_id" class="block text-sm font-medium text-stone-700 dark:text-slate-300">Category</label>
					<select id="category_id" name="category_id">
						<option value="">No Category</option>
						for _, cat := range props.Categories {
							<option
								value={ cat.ID }
								if props.Recipe != nil && props.Recipe.CategoryID != nil && *props.Recipe.CategoryID == cat.ID {
									selected
								}
							>{ cat.Name }</option>
						}
					</select>
				</div>

				<div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
					<div>
						<label for="servings" class="block text-sm font-medium text-stone-700 dark:text-slate-300">Servings</label>
						<input
							type="number"
							id="servings"
							name="servings"
							min="1"
							if props.Recipe != nil && props.Recipe.Servings != nil {
								value={ strconv.Itoa(*props.Recipe.Servings) }
							}
						/>
					</div>
					<div>
						<label for="prep_time" class="block text-sm font-medium text-stone-700 dark:text-slate-300">Prep Time</label>
						<input
							type="text"
							id="prep_time"
							name="prep_time"
							placeholder="e.g. 15 min"
							if props.Recipe != nil && props.Recipe.PrepTime != nil {
								value={ *props.Recipe.PrepTime }
							}
						/>
					</div>
					<div>
						<label for="cook_time" class="block text-sm font-medium text-stone-700 dark:text-slate-300">Cook Time</label>
						<input
							type="text"
							id="cook_time"
							name="cook_time"
							placeholder="e.g. 30 min"
							if props.Recipe != nil && props.Recipe.CookTime != nil {
								value={ *props.Recipe.CookTime }
							}
						/>
					</div>
				</div>

				<div>
					<label for="source_url" class="block text-sm font-medium text-stone-700 dark:text-slate-300">Source URL</label>
					<input
						type="url"
						id="source_url"
						name="source_url"
						placeholder="https://..."
						if props.Recipe != nil && props.Recipe.SourceURL != nil {
							value={ *props.Recipe.SourceURL }
						}
					/>
				</div>

				<!-- Ingredient Groups -->
				<div>
					<label class="block text-sm font-medium text-stone-700 dark:text-slate-300 mb-2">Ingredients</label>
					<div id="ingredient-groups" class="space-y-4">
						if props.Recipe != nil && len(props.Recipe.Ingredients) > 0 {
							for index, group := range props.Recipe.Ingredients {
								@IngredientGroupFields(index, group)
							}
						} else {
							@IngredientGroupFields(0, models.IngredientGroup{Name: "Main"})
						}
					</div>
					<button
						type="button"
						hx-get="/recipes/ingredient-group"
						hx-vals="js:{index: document.querySelectorAll('#ingredient-groups > div').length}"
						hx-target="#ingredient-groups"
						hx-swap="beforeend"
						class="mt-3 text-sm text-stone-600 dark:text-slate-300 hover:text-stone-900 dark:hover:text-slate-100 font-medium transition-colors duration-150"
					>
						+ Add Ingredient Group
					</button>
				</div>

				<!-- Steps -->
				<div>
					<label class="block text-sm font-medium text-stone-700 dark:text-slate-300 mb-2">Instructions</label>
					<div id="recipe-steps" class="space-y-3">
						if props.Recipe != nil && len(props.Recipe.Steps) > 0 {
							for index, step := range props.Recipe.Steps {
								@RecipeStepField(index, step)
							}
						} else {
							@RecipeStepField(0, "")
						}
					</div>
					<button
						type="button"
						hx-get="/recipes/step"
						hx-vals="js:{index: document.querySelectorAll('#recipe-steps > div').length}"
						hx-target="#recipe-steps"
						hx-swap="beforeend"
						class="mt-3 text-sm text-stone-600 dark:text-slate-300 hover:text-stone-900 dark:hover:text-slate-100 font-medium transition-colors duration-150"
					>
						+ Add Step
					</button>
				</div>

				<div class="flex justify-end space-x-3">
					<a href="/recipes" class="bg-white dark:bg-slate-700 py-2 px-4 border border-zinc-200 dark:border-slate-600 rounded-xl shadow-sm text-sm font-medium text-stone-700 dark:text-slate-100 hover:bg-zinc-50 dark:hover:bg-slate-600 transition-colors duration-150">Cancel</a>
					<button type="submit" class="bg-indigo-600 py-2 px-4 border border-transparent rounded-xl shadow-sm text-sm font-medium text-white hover:bg-indigo-500 transition-colors duration-150 hover:-translate-y-px active:translate-y-0">
						if props.IsEdit {
							Update
						} else {
							Create
						}
					</button>
				</div>
			</form>
		</div>
	}
}
```

**Step 3: Add helper `recipeMealTypeOptions`**

Add to the Go helper functions at the bottom of `recipes.templ`:

```go
type mealTypeOption struct {
	value string
	label string
}

func recipeMealTypeOptions() []mealTypeOption {
	return []mealTypeOption{
		{value: string(models.RecipeMealTypeBreakfast), label: "Breakfast"},
		{value: string(models.RecipeMealTypeLunch), label: "Lunch"},
		{value: string(models.RecipeMealTypeDinner), label: "Dinner"},
		{value: string(models.RecipeMealTypeSide), label: "Side"},
		{value: string(models.RecipeMealTypeDessert), label: "Dessert"},
	}
}
```

**Step 4: Run templ generate and tests**

Run: `make templ && make test`
Expected: compiles and all tests pass.

**Step 5: Commit**

```bash
git add templates/pages/recipes.templ templates/pages/recipes_templ.go
git commit -m "feat: update recipe form with meal type select and step editor"
```

---

## Task 11: Add cook mode template

**Files:**
- Modify: `templates/pages/recipes.templ`

**Step 1: Add `RecipeCookProps` struct and `RecipeCook` templ**

Add after the `RecipeForm` templ:

```go
type RecipeCookProps struct {
	User   models.User
	Recipe models.Recipe
	Steps  []string
}
```

```templ
templ RecipeCook(props RecipeCookProps) {
	@layouts.Base(props.Recipe.Title + " — Cook Mode", props.User, "/recipes") {
		<div class="max-w-2xl mx-auto space-y-6" data-step-count={ strconv.Itoa(len(props.Steps)) }>
			<!-- Header -->
			<div class="flex items-center justify-between">
				<div>
					<h1 class="text-xl font-semibold text-stone-800 dark:text-slate-100">{ props.Recipe.Title }</h1>
					if len(props.Steps) > 0 {
						<p id="step-counter" class="text-sm text-stone-500 dark:text-slate-400 mt-1">Step 1 of { strconv.Itoa(len(props.Steps)) }</p>
					}
				</div>
				<a href={ templ.SafeURL(fmt.Sprintf("/recipes/%s", props.Recipe.ID)) } class="text-sm text-stone-500 dark:text-slate-400 hover:text-stone-900 dark:hover:text-slate-100 transition-colors duration-150">
					Exit Cook Mode
				</a>
			</div>

			if len(props.Steps) == 0 {
				<div class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-8 text-center text-stone-500 dark:text-slate-400">
					<p>No steps available. <a href={ templ.SafeURL(fmt.Sprintf("/recipes/%s/edit", props.Recipe.ID)) } class="text-indigo-600 dark:text-indigo-400 hover:underline">Edit the recipe</a> to add steps.</p>
				</div>
			} else {
				<!-- Step display -->
				<div class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl p-8 min-h-48">
					for i, step := range props.Steps {
						<div
							id={ fmt.Sprintf("cook-step-%d", i) }
							if i != 0 {
								class="hidden"
							}
						>
							<div class="flex gap-4 items-start">
								<span class="flex-shrink-0 w-10 h-10 rounded-full bg-indigo-100 dark:bg-indigo-500/20 text-indigo-700 dark:text-indigo-400 text-lg font-bold flex items-center justify-center">{ strconv.Itoa(i + 1) }</span>
								<p class="text-lg text-stone-800 dark:text-slate-100 leading-relaxed pt-1">{ step }</p>
							</div>
						</div>
					}
				</div>

				<!-- Navigation -->
				<div class="flex justify-between items-center">
					<button
						id="prev-btn"
						onclick="cookNavPrev()"
						disabled
						class="inline-flex items-center gap-1.5 bg-white dark:bg-slate-700 border border-zinc-200 dark:border-slate-600 text-stone-700 dark:text-slate-100 px-4 py-2 rounded-xl text-sm font-medium hover:bg-zinc-50 dark:hover:bg-slate-600 transition-colors duration-150 disabled:opacity-40 disabled:cursor-not-allowed"
					>
						@components.IconChevronLeft("h-4 w-4")
						Previous
					</button>
					<button
						id="next-btn"
						onclick="cookNavNext()"
						if len(props.Steps) <= 1 {
							disabled
						}
						class="inline-flex items-center gap-1.5 bg-indigo-600 text-white px-4 py-2 rounded-xl text-sm font-medium hover:bg-indigo-500 transition-colors duration-150 disabled:opacity-40 disabled:cursor-not-allowed"
					>
						Next
						@components.IconChevronRight("h-4 w-4")
					</button>
				</div>
			}

			<!-- Ingredients collapsible -->
			if len(props.Recipe.Ingredients) > 0 {
				<details class="bg-white dark:bg-slate-800 ring-1 ring-zinc-200 dark:ring-slate-700 shadow-card rounded-xl">
					<summary class="px-6 py-4 text-base font-medium text-stone-900 dark:text-slate-100 cursor-pointer select-none list-none flex justify-between items-center">
						Ingredients
						<svg class="h-4 w-4 text-stone-400 dark:text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/></svg>
					</summary>
					<div class="px-6 pb-6 pt-2">
						for _, group := range props.Recipe.Ingredients {
							if group.Name != "" && group.Name != "Main" {
								<h3 class="text-sm font-medium text-stone-700 dark:text-slate-300 mt-3 mb-1">{ group.Name }</h3>
							}
							<ul class="list-disc list-inside space-y-1 text-sm text-stone-700 dark:text-slate-300">
								for _, item := range group.Items {
									<li>{ item }</li>
								}
							</ul>
						}
					</div>
				</details>
			}
		</div>

		<script>
			(function() {
				var current = 0;
				var total = parseInt(document.querySelector('[data-step-count]').dataset.stepCount, 10);

				function showStep(n) {
					document.querySelectorAll('[id^="cook-step-"]').forEach(function(el, i) {
						el.classList.toggle('hidden', i !== n);
					});
					document.getElementById('step-counter').textContent = 'Step ' + (n + 1) + ' of ' + total;
					document.getElementById('prev-btn').disabled = n === 0;
					document.getElementById('next-btn').disabled = n === total - 1;
					current = n;
				}

				window.cookNavPrev = function() { if (current > 0) showStep(current - 1); };
				window.cookNavNext = function() { if (current < total - 1) showStep(current + 1); };
			})();
		</script>
	}
}
```

Note: `components.IconChevronRight` may need to be added if not already in the components package. Check `templates/components/` — if it doesn't exist, use an inline SVG instead:

```templ
<svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 5l7 7-7 7"/></svg>
```

**Step 2: Run templ generate**

Run: `make templ`
Expected: no errors.

**Step 3: Run full test suite**

Run: `make test`
Expected: all tests pass.

**Step 4: Full build verification**

Run: `go build ./...`
Expected: no errors.

**Step 5: Commit**

```bash
git add templates/pages/recipes.templ templates/pages/recipes_templ.go internal/handlers/recipes.go internal/server/server.go
git commit -m "feat: add cook mode page and wire all recipe improvement routes"
```

---

## Final verification

Run: `make test && go build ./...`

All tests should pass. The feature is complete:
- Recipe list cards show image thumbnails and amber meal-type badges
- Recipe detail shows full image, numbered steps (with legacy fallback), cook mode button, and image upload widget
- Recipe form has meal type selector and step-by-step instruction editor (pre-populated from legacy instructions on edit)
- Cook mode at `/recipes/:id/cook` steps through instructions one at a time with prev/next navigation

# iOS Recipe CRUD Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add full recipe create/update/delete support to the iOS app, backed by new JSON API endpoints.

**Architecture:** Three new API endpoints in `api.go` (POST/PUT/DELETE) following the existing meal CRUD pattern. iOS gets a new `RecipeFormView` sheet for create/edit, plus delete support on the detail view. Image upload uses base64 data URI in the JSON body.

**Tech Stack:** Go (chi router, SQLite), Swift (SwiftUI, PhotosPicker, Observation)

---

## File Structure

### Backend (Go)
| Action | File | Responsibility |
|--------|------|---------------|
| Modify | `server/internal/handlers/api.go` | Add `CreateRecipe`, `UpdateRecipe`, `DeleteRecipe` handlers |
| Modify | `server/internal/handlers/api_test.go` | Tests for new endpoints |
| Modify | `server/internal/server/server.go` | Register new routes |

### iOS (Swift)
| Action | File | Responsibility |
|--------|------|---------------|
| Modify | `ios/FamilyHub/FamilyHub/Models/Recipe.swift` | Add `mealType`, `sourceURL` fields + `RecipeRequest` struct |
| Modify | `ios/FamilyHub/FamilyHub/Networking/APIClientProtocol.swift` | Add create/update/delete protocol methods |
| Modify | `ios/FamilyHub/FamilyHub/Networking/APIClient.swift` | Add `put` helper + implement new methods |
| Modify | `ios/FamilyHub/FamilyHub/Features/Meals/RecipesViewModel.swift` | Add create/update/delete methods |
| Create | `ios/FamilyHub/FamilyHub/Features/Meals/RecipeFormView.swift` | Form sheet for create/edit with photo picker |
| Modify | `ios/FamilyHub/FamilyHub/Features/Meals/RecipesView.swift` | Add "+" button, pass viewModel to detail |
| Modify | `ios/FamilyHub/FamilyHub/Features/Meals/RecipeDetailView.swift` | Add edit/delete toolbar actions |
| Modify | `ios/FamilyHub/FamilyHubTests/FakeAPIClient.swift` | Add fake implementations for new methods |

---

### Task 1: Backend — CreateRecipe API endpoint

**Files:**
- Modify: `server/internal/handlers/api_test.go`
- Modify: `server/internal/handlers/api.go`
- Modify: `server/internal/server/server.go`

- [ ] **Step 1: Write the failing test for CreateRecipe**

Add to `server/internal/handlers/api_test.go`:

```go
func TestCreateRecipe_API(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-create-recipe",
		Email:       "create@example.com",
		Name:        "Creator",
		Role:        models.RoleMember,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, "", "", "")

	router := chi.NewRouter()
	router.Post("/api/recipes", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
		handler.CreateRecipe(w, r.WithContext(ctx))
	})

	body := `{"title":"Pasta Carbonara","steps":["Boil pasta","Mix eggs"],"ingredients":[{"name":"Main","items":["pasta","eggs"]}],"mealType":"dinner","servings":4,"prepTime":"10 min","cookTime":"20 min","sourceURL":"https://example.com"}`
	request := httptest.NewRequest(http.MethodPost, "/api/recipes", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var recipe models.Recipe
	json.NewDecoder(recorder.Body).Decode(&recipe)
	if recipe.Title != "Pasta Carbonara" {
		t.Errorf("expected title Pasta Carbonara, got %s", recipe.Title)
	}
	if recipe.MealType == nil || string(*recipe.MealType) != "dinner" {
		t.Errorf("expected mealType dinner, got %v", recipe.MealType)
	}
	if recipe.Servings == nil || *recipe.Servings != 4 {
		t.Errorf("expected servings 4, got %v", recipe.Servings)
	}
	if len(recipe.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(recipe.Steps))
	}
}

func TestCreateRecipe_API_MissingTitle(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, "", "", "")

	router := chi.NewRouter()
	router.Post("/api/recipes", handler.CreateRecipe)

	body := `{"steps":["step 1"]}`
	request := httptest.NewRequest(http.MethodPost, "/api/recipes", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing title, got %d", recorder.Code)
	}
}

func TestCreateRecipe_API_WithImage(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-create-recipe-img",
		Email:       "img@example.com",
		Name:        "Img User",
		Role:        models.RoleMember,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, "", "", "")

	router := chi.NewRouter()
	router.Post("/api/recipes", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), middleware.UserContextKey, user)
		handler.CreateRecipe(w, r.WithContext(ctx))
	})

	body := `{"title":"Photo Recipe","imageData":"data:image/png;base64,iVBORw0KGgo="}`
	request := httptest.NewRequest(http.MethodPost, "/api/recipes", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var recipe models.Recipe
	json.NewDecoder(recorder.Body).Decode(&recipe)
	if !recipe.HasImage {
		t.Error("expected HasImage to be true after uploading image data")
	}

	imageData, _ := recipeRepo.FindImageData(ctx, recipe.ID)
	if imageData != "data:image/png;base64,iVBORw0KGgo=" {
		t.Errorf("expected stored image data URI, got %q", imageData)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./internal/handlers/ -run TestCreateRecipe_API -v`
Expected: FAIL — `handler.CreateRecipe` undefined

- [ ] **Step 3: Implement CreateRecipe handler**

Add to `server/internal/handlers/api.go` (before the `generateToken` function near line 552):

```go
func (handler *APIHandler) CreateRecipe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user := middleware.GetUser(ctx)

	var body struct {
		Title       string                `json:"title"`
		Steps       []string              `json:"steps"`
		Ingredients []models.IngredientGroup `json:"ingredients"`
		MealType    string                `json:"mealType,omitempty"`
		Servings    *int                  `json:"servings,omitempty"`
		PrepTime    *string               `json:"prepTime,omitempty"`
		CookTime    *string               `json:"cookTime,omitempty"`
		SourceURL   *string               `json:"sourceURL,omitempty"`
		ImageData   *string               `json:"imageData,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if body.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}

	recipe := models.Recipe{
		Title:           body.Title,
		Steps:           body.Steps,
		Ingredients:     body.Ingredients,
		Servings:        body.Servings,
		PrepTime:        body.PrepTime,
		CookTime:        body.CookTime,
		SourceURL:       body.SourceURL,
		CreatedByUserID: user.ID,
	}
	if body.MealType != "" {
		mt := models.RecipeMealType(body.MealType)
		recipe.MealType = &mt
	}

	created, err := handler.recipeRepo.Create(ctx, recipe)
	if err != nil {
		slog.Error("creating recipe via API", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create recipe"})
		return
	}

	if body.ImageData != nil && *body.ImageData != "" {
		if err := handler.recipeRepo.UpdateImage(ctx, created.ID, *body.ImageData); err != nil {
			slog.Error("saving recipe image via API", "error", err)
		}
		created.HasImage = true
	}

	writeJSON(w, http.StatusCreated, created)
}
```

- [ ] **Step 4: Register the route**

In `server/internal/server/server.go`, add after the `r.Get("/api/recipes/{id}", apiHandler.GetRecipe)` line (around line 190):

```go
		r.Post("/api/recipes", apiHandler.CreateRecipe)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd server && go test ./internal/handlers/ -run TestCreateRecipe_API -v`
Expected: All 3 tests PASS

- [ ] **Step 6: Commit**

```bash
cd server && git add internal/handlers/api.go internal/handlers/api_test.go internal/server/server.go
git commit -m "feat(api): add POST /api/recipes endpoint for recipe creation"
```

---

### Task 2: Backend — UpdateRecipe API endpoint

**Files:**
- Modify: `server/internal/handlers/api_test.go`
- Modify: `server/internal/handlers/api.go`
- Modify: `server/internal/server/server.go`

- [ ] **Step 1: Write the failing test for UpdateRecipe**

Add to `server/internal/handlers/api_test.go`:

```go
func TestUpdateRecipe_API(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-update-recipe",
		Email:       "update@example.com",
		Name:        "Updater",
		Role:        models.RoleMember,
	})

	created, _ := recipeRepo.Create(ctx, models.Recipe{
		Title:           "Old Title",
		CreatedByUserID: user.ID,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, "", "", "")

	router := chi.NewRouter()
	router.Put("/api/recipes/{id}", handler.UpdateRecipe)

	body := `{"title":"New Title","steps":["step 1"],"servings":2,"prepTime":"5 min"}`
	request := httptest.NewRequest(http.MethodPut, "/api/recipes/"+created.ID, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var recipe models.Recipe
	json.NewDecoder(recorder.Body).Decode(&recipe)
	if recipe.Title != "New Title" {
		t.Errorf("expected title New Title, got %s", recipe.Title)
	}
	if recipe.Servings == nil || *recipe.Servings != 2 {
		t.Errorf("expected servings 2, got %v", recipe.Servings)
	}
}

func TestUpdateRecipe_API_NotFound(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, "", "", "")

	router := chi.NewRouter()
	router.Put("/api/recipes/{id}", handler.UpdateRecipe)

	body := `{"title":"Nope"}`
	request := httptest.NewRequest(http.MethodPut, "/api/recipes/nonexistent", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}

func TestUpdateRecipe_API_ImageHandling(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-update-recipe-img",
		Email:       "updimg@example.com",
		Name:        "Img Updater",
		Role:        models.RoleMember,
	})

	created, _ := recipeRepo.Create(ctx, models.Recipe{
		Title:           "Img Recipe",
		CreatedByUserID: user.ID,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, nil, recipeRepo, nil, "", "", "")

	router := chi.NewRouter()
	router.Put("/api/recipes/{id}", handler.UpdateRecipe)

	// Add image
	body := `{"title":"Img Recipe","imageData":"data:image/png;base64,iVBORw0KGgo="}`
	request := httptest.NewRequest(http.MethodPut, "/api/recipes/"+created.ID, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}

	var recipe models.Recipe
	json.NewDecoder(recorder.Body).Decode(&recipe)
	if !recipe.HasImage {
		t.Error("expected HasImage true after setting image")
	}

	// Clear image with empty string
	body = `{"title":"Img Recipe","imageData":""}`
	request = httptest.NewRequest(http.MethodPut, "/api/recipes/"+created.ID, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	json.NewDecoder(recorder.Body).Decode(&recipe)
	if recipe.HasImage {
		t.Error("expected HasImage false after clearing image")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./internal/handlers/ -run TestUpdateRecipe_API -v`
Expected: FAIL — `handler.UpdateRecipe` undefined

- [ ] **Step 3: Implement UpdateRecipe handler**

Add to `server/internal/handlers/api.go` (after `CreateRecipe`):

```go
func (handler *APIHandler) UpdateRecipe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipeID := chi.URLParam(r, "id")

	existing, err := handler.recipeRepo.FindByID(ctx, recipeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "recipe not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load recipe"})
		}
		return
	}

	var body struct {
		Title       string                   `json:"title"`
		Steps       []string                 `json:"steps"`
		Ingredients []models.IngredientGroup `json:"ingredients"`
		MealType    string                   `json:"mealType,omitempty"`
		Servings    *int                     `json:"servings,omitempty"`
		PrepTime    *string                  `json:"prepTime,omitempty"`
		CookTime    *string                  `json:"cookTime,omitempty"`
		SourceURL   *string                  `json:"sourceURL,omitempty"`
		ImageData   *string                  `json:"imageData,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if body.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "title is required"})
		return
	}

	existing.Title = body.Title
	existing.Steps = body.Steps
	existing.Ingredients = body.Ingredients
	existing.Servings = body.Servings
	existing.PrepTime = body.PrepTime
	existing.CookTime = body.CookTime
	existing.SourceURL = body.SourceURL
	if body.MealType != "" {
		mt := models.RecipeMealType(body.MealType)
		existing.MealType = &mt
	} else {
		existing.MealType = nil
	}

	if err := handler.recipeRepo.Update(ctx, existing); err != nil {
		slog.Error("updating recipe via API", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update recipe"})
		return
	}

	if body.ImageData != nil {
		if *body.ImageData == "" {
			handler.recipeRepo.ClearImage(ctx, recipeID)
			existing.HasImage = false
		} else {
			handler.recipeRepo.UpdateImage(ctx, recipeID, *body.ImageData)
			existing.HasImage = true
		}
	}

	updated, err := handler.recipeRepo.FindByID(ctx, recipeID)
	if err != nil {
		writeJSON(w, http.StatusOK, existing)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
```

- [ ] **Step 4: Register the route**

In `server/internal/server/server.go`, add after the `r.Post("/api/recipes", apiHandler.CreateRecipe)` line:

```go
		r.Put("/api/recipes/{id}", apiHandler.UpdateRecipe)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd server && go test ./internal/handlers/ -run TestUpdateRecipe_API -v`
Expected: All 3 tests PASS

- [ ] **Step 6: Commit**

```bash
cd server && git add internal/handlers/api.go internal/handlers/api_test.go internal/server/server.go
git commit -m "feat(api): add PUT /api/recipes/{id} endpoint for recipe updates"
```

---

### Task 3: Backend — DeleteRecipe API endpoint

**Files:**
- Modify: `server/internal/handlers/api_test.go`
- Modify: `server/internal/handlers/api.go`
- Modify: `server/internal/server/server.go`

- [ ] **Step 1: Write the failing test for DeleteRecipe**

Add to `server/internal/handlers/api_test.go`:

```go
func TestDeleteRecipe_API(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	mealPlanRepo := repository.NewMealPlanRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, _ := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-delete-recipe",
		Email:       "delete@example.com",
		Name:        "Deleter",
		Role:        models.RoleMember,
	})

	created, _ := recipeRepo.Create(ctx, models.Recipe{
		Title:           "To Delete",
		CreatedByUserID: user.ID,
	})

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, mealPlanRepo, recipeRepo, nil, "", "", "")

	router := chi.NewRouter()
	router.Delete("/api/recipes/{id}", handler.DeleteRecipe)

	request := httptest.NewRequest(http.MethodDelete, "/api/recipes/"+created.ID, nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", recorder.Code, recorder.Body.String())
	}

	_, err := recipeRepo.FindByID(ctx, created.ID)
	if err == nil {
		t.Error("expected recipe to be deleted but it still exists")
	}
}

func TestDeleteRecipe_API_NotFound(t *testing.T) {
	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	mealPlanRepo := repository.NewMealPlanRepository(database)

	handler := NewAPIHandler(nil, nil, nil, nil, nil, nil, mealPlanRepo, recipeRepo, nil, "", "", "")

	router := chi.NewRouter()
	router.Delete("/api/recipes/{id}", handler.DeleteRecipe)

	request := httptest.NewRequest(http.MethodDelete, "/api/recipes/nonexistent", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", recorder.Code)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd server && go test ./internal/handlers/ -run TestDeleteRecipe_API -v`
Expected: FAIL — `handler.DeleteRecipe` undefined

- [ ] **Step 3: Implement DeleteRecipe handler**

Add to `server/internal/handlers/api.go` (after `UpdateRecipe`):

```go
func (handler *APIHandler) DeleteRecipe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	recipeID := chi.URLParam(r, "id")

	if _, err := handler.recipeRepo.FindByID(ctx, recipeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "recipe not found"})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load recipe"})
		}
		return
	}

	if err := handler.mealPlanRepo.ClearRecipeID(ctx, recipeID); err != nil {
		slog.Error("clearing recipe from meal plans via API", "error", err)
	}

	if err := handler.recipeRepo.Delete(ctx, recipeID); err != nil {
		slog.Error("deleting recipe via API", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete recipe"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
```

- [ ] **Step 4: Register the route**

In `server/internal/server/server.go`, add after the `r.Put("/api/recipes/{id}", apiHandler.UpdateRecipe)` line:

```go
		r.Delete("/api/recipes/{id}", apiHandler.DeleteRecipe)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd server && go test ./internal/handlers/ -run TestDeleteRecipe_API -v`
Expected: Both tests PASS

- [ ] **Step 6: Run all tests**

Run: `cd server && make test`
Expected: All tests PASS (including all existing tests)

- [ ] **Step 7: Commit**

```bash
cd server && git add internal/handlers/api.go internal/handlers/api_test.go internal/server/server.go
git commit -m "feat(api): add DELETE /api/recipes/{id} endpoint"
```

---

### Task 4: iOS — Update Recipe model and API client

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Models/Recipe.swift`
- Modify: `ios/FamilyHub/FamilyHub/Networking/APIClientProtocol.swift`
- Modify: `ios/FamilyHub/FamilyHub/Networking/APIClient.swift`
- Modify: `ios/FamilyHub/FamilyHubTests/FakeAPIClient.swift`

- [ ] **Step 1: Update Recipe model with missing fields and add RecipeRequest**

Replace the contents of `ios/FamilyHub/FamilyHub/Models/Recipe.swift` with:

```swift
import Foundation

struct IngredientGroup: Codable, Equatable {
    var name: String
    var items: [String]
}

struct Recipe: Codable, Identifiable {
    let id: String
    let title: String
    let steps: [String]?
    let ingredients: [IngredientGroup]?
    let mealType: String?
    let servings: Int?
    let prepTime: String?
    let cookTime: String?
    let sourceURL: String?
    let hasImage: Bool

    enum CodingKeys: String, CodingKey {
        case id = "ID"
        case title = "Title"
        case steps = "Steps"
        case ingredients = "Ingredients"
        case mealType = "MealType"
        case servings = "Servings"
        case prepTime = "PrepTime"
        case cookTime = "CookTime"
        case sourceURL = "SourceURL"
        case hasImage = "HasImage"
    }
}

struct RecipeRequest: Encodable {
    let title: String
    var steps: [String]?
    var ingredients: [IngredientGroup]?
    var mealType: String?
    var servings: Int?
    var prepTime: String?
    var cookTime: String?
    var sourceURL: String?
    var imageData: String?
}
```

- [ ] **Step 2: Add protocol methods**

Add to `ios/FamilyHub/FamilyHub/Networking/APIClientProtocol.swift`, after the `fetchRecipeImage` line:

```swift
    func createRecipe(_ request: RecipeRequest) async throws -> Recipe
    func updateRecipe(id: String, _ request: RecipeRequest) async throws -> Recipe
    func deleteRecipe(id: String) async throws
```

- [ ] **Step 3: Add put helper and implement methods in APIClient**

Add a `put` helper method in `ios/FamilyHub/FamilyHub/Networking/APIClient.swift` after the existing `post` methods (around line 34):

```swift
    private func put<T: Decodable>(_ path: String, body: some Encodable) async throws -> T {
        var request = try await buildRequest(path: path, method: "PUT")
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(body)
        let (data, response) = try await perform(request)
        return try decode(T.self, from: data, response: response)
    }
```

Then add the recipe CRUD methods after the `fetchRecipeImage` method:

```swift
    func createRecipe(_ request: RecipeRequest) async throws -> Recipe {
        try await post("api/recipes", body: request)
    }

    func updateRecipe(id: String, _ request: RecipeRequest) async throws -> Recipe {
        try await put("api/recipes/\(id)", body: request)
    }

    func deleteRecipe(id: String) async throws {
        try await delete("api/recipes/\(id)")
    }
```

- [ ] **Step 4: Update FakeAPIClient**

Add to `ios/FamilyHub/FamilyHubTests/FakeAPIClient.swift`, after the `recipeImageResult` property:

```swift
    var createRecipeResult: Result<Recipe, Error> = .success(
        Recipe(id: "new", title: "New", steps: nil, ingredients: nil, mealType: nil, servings: nil, prepTime: nil, cookTime: nil, sourceURL: nil, hasImage: false)
    )
    var updateRecipeResult: Result<Recipe, Error> = .success(
        Recipe(id: "updated", title: "Updated", steps: nil, ingredients: nil, mealType: nil, servings: nil, prepTime: nil, cookTime: nil, sourceURL: nil, hasImage: false)
    )
    var deleteRecipeResult: Result<Void, Error> = .success(())
```

Add the method implementations after the `fetchRecipeImage` method:

```swift
    func createRecipe(_ request: RecipeRequest) async throws -> Recipe { try createRecipeResult.get() }
    func updateRecipe(id: String, _ request: RecipeRequest) async throws -> Recipe { try updateRecipeResult.get() }
    func deleteRecipe(id: String) async throws { try deleteRecipeResult.get() }
```

- [ ] **Step 5: Build to verify compilation**

Run: `cd /Users/bensuskins/workspace/family-hub/ios/FamilyHub && xcodebuild -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 17 Pro' build 2>&1 | tail -5`
Expected: BUILD SUCCEEDED

- [ ] **Step 6: Commit**

```bash
cd /Users/bensuskins/workspace/family-hub && git add ios/
git commit -m "feat(ios): add recipe CRUD to model, API client, and fake"
```

---

### Task 5: iOS — RecipesViewModel CRUD methods

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Features/Meals/RecipesViewModel.swift`

- [ ] **Step 1: Add create, update, and delete methods**

Add to `ios/FamilyHub/FamilyHub/Features/Meals/RecipesViewModel.swift`, after the `load()` function:

```swift
    func createRecipe(_ request: RecipeRequest) async -> Bool {
        do {
            _ = try await apiClient.createRecipe(request)
            await load()
            return true
        } catch {
            return false
        }
    }

    func updateRecipe(id: String, _ request: RecipeRequest) async -> Bool {
        do {
            _ = try await apiClient.updateRecipe(id: id, request)
            await load()
            return true
        } catch {
            return false
        }
    }

    func deleteRecipe(id: String) async -> Bool {
        do {
            try await apiClient.deleteRecipe(id: id)
            await load()
            return true
        } catch {
            return false
        }
    }
```

- [ ] **Step 2: Build to verify**

Run: `cd /Users/bensuskins/workspace/family-hub/ios/FamilyHub && xcodebuild -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 17 Pro' build 2>&1 | tail -5`
Expected: BUILD SUCCEEDED

- [ ] **Step 3: Commit**

```bash
cd /Users/bensuskins/workspace/family-hub && git add ios/
git commit -m "feat(ios): add CRUD methods to RecipesViewModel"
```

---

### Task 6: iOS — RecipeFormView

**Files:**
- Create: `ios/FamilyHub/FamilyHub/Features/Meals/RecipeFormView.swift`

- [ ] **Step 1: Create RecipeFormView**

Create `ios/FamilyHub/FamilyHub/Features/Meals/RecipeFormView.swift`:

```swift
import SwiftUI
import PhotosUI

struct RecipeFormView: View {
    let recipe: Recipe?
    let viewModel: RecipesViewModel
    let apiClient: any APIClientProtocol

    @State private var title: String
    @State private var mealType: String
    @State private var servings: String
    @State private var prepTime: String
    @State private var cookTime: String
    @State private var sourceURL: String
    @State private var ingredientGroups: [EditableIngredientGroup]
    @State private var steps: [String]
    @State private var selectedPhoto: PhotosPickerItem?
    @State private var imageData: Data?
    @State private var existingImageData: Data?
    @State private var removeImage = false
    @State private var isSaving = false
    @Environment(\.dismiss) private var dismiss

    private var isEditing: Bool { recipe != nil }

    init(recipe: Recipe?, viewModel: RecipesViewModel, apiClient: any APIClientProtocol) {
        self.recipe = recipe
        self.viewModel = viewModel
        self.apiClient = apiClient
        _title = State(initialValue: recipe?.title ?? "")
        _mealType = State(initialValue: recipe?.mealType ?? "")
        _servings = State(initialValue: recipe?.servings.map { String($0) } ?? "")
        _prepTime = State(initialValue: recipe?.prepTime ?? "")
        _cookTime = State(initialValue: recipe?.cookTime ?? "")
        _sourceURL = State(initialValue: recipe?.sourceURL ?? "")
        _ingredientGroups = State(initialValue: recipe?.ingredients?.map {
            EditableIngredientGroup(name: $0.name, items: $0.items.isEmpty ? [""] : $0.items)
        } ?? [EditableIngredientGroup(name: "Main", items: [""])])
        _steps = State(initialValue: {
            let s = recipe?.steps ?? []
            return s.isEmpty ? [""] : s
        }())
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Recipe title", text: $title)
                }

                Section("Details") {
                    Picker("Meal Type", selection: $mealType) {
                        Text("None").tag("")
                        Text("Breakfast").tag("breakfast")
                        Text("Lunch").tag("lunch")
                        Text("Dinner").tag("dinner")
                        Text("Side").tag("side")
                        Text("Dessert").tag("dessert")
                    }
                    HStack {
                        Text("Servings")
                        Spacer()
                        TextField("", text: $servings)
                            .keyboardType(.numberPad)
                            .multilineTextAlignment(.trailing)
                            .frame(width: 60)
                    }
                    HStack {
                        Text("Prep Time")
                        Spacer()
                        TextField("e.g. 15 min", text: $prepTime)
                            .multilineTextAlignment(.trailing)
                    }
                    HStack {
                        Text("Cook Time")
                        Spacer()
                        TextField("e.g. 30 min", text: $cookTime)
                            .multilineTextAlignment(.trailing)
                    }
                    HStack {
                        Text("Source URL")
                        Spacer()
                        TextField("https://...", text: $sourceURL)
                            .multilineTextAlignment(.trailing)
                            .textContentType(.URL)
                            .autocapitalization(.none)
                    }
                }

                photoSection

                ForEach(ingredientGroups.indices, id: \.self) { groupIndex in
                    ingredientGroupSection(at: groupIndex)
                }

                Button("Add Ingredient Group") {
                    ingredientGroups.append(EditableIngredientGroup(name: "", items: [""]))
                }

                stepsSection
            }
            .navigationTitle(isEditing ? "Edit Recipe" : "New Recipe")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    if isSaving {
                        ProgressView()
                    } else {
                        Button("Save") { Task { await save() } }
                            .disabled(title.trimmingCharacters(in: .whitespaces).isEmpty)
                    }
                }
            }
            .onChange(of: selectedPhoto) { _, newValue in
                Task {
                    if let data = try? await newValue?.loadTransferable(type: Data.self) {
                        imageData = data
                        removeImage = false
                    }
                }
            }
            .task {
                if let recipe, recipe.hasImage {
                    existingImageData = try? await apiClient.fetchRecipeImage(id: recipe.id)
                }
            }
        }
        .presentationDetents([.large])
    }

    @ViewBuilder
    private var photoSection: some View {
        Section("Photo") {
            if let imageData, let uiImage = UIImage(data: imageData) {
                Image(uiImage: uiImage)
                    .resizable()
                    .aspectRatio(contentMode: .fill)
                    .frame(height: 150)
                    .clipped()
                    .listRowInsets(EdgeInsets())
                Button("Remove Photo", role: .destructive) {
                    self.imageData = nil
                    selectedPhoto = nil
                    removeImage = true
                }
            } else if !removeImage, let existingImageData, let uiImage = UIImage(data: existingImageData) {
                Image(uiImage: uiImage)
                    .resizable()
                    .aspectRatio(contentMode: .fill)
                    .frame(height: 150)
                    .clipped()
                    .listRowInsets(EdgeInsets())
                Button("Remove Photo", role: .destructive) {
                    removeImage = true
                }
            }
            PhotosPicker(selection: $selectedPhoto, matching: .images) {
                Label(hasAnyImage ? "Change Photo" : "Add Photo", systemImage: "photo")
            }
        }
    }

    private var hasAnyImage: Bool {
        imageData != nil || (!removeImage && existingImageData != nil)
    }

    private func ingredientGroupSection(at groupIndex: Int) -> some View {
        Section {
            TextField("Group name", text: $ingredientGroups[groupIndex].name)
            ForEach(ingredientGroups[groupIndex].items.indices, id: \.self) { itemIndex in
                TextField("Ingredient", text: $ingredientGroups[groupIndex].items[itemIndex])
            }
            .onDelete { offsets in
                ingredientGroups[groupIndex].items.remove(atOffsets: offsets)
                if ingredientGroups[groupIndex].items.isEmpty {
                    ingredientGroups[groupIndex].items.append("")
                }
            }
            Button("Add Ingredient") {
                ingredientGroups[groupIndex].items.append("")
            }
            if ingredientGroups.count > 1 {
                Button("Remove Group", role: .destructive) {
                    ingredientGroups.remove(at: groupIndex)
                }
            }
        } header: {
            Text(ingredientGroups[groupIndex].name.isEmpty ? "Ingredients" : ingredientGroups[groupIndex].name)
        }
    }

    private var stepsSection: some View {
        Section("Steps") {
            ForEach(steps.indices, id: \.self) { index in
                HStack(alignment: .top) {
                    Text("\(index + 1).")
                        .foregroundStyle(.secondary)
                        .frame(width: 24, alignment: .trailing)
                    TextField("Step", text: $steps[index], axis: .vertical)
                }
            }
            .onDelete { offsets in
                steps.remove(atOffsets: offsets)
                if steps.isEmpty { steps.append("") }
            }
            Button("Add Step") {
                steps.append("")
            }
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }

        var imageDataString: String?
        if removeImage {
            imageDataString = ""
        } else if let imageData {
            let base64 = imageData.base64EncodedString()
            let mimeType = detectMimeType(imageData)
            imageDataString = "data:\(mimeType);base64,\(base64)"
        }

        let filteredGroups = ingredientGroups.map {
            IngredientGroup(name: $0.name, items: $0.items.filter { !$0.trimmingCharacters(in: .whitespaces).isEmpty })
        }.filter { !$0.items.isEmpty }

        let filteredSteps = steps.filter { !$0.trimmingCharacters(in: .whitespaces).isEmpty }

        let request = RecipeRequest(
            title: title.trimmingCharacters(in: .whitespaces),
            steps: filteredSteps.isEmpty ? nil : filteredSteps,
            ingredients: filteredGroups.isEmpty ? nil : filteredGroups,
            mealType: mealType.isEmpty ? nil : mealType,
            servings: Int(servings),
            prepTime: prepTime.isEmpty ? nil : prepTime,
            cookTime: cookTime.isEmpty ? nil : cookTime,
            sourceURL: sourceURL.isEmpty ? nil : sourceURL,
            imageData: imageDataString
        )

        let success: Bool
        if let recipe {
            success = await viewModel.updateRecipe(id: recipe.id, request)
        } else {
            success = await viewModel.createRecipe(request)
        }
        if success { dismiss() }
    }

    private func detectMimeType(_ data: Data) -> String {
        if data.starts(with: [0x89, 0x50, 0x4E, 0x47]) { return "image/png" }
        if data.starts(with: [0xFF, 0xD8, 0xFF]) { return "image/jpeg" }
        if data.starts(with: [0x47, 0x49, 0x46]) { return "image/gif" }
        if data.count >= 12 {
            let headerRange = data[8..<12]
            if headerRange.elementsEqual([0x57, 0x45, 0x42, 0x50]) { return "image/webp" }
            if headerRange.elementsEqual([0x41, 0x56, 0x49, 0x46]) { return "image/avif" }
            if headerRange.elementsEqual([0x68, 0x65, 0x69, 0x63]) { return "image/heic" }
        }
        return "image/jpeg"
    }
}

private struct EditableIngredientGroup {
    var name: String
    var items: [String]
}
```

- [ ] **Step 2: Add file to Xcode project**

The file needs to be in the correct directory. Xcode with the new build system should pick up files automatically, but verify the build:

Run: `cd /Users/bensuskins/workspace/family-hub/ios/FamilyHub && xcodebuild -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 17 Pro' build 2>&1 | tail -5`
Expected: BUILD SUCCEEDED

- [ ] **Step 3: Commit**

```bash
cd /Users/bensuskins/workspace/family-hub && git add ios/
git commit -m "feat(ios): add RecipeFormView for recipe create/edit with photo picker"
```

---

### Task 7: iOS — Wire up RecipesView and RecipeDetailView

**Files:**
- Modify: `ios/FamilyHub/FamilyHub/Features/Meals/RecipesView.swift`
- Modify: `ios/FamilyHub/FamilyHub/Features/Meals/RecipeDetailView.swift`

- [ ] **Step 1: Add create button and pass viewModel to RecipesView**

Replace `RecipesView` in `ios/FamilyHub/FamilyHub/Features/Meals/RecipesView.swift`. The key changes are: add `@State private var showCreateForm = false`, pass `viewModel` to `RecipeDetailView`, add a toolbar "+" button, and add the `.sheet` modifier.

Replace the `RecipesView` struct (lines 4-42) with:

```swift
struct RecipesView: View {
    @State private var viewModel: RecipesViewModel
    @State private var showCreateForm = false
    private let apiClient: any APIClientProtocol
    private let columns = [GridItem(.flexible()), GridItem(.flexible())]

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(wrappedValue: RecipesViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            Group {
                if case .failed(let error) = viewModel.state {
                    ContentUnavailableView("Error", systemImage: "exclamationmark.triangle", description: Text(error.localizedDescription))
                } else {
                    ScrollView {
                        LazyVGrid(columns: columns, spacing: 10) {
                            ForEach(viewModel.filteredRecipes) { recipe in
                                NavigationLink {
                                    RecipeDetailView(recipe: recipe, apiClient: apiClient, viewModel: viewModel)
                                } label: {
                                    RecipeCardView(recipe: recipe, apiClient: apiClient)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                        .padding(.horizontal, 14)
                        .padding(.top, 8)
                    }
                    .refreshable { await viewModel.load() }
                }
            }
            .navigationTitle("Recipes")
            .navigationBarTitleDisplayMode(.large)
            .searchable(text: $viewModel.searchQuery, prompt: "Search recipes")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        showCreateForm = true
                    } label: {
                        Image(systemName: "plus")
                    }
                }
            }
            .sheet(isPresented: $showCreateForm) {
                RecipeFormView(recipe: nil, viewModel: viewModel, apiClient: apiClient)
            }
        }
        .task { await viewModel.load() }
    }
}
```

- [ ] **Step 2: Update RecipeDetailView with edit/delete actions**

Replace the `RecipeDetailView` struct in `ios/FamilyHub/FamilyHub/Features/Meals/RecipeDetailView.swift` with:

```swift
struct RecipeDetailView: View {
    let recipe: Recipe
    let apiClient: any APIClientProtocol
    let viewModel: RecipesViewModel

    @State private var showCookMode = false
    @State private var showEditForm = false
    @State private var showDeleteConfirmation = false
    @State private var fullRecipe: Recipe?
    @State private var isLoading = true
    @State private var fetchError = false
    @State private var imageData: Data?
    @Environment(\.dismiss) private var dismiss

    private var displayRecipe: Recipe { fullRecipe ?? recipe }

    var body: some View {
        Group {
            if isLoading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if fetchError && fullRecipe == nil {
                ContentUnavailableView("Failed to load", systemImage: "exclamationmark.triangle")
            } else {
                recipeContent(displayRecipe)
            }
        }
        .navigationTitle(recipe.title)
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Menu {
                    Button {
                        showCookMode = true
                    } label: {
                        Label("Cook Mode", systemImage: "flame")
                    }
                    Button {
                        showEditForm = true
                    } label: {
                        Label("Edit", systemImage: "pencil")
                    }
                    Button(role: .destructive) {
                        showDeleteConfirmation = true
                    } label: {
                        Label("Delete", systemImage: "trash")
                    }
                } label: {
                    Image(systemName: "ellipsis.circle")
                }
                .disabled(isLoading)
            }
        }
        .fullScreenCover(isPresented: $showCookMode) {
            CookModeView(recipe: displayRecipe)
        }
        .sheet(isPresented: $showEditForm) {
            RecipeFormView(recipe: displayRecipe, viewModel: viewModel, apiClient: apiClient)
        }
        .alert("Delete Recipe?", isPresented: $showDeleteConfirmation) {
            Button("Delete", role: .destructive) {
                Task {
                    let deleted = await viewModel.deleteRecipe(id: recipe.id)
                    if deleted { dismiss() }
                }
            }
            Button("Cancel", role: .cancel) {}
        } message: {
            Text("This cannot be undone.")
        }
        .task {
            await loadRecipe()
        }
        .onChange(of: showEditForm) { _, isShowing in
            if !isShowing {
                Task { await loadRecipe() }
            }
        }
    }

    private func loadRecipe() async {
        do {
            fullRecipe = try await apiClient.fetchRecipe(id: recipe.id)
        } catch {
            fetchError = true
        }
        isLoading = false
        if displayRecipe.hasImage {
            imageData = try? await apiClient.fetchRecipeImage(id: recipe.id)
        } else {
            imageData = nil
        }
    }

    private func recipeContent(_ r: Recipe) -> some View {
        List {
            if let imageData, let uiImage = UIImage(data: imageData) {
                Section {
                    Image(uiImage: uiImage)
                        .resizable()
                        .aspectRatio(contentMode: .fill)
                        .frame(maxHeight: 250)
                        .clipped()
                        .listRowInsets(EdgeInsets())
                }
            }

            Section {
                HStack(spacing: 16) {
                    if let prep = r.prepTime {
                        metaStat(label: "Prep", value: prep)
                    }
                    if let cook = r.cookTime {
                        metaStat(label: "Cook", value: cook)
                    }
                    if let servings = r.servings {
                        metaStat(label: "Serves", value: "\(servings)")
                    }
                }
            }

            if let ingredients = r.ingredients, !ingredients.isEmpty {
                ForEach(ingredients, id: \.name) { group in
                    Section(group.name.isEmpty ? "Ingredients" : group.name) {
                        ForEach(group.items, id: \.self) { item in
                            Text(item)
                                .font(.subheadline)
                        }
                    }
                }
            }

            if let steps = r.steps, !steps.isEmpty {
                Section("Steps") {
                    ForEach(Array(steps.enumerated()), id: \.offset) { index, step in
                        HStack(alignment: .top, spacing: 12) {
                            Text("\(index + 1)")
                                .font(.subheadline.bold())
                                .foregroundStyle(Color.accentColor)
                                .frame(width: 22, alignment: .trailing)
                            Text(step)
                                .font(.subheadline)
                        }
                    }
                }
            }

            Section {
                Button {
                    showCookMode = true
                } label: {
                    Label("Start Cooking", systemImage: "flame.fill")
                        .font(.headline)
                        .frame(maxWidth: .infinity)
                }
                .disabled(isLoading)
            }
        }
        .listStyle(.insetGrouped)
    }

    private func metaStat(label: String, value: String) -> some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.title3.bold())
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
    }
}
```

- [ ] **Step 3: Build to verify**

Run: `cd /Users/bensuskins/workspace/family-hub/ios/FamilyHub && xcodebuild -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 17 Pro' build 2>&1 | tail -5`
Expected: BUILD SUCCEEDED

- [ ] **Step 4: Commit**

```bash
cd /Users/bensuskins/workspace/family-hub && git add ios/
git commit -m "feat(ios): wire up recipe create/edit/delete in RecipesView and RecipeDetailView"
```

---

### Task 8: Regenerate templ and verify full build

**Files:**
- Modify: `server/templates/pages/recipes_templ.go` (auto-generated)

- [ ] **Step 1: Regenerate templ files**

Run: `cd /Users/bensuskins/workspace/family-hub/server && make templ`

- [ ] **Step 2: Run all Go tests**

Run: `cd /Users/bensuskins/workspace/family-hub/server && make test`
Expected: All tests PASS

- [ ] **Step 3: Build iOS**

Run: `cd /Users/bensuskins/workspace/family-hub/ios/FamilyHub && xcodebuild -scheme FamilyHub -destination 'platform=iOS Simulator,name=iPhone 17 Pro' build 2>&1 | tail -5`
Expected: BUILD SUCCEEDED

- [ ] **Step 4: Commit any generated file changes**

```bash
cd /Users/bensuskins/workspace/family-hub && git add -A
git status
# Only commit if there are changes
git commit -m "chore: regenerate templ files"
```

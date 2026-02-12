package repository_test

import (
	"context"
	"testing"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestRecipeRepository_CreateAndFindByID(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	recipeRepo := repository.NewRecipeRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	servings := 4
	prepTime := "15 min"
	cookTime := "30 min"
	sourceURL := "https://example.com/recipe"

	recipe := models.Recipe{
		Title:        "Pasta Carbonara",
		Instructions: "Cook the pasta.\nMake the sauce.",
		Ingredients: []models.IngredientGroup{
			{Name: "Pasta", Items: []string{"400g spaghetti", "Salt"}},
			{Name: "Sauce", Items: []string{"4 eggs", "200g pancetta", "100g parmesan"}},
		},
		Servings:        &servings,
		PrepTime:        &prepTime,
		CookTime:        &cookTime,
		SourceURL:       &sourceURL,
		CreatedByUserID: user.ID,
	}

	created, err := recipeRepo.Create(ctx, recipe)
	if err != nil {
		t.Fatalf("creating recipe: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected non-empty ID")
	}

	found, err := recipeRepo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("finding recipe: %v", err)
	}
	if found.Title != "Pasta Carbonara" {
		t.Errorf("expected title 'Pasta Carbonara', got '%s'", found.Title)
	}
	if found.Instructions != "Cook the pasta.\nMake the sauce." {
		t.Errorf("unexpected instructions: %s", found.Instructions)
	}
	if len(found.Ingredients) != 2 {
		t.Fatalf("expected 2 ingredient groups, got %d", len(found.Ingredients))
	}
	if found.Ingredients[0].Name != "Pasta" {
		t.Errorf("expected group name 'Pasta', got '%s'", found.Ingredients[0].Name)
	}
	if len(found.Ingredients[0].Items) != 2 {
		t.Errorf("expected 2 items in Pasta group, got %d", len(found.Ingredients[0].Items))
	}
	if found.Ingredients[1].Name != "Sauce" {
		t.Errorf("expected group name 'Sauce', got '%s'", found.Ingredients[1].Name)
	}
	if len(found.Ingredients[1].Items) != 3 {
		t.Errorf("expected 3 items in Sauce group, got %d", len(found.Ingredients[1].Items))
	}
	if found.Servings == nil || *found.Servings != 4 {
		t.Errorf("expected servings 4, got %v", found.Servings)
	}
	if found.PrepTime == nil || *found.PrepTime != "15 min" {
		t.Errorf("expected prep time '15 min', got %v", found.PrepTime)
	}
	if found.CookTime == nil || *found.CookTime != "30 min" {
		t.Errorf("expected cook time '30 min', got %v", found.CookTime)
	}
	if found.SourceURL == nil || *found.SourceURL != "https://example.com/recipe" {
		t.Errorf("expected source URL, got %v", found.SourceURL)
	}
}

func TestRecipeRepository_CreateWithNilOptionalFields(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	recipeRepo := repository.NewRecipeRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	recipe := models.Recipe{
		Title:           "Simple Recipe",
		Instructions:    "Just cook it.",
		Ingredients:     []models.IngredientGroup{{Name: "Main", Items: []string{"Food"}}},
		CreatedByUserID: user.ID,
	}

	created, err := recipeRepo.Create(ctx, recipe)
	if err != nil {
		t.Fatalf("creating recipe: %v", err)
	}

	found, err := recipeRepo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("finding recipe: %v", err)
	}
	if found.Servings != nil {
		t.Errorf("expected nil servings, got %v", found.Servings)
	}
	if found.PrepTime != nil {
		t.Errorf("expected nil prep time, got %v", found.PrepTime)
	}
	if found.CookTime != nil {
		t.Errorf("expected nil cook time, got %v", found.CookTime)
	}
	if found.SourceURL != nil {
		t.Errorf("expected nil source url, got %v", found.SourceURL)
	}
	if found.CategoryID != nil {
		t.Errorf("expected nil category id, got %v", found.CategoryID)
	}
}

func TestRecipeRepository_FindAll(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	recipeRepo := repository.NewRecipeRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	recipeRepo.Create(ctx, models.Recipe{
		Title: "Zebra Cake", Instructions: "Bake it.", CreatedByUserID: user.ID,
		Ingredients: []models.IngredientGroup{},
	})
	recipeRepo.Create(ctx, models.Recipe{
		Title: "Apple Pie", Instructions: "Bake it.", CreatedByUserID: user.ID,
		Ingredients: []models.IngredientGroup{},
	})

	recipes, err := recipeRepo.FindAll(ctx)
	if err != nil {
		t.Fatalf("finding recipes: %v", err)
	}
	if len(recipes) != 2 {
		t.Fatalf("expected 2 recipes, got %d", len(recipes))
	}
	if recipes[0].Title != "Apple Pie" {
		t.Errorf("expected 'Apple Pie' first (alphabetical), got '%s'", recipes[0].Title)
	}
	if recipes[1].Title != "Zebra Cake" {
		t.Errorf("expected 'Zebra Cake' second, got '%s'", recipes[1].Title)
	}
}

func TestRecipeRepository_Update(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	recipeRepo := repository.NewRecipeRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	created, _ := recipeRepo.Create(ctx, models.Recipe{
		Title: "Original", Instructions: "Old instructions", CreatedByUserID: user.ID,
		Ingredients: []models.IngredientGroup{{Name: "Main", Items: []string{"item1"}}},
	})

	created.Title = "Updated"
	created.Instructions = "New instructions"
	created.Ingredients = []models.IngredientGroup{{Name: "New Group", Items: []string{"new item"}}}
	servings := 6
	created.Servings = &servings

	if err := recipeRepo.Update(ctx, created); err != nil {
		t.Fatalf("updating recipe: %v", err)
	}

	found, _ := recipeRepo.FindByID(ctx, created.ID)
	if found.Title != "Updated" {
		t.Errorf("expected 'Updated', got '%s'", found.Title)
	}
	if found.Instructions != "New instructions" {
		t.Errorf("expected 'New instructions', got '%s'", found.Instructions)
	}
	if len(found.Ingredients) != 1 || found.Ingredients[0].Name != "New Group" {
		t.Errorf("unexpected ingredients: %v", found.Ingredients)
	}
	if found.Servings == nil || *found.Servings != 6 {
		t.Errorf("expected servings 6, got %v", found.Servings)
	}
}

func TestRecipeRepository_Delete(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	recipeRepo := repository.NewRecipeRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	created, _ := recipeRepo.Create(ctx, models.Recipe{
		Title: "To Delete", Instructions: "Delete me", CreatedByUserID: user.ID,
		Ingredients: []models.IngredientGroup{},
	})

	if err := recipeRepo.Delete(ctx, created.ID); err != nil {
		t.Fatalf("deleting recipe: %v", err)
	}

	_, err := recipeRepo.FindByID(ctx, created.ID)
	if err == nil {
		t.Fatal("expected error finding deleted recipe")
	}
}

func TestRecipeRepository_CategoryFK_OnDeleteSetNull(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	recipeRepo := repository.NewRecipeRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	category, err := categoryRepo.Create(ctx, models.Category{
		Name: "Dinner", CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating category: %v", err)
	}

	created, err := recipeRepo.Create(ctx, models.Recipe{
		Title: "Categorized Recipe", Instructions: "Cook it", CreatedByUserID: user.ID,
		Ingredients: []models.IngredientGroup{},
		CategoryID:  &category.ID,
	})
	if err != nil {
		t.Fatalf("creating recipe: %v", err)
	}

	found, _ := recipeRepo.FindByID(ctx, created.ID)
	if found.CategoryID == nil || *found.CategoryID != category.ID {
		t.Fatalf("expected category ID %s, got %v", category.ID, found.CategoryID)
	}

	// Enable foreign keys and delete category
	db.ExecContext(ctx, "PRAGMA foreign_keys = ON")
	if err := categoryRepo.Delete(ctx, category.ID); err != nil {
		t.Fatalf("deleting category: %v", err)
	}

	found, err = recipeRepo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("finding recipe after category delete: %v", err)
	}
	if found.CategoryID != nil {
		t.Errorf("expected nil category ID after delete, got %v", found.CategoryID)
	}
}

func TestRecipeRepository_JSONRoundtrip_EmptyIngredients(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	recipeRepo := repository.NewRecipeRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	created, err := recipeRepo.Create(ctx, models.Recipe{
		Title: "No Ingredients", Instructions: "Magic", CreatedByUserID: user.ID,
	})
	if err != nil {
		t.Fatalf("creating recipe: %v", err)
	}

	found, err := recipeRepo.FindByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("finding recipe: %v", err)
	}
	if found.Ingredients == nil {
		t.Fatal("expected non-nil empty ingredients slice")
	}
	if len(found.Ingredients) != 0 {
		t.Errorf("expected 0 ingredient groups, got %d", len(found.Ingredients))
	}
}

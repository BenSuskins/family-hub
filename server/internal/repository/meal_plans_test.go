package repository_test

import (
	"context"
	"testing"

	mealapp "github.com/bensuskins/family-hub/internal/application/meals"
	recipeapp "github.com/bensuskins/family-hub/internal/application/recipes"

	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestMealPlanRepository_UpsertAndFindByDateAndType(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	mealRepo := repository.NewMealPlanRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	meal := mealapp.Meal{
		Date:            "2025-06-15",
		MealType:        mealapp.Dinner,
		Name:            "Pasta Carbonara",
		Notes:           "Use the good parmesan",
		CreatedByUserID: user.ID,
	}

	if err := mealRepo.Upsert(ctx, meal); err != nil {
		t.Fatalf("upserting meal: %v", err)
	}

	found, err := mealRepo.FindByDateAndType(ctx, "2025-06-15", mealapp.Dinner)
	if err != nil {
		t.Fatalf("finding meal: %v", err)
	}
	if found.Name != "Pasta Carbonara" {
		t.Errorf("expected name 'Pasta Carbonara', got '%s'", found.Name)
	}
	if found.Notes != "Use the good parmesan" {
		t.Errorf("expected notes, got '%s'", found.Notes)
	}
}

func TestMealPlanRepository_UpsertOverwrites(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	mealRepo := repository.NewMealPlanRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	mealRepo.Upsert(ctx, mealapp.Meal{
		Date: "2025-06-15", MealType: mealapp.Lunch,
		Name: "Original", CreatedByUserID: user.ID,
	})

	mealRepo.Upsert(ctx, mealapp.Meal{
		Date: "2025-06-15", MealType: mealapp.Lunch,
		Name: "Updated", Notes: "New notes", CreatedByUserID: user.ID,
	})

	found, err := mealRepo.FindByDateAndType(ctx, "2025-06-15", mealapp.Lunch)
	if err != nil {
		t.Fatalf("finding meal: %v", err)
	}
	if found.Name != "Updated" {
		t.Errorf("expected 'Updated', got '%s'", found.Name)
	}
	if found.Notes != "New notes" {
		t.Errorf("expected 'New notes', got '%s'", found.Notes)
	}
}

func TestMealPlanRepository_FindByDate_Ordered(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	mealRepo := repository.NewMealPlanRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	// Insert in wrong order
	mealRepo.Upsert(ctx, mealapp.Meal{
		Date: "2025-06-15", MealType: mealapp.Dinner,
		Name: "Steak", CreatedByUserID: user.ID,
	})
	mealRepo.Upsert(ctx, mealapp.Meal{
		Date: "2025-06-15", MealType: mealapp.Breakfast,
		Name: "Eggs", CreatedByUserID: user.ID,
	})
	mealRepo.Upsert(ctx, mealapp.Meal{
		Date: "2025-06-15", MealType: mealapp.Lunch,
		Name: "Sandwich", CreatedByUserID: user.ID,
	})

	meals, err := mealRepo.FindByDate(ctx, "2025-06-15")
	if err != nil {
		t.Fatalf("finding meals: %v", err)
	}
	if len(meals) != 3 {
		t.Fatalf("expected 3 meals, got %d", len(meals))
	}
	if meals[0].MealType != mealapp.Breakfast {
		t.Errorf("expected breakfast first, got %s", meals[0].MealType)
	}
	if meals[1].MealType != mealapp.Lunch {
		t.Errorf("expected lunch second, got %s", meals[1].MealType)
	}
	if meals[2].MealType != mealapp.Dinner {
		t.Errorf("expected dinner third, got %s", meals[2].MealType)
	}
}

func TestMealPlanRepository_FindAll_DateRange(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	mealRepo := repository.NewMealPlanRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	mealRepo.Upsert(ctx, mealapp.Meal{
		Date: "2025-06-14", MealType: mealapp.Dinner,
		Name: "Before range", CreatedByUserID: user.ID,
	})
	mealRepo.Upsert(ctx, mealapp.Meal{
		Date: "2025-06-15", MealType: mealapp.Dinner,
		Name: "In range", CreatedByUserID: user.ID,
	})
	mealRepo.Upsert(ctx, mealapp.Meal{
		Date: "2025-06-16", MealType: mealapp.Lunch,
		Name: "Also in range", CreatedByUserID: user.ID,
	})
	mealRepo.Upsert(ctx, mealapp.Meal{
		Date: "2025-06-17", MealType: mealapp.Dinner,
		Name: "After range", CreatedByUserID: user.ID,
	})

	meals, err := mealRepo.FindAll(ctx, mealapp.Filter{
		DateFrom: "2025-06-15",
		DateTo:   "2025-06-16",
	})
	if err != nil {
		t.Fatalf("finding meals: %v", err)
	}
	if len(meals) != 2 {
		t.Fatalf("expected 2 meals in range, got %d", len(meals))
	}
	if meals[0].Name != "In range" {
		t.Errorf("expected 'In range', got '%s'", meals[0].Name)
	}
	if meals[1].Name != "Also in range" {
		t.Errorf("expected 'Also in range', got '%s'", meals[1].Name)
	}
}

func TestMealPlanRepository_Delete(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	mealRepo := repository.NewMealPlanRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	mealRepo.Upsert(ctx, mealapp.Meal{
		Date: "2025-06-15", MealType: mealapp.Dinner,
		Name: "To Delete", CreatedByUserID: user.ID,
	})

	if err := mealRepo.Delete(ctx, "2025-06-15", mealapp.Dinner); err != nil {
		t.Fatalf("deleting meal: %v", err)
	}

	_, err := mealRepo.FindByDateAndType(ctx, "2025-06-15", mealapp.Dinner)
	if err == nil {
		t.Fatal("expected error finding deleted meal")
	}
}

func TestMealPlanRepository_ClearRecipeID_PreservesName(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	recipeRepo := repository.NewRecipeRepository(db)
	mealRepo := repository.NewMealPlanRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	recipe, _ := recipeRepo.Create(ctx, recipeapp.SavedRecipe{
		Title: "Test Recipe", Instructions: "Cook it", CreatedByUserID: user.ID,
		Ingredients: []recipeapp.IngredientGroup{},
	})

	mealRepo.Upsert(ctx, mealapp.Meal{
		Date: "2025-06-15", MealType: mealapp.Dinner,
		RecipeID: &recipe.ID, Name: "Test Recipe",
		CreatedByUserID: user.ID,
	})

	// Also create a meal without this recipe
	mealRepo.Upsert(ctx, mealapp.Meal{
		Date: "2025-06-15", MealType: mealapp.Lunch,
		Name: "Freeform Meal", CreatedByUserID: user.ID,
	})

	if err := mealRepo.ClearRecipeID(ctx, recipe.ID); err != nil {
		t.Fatalf("clearing recipe id: %v", err)
	}

	// The dinner meal should still have the name but no recipe_id
	dinner, err := mealRepo.FindByDateAndType(ctx, "2025-06-15", mealapp.Dinner)
	if err != nil {
		t.Fatalf("finding dinner: %v", err)
	}
	if dinner.RecipeID != nil {
		t.Errorf("expected nil recipe id, got %v", dinner.RecipeID)
	}
	if dinner.Name != "Test Recipe" {
		t.Errorf("expected name preserved as 'Test Recipe', got '%s'", dinner.Name)
	}

	// The lunch meal should be unaffected
	lunch, err := mealRepo.FindByDateAndType(ctx, "2025-06-15", mealapp.Lunch)
	if err != nil {
		t.Fatalf("finding lunch: %v", err)
	}
	if lunch.Name != "Freeform Meal" {
		t.Errorf("expected 'Freeform Meal', got '%s'", lunch.Name)
	}
}

func TestMealPlanRepository_WithRecipeID(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	userRepo := repository.NewUserRepository(db)
	recipeRepo := repository.NewRecipeRepository(db)
	mealRepo := repository.NewMealPlanRepository(db)
	ctx := context.Background()

	user := createTestUser(t, userRepo)

	recipe, _ := recipeRepo.Create(ctx, recipeapp.SavedRecipe{
		Title: "Linked Recipe", Instructions: "Cook it", CreatedByUserID: user.ID,
		Ingredients: []recipeapp.IngredientGroup{},
	})

	mealRepo.Upsert(ctx, mealapp.Meal{
		Date: "2025-06-15", MealType: mealapp.Breakfast,
		RecipeID: &recipe.ID, Name: "Linked Recipe",
		CreatedByUserID: user.ID,
	})

	found, err := mealRepo.FindByDateAndType(ctx, "2025-06-15", mealapp.Breakfast)
	if err != nil {
		t.Fatalf("finding meal: %v", err)
	}
	if found.RecipeID == nil || *found.RecipeID != recipe.ID {
		t.Errorf("expected recipe ID %s, got %v", recipe.ID, found.RecipeID)
	}
}

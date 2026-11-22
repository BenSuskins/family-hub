package contracttests

import (
	"context"
	"errors"
	app "github.com/bensuskins/family-hub/internal/application/meals"
	recipeapp "github.com/bensuskins/family-hub/internal/application/recipes"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"sort"
	"testing"
	"time"
)

type mealStoreFake struct{ meals map[string]app.Meal }

func mealKey(date string, typ app.Type) string { return date + "/" + string(typ) }
func cloneMeal(meal app.Meal) app.Meal {
	if meal.RecipeID != nil {
		value := *meal.RecipeID
		meal.RecipeID = &value
	}
	return meal
}
func (fake *mealStoreFake) FindByDateAndType(ctx context.Context, date string, typ app.Type) (app.Meal, error) {
	if err := ctx.Err(); err != nil {
		return app.Meal{}, err
	}
	meal, ok := fake.meals[mealKey(date, typ)]
	if !ok {
		return app.Meal{}, app.ErrNotFound
	}
	return cloneMeal(meal), nil
}
func (fake *mealStoreFake) FindAll(ctx context.Context, filter app.Filter) ([]app.Meal, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var result []app.Meal
	for _, meal := range fake.meals {
		if (filter.DateFrom == "" || meal.Date >= filter.DateFrom) && (filter.DateTo == "" || meal.Date <= filter.DateTo) {
			result = append(result, cloneMeal(meal))
		}
	}
	rank := func(value app.Type) int {
		switch value {
		case app.Breakfast:
			return 1
		case app.Lunch:
			return 2
		case app.Dinner:
			return 3
		}
		return 0
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Date != result[j].Date {
			return result[i].Date < result[j].Date
		}
		return rank(result[i].MealType) < rank(result[j].MealType)
	})
	return result, nil
}
func (fake *mealStoreFake) FindByDate(ctx context.Context, date string) ([]app.Meal, error) {
	return fake.FindAll(ctx, app.Filter{DateFrom: date, DateTo: date})
}
func (fake *mealStoreFake) Upsert(ctx context.Context, meal app.Meal) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if meal.MealType != app.Breakfast && meal.MealType != app.Lunch && meal.MealType != app.Dinner {
		return errors.New("invalid meal type")
	}
	key := mealKey(meal.Date, meal.MealType)
	now := time.Now()
	if existing, ok := fake.meals[key]; ok {
		meal.CreatedByUserID = existing.CreatedByUserID
		meal.CreatedAt = existing.CreatedAt
	} else {
		meal.CreatedAt = now
	}
	meal.UpdatedAt = now
	fake.meals[key] = cloneMeal(meal)
	return nil
}
func (fake *mealStoreFake) Delete(ctx context.Context, date string, typ app.Type) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	delete(fake.meals, mealKey(date, typ))
	return nil
}
func (fake *mealStoreFake) ClearRecipeID(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	for key, meal := range fake.meals {
		if meal.RecipeID != nil && *meal.RecipeID == id {
			meal.RecipeID = nil
			meal.UpdatedAt = time.Now()
			fake.meals[key] = meal
		}
	}
	return nil
}

func TestMealStoreContract(t *testing.T) {
	for _, name := range []string{"fake", "sqlite"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			var store app.Store
			creator := "user"
			recipeID := "recipe"
			if name == "sqlite" {
				db := testutil.NewTestDatabase(t)
				user, err := repository.NewUserRepository(db).Create(ctx, models.User{OIDCSubject: "meal-contract", Email: "meal@example.com", Name: "Member", Role: models.RoleMember})
				if err != nil {
					t.Fatal(err)
				}
				creator = user.ID
				recipe, err := repository.NewRecipeRepository(db).Create(ctx, recipeapp.SavedRecipe{Title: "Soup", CreatedByUserID: creator})
				if err != nil {
					t.Fatal(err)
				}
				recipeID = recipe.ID
				store = repository.NewMealPlanRepository(db)
			} else {
				store = &mealStoreFake{meals: map[string]app.Meal{}}
			}
			if _, err := store.FindByDateAndType(ctx, "2026-01-01", app.Dinner); !errors.Is(err, app.ErrNotFound) {
				t.Fatalf("missing: %v", err)
			}
			if err := store.Upsert(ctx, app.Meal{Date: "2026-01-01", MealType: "invalid", CreatedByUserID: creator}); err == nil {
				t.Fatal("invalid meal type was accepted")
			}
			initial := app.Meal{Date: "2026-01-01", MealType: app.Dinner, Name: "Soup", RecipeID: &recipeID, CreatedByUserID: creator}
			if err := store.Upsert(ctx, initial); err != nil {
				t.Fatal(err)
			}
			first, err := store.FindByDateAndType(ctx, initial.Date, initial.MealType)
			if err != nil {
				t.Fatal(err)
			}
			if first.CreatedAt.IsZero() || first.UpdatedAt.IsZero() || first.CreatedByUserID != creator {
				t.Fatalf("creation metadata: %+v", first)
			}
			updated := initial
			updated.Name = "Updated soup"
			updated.CreatedByUserID = ""
			if err := store.Upsert(ctx, updated); err != nil {
				t.Fatal(err)
			}
			second, err := store.FindByDateAndType(ctx, initial.Date, initial.MealType)
			if err != nil || second.Name != updated.Name || second.CreatedByUserID != creator || !second.CreatedAt.Equal(first.CreatedAt) || second.UpdatedAt.Before(first.UpdatedAt) {
				t.Fatalf("update metadata: %+v %v", second, err)
			}
			*first.RecipeID = "changed"
			saved, err := store.FindByDateAndType(ctx, initial.Date, initial.MealType)
			if err != nil || *saved.RecipeID != recipeID {
				t.Fatalf("storage alias: %+v %v", saved, err)
			}
			if err := store.Upsert(ctx, app.Meal{Date: initial.Date, MealType: app.Breakfast, Name: "Toast", CreatedByUserID: creator}); err != nil {
				t.Fatal(err)
			}
			list, err := store.FindByDate(ctx, initial.Date)
			if err != nil || len(list) != 2 || list[0].MealType != app.Breakfast || list[1].MealType != app.Dinner {
				t.Fatalf("order: %+v %v", list, err)
			}
			if err := store.ClearRecipeID(ctx, recipeID); err != nil {
				t.Fatal(err)
			}
			saved, err = store.FindByDateAndType(ctx, initial.Date, initial.MealType)
			if err != nil || saved.RecipeID != nil || saved.Name != "Updated soup" {
				t.Fatalf("unlink: %+v %v", saved, err)
			}
			if err := store.Delete(ctx, initial.Date, app.Dinner); err != nil {
				t.Fatal(err)
			}
			if _, err := store.FindByDateAndType(ctx, initial.Date, app.Dinner); !errors.Is(err, app.ErrNotFound) {
				t.Fatalf("deleted: %v", err)
			}
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			if _, err := store.FindAll(canceled, app.Filter{}); !errors.Is(err, context.Canceled) {
				t.Fatalf("canceled: %v", err)
			}
		})
	}
}

package contracttests

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/recipes"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

type recipeStoreFake struct {
	items map[string]app.SavedRecipe
	next  int
}

func fakeRecipePointer[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
func fakeRecipeCopy(item app.SavedRecipe) app.SavedRecipe {
	item.MealType = fakeRecipePointer(item.MealType)
	item.Servings = fakeRecipePointer(item.Servings)
	item.PrepTime = fakeRecipePointer(item.PrepTime)
	item.CookTime = fakeRecipePointer(item.CookTime)
	item.SourceURL = fakeRecipePointer(item.SourceURL)
	item.CategoryID = fakeRecipePointer(item.CategoryID)
	if item.StepDurations != nil {
		durations := make([]*int, len(item.StepDurations))
		for i, duration := range item.StepDurations {
			durations[i] = fakeRecipePointer(duration)
		}
		item.StepDurations = durations
	}
	if item.Steps != nil {
		item.Steps = append([]string{}, item.Steps...)
	}
	if item.Ingredients != nil {
		groups := make([]app.IngredientGroup, len(item.Ingredients))
		for i, group := range item.Ingredients {
			groups[i] = group
			if group.Items != nil {
				groups[i].Items = append([]string{}, group.Items...)
			}
		}
		item.Ingredients = groups
	}
	return item
}
func fakeRecipeNormalized(item app.SavedRecipe) app.SavedRecipe {
	item = fakeRecipeCopy(item)
	if item.Steps == nil {
		item.Steps = []string{}
	}
	if item.Ingredients == nil {
		item.Ingredients = []app.IngredientGroup{}
	}
	for i := range item.Ingredients {
		if item.Ingredients[i].Items == nil {
			item.Ingredients[i].Items = []string{}
		}
	}
	return item
}
func (fake *recipeStoreFake) FindByID(_ context.Context, id string) (app.SavedRecipe, error) {
	item, ok := fake.items[id]
	if !ok {
		return app.SavedRecipe{}, app.ErrNotFound
	}
	return fakeRecipeCopy(item), nil
}
func (fake *recipeStoreFake) FindAll(context.Context) ([]app.SavedRecipe, error) {
	var items []app.SavedRecipe
	for _, item := range fake.items {
		item = fakeRecipeCopy(item)
		item.Steps = nil
		item.Instructions = ""
		item.StepDurations = nil
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Title < items[j].Title })
	return items, nil
}
func (fake *recipeStoreFake) Create(_ context.Context, item app.SavedRecipe) (app.SavedRecipe, error) {
	fake.next++
	item = fakeRecipeCopy(item)
	if item.ID == "" {
		item.ID = fmt.Sprint(fake.next)
	}
	if _, exists := fake.items[item.ID]; exists {
		return app.SavedRecipe{}, errors.New("duplicate recipe ID")
	}
	stored := fakeRecipeNormalized(item)
	stored.HasImage = false
	stored.StepDurations = nil
	fake.items[item.ID] = stored
	if item.Ingredients != nil {
		item.Ingredients = fakeRecipeCopy(stored).Ingredients
	}
	return item, nil
}
func (fake *recipeStoreFake) Update(_ context.Context, item app.SavedRecipe) error {
	old, exists := fake.items[item.ID]
	if !exists {
		return nil
	}
	item = fakeRecipeNormalized(item)
	old.Title = item.Title
	old.Ingredients = item.Ingredients
	old.Steps = item.Steps
	old.Servings = item.Servings
	old.PrepTime = item.PrepTime
	old.CookTime = item.CookTime
	old.SourceURL = item.SourceURL
	old.CategoryID = item.CategoryID
	old.MealType = item.MealType
	fake.items[item.ID] = old
	return nil
}
func (fake *recipeStoreFake) Delete(_ context.Context, id string) error {
	delete(fake.items, id)
	return nil
}
func TestRecipeCatalogueStoreContract(t *testing.T) {
	factories := map[string]func(*testing.T) (app.CatalogueStore, string){
		"fake": func(*testing.T) (app.CatalogueStore, string) {
			return &recipeStoreFake{items: map[string]app.SavedRecipe{}}, "owner"
		},
		"sqlite": func(t *testing.T) (app.CatalogueStore, string) {
			db := testutil.NewTestDatabase(t)
			user, err := repository.NewUserRepository(db).Create(context.Background(), models.User{OIDCSubject: "recipe-contract", Email: "recipe-contract@example.com", Name: "Cook", Role: models.RoleAdmin})
			if err != nil {
				t.Fatal(err)
			}
			return repository.NewRecipeRepository(db), user.ID
		},
	}
	for name, factory := range factories {
		t.Run(name, func(t *testing.T) {
			store, user := factory(t)
			ctx := context.Background()
			prep := "10 min"
			duration := 60
			item, err := store.Create(ctx, app.SavedRecipe{ID: "fixed", Title: "Soup", PrepTime: &prep, HasImage: true, StepDurations: []*int{&duration}, CreatedByUserID: user, Ingredients: []app.IngredientGroup{{Name: "Main"}}})
			if err != nil {
				t.Fatal(err)
			}
			*item.PrepTime = "changed"
			if prep != "10 min" {
				t.Fatal("create result shares caller pointer")
			}
			if item.Ingredients[0].Items == nil {
				t.Fatal("create response has nil ingredient items")
			}
			found, err := store.FindByID(ctx, item.ID)
			if err != nil || found.Title != "Soup" || found.HasImage || found.StepDurations != nil || found.PrepTime == nil || *found.PrepTime != "10 min" || found.Ingredients[0].Items == nil || found.Steps == nil {
				t.Fatalf("found=%+v err=%v", found, err)
			}
			found.Title = "Stew"
			found.Steps = []string{"Cook"}
			if err := store.Update(ctx, found); err != nil {
				t.Fatal(err)
			}
			all, err := store.FindAll(ctx)
			if err != nil || len(all) != 1 || all[0].Title != "Stew" || all[0].Steps != nil {
				t.Fatalf("list=%+v err=%v", all, err)
			}
			if _, err := store.Create(ctx, app.SavedRecipe{ID: item.ID, Title: "Duplicate", CreatedByUserID: user}); err == nil {
				t.Fatal("duplicate ID accepted")
			}
			second, err := store.Create(ctx, app.SavedRecipe{Title: "Bread", Instructions: "Legacy", CreatedByUserID: user, Ingredients: []app.IngredientGroup{{Name: "Main", Items: []string{"Flour"}}}})
			if err != nil || second.ID == item.ID {
				t.Fatalf("second=%+v err=%v", second, err)
			}
			loaded, err := store.FindByID(ctx, second.ID)
			if err != nil {
				t.Fatal(err)
			}
			loaded.Ingredients[0].Items[0] = "changed"
			again, err := store.FindByID(ctx, second.ID)
			if err != nil || again.Ingredients[0].Items[0] != "Flour" {
				t.Fatalf("stored=%+v err=%v", again, err)
			}
			again.Instructions = "replace"
			again.Ingredients = []app.IngredientGroup{{Name: "Main"}}
			again.Steps = nil
			if err := store.Update(ctx, again); err != nil {
				t.Fatal(err)
			}
			again, err = store.FindByID(ctx, second.ID)
			if err != nil || again.Instructions != "Legacy" || again.Steps == nil || again.Ingredients[0].Items == nil {
				t.Fatalf("updated=%+v err=%v", again, err)
			}
			all, err = store.FindAll(ctx)
			if err != nil || len(all) != 2 || all[0].Title != "Bread" || all[0].Instructions != "" {
				t.Fatalf("list=%+v err=%v", all, err)
			}
			if err := store.Delete(ctx, item.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := store.FindByID(ctx, item.ID); !errors.Is(err, app.ErrNotFound) {
				t.Fatalf("missing=%v", err)
			}
		})
	}
}

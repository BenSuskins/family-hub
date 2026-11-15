package repository_test

import (
	"context"
	"errors"
	"testing"

	app "github.com/bensuskins/family-hub/internal/application/recipes"
	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
)

func TestRecipeStoreMapsMissingAndPreservesCallerValues(t *testing.T) {
	db := testutil.NewTestDatabase(t)
	ctx := context.Background()
	user, err := repository.NewUserRepository(db).Create(ctx, models.User{OIDCSubject: "recipe-store", Email: "recipe@example.com", Name: "Cook", Role: models.RoleAdmin})
	if err != nil {
		t.Fatal(err)
	}
	store := repository.NewRecipeRepository(db)
	if _, err := store.FindByID(ctx, "missing"); !errors.Is(err, app.ErrNotFound) {
		t.Fatalf("missing error=%v", err)
	}
	preparation := "15 min"
	input := app.SavedRecipe{Title: "Soup", PrepTime: &preparation, Ingredients: []app.IngredientGroup{{Name: "Main"}}, CreatedByUserID: user.ID}
	created, err := store.Create(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	*created.PrepTime = "changed"
	if preparation != "15 min" {
		t.Fatal("adapter retained caller pointer")
	}
	if input.Ingredients[0].Items != nil {
		t.Fatal("adapter changed caller input")
	}
	found, err := store.FindByID(ctx, created.ID)
	if err != nil || found.Ingredients[0].Items == nil {
		t.Fatalf("found=%+v err=%v", found, err)
	}
}

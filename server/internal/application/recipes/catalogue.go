package recipes

import (
	"context"
	"errors"
	"fmt"
	"time"

	domain "github.com/bensuskins/family-hub/internal/domain/recipes"
)

var ErrNotFound = errors.New("recipe not found")
var ErrLookup = errors.New("recipe lookup failed")

type IngredientGroup struct {
	Name  string
	Items []string
}
type SavedRecipe struct {
	ID, Title, Instructions                   string
	Steps                                     []string
	Ingredients                               []IngredientGroup
	MealType                                  *string
	Servings                                  *int
	PrepTime, CookTime, SourceURL, CategoryID *string
	HasImage                                  bool
	StepDurations                             []*int
	CreatedByUserID                           string
	CreatedAt, UpdatedAt                      time.Time
}
type CatalogueStore interface {
	FindByID(context.Context, string) (SavedRecipe, error)
	FindAll(context.Context) ([]SavedRecipe, error)
	Create(context.Context, SavedRecipe) (SavedRecipe, error)
	Update(context.Context, SavedRecipe) error
	Delete(context.Context, string) error
}
type ImageWriter interface {
	UpdateImage(context.Context, string, string) error
	ClearImage(context.Context, string) error
}
type MealLinks interface {
	ClearRecipeID(context.Context, string) error
}
type PreparedUpdate struct{ existing SavedRecipe }
type SaveResult struct {
	Recipe         SavedRecipe
	ImageError     error
	ReloadFallback bool
}
type DeleteResult struct{ MealUnlinkError error }
type Catalogue struct {
	store  CatalogueStore
	images ImageWriter
	meals  MealLinks
}

func NewCatalogue(store CatalogueStore, images ImageWriter, meals MealLinks) *Catalogue {
	return &Catalogue{store: store, images: images, meals: meals}
}
func (catalogue *Catalogue) List(ctx context.Context) ([]SavedRecipe, error) {
	items, err := catalogue.store.FindAll(ctx)
	if items != nil {
		copies := make([]SavedRecipe, len(items))
		for i, item := range items {
			copies[i] = copySavedRecipe(item)
		}
		items = copies
	}
	return items, err
}
func (catalogue *Catalogue) PrepareUpdate(ctx context.Context, id string) (PreparedUpdate, error) {
	item, err := catalogue.store.FindByID(ctx, id)
	if err != nil {
		return PreparedUpdate{}, err
	}
	return PreparedUpdate{existing: copySavedRecipe(item)}, nil
}
func (catalogue *Catalogue) Detail(ctx context.Context, id string) (SavedRecipe, error) {
	item, err := catalogue.store.FindByID(ctx, id)
	if err != nil {
		return SavedRecipe{}, err
	}
	item = copySavedRecipe(item)
	if len(item.Steps) == 0 && item.Instructions != "" {
		item.Steps = domain.SplitIntoSteps(item.Instructions)
	}
	item.StepDurations = domain.StepDurations(item.Steps)
	return item, nil
}
func (catalogue *Catalogue) Create(ctx context.Context, item SavedRecipe, image *string) (SaveResult, error) {
	created, err := catalogue.store.Create(ctx, copySavedRecipe(item))
	if err != nil {
		return SaveResult{}, err
	}
	var warning error
	if image != nil && *image != "" {
		warning = catalogue.images.UpdateImage(ctx, created.ID, *image)
		if warning == nil {
			created.HasImage = true
		}
	}
	return SaveResult{Recipe: copySavedRecipe(created), ImageError: warning}, nil
}
func (catalogue *Catalogue) Update(ctx context.Context, prepared PreparedUpdate, change SavedRecipe, image *string) (SaveResult, error) {
	if prepared.existing.ID == "" {
		return SaveResult{}, ErrNotFound
	}
	existing := copySavedRecipe(prepared.existing)
	change = copySavedRecipe(change)
	existing.Title = change.Title
	existing.Steps = change.Steps
	existing.Ingredients = change.Ingredients
	existing.Servings = change.Servings
	existing.PrepTime = change.PrepTime
	existing.CookTime = change.CookTime
	existing.SourceURL = change.SourceURL
	existing.MealType = change.MealType
	if err := catalogue.store.Update(ctx, copySavedRecipe(existing)); err != nil {
		return SaveResult{}, err
	}
	var warning error
	if image != nil {
		if *image == "" {
			warning = catalogue.images.ClearImage(ctx, existing.ID)
		} else {
			warning = catalogue.images.UpdateImage(ctx, existing.ID, *image)
		}
	}
	updated, err := catalogue.store.FindByID(ctx, existing.ID)
	if err != nil {
		return SaveResult{Recipe: copySavedRecipe(existing), ImageError: warning, ReloadFallback: true}, nil
	}
	return SaveResult{Recipe: copySavedRecipe(updated), ImageError: warning}, nil
}
func (catalogue *Catalogue) Delete(ctx context.Context, id string) (DeleteResult, error) {
	if _, err := catalogue.store.FindByID(ctx, id); err != nil {
		return DeleteResult{}, fmt.Errorf("%w: %w", ErrLookup, err)
	}
	warning := catalogue.meals.ClearRecipeID(ctx, id)
	return DeleteResult{MealUnlinkError: warning}, catalogue.store.Delete(ctx, id)
}
func copySavedRecipe(item SavedRecipe) SavedRecipe {
	if item.Steps != nil {
		item.Steps = append([]string{}, item.Steps...)
	}
	if item.Ingredients != nil {
		groups := make([]IngredientGroup, len(item.Ingredients))
		for i, group := range item.Ingredients {
			groups[i] = group
			if group.Items != nil {
				groups[i].Items = append([]string{}, group.Items...)
			}
		}
		item.Ingredients = groups
	}
	copyString := func(value *string) *string {
		if value == nil {
			return nil
		}
		copy := *value
		return &copy
	}
	copyInt := func(value *int) *int {
		if value == nil {
			return nil
		}
		copy := *value
		return &copy
	}
	item.MealType = copyString(item.MealType)
	item.Servings = copyInt(item.Servings)
	item.PrepTime = copyString(item.PrepTime)
	item.CookTime = copyString(item.CookTime)
	item.SourceURL = copyString(item.SourceURL)
	item.CategoryID = copyString(item.CategoryID)
	if item.StepDurations != nil {
		durations := make([]*int, len(item.StepDurations))
		for i, duration := range item.StepDurations {
			durations[i] = copyInt(duration)
		}
		item.StepDurations = durations
	}
	return item
}

package recipes

import (
	"context"
	"strings"
)

type Recipe struct {
	Title       string
	Ingredients []string
	Steps       []string
	PrepTime    string
	CookTime    string
	Servings    *int
	ImageURL    string
}
type Metadata struct{ Title, ImageURL, Description string }
type Page interface {
	Structured() (Recipe, bool)
	Video() (Recipe, bool)
	Metadata() Metadata
}
type Source interface {
	Validate(string) error
	Resolve(context.Context, string) string
	Load(context.Context, string) (Page, string, error)
	OEmbed(context.Context, string) (Recipe, bool)
	CaptionEmbed(context.Context, string) (Recipe, bool)
}
type Parser interface {
	CleanCaption(string) string
	ApplyCaption(Recipe, string) Recipe
	ResolveURL(string, string) string
}
type Service struct {
	source Source
	parser Parser
}

func New(source Source, parser Parser) *Service { return &Service{source: source, parser: parser} }
func (service *Service) Import(ctx context.Context, rawURL string) (Recipe, error) {
	if err := service.source.Validate(rawURL); err != nil {
		return Recipe{}, err
	}
	target := service.source.Resolve(ctx, rawURL)
	page, finalURL, err := service.source.Load(ctx, target)
	if err != nil {
		if recipe, found := service.source.OEmbed(ctx, target); found {
			return copyRecipe(recipe), nil
		}
		if recipe, found := service.source.CaptionEmbed(ctx, target); found {
			return copyRecipe(recipe), nil
		}
		return Recipe{}, err
	}
	recipe, found := page.Structured()
	if found && len(recipe.Ingredients) > 0 && len(recipe.Steps) > 0 {
		recipe.ImageURL = service.parser.ResolveURL(finalURL, recipe.ImageURL)
		return copyRecipe(recipe), nil
	}
	if addition, ok := service.source.OEmbed(ctx, finalURL); ok {
		merge(&recipe, addition)
	}
	if addition, ok := page.Video(); ok {
		merge(&recipe, addition)
	}
	if addition, ok := service.source.CaptionEmbed(ctx, finalURL); ok {
		merge(&recipe, addition)
	}
	metadata := page.Metadata()
	if recipe.Title == "" {
		recipe.Title = service.parser.CleanCaption(metadata.Title)
	}
	if recipe.ImageURL == "" {
		recipe.ImageURL = metadata.ImageURL
	}
	caption := metadata.Description
	if strings.TrimSpace(caption) == "" {
		caption = metadata.Title
	}
	recipe = service.parser.ApplyCaption(recipe, service.parser.CleanCaption(caption))
	recipe.ImageURL = service.parser.ResolveURL(finalURL, recipe.ImageURL)
	return copyRecipe(recipe), nil
}
func merge(recipe *Recipe, addition Recipe) {
	if recipe.Title == "" {
		recipe.Title = addition.Title
	}
	if len(recipe.Ingredients) == 0 {
		recipe.Ingredients = addition.Ingredients
	}
	if len(recipe.Steps) == 0 {
		recipe.Steps = addition.Steps
	}
	if recipe.ImageURL == "" {
		recipe.ImageURL = addition.ImageURL
	}
	if recipe.PrepTime == "" {
		recipe.PrepTime = addition.PrepTime
	}
	if recipe.CookTime == "" {
		recipe.CookTime = addition.CookTime
	}
	if recipe.Servings == nil {
		recipe.Servings = addition.Servings
	}
}
func copyRecipe(recipe Recipe) Recipe {
	if recipe.Ingredients != nil {
		recipe.Ingredients = append([]string{}, recipe.Ingredients...)
	}
	if recipe.Steps != nil {
		recipe.Steps = append([]string{}, recipe.Steps...)
	}
	if recipe.Servings != nil {
		value := *recipe.Servings
		recipe.Servings = &value
	}
	return recipe
}

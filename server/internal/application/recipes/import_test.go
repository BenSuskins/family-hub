package recipes_test

import (
	"context"
	"errors"
	"testing"

	"github.com/bensuskins/family-hub/internal/application/recipes"
)

type fakePage struct {
	structured recipes.Recipe
	complete   bool
	metadata   recipes.Metadata
}

func (p fakePage) Structured() (recipes.Recipe, bool) { return p.structured, p.complete }
func (p fakePage) Video() (recipes.Recipe, bool)      { return recipes.Recipe{}, false }
func (p fakePage) Metadata() recipes.Metadata         { return p.metadata }

type fakeImporter struct {
	page       fakePage
	loadErr    error
	embed      recipes.Recipe
	embedCalls int
	caption    string
}

func (f *fakeImporter) Validate(string) error                        { return nil }
func (f *fakeImporter) Resolve(_ context.Context, url string) string { return url }
func (f *fakeImporter) Load(_ context.Context, url string) (recipes.Page, string, error) {
	return f.page, url, f.loadErr
}
func (f *fakeImporter) OEmbed(context.Context, string) (recipes.Recipe, bool) {
	f.embedCalls++
	return f.embed, f.embed.Title != ""
}
func (f *fakeImporter) CaptionEmbed(context.Context, string) (recipes.Recipe, bool) {
	return recipes.Recipe{}, false
}
func (f *fakeImporter) CleanCaption(value string) string { return value }
func (f *fakeImporter) ApplyCaption(recipe recipes.Recipe, caption string) recipes.Recipe {
	f.caption = caption
	return recipe
}
func (f *fakeImporter) ResolveURL(_ string, value string) string { return value }
func TestImportPrefersCompleteStructuredRecipe(t *testing.T) {
	provider := &fakeImporter{page: fakePage{structured: recipes.Recipe{Title: "Structured", Ingredients: []string{"flour"}, Steps: []string{"mix"}}, complete: true}, embed: recipes.Recipe{Title: "Fallback"}}
	result, err := recipes.New(provider, provider).Import(context.Background(), "https://example.com")
	if err != nil || result.Title != "Structured" || provider.embedCalls != 0 {
		t.Fatalf("result=%+v embeds=%d error=%v", result, provider.embedCalls, err)
	}
}
func TestImportUsesOEmbedWhenPageFails(t *testing.T) {
	provider := &fakeImporter{loadErr: errors.New("blocked"), embed: recipes.Recipe{Title: "Fallback"}}
	result, err := recipes.New(provider, provider).Import(context.Background(), "https://example.com")
	if err != nil || result.Title != "Fallback" {
		t.Fatalf("result=%+v error=%v", result, err)
	}
}

func TestImportUsesTitleWhenDescriptionIsBlank(t *testing.T) {
	provider := &fakeImporter{page: fakePage{metadata: recipes.Metadata{Title: "Recipe title", Description: "  "}}}
	_, err := recipes.New(provider, provider).Import(context.Background(), "https://example.com")
	if err != nil || provider.caption != "Recipe title" {
		t.Fatalf("caption=%q error=%v", provider.caption, err)
	}
}

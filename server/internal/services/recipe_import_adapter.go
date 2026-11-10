package services

import (
	"context"

	recipeapp "github.com/bensuskins/family-hub/internal/application/recipes"
	"golang.org/x/net/html"
)

type recipePage struct{ document *html.Node }

func (page recipePage) Structured() (recipeapp.Recipe, bool) {
	return extractStructuredRecipe(page.document)
}
func (page recipePage) Video() (recipeapp.Recipe, bool) { return extractFromVideoJSONLD(page.document) }
func (page recipePage) Metadata() recipeapp.Metadata {
	metadata := extractMetadata(page.document)
	return recipeapp.Metadata{Title: metadata.Title, ImageURL: metadata.ImageURL, Description: metadata.Description}
}
func (extractor *RecipeExtractor) Validate(rawURL string) error { return extractor.validateURL(rawURL) }
func (extractor *RecipeExtractor) Resolve(ctx context.Context, rawURL string) string {
	return extractor.resolveShortLink(ctx, rawURL)
}
func (extractor *RecipeExtractor) Load(ctx context.Context, rawURL string) (recipeapp.Page, string, error) {
	document, finalURL, err := extractor.fetchDocument(ctx, rawURL)
	if err != nil {
		return nil, finalURL, err
	}
	return recipePage{document: document}, finalURL, nil
}
func (extractor *RecipeExtractor) OEmbed(ctx context.Context, rawURL string) (recipeapp.Recipe, bool) {
	return extractor.extractViaOEmbedFor(ctx, rawURL)
}
func (extractor *RecipeExtractor) CaptionEmbed(ctx context.Context, rawURL string) (recipeapp.Recipe, bool) {
	return extractor.extractViaCaptionEmbed(ctx, rawURL)
}
func (extractor *RecipeExtractor) CleanCaption(value string) string { return cleanSocialCaption(value) }
func (extractor *RecipeExtractor) ApplyCaption(recipe recipeapp.Recipe, caption string) recipeapp.Recipe {
	applyCaption(&recipe, caption)
	return recipe
}
func (extractor *RecipeExtractor) ResolveURL(base, reference string) string {
	return resolveURL(base, reference)
}

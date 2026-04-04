package services_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bensuskins/family-hub/internal/services"
)

func intPtr(n int) *int { return &n }

func TestRecipeExtractor_Extract(t *testing.T) {
	tests := []struct {
		name string
		html string
		want services.ExtractedRecipe
	}{
		{
			name: "JSON-LD top-level Recipe",
			html: `<html><head><script type="application/ld+json">{
				"@type": "Recipe",
				"name": "Pancakes",
				"recipeIngredient": ["2 cups flour", "1 cup milk", "2 eggs"],
				"recipeInstructions": [
					{"@type": "HowToStep", "text": "Mix dry ingredients."},
					{"@type": "HowToStep", "text": "Add wet ingredients."},
					{"@type": "HowToStep", "text": "Cook on griddle."}
				],
				"prepTime": "PT10M",
				"cookTime": "PT20M",
				"recipeYield": "4",
				"image": "https://example.com/pancakes.jpg"
			}</script></head><body></body></html>`,
			want: services.ExtractedRecipe{
				Title:       "Pancakes",
				Ingredients: []string{"2 cups flour", "1 cup milk", "2 eggs"},
				Steps:       []string{"Mix dry ingredients.", "Add wet ingredients.", "Cook on griddle."},
				PrepTime:    "10 mins",
				CookTime:    "20 mins",
				Servings:    intPtr(4),
				ImageURL:    "https://example.com/pancakes.jpg",
			},
		},
		{
			name: "JSON-LD in @graph array",
			html: `<html><head><script type="application/ld+json">{
				"@graph": [
					{"@type": "WebPage", "name": "My Blog"},
					{
						"@type": "Recipe",
						"name": "Soup",
						"recipeIngredient": ["1 onion", "2 carrots"],
						"recipeInstructions": [{"@type": "HowToStep", "text": "Chop vegetables."}, {"@type": "HowToStep", "text": "Simmer for 30 minutes."}]
					}
				]
			}</script></head><body></body></html>`,
			want: services.ExtractedRecipe{
				Title:       "Soup",
				Ingredients: []string{"1 onion", "2 carrots"},
				Steps:       []string{"Chop vegetables.", "Simmer for 30 minutes."},
			},
		},
		{
			name: "JSON-LD plain array",
			html: `<html><head><script type="application/ld+json">[
				{"@type": "Recipe", "name": "Toast", "recipeIngredient": ["2 slices bread"]}
			]</script></head><body></body></html>`,
			want: services.ExtractedRecipe{
				Title:       "Toast",
				Ingredients: []string{"2 slices bread"},
			},
		},
		{
			name: "HowToSection instructions",
			html: `<html><head><script type="application/ld+json">{
				"@type": "Recipe",
				"name": "Layered Cake",
				"recipeInstructions": [
					{
						"@type": "HowToSection",
						"name": "Make the batter",
						"itemListElement": [
							{"@type": "HowToStep", "text": "Cream butter and sugar."},
							{"@type": "HowToStep", "text": "Add eggs one at a time."}
						]
					},
					{
						"@type": "HowToSection",
						"name": "Bake",
						"itemListElement": [
							{"@type": "HowToStep", "text": "Pour into pan."},
							{"@type": "HowToStep", "text": "Bake at 350F for 30 minutes."}
						]
					}
				]
			}</script></head><body></body></html>`,
			want: services.ExtractedRecipe{
				Title: "Layered Cake",
				Steps: []string{
					"Cream butter and sugar.",
					"Add eggs one at a time.",
					"Pour into pan.",
					"Bake at 350F for 30 minutes.",
				},
			},
		},
		{
			name: "string instructions split by newlines",
			html: `<html><head><script type="application/ld+json">{
				"@type": "Recipe",
				"name": "Quick Salad",
				"recipeInstructions": "Chop lettuce.\nAdd tomatoes.\nDrizzle dressing."
			}</script></head><body></body></html>`,
			want: services.ExtractedRecipe{
				Title: "Quick Salad",
				Steps: []string{"Chop lettuce.", "Add tomatoes.", "Drizzle dressing."},
			},
		},
		{
			name: "plain string array instructions",
			html: `<html><head><script type="application/ld+json">{
				"@type": "Recipe",
				"name": "Simple Recipe",
				"recipeInstructions": ["Step one.", "Step two.", "Step three."]
			}</script></head><body></body></html>`,
			want: services.ExtractedRecipe{
				Title: "Simple Recipe",
				Steps: []string{"Step one.", "Step two.", "Step three."},
			},
		},
		{
			name: "ISO 8601 duration with hours and minutes",
			html: `<html><head><script type="application/ld+json">{
				"@type": "Recipe",
				"name": "Slow Roast",
				"prepTime": "PT1H30M",
				"cookTime": "PT2H"
			}</script></head><body></body></html>`,
			want: services.ExtractedRecipe{
				Title:    "Slow Roast",
				PrepTime: "1 hr 30 mins",
				CookTime: "2 hrs",
			},
		},
		{
			name: "recipeYield as integer",
			html: `<html><head><script type="application/ld+json">{
				"@type": "Recipe",
				"name": "Cookies",
				"recipeYield": 24
			}</script></head><body></body></html>`,
			want: services.ExtractedRecipe{
				Title:    "Cookies",
				Servings: intPtr(24),
			},
		},
		{
			name: "recipeYield as string with text",
			html: `<html><head><script type="application/ld+json">{
				"@type": "Recipe",
				"name": "Stew",
				"recipeYield": "6 servings"
			}</script></head><body></body></html>`,
			want: services.ExtractedRecipe{
				Title:    "Stew",
				Servings: intPtr(6),
			},
		},
		{
			name: "recipeYield as array",
			html: `<html><head><script type="application/ld+json">{
				"@type": "Recipe",
				"name": "Pizza",
				"recipeYield": ["8 slices"]
			}</script></head><body></body></html>`,
			want: services.ExtractedRecipe{
				Title:    "Pizza",
				Servings: intPtr(8),
			},
		},
		{
			name: "image as object with url",
			html: `<html><head><script type="application/ld+json">{
				"@type": "Recipe",
				"name": "Risotto",
				"image": {"@type": "ImageObject", "url": "https://example.com/risotto.jpg"}
			}</script></head><body></body></html>`,
			want: services.ExtractedRecipe{
				Title:    "Risotto",
				ImageURL: "https://example.com/risotto.jpg",
			},
		},
		{
			name: "image as string array",
			html: `<html><head><script type="application/ld+json">{
				"@type": "Recipe",
				"name": "Tacos",
				"image": ["https://example.com/tacos1.jpg", "https://example.com/tacos2.jpg"]
			}</script></head><body></body></html>`,
			want: services.ExtractedRecipe{
				Title:    "Tacos",
				ImageURL: "https://example.com/tacos1.jpg",
			},
		},
		{
			name: "@type as array",
			html: `<html><head><script type="application/ld+json">{
				"@type": ["Recipe", "Thing"],
				"name": "Multi-Type Recipe",
				"recipeIngredient": ["flour"]
			}</script></head><body></body></html>`,
			want: services.ExtractedRecipe{
				Title:       "Multi-Type Recipe",
				Ingredients: []string{"flour"},
			},
		},
		{
			name: "no recipe data returns empty",
			html: `<html><head><title>Blog Post</title></head><body><p>Hello world</p></body></html>`,
			want: services.ExtractedRecipe{},
		},
		{
			name: "microdata fallback",
			html: `<html><body>
				<div itemscope itemtype="http://schema.org/Recipe">
					<h1 itemprop="name">Microdata Soup</h1>
					<img itemprop="image" src="https://example.com/soup.jpg" />
					<span itemprop="recipeIngredient">1 onion</span>
					<span itemprop="recipeIngredient">2 potatoes</span>
					<div itemprop="recipeInstructions">Boil everything together.</div>
					<meta itemprop="prepTime" content="PT15M" />
					<meta itemprop="cookTime" content="PT45M" />
					<meta itemprop="recipeYield" content="4" />
				</div>
			</body></html>`,
			want: services.ExtractedRecipe{
				Title:       "Microdata Soup",
				Ingredients: []string{"1 onion", "2 potatoes"},
				Steps:       []string{"Boil everything together."},
				PrepTime:    "15 mins",
				CookTime:    "45 mins",
				Servings:    intPtr(4),
				ImageURL:    "https://example.com/soup.jpg",
			},
		},
		{
			name: "HTML entities in ingredients are decoded",
			html: `<html><head><script type="application/ld+json">{
				"@type": "Recipe",
				"name": "Test",
				"recipeIngredient": ["1 cup flour &amp; sugar", "2 &quot;large&quot; eggs"]
			}</script></head><body></body></html>`,
			want: services.ExtractedRecipe{
				Title:       "Test",
				Ingredients: []string{`1 cup flour & sugar`, `2 "large" eggs`},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				w.Write([]byte(tt.html))
			}))
			defer server.Close()

			extractor := services.NewRecipeExtractor()
			got, err := extractor.Extract(context.Background(), server.URL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			assertStringEqual(t, "Title", got.Title, tt.want.Title)
			assertStringSliceEqual(t, "Ingredients", got.Ingredients, tt.want.Ingredients)
			assertStringSliceEqual(t, "Steps", got.Steps, tt.want.Steps)
			assertStringEqual(t, "PrepTime", got.PrepTime, tt.want.PrepTime)
			assertStringEqual(t, "CookTime", got.CookTime, tt.want.CookTime)
			assertStringEqual(t, "ImageURL", got.ImageURL, tt.want.ImageURL)
			assertIntPtrEqual(t, "Servings", got.Servings, tt.want.Servings)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"PT30M", "30 mins"},
		{"PT1H", "1 hr"},
		{"PT1H30M", "1 hr 30 mins"},
		{"PT2H", "2 hrs"},
		{"PT2H15M", "2 hrs 15 mins"},
		{"PT1M", "1 min"},
		{"P0DT0H45M", "45 mins"},
		{"pt30m", "30 mins"},
		{"", ""},
		{"invalid", ""},
		{"PT0M", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := services.FormatDuration(tt.input)
			if got != tt.want {
				t.Errorf("FormatDuration(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func assertStringEqual(t *testing.T, field, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %q, want %q", field, got, want)
	}
}

func assertStringSliceEqual(t *testing.T, field string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Errorf("%s length = %d, want %d\ngot:  %v\nwant: %v", field, len(got), len(want), got, want)
		return
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("%s[%d] = %q, want %q", field, i, got[i], want[i])
		}
	}
}

func assertIntPtrEqual(t *testing.T, field string, got, want *int) {
	t.Helper()
	if got == nil && want == nil {
		return
	}
	if got == nil || want == nil {
		t.Errorf("%s = %v, want %v", field, got, want)
		return
	}
	if *got != *want {
		t.Errorf("%s = %d, want %d", field, *got, *want)
	}
}

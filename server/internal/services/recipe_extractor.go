package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type ExtractedRecipe struct {
	Title       string   `json:"title"`
	Ingredients []string `json:"ingredients"`
	Steps       []string `json:"steps"`
	PrepTime    string   `json:"prepTime,omitempty"`
	CookTime    string   `json:"cookTime,omitempty"`
	Servings    *int     `json:"servings,omitempty"`
	ImageURL    string   `json:"imageURL,omitempty"`
}

type RecipeExtractor struct {
	client             *http.Client
	skipURLValidation  bool
}

func NewRecipeExtractor() *RecipeExtractor {
	return &RecipeExtractor{
		client: NewSafeHTTPClient(15 * time.Second),
	}
}

func NewRecipeExtractorWithoutSSRFProtection() *RecipeExtractor {
	return &RecipeExtractor{
		client:            &http.Client{Timeout: 15 * time.Second},
		skipURLValidation: true,
	}
}

func (extractor *RecipeExtractor) Extract(ctx context.Context, rawURL string) (ExtractedRecipe, error) {
	if !extractor.skipURLValidation {
		if err := ValidateExternalURL(rawURL); err != nil {
			return ExtractedRecipe{}, err
		}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return ExtractedRecipe{}, fmt.Errorf("creating request: %w", err)
	}
	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; FamilyHub/1.0)")

	response, err := extractor.client.Do(request)
	if err != nil {
		return ExtractedRecipe{}, fmt.Errorf("fetching URL: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ExtractedRecipe{}, fmt.Errorf("unexpected status %d", response.StatusCode)
	}

	limitedBody := io.LimitReader(response.Body, 5*1024*1024)
	document, err := html.Parse(limitedBody)
	if err != nil {
		return ExtractedRecipe{}, fmt.Errorf("parsing HTML: %w", err)
	}

	if recipe, found := extractFromJSONLD(document); found {
		return recipe, nil
	}

	if recipe, found := extractFromMicrodata(document); found {
		return recipe, nil
	}

	return ExtractedRecipe{}, nil
}

// --- JSON-LD extraction ---

func extractFromJSONLD(document *html.Node) (ExtractedRecipe, bool) {
	var scripts []string
	findJSONLDScripts(document, &scripts)

	for _, scriptContent := range scripts {
		var raw any
		if err := json.Unmarshal([]byte(scriptContent), &raw); err != nil {
			continue
		}
		if recipeMap, found := findRecipeObject(raw); found {
			return parseRecipeMap(recipeMap), true
		}
	}
	return ExtractedRecipe{}, false
}

func findJSONLDScripts(node *html.Node, results *[]string) {
	if node.Type == html.ElementNode && node.Data == "script" {
		if getAttr(node, "type") == "application/ld+json" {
			if node.FirstChild != nil && node.FirstChild.Type == html.TextNode {
				*results = append(*results, node.FirstChild.Data)
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		findJSONLDScripts(child, results)
	}
}

func findRecipeObject(data any) (map[string]any, bool) {
	switch value := data.(type) {
	case map[string]any:
		if isRecipeType(value) {
			return value, true
		}
		if graph, ok := value["@graph"].([]any); ok {
			for _, item := range graph {
				if dict, ok := item.(map[string]any); ok && isRecipeType(dict) {
					return dict, true
				}
			}
		}
	case []any:
		for _, item := range value {
			if dict, ok := item.(map[string]any); ok && isRecipeType(dict) {
				return dict, true
			}
		}
	}
	return nil, false
}

func isRecipeType(dict map[string]any) bool {
	switch typeValue := dict["@type"].(type) {
	case string:
		return typeValue == "Recipe"
	case []any:
		for _, item := range typeValue {
			if str, ok := item.(string); ok && str == "Recipe" {
				return true
			}
		}
	}
	return false
}

func parseRecipeMap(recipe map[string]any) ExtractedRecipe {
	result := ExtractedRecipe{
		Title:       stringField(recipe, "name"),
		Ingredients: extractIngredients(recipe),
		Steps:       extractSteps(recipe),
		ImageURL:    extractImageURL(recipe),
	}

	if prepTime := stringField(recipe, "prepTime"); prepTime != "" {
		result.PrepTime = FormatDuration(prepTime)
	}
	if cookTime := stringField(recipe, "cookTime"); cookTime != "" {
		result.CookTime = FormatDuration(cookTime)
	}
	if servings := extractServings(recipe); servings != nil {
		result.Servings = servings
	}

	return result
}

func stringField(dict map[string]any, key string) string {
	if value, ok := dict[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func extractIngredients(recipe map[string]any) []string {
	items, ok := recipe["recipeIngredient"].([]any)
	if !ok {
		return nil
	}
	var result []string
	for _, item := range items {
		if str, ok := item.(string); ok {
			trimmed := strings.TrimSpace(htmlDecode(str))
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
	}
	return result
}

func extractSteps(recipe map[string]any) []string {
	instructions, ok := recipe["recipeInstructions"]
	if !ok {
		return nil
	}

	var result []string

	switch value := instructions.(type) {
	case string:
		for _, line := range strings.Split(value, "\n") {
			trimmed := strings.TrimSpace(htmlDecode(line))
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
	case []any:
		for _, item := range value {
			switch step := item.(type) {
			case string:
				trimmed := strings.TrimSpace(htmlDecode(step))
				if trimmed != "" {
					result = append(result, trimmed)
				}
			case map[string]any:
				stepType, _ := step["@type"].(string)
				if stepType == "HowToSection" {
					if sectionItems, ok := step["itemListElement"].([]any); ok {
						for _, sectionItem := range sectionItems {
							if sectionStep, ok := sectionItem.(map[string]any); ok {
								if text := stringField(sectionStep, "text"); text != "" {
									result = append(result, htmlDecode(text))
								}
							}
						}
					}
				} else if text := stringField(step, "text"); text != "" {
					result = append(result, htmlDecode(text))
				}
			}
		}
	}

	return result
}

func extractImageURL(recipe map[string]any) string {
	switch value := recipe["image"].(type) {
	case string:
		return value
	case []any:
		if len(value) > 0 {
			if str, ok := value[0].(string); ok {
				return str
			}
			if obj, ok := value[0].(map[string]any); ok {
				if u, ok := obj["url"].(string); ok {
					return u
				}
			}
		}
	case map[string]any:
		if u, ok := value["url"].(string); ok {
			return u
		}
	}
	return ""
}

func extractServings(recipe map[string]any) *int {
	switch value := recipe["recipeYield"].(type) {
	case float64:
		n := int(value)
		return &n
	case string:
		if n := firstInt(value); n > 0 {
			return &n
		}
	case []any:
		if len(value) > 0 {
			switch first := value[0].(type) {
			case float64:
				n := int(first)
				return &n
			case string:
				if n := firstInt(first); n > 0 {
					return &n
				}
			}
		}
	}
	return nil
}

var digitsRegex = regexp.MustCompile(`\d+`)

func firstInt(text string) int {
	match := digitsRegex.FindString(text)
	if match == "" {
		return 0
	}
	n, _ := strconv.Atoi(match)
	return n
}

// FormatDuration converts an ISO 8601 duration (e.g. "PT1H30M") to a human-readable string.
func FormatDuration(iso string) string {
	upper := strings.ToUpper(strings.TrimSpace(iso))
	if !strings.HasPrefix(upper, "P") {
		return ""
	}

	remaining := upper[1:]
	if idx := strings.Index(remaining, "T"); idx >= 0 {
		remaining = remaining[idx+1:]
	}

	var hours, minutes int

	if idx := strings.Index(remaining, "H"); idx >= 0 {
		hours, _ = strconv.Atoi(remaining[:idx])
		remaining = remaining[idx+1:]
	}
	if idx := strings.Index(remaining, "M"); idx >= 0 {
		minutes, _ = strconv.Atoi(remaining[:idx])
	}

	if hours == 0 && minutes == 0 {
		return ""
	}

	var parts []string
	if hours > 0 {
		suffix := " hr"
		if hours > 1 {
			suffix = " hrs"
		}
		parts = append(parts, strconv.Itoa(hours)+suffix)
	}
	if minutes > 0 {
		suffix := " min"
		if minutes > 1 {
			suffix = " mins"
		}
		parts = append(parts, strconv.Itoa(minutes)+suffix)
	}
	return strings.Join(parts, " ")
}

// --- Microdata extraction ---

func extractFromMicrodata(document *html.Node) (ExtractedRecipe, bool) {
	var ingredients []string
	var steps []string
	var title string
	var imageURL string
	var prepTime string
	var cookTime string
	var servings *int

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			itemprop := getAttr(node, "itemprop")
			switch itemprop {
			case "recipeIngredient", "ingredients":
				if text := textContent(node); text != "" {
					ingredients = append(ingredients, text)
				}
			case "recipeInstructions":
				if text := textContent(node); text != "" {
					for _, line := range strings.Split(text, "\n") {
						trimmed := strings.TrimSpace(line)
						if trimmed != "" {
							steps = append(steps, trimmed)
						}
					}
				}
			case "name":
				if title == "" {
					if text := textContent(node); text != "" {
						title = text
					}
				}
			case "image":
				if imageURL == "" {
					if src := getAttr(node, "src"); src != "" {
						imageURL = src
					} else if content := getAttr(node, "content"); content != "" {
						imageURL = content
					}
				}
			case "prepTime":
				if content := getAttr(node, "content"); content != "" {
					prepTime = FormatDuration(content)
				} else if datetime := getAttr(node, "datetime"); datetime != "" {
					prepTime = FormatDuration(datetime)
				}
			case "cookTime":
				if content := getAttr(node, "content"); content != "" {
					cookTime = FormatDuration(content)
				} else if datetime := getAttr(node, "datetime"); datetime != "" {
					cookTime = FormatDuration(datetime)
				}
			case "recipeYield":
				if content := getAttr(node, "content"); content != "" {
					if n := firstInt(content); n > 0 {
						servings = &n
					}
				} else if text := textContent(node); text != "" {
					if n := firstInt(text); n > 0 {
						servings = &n
					}
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)

	found := len(ingredients) > 0 || len(steps) > 0 || title != ""
	if !found {
		return ExtractedRecipe{}, false
	}

	return ExtractedRecipe{
		Title:       title,
		Ingredients: ingredients,
		Steps:       steps,
		PrepTime:    prepTime,
		CookTime:    cookTime,
		Servings:    servings,
		ImageURL:    imageURL,
	}, true
}

// --- HTML helpers ---

func getAttr(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, key) {
			return attr.Val
		}
	}
	return ""
}

func textContent(node *html.Node) string {
	var builder strings.Builder
	var collect func(*html.Node)
	collect = func(n *html.Node) {
		if n.Type == html.TextNode {
			builder.WriteString(n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			collect(child)
		}
	}
	collect(node)
	return strings.TrimSpace(builder.String())
}

func htmlDecode(text string) string {
	replacer := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", "\"",
		"&#39;", "'",
		"&apos;", "'",
		"&nbsp;", " ",
	)
	return replacer.Replace(text)
}

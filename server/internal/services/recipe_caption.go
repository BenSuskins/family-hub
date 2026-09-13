package services

import (
	"regexp"
	"strings"
	"unicode"
)

// TikTok and Instagram posts have no structured recipe data — the cook writes
// the whole thing into the caption. This file turns such a caption into a
// title, an ingredient list and a method, and degrades to "just the title"
// when the caption is only a sentence of hype.

const maxCaptionTitleLength = 80

var (
	// Engagement preamble Instagram prepends to og:description, e.g.
	// `12K likes, 340 comments - joe on January 1, 2026: "Caption"`.
	instagramEngagementPrefix = regexp.MustCompile(`(?i)^[\d.,]+\s*[km]?\s+likes?,\s*[\d.,]+\s*[km]?\s+comments?\s*[-–]\s*`)
	// Author preamble both platforms use on og:title, e.g.
	// `joe on Instagram: "Caption"` or `joe on January 1, 2026: "Caption"`.
	socialAuthorPrefix = regexp.MustCompile(`(?i)^[^:\n]{1,120}\bon\b[^:\n]{0,60}:\s*`)

	ingredientsHeading = regexp.MustCompile(`(?i)^\W*(ingredients|you(?:'|’)?ll need|what you(?:'|’)?ll need|you need|shopping list)\b\s*:?`)
	methodHeading      = regexp.MustCompile(`(?i)^\W*(method|instructions|directions|steps|how to make(?: it| this)?|how i(?:'|’)?d? made? (?:it|this))\b\s*:?`)
	stepNumberPrefix   = regexp.MustCompile(`(?i)^\W*(?:step\s*)?\d{1,2}\s*[.)\-:]\s*`)
	hashtagToken       = regexp.MustCompile(`(^|\s)#[\p{L}\p{N}_]+`)
	trailingCredit     = regexp.MustCompile(`(?i)\s*[-–—|]\s*(full recipe|recipe|link in bio).*$`)
	whitespaceRun      = regexp.MustCompile(`\s+`)

	// Captions often run the whole recipe onto one line, so headings and step
	// numbers have to be recognised mid-line too.
	inlineHeading    = regexp.MustCompile(`(?i)\s+((?:ingredients|you(?:'|’)?ll need|what you(?:'|’)?ll need|you need|shopping list|method|instructions|directions|steps|how to make(?: it| this)?)\s*:)`)
	numberedStepMark = regexp.MustCompile(`(?:^|\s)\d{1,2}\s*[.)]\s+`)

	servingsPattern = regexp.MustCompile(`(?i)\b(?:serves|servings?|feeds|makes)\s*:?\s*(\d{1,3})\b`)
	prepTimePattern = regexp.MustCompile(`(?i)\bprep(?:aration)?(?:\s*time)?\s*:?\s*(\d{1,3})\s*(m|min|mins|minute|minutes|h|hr|hrs|hour|hours)\b`)
	cookTimePattern = regexp.MustCompile(`(?i)\b(?:cook|cooking|bake|baking|oven)(?:\s*time)?\s*:?\s*(\d{1,3})\s*(m|min|mins|minute|minutes|h|hr|hrs|hour|hours)\b`)
)

// cleanSocialCaption strips the boilerplate the platforms wrap around a caption
// before it reaches an og: tag or an oEmbed title.
func cleanSocialCaption(text string) string {
	caption := htmlDecode(strings.TrimSpace(text))
	if caption == "" {
		return ""
	}

	if location := instagramEngagementPrefix.FindStringIndex(caption); location != nil {
		caption = strings.TrimSpace(caption[location[1]:])
	}
	// Only strip an author preamble when it is followed by a quoted caption,
	// otherwise a legitimate title like "Pasta: a guide" would lose its start.
	if location := socialAuthorPrefix.FindStringIndex(caption); location != nil {
		remainder := strings.TrimSpace(caption[location[1]:])
		if strings.HasPrefix(remainder, `"`) || strings.HasPrefix(remainder, "“") {
			caption = remainder
		}
	}

	return strings.TrimSpace(strings.Trim(strings.TrimSpace(caption), `"“”`))
}

// applyCaption fills in whatever of recipe is still missing from a caption.
// Existing structured data always wins.
func applyCaption(recipe *ExtractedRecipe, caption string) {
	if strings.TrimSpace(caption) == "" {
		return
	}

	parsed := parseCaption(caption)

	if recipe.Title == "" {
		recipe.Title = parsed.Title
	}
	if len(recipe.Ingredients) == 0 {
		recipe.Ingredients = parsed.Ingredients
	}
	if len(recipe.Steps) == 0 {
		recipe.Steps = parsed.Steps
	}
	if recipe.Servings == nil {
		recipe.Servings = parsed.Servings
	}
	if recipe.PrepTime == "" {
		recipe.PrepTime = parsed.PrepTime
	}
	if recipe.CookTime == "" {
		recipe.CookTime = parsed.CookTime
	}
}

// parseCaption splits a caption into a recipe. Ingredients and steps are only
// returned when the caption labels them, because guessing at unlabelled lines
// produces more noise than it saves.
func parseCaption(caption string) ExtractedRecipe {
	lines := captionLines(caption)

	const (
		sectionNone = iota
		sectionIngredients
		sectionSteps
	)

	section := sectionNone
	var titleCandidates []string
	var ingredients []string
	var steps []string

	for _, line := range lines {
		if remainder, matched := matchHeading(line, ingredientsHeading); matched {
			section = sectionIngredients
			if remainder != "" {
				ingredients = append(ingredients, splitInlineItems(remainder)...)
			}
			continue
		}
		if remainder, matched := matchHeading(line, methodHeading); matched {
			section = sectionSteps
			if remainder != "" {
				steps = append(steps, splitNumberedSteps(remainder)...)
			}
			continue
		}

		switch section {
		case sectionIngredients:
			if item := cleanCaptionLine(line); item != "" {
				ingredients = append(ingredients, item)
			}
		case sectionSteps:
			steps = append(steps, splitNumberedSteps(cleanCaptionLine(line))...)
		default:
			if candidate := cleanCaptionLine(line); candidate != "" {
				titleCandidates = append(titleCandidates, candidate)
			}
		}
	}

	parsed := ExtractedRecipe{
		Title:       captionTitle(titleCandidates),
		Ingredients: ingredients,
		Steps:       steps,
	}
	parsed.Servings = captionServings(caption)
	parsed.PrepTime = captionDuration(caption, prepTimePattern)
	parsed.CookTime = captionDuration(caption, cookTimePattern)

	return parsed
}

func captionServings(caption string) *int {
	match := servingsPattern.FindStringSubmatch(caption)
	if match == nil {
		return nil
	}
	if servings := firstInt(match[1]); servings > 0 {
		return &servings
	}
	return nil
}

// captionDuration reads "prep 15 mins" / "cook time: 1 hour" out of a caption
// and reuses the ISO 8601 formatter so the output matches structured data.
func captionDuration(caption string, pattern *regexp.Regexp) string {
	match := pattern.FindStringSubmatch(caption)
	if match == nil {
		return ""
	}

	unit := "M"
	if strings.HasPrefix(strings.ToLower(match[2]), "h") {
		unit = "H"
	}
	return FormatDuration("PT" + match[1] + unit)
}

// captionLines normalises a caption into candidate lines. Single-line captions
// are split on the bullet characters social cooks use instead of line breaks.
func captionLines(caption string) []string {
	normalised := strings.NewReplacer("\r\n", "\n", "\r", "\n", `\n`, "\n").Replace(caption)

	normalised = inlineHeading.ReplaceAllString(normalised, "\n$1")

	var lines []string
	for _, line := range strings.Split(normalised, "\n") {
		for _, part := range splitOnBullets(line) {
			if trimmed := strings.TrimSpace(part); trimmed != "" {
				lines = append(lines, trimmed)
			}
		}
	}
	return lines
}

// splitNumberedSteps breaks "1. Sear the chicken 2. Add cream" into one entry
// per step, leaving unnumbered text as a single step.
func splitNumberedSteps(text string) []string {
	markers := numberedStepMark.FindAllStringIndex(text, -1)
	if len(markers) < 2 {
		if step := stripStepNumber(text); step != "" {
			return []string{step}
		}
		return nil
	}

	var steps []string
	if preamble := strings.TrimSpace(text[:markers[0][0]]); preamble != "" {
		steps = append(steps, preamble)
	}
	for index, marker := range markers {
		end := len(text)
		if index+1 < len(markers) {
			end = markers[index+1][0]
		}
		if step := strings.TrimSpace(text[marker[1]:end]); step != "" {
			steps = append(steps, step)
		}
	}
	return steps
}

func splitOnBullets(line string) []string {
	if !strings.ContainsAny(line, "•·‣▪●") {
		return []string{line}
	}
	return strings.FieldsFunc(line, func(r rune) bool {
		return strings.ContainsRune("•·‣▪●", r)
	})
}

// matchHeading reports whether a line is a section heading, returning any
// content that followed it on the same line.
func matchHeading(line string, heading *regexp.Regexp) (string, bool) {
	location := heading.FindStringIndex(line)
	if location == nil || location[0] != 0 {
		return "", false
	}

	matched := line[location[0]:location[1]]
	remainder := strings.TrimSpace(line[location[1]:])

	// "Ingredients:" or a heading on a line of its own. Anything else (say
	// "Steps I wish I'd known") is ordinary prose.
	if strings.HasSuffix(strings.TrimSpace(matched), ":") || remainder == "" {
		return cleanCaptionLine(remainder), true
	}
	return "", false
}

// splitInlineItems handles "Ingredients: 200g flour, 2 eggs, a pinch of salt".
func splitInlineItems(text string) []string {
	if !strings.Contains(text, ",") {
		if cleaned := cleanCaptionLine(text); cleaned != "" {
			return []string{cleaned}
		}
		return nil
	}

	var items []string
	for _, part := range strings.Split(text, ",") {
		if item := cleanCaptionLine(part); item != "" {
			items = append(items, item)
		}
	}
	return items
}

func stripStepNumber(line string) string {
	return strings.TrimSpace(stepNumberPrefix.ReplaceAllString(line, ""))
}

// cleanCaptionLine removes hashtags and collapses whitespace.
func cleanCaptionLine(line string) string {
	cleaned := hashtagToken.ReplaceAllString(line, "")
	cleaned = whitespaceRun.ReplaceAllString(cleaned, " ")
	return strings.TrimSpace(cleaned)
}

// captionTitle picks the first line that reads like a dish name and trims it to
// something that fits a recipe card.
func captionTitle(candidates []string) string {
	for _, candidate := range candidates {
		title := trimTrailingPunctuation(trailingCredit.ReplaceAllString(candidate, ""))
		title = trimTrailingPunctuation(stripTrailingServings(title))
		if !hasLetters(title) {
			continue
		}
		return truncateOnWordBoundary(title, maxCaptionTitleLength)
	}
	return ""
}

// stripTrailingServings removes a "Serves 4" tacked onto a dish name.
func stripTrailingServings(title string) string {
	location := servingsPattern.FindStringIndex(title)
	if location == nil || location[0] == 0 || strings.TrimSpace(title[location[1]:]) != "" {
		return title
	}
	return strings.TrimSpace(title[:location[0]])
}

func hasLetters(text string) bool {
	for _, r := range text {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

func trimTrailingPunctuation(text string) string {
	return strings.TrimSpace(strings.TrimRight(strings.TrimSpace(text), " .!-–—:|"))
}

func truncateOnWordBoundary(text string, limit int) string {
	if len([]rune(text)) <= limit {
		return text
	}

	runes := []rune(text)[:limit]
	truncated := string(runes)
	if index := strings.LastIndex(truncated, " "); index > limit/2 {
		truncated = truncated[:index]
	}
	return strings.TrimSpace(strings.TrimRight(truncated, " ,;:-–—")) + "…"
}

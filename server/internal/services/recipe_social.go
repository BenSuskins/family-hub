package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

// Social video platforms render their pages client side, so the usual
// JSON-LD / microdata scrape finds nothing. They do serve Open Graph tags to
// crawlers (and TikTok publishes an unauthenticated oEmbed endpoint), which is
// enough for a title, an image, and the caption — captions are where TikTok and
// Instagram cooks actually write the recipe.

const (
	defaultUserAgent = "Mozilla/5.0 (compatible; FamilyHub/1.0)"
	// Social hosts only serve Open Graph tags to recognised crawlers.
	crawlerUserAgent = "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)"
)

// defaultOEmbedProviders maps a registrable host to an oEmbed endpoint that
// works without an API key.
func defaultOEmbedProviders() map[string]string {
	return map[string]string{
		"tiktok.com": "https://www.tiktok.com/oembed",
	}
}

// defaultCaptionEmbedHosts publish a public, server-rendered embed page whose
// caption is the only place a signed-out client can read the recipe.
func defaultCaptionEmbedHosts() map[string]bool {
	return map[string]bool{"instagram.com": true}
}

// shortLinkHosts redirect to a canonical URL that the oEmbed providers accept.
var shortLinkHosts = map[string]bool{
	"vm.tiktok.com": true,
	"vt.tiktok.com": true,
}

// crawlerUserAgentHosts are served Open Graph tags only when the request looks
// like a social crawler.
var crawlerUserAgentHosts = map[string]bool{
	"instagram.com": true,
	"tiktok.com":    true,
	"facebook.com":  true,
	"fb.watch":      true,
	"threads.net":   true,
	"threads.com":   true,
	"pinterest.com": true,
	"pin.it":        true,
}

// hostOf returns the lowercased hostname of rawURL without any "www." prefix.
func hostOf(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(parsed.Hostname()), "www.")
}

// registrableHost trims subdomains down to the last two labels, so
// "m.tiktok.com" matches a "tiktok.com" provider entry.
func registrableHost(host string) string {
	labels := strings.Split(host, ".")
	if len(labels) <= 2 {
		return host
	}
	return strings.Join(labels[len(labels)-2:], ".")
}

func userAgentForURL(rawURL string) string {
	host := hostOf(rawURL)
	if crawlerUserAgentHosts[host] || crawlerUserAgentHosts[registrableHost(host)] {
		return crawlerUserAgent
	}
	return defaultUserAgent
}

// --- oEmbed ---

type oembedResponse struct {
	Title        string `json:"title"`
	AuthorName   string `json:"author_name"`
	ThumbnailURL string `json:"thumbnail_url"`
}

func (extractor *RecipeExtractor) oembedEndpointFor(rawURL string) (string, bool) {
	host := hostOf(rawURL)
	if endpoint, ok := extractor.oembedProviders[host]; ok {
		return endpoint, true
	}
	endpoint, ok := extractor.oembedProviders[registrableHost(host)]
	return endpoint, ok
}

// extractViaOEmbed asks the platform for the post's caption and thumbnail. The
// caption doubles as the recipe body on TikTok, so it is run through the same
// caption parser as an Open Graph description.
func (extractor *RecipeExtractor) extractViaOEmbed(ctx context.Context, endpoint, rawURL string) (ExtractedRecipe, bool) {
	requestURL := endpoint + "?url=" + url.QueryEscape(rawURL)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return ExtractedRecipe{}, false
	}
	request.Header.Set("User-Agent", defaultUserAgent)
	request.Header.Set("Accept", "application/json")

	response, err := extractor.client.Do(request)
	if err != nil {
		return ExtractedRecipe{}, false
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ExtractedRecipe{}, false
	}

	var payload oembedResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, 512*1024)).Decode(&payload); err != nil {
		return ExtractedRecipe{}, false
	}

	caption := cleanSocialCaption(payload.Title)
	if caption == "" && payload.ThumbnailURL == "" {
		return ExtractedRecipe{}, false
	}

	recipe := ExtractedRecipe{ImageURL: payload.ThumbnailURL}
	applyCaption(&recipe, caption)
	if recipe.Title == "" {
		recipe.Title = payload.AuthorName
	}

	return recipe, recipe.Title != "" || recipe.ImageURL != ""
}

// resolveShortLink follows a share-sheet short link to its canonical URL so the
// oEmbed providers recognise it.
func (extractor *RecipeExtractor) resolveShortLink(ctx context.Context, rawURL string) string {
	if !isShortLink(rawURL) {
		return rawURL
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return rawURL
	}
	request.Header.Set("User-Agent", userAgentForURL(rawURL))

	response, err := extractor.client.Do(request)
	if err != nil {
		return rawURL
	}
	defer response.Body.Close()
	io.Copy(io.Discard, io.LimitReader(response.Body, 4096))

	if response.Request == nil || response.Request.URL == nil {
		return rawURL
	}
	resolved := response.Request.URL.String()
	if err := extractor.validateURL(resolved); err != nil {
		return rawURL
	}
	return resolved
}

// isShortLink reports whether a URL needs following before a platform will
// recognise it: the share-sheet hosts, plus TikTok's own /t/ short paths.
func isShortLink(rawURL string) bool {
	host := hostOf(rawURL)
	if shortLinkHosts[host] {
		return true
	}
	if registrableHost(host) != "tiktok.com" {
		return false
	}
	parsed, err := url.Parse(rawURL)
	return err == nil && strings.HasPrefix(parsed.Path, "/t/")
}

// --- Open Graph / meta tags ---

type pageMetadata struct {
	Title       string
	ImageURL    string
	Description string
}

// extractMetadata reads the Open Graph / Twitter card tags, falling back to the
// document title. This is the "if nothing else, find a title and an image" path
// and applies to every site, not just social ones.
func extractMetadata(document *html.Node) pageMetadata {
	properties := map[string]string{}
	var documentTitle string

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch node.Data {
			case "meta":
				key := getAttr(node, "property")
				if key == "" {
					key = getAttr(node, "name")
				}
				key = strings.ToLower(strings.TrimSpace(key))
				content := strings.TrimSpace(getAttr(node, "content"))
				if key != "" && content != "" {
					if _, exists := properties[key]; !exists {
						properties[key] = content
					}
				}
			case "title":
				if documentTitle == "" {
					documentTitle = textContent(node)
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(document)

	first := func(keys ...string) string {
		for _, key := range keys {
			if value := properties[key]; value != "" {
				return htmlDecode(value)
			}
		}
		return ""
	}

	title := first("og:title", "twitter:title")
	if title == "" {
		title = strings.TrimSpace(htmlDecode(documentTitle))
	}

	return pageMetadata{
		Title:       title,
		ImageURL:    first("og:image", "og:image:secure_url", "og:image:url", "twitter:image", "twitter:image:src"),
		Description: first("og:description", "twitter:description", "description"),
	}
}

// resolveURL turns a possibly relative image URL into an absolute one.
func resolveURL(baseURL, reference string) string {
	if reference == "" {
		return ""
	}
	if strings.HasPrefix(reference, "http://") || strings.HasPrefix(reference, "https://") {
		return reference
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return reference
	}
	resolved, err := base.Parse(reference)
	if err != nil {
		return reference
	}
	return resolved.String()
}

// --- Video JSON-LD ---

// extractFromVideoJSONLD handles pages whose only structured data describes a
// video (common on TikTok, Instagram Reels and YouTube).
func extractFromVideoJSONLD(document *html.Node) (ExtractedRecipe, bool) {
	var scripts []string
	findJSONLDScripts(document, &scripts)

	for _, scriptContent := range scripts {
		var raw any
		if err := json.Unmarshal([]byte(scriptContent), &raw); err != nil {
			continue
		}
		video, found := findObjectOfType(raw, "VideoObject")
		if !found {
			continue
		}

		recipe := ExtractedRecipe{
			Title:    htmlDecode(stringField(video, "name")),
			ImageURL: videoThumbnail(video),
		}
		if description := stringField(video, "description"); description != "" {
			applyCaption(&recipe, cleanSocialCaption(description))
		}
		if recipe.Title != "" || recipe.ImageURL != "" {
			return recipe, true
		}
	}

	return ExtractedRecipe{}, false
}

func videoThumbnail(video map[string]any) string {
	if thumbnail := extractImageURL(map[string]any{"image": video["thumbnailUrl"]}); thumbnail != "" {
		return thumbnail
	}
	return extractImageURL(video)
}

func findObjectOfType(data any, wanted string) (map[string]any, bool) {
	switch value := data.(type) {
	case map[string]any:
		if objectHasType(value, wanted) {
			return value, true
		}
		if graph, ok := value["@graph"].([]any); ok {
			for _, item := range graph {
				if found, ok := findObjectOfType(item, wanted); ok {
					return found, true
				}
			}
		}
	case []any:
		for _, item := range value {
			if found, ok := findObjectOfType(item, wanted); ok {
				return found, true
			}
		}
	}
	return nil, false
}

func objectHasType(dict map[string]any, wanted string) bool {
	switch typeValue := dict["@type"].(type) {
	case string:
		return typeValue == wanted
	case []any:
		for _, item := range typeValue {
			if str, ok := item.(string); ok && str == wanted {
				return true
			}
		}
	}
	return false
}

// --- Instagram embed fallback ---

// Instagram's post page is a login wall for signed-out clients, but its embed
// page is server rendered and public. It carries the caption and the poster
// image, which is everything we can hope for from a Reel.
func (extractor *RecipeExtractor) extractViaCaptionEmbed(ctx context.Context, rawURL string) (ExtractedRecipe, bool) {
	host := hostOf(rawURL)
	if !extractor.captionEmbedHosts[host] && !extractor.captionEmbedHosts[registrableHost(host)] {
		return ExtractedRecipe{}, false
	}

	embedURL, ok := captionEmbedURL(rawURL)
	if !ok {
		return ExtractedRecipe{}, false
	}
	if err := extractor.validateURL(embedURL); err != nil {
		return ExtractedRecipe{}, false
	}

	document, _, err := extractor.fetchDocument(ctx, embedURL)
	if err != nil {
		return ExtractedRecipe{}, false
	}

	return parseCaptionEmbed(document)
}

var postPathPattern = regexp.MustCompile(`^/(?:p|reel|reels|tv)/[^/]+`)

// captionEmbedURL turns ".../p/ABC123/" into ".../p/ABC123/embed/captioned/".
func captionEmbedURL(rawURL string) (string, bool) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}

	postPath := postPathPattern.FindString(parsed.Path)
	if postPath == "" {
		return "", false
	}

	parsed.Path = postPath + "/embed/captioned/"
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String(), true
}

func parseCaptionEmbed(document *html.Node) (ExtractedRecipe, bool) {
	recipe := ExtractedRecipe{ImageURL: embeddedMediaImage(document)}

	if caption := embeddedCaption(document); caption != "" {
		applyCaption(&recipe, cleanSocialCaption(caption))
	}
	if recipe.Title == "" && recipe.ImageURL == "" {
		return ExtractedRecipe{}, false
	}
	return recipe, true
}

func embeddedMediaImage(document *html.Node) string {
	var fallback string
	var found string

	forEachElement(document, func(node *html.Node) {
		if found != "" || node.Data != "img" {
			return
		}
		source := getAttr(node, "src")
		if source == "" {
			return
		}
		if strings.Contains(getAttr(node, "class"), "EmbeddedMediaImage") {
			found = source
			return
		}
		if fallback == "" && (strings.Contains(source, "cdninstagram") || strings.Contains(source, "fbcdn")) {
			fallback = source
		}
	})

	if found != "" {
		return found
	}
	return fallback
}

func embeddedCaption(document *html.Node) string {
	var caption string
	var username string

	forEachElement(document, func(node *html.Node) {
		class := getAttr(node, "class")
		switch {
		case username == "" && strings.Contains(class, "CaptionUsername"):
			username = textContent(node)
		case caption == "" && strings.Contains(class, "Caption") && !strings.Contains(class, "CaptionUsername"):
			caption = textWithLineBreaks(node)
		}
	})

	caption = strings.TrimSpace(caption)
	if username != "" {
		caption = strings.TrimSpace(strings.TrimPrefix(caption, strings.TrimSpace(username)))
	}
	return caption
}

func forEachElement(node *html.Node, visit func(*html.Node)) {
	if node.Type == html.ElementNode {
		visit(node)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		forEachElement(child, visit)
	}
}

// textWithLineBreaks preserves the line structure of a caption, which is what
// separates one ingredient from the next.
func textWithLineBreaks(node *html.Node) string {
	var builder strings.Builder

	var collect func(*html.Node)
	collect = func(current *html.Node) {
		if current.Type == html.TextNode {
			builder.WriteString(current.Data)
		}
		if current.Type == html.ElementNode {
			switch current.Data {
			case "br", "p", "div", "li":
				builder.WriteString("\n")
			}
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			collect(child)
		}
	}
	collect(node)

	return strings.TrimSpace(builder.String())
}

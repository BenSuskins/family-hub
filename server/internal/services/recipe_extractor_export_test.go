package services

import "net/http"

// NewRecipeExtractorForTest builds a RecipeExtractor wired for use with
// httptest servers: it skips URL validation (so loopback addresses work)
// and uses the supplied HTTP client. This file has the _test.go suffix
// so the symbol is only compiled into test binaries.
func NewRecipeExtractorForTest(client *http.Client) *RecipeExtractor {
	return &RecipeExtractor{
		client:      client,
		validateURL: func(string) error { return nil },
	}
}

// WithOEmbedProvider points the extractor at a stub oEmbed endpoint for the
// given host, so the TikTok path can be exercised without leaving the test.
func (extractor *RecipeExtractor) WithOEmbedProvider(host, endpoint string) *RecipeExtractor {
	if extractor.oembedProviders == nil {
		extractor.oembedProviders = map[string]string{}
	}
	extractor.oembedProviders[host] = endpoint
	return extractor
}

// WithCaptionEmbedOrigin enables the Instagram-style embed fallback for a stub
// host, pointing it at a stub origin (which must end in "/"), so the fallback
// can be exercised without leaving the test.
func (extractor *RecipeExtractor) WithCaptionEmbedOrigin(host, origin string) *RecipeExtractor {
	if extractor.captionEmbedOrigins == nil {
		extractor.captionEmbedOrigins = map[string]string{}
	}
	extractor.captionEmbedOrigins[host] = origin
	return extractor
}

// ShortLinkRequestURLForTest exposes the short-link canonicaliser so its
// host-pinning can be asserted directly.
func ShortLinkRequestURLForTest(rawURL string) (string, bool) {
	return shortLinkRequestURL(rawURL)
}

// CaptionEmbedPathForTest exposes the embed-path builder so its post-id
// matching can be asserted directly.
func CaptionEmbedPathForTest(rawURL string) (string, bool) {
	return captionEmbedPath(rawURL)
}

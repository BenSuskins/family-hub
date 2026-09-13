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

// WithCaptionEmbedHost enables the Instagram-style embed fallback for a stub
// host, so the fallback can be exercised without leaving the test.
func (extractor *RecipeExtractor) WithCaptionEmbedHost(host string) *RecipeExtractor {
	if extractor.captionEmbedHosts == nil {
		extractor.captionEmbedHosts = map[string]bool{}
	}
	extractor.captionEmbedHosts[host] = true
	return extractor
}

// ShortLinkRequestURLForTest exposes the short-link canonicaliser so its
// host-pinning can be asserted directly.
func ShortLinkRequestURLForTest(rawURL string) (string, bool) {
	return shortLinkRequestURL(rawURL)
}

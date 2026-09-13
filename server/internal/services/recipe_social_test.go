package services_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/bensuskins/family-hub/internal/services"
)

const tiktokOEmbedBody = `{
	"title": "Smash Burger Tacos\nIngredients:\n500g beef mince\n6 mini tortillas\nMethod:\n1. Press the mince onto the tortilla.\n2. Fry for 3 minutes.",
	"author_name": "tacoguy",
	"thumbnail_url": "https://cdn.example.com/thumb.jpg"
}`

func TestRecipeExtractor_OEmbedFallback(t *testing.T) {
	tests := []struct {
		name     string
		pageBody string
		status   int
	}{
		{
			name:   "platform blocks the scrape",
			status: http.StatusForbidden,
		},
		{
			name:     "platform serves a login wall with no metadata",
			status:   http.StatusOK,
			pageBody: `<html><head><title>TikTok</title></head><body>Log in to continue</body></html>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var oembedRequests []string

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/oembed" {
					oembedRequests = append(oembedRequests, r.URL.Query().Get("url"))
					w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(tiktokOEmbedBody))
					return
				}
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.pageBody))
			}))
			defer server.Close()

			host := hostOfTestServer(t, server.URL)
			extractor := services.NewRecipeExtractorForTest(server.Client()).
				WithOEmbedProvider(host, server.URL+"/oembed")

			videoURL := server.URL + "/@tacoguy/video/123"
			got, err := extractor.Extract(context.Background(), videoURL)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(oembedRequests) != 1 || oembedRequests[0] != videoURL {
				t.Fatalf("oEmbed requests = %v, want exactly [%s]", oembedRequests, videoURL)
			}

			assertStringEqual(t, "Title", got.Title, "Smash Burger Tacos")
			assertStringSliceEqual(t, "Ingredients", got.Ingredients, []string{"500g beef mince", "6 mini tortillas"})
			assertStringSliceEqual(t, "Steps", got.Steps, []string{"Press the mince onto the tortilla.", "Fry for 3 minutes."})
			assertStringEqual(t, "ImageURL", got.ImageURL, "https://cdn.example.com/thumb.jpg")
		})
	}
}

// A caption that is only hype still has to yield a title and an image, since
// that is all the user needs before filling in the rest by hand.
func TestRecipeExtractor_TitleAndImageOnly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oembed" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"title":"the BEST garlic butter prawns you will ever make #seafood","thumbnail_url":"https://cdn.example.com/prawns.jpg"}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	extractor := services.NewRecipeExtractorForTest(server.Client()).
		WithOEmbedProvider(hostOfTestServer(t, server.URL), server.URL+"/oembed")

	got, err := extractor.Extract(context.Background(), server.URL+"/@cook/video/9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertStringEqual(t, "Title", got.Title, "the BEST garlic butter prawns you will ever make")
	assertStringEqual(t, "ImageURL", got.ImageURL, "https://cdn.example.com/prawns.jpg")
	if len(got.Ingredients) != 0 || len(got.Steps) != 0 {
		t.Errorf("expected no ingredients or steps, got %v / %v", got.Ingredients, got.Steps)
	}
}

func TestRecipeExtractor_ResolvesRelativeImageURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<html><head>
			<meta property="og:title" content="Toad in the Hole">
			<meta property="og:image" content="/media/toad.jpg">
		</head><body></body></html>`))
	}))
	defer server.Close()

	extractor := services.NewRecipeExtractorForTest(server.Client())
	got, err := extractor.Extract(context.Background(), server.URL+"/recipes/toad")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := server.URL + "/media/toad.jpg"
	assertStringEqual(t, "ImageURL", got.ImageURL, want)
	if !strings.HasPrefix(got.ImageURL, "http") {
		t.Errorf("ImageURL %q is not absolute", got.ImageURL)
	}
}

func hostOfTestServer(t *testing.T, rawURL string) string {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parsing test server URL: %v", err)
	}
	return parsed.Hostname()
}

// Instagram serves signed-out clients a login wall, so the extractor falls back
// to the public embed page for the caption and the poster image.
func TestRecipeExtractor_InstagramEmbedFallback(t *testing.T) {
	const embedPage = `<html><body>
		<div class="EmbedFrame">
			<img class="EmbeddedMediaImage" src="https://scontent.cdninstagram.com/v/reel.jpg">
			<div class="Caption">
				<a class="CaptionUsername">weeknight_dinners</a>
				Chilli Paneer<br>Serves 2<br>
				Ingredients:<br>250g paneer<br>2 tbsp cornflour<br>1 green chilli<br>
				Method:<br>1. Toss the paneer in cornflour.<br>2. Fry until golden.<br>
				<a href="#">#paneer</a>
			</div>
		</div>
	</body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		if strings.HasSuffix(r.URL.Path, "/embed/captioned/") {
			w.Write([]byte(embedPage))
			return
		}
		w.Write([]byte(`<html><head><title>Instagram</title></head><body>Log in to continue</body></html>`))
	}))
	defer server.Close()

	extractor := services.NewRecipeExtractorForTest(server.Client()).
		WithCaptionEmbedOrigin(hostOfTestServer(t, server.URL), server.URL+"/")

	got, err := extractor.Extract(context.Background(), server.URL+"/reel/ABC123/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertStringEqual(t, "Title", got.Title, "Chilli Paneer")
	assertStringSliceEqual(t, "Ingredients", got.Ingredients, []string{"250g paneer", "2 tbsp cornflour", "1 green chilli"})
	assertStringSliceEqual(t, "Steps", got.Steps, []string{"Toss the paneer in cornflour.", "Fry until golden."})
	assertStringEqual(t, "ImageURL", got.ImageURL, "https://scontent.cdninstagram.com/v/reel.jpg")
	assertIntPtrEqual(t, "Servings", got.Servings, intPtr(2))
}

// The short-link lookup rebuilds the URL against a constant origin, so a
// crafted link cannot steer the request at another host, port or scheme.
func TestShortLinkRequestURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{name: "vm short link", input: "https://vm.tiktok.com/ZGeAbCdEf/", want: "https://vm.tiktok.com/ZGeAbCdEf/", ok: true},
		{name: "vt short link", input: "https://vt.tiktok.com/ZSxYz/", want: "https://vt.tiktok.com/ZSxYz/", ok: true},
		{name: "tiktok /t/ path", input: "https://www.tiktok.com/t/ZTabc/", want: "https://www.tiktok.com/t/ZTabc/", ok: true},
		{name: "query string is dropped", input: "https://vm.tiktok.com/ZGe/?redirect=http://169.254.169.254/", want: "https://vm.tiktok.com/ZGe/", ok: true},
		{name: "credentials are dropped", input: "https://evil.example.com@vm.tiktok.com/ZGe/", want: "https://vm.tiktok.com/ZGe/", ok: true},
		{name: "port is dropped", input: "https://vm.tiktok.com:8080/ZGe/", want: "https://vm.tiktok.com/ZGe/", ok: true},
		{name: "scheme is forced to https", input: "http://vm.tiktok.com/ZGe/", want: "https://vm.tiktok.com/ZGe/", ok: true},
		{name: "lookalike host is rejected", input: "https://vm.tiktok.com.evil.example.com/ZGe/", ok: false},
		{name: "userinfo lookalike is rejected", input: "https://vm.tiktok.com@evil.example.com/ZGe/", ok: false},
		{name: "full video URL is not a short link", input: "https://www.tiktok.com/@cook/video/123", ok: false},
		{name: "other host is not a short link", input: "https://example.com/recipe", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := services.ShortLinkRequestURLForTest(tt.input)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v (got %q)", ok, tt.ok, got)
			}
			if ok {
				assertStringEqual(t, "URL", got, tt.want)
			}
		})
	}
}

// The embed path is appended to a constant origin, so the only part of a
// shared link that reaches the request is a strictly matched post id.
func TestCaptionEmbedPath(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
		ok    bool
	}{
		{name: "post", input: "https://www.instagram.com/p/ABC123/", want: "p/ABC123/embed/captioned/", ok: true},
		{name: "reel", input: "https://www.instagram.com/reel/A-b_c9/", want: "reel/A-b_c9/embed/captioned/", ok: true},
		{name: "tv", input: "https://www.instagram.com/tv/ABC123/", want: "tv/ABC123/embed/captioned/", ok: true},
		{name: "no trailing slash", input: "https://www.instagram.com/p/ABC123", want: "p/ABC123/embed/captioned/", ok: true},
		{name: "tracking query is dropped", input: "https://www.instagram.com/p/ABC123/?igsh=abc123&img_index=2", want: "p/ABC123/embed/captioned/", ok: true},
		{name: "encoded traversal is rejected", input: "https://www.instagram.com/p/..%2F..%2Fadmin/", ok: false},
		{name: "plain traversal is rejected", input: "https://www.instagram.com/p/../../etc/passwd", ok: false},
		{name: "over-long post id is rejected", input: "https://www.instagram.com/p/" + strings.Repeat("A", 65) + "/", ok: false},
		{name: "empty post id is rejected", input: "https://www.instagram.com/p//", ok: false},
		{name: "profile page is rejected", input: "https://www.instagram.com/somecook/", ok: false},
		{name: "explore page is rejected", input: "https://www.instagram.com/explore/tags/food/", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := services.CaptionEmbedPathForTest(tt.input)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v (got %q)", ok, tt.ok, got)
			}
			if ok {
				assertStringEqual(t, "path", got, tt.want)
			}
		})
	}
}

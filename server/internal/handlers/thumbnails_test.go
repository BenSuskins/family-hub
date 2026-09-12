package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bensuskins/family-hub/internal/models"
	"github.com/bensuskins/family-hub/internal/repository"
	"github.com/bensuskins/family-hub/internal/testutil"
	"github.com/go-chi/chi/v5"
)

// pngDataURI renders a solid PNG of the given size and returns it as the
// `data:` URI the repository stores.
func pngDataURI(t *testing.T, width, height int) string {
	t.Helper()

	source := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			source.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 120, A: 255})
		}
	}

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, source); err != nil {
		t.Fatalf("encoding fixture png: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes())
}

// serveRecipeImage seeds a recipe carrying dataURI and performs a GET against
// ServeImage with the given query string and headers.
func serveRecipeImage(t *testing.T, dataURI, query string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()

	database := testutil.NewTestDatabase(t)
	recipeRepo := repository.NewRecipeRepository(database)
	userRepo := repository.NewUserRepository(database)
	ctx := context.Background()

	user, err := userRepo.Create(ctx, models.User{
		OIDCSubject: "sub-image",
		Email:       "image@example.com",
		Name:        "Image User",
		Role:        models.RoleMember,
	})
	if err != nil {
		t.Fatalf("creating user: %v", err)
	}

	created, err := recipeRepo.Create(ctx, models.Recipe{Title: "Photo", CreatedByUserID: user.ID})
	if err != nil {
		t.Fatalf("creating recipe: %v", err)
	}
	if err := recipeRepo.UpdateImage(ctx, created.ID, dataURI); err != nil {
		t.Fatalf("storing image: %v", err)
	}

	handler := NewRecipeHandler(recipeRepo, nil, nil, nil)
	router := chi.NewRouter()
	router.Get("/api/recipes/{id}/image", handler.ServeImage)

	request := httptest.NewRequest(http.MethodGet, "/api/recipes/"+created.ID+"/image"+query, nil)
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func TestServeImage_FullSetsValidatorAndCacheHeaders(t *testing.T) {
	recorder := serveRecipeImage(t, pngDataURI(t, 400, 300), "", nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("ETag"); got == "" {
		t.Error("expected an ETag so clients can revalidate")
	}
	if got := recorder.Header().Get("Cache-Control"); got != "private, max-age=86400" {
		t.Errorf("got Cache-Control %q, want private, max-age=86400", got)
	}
	if got := recorder.Header().Get("Content-Type"); got != "image/png" {
		t.Errorf("got Content-Type %q, want image/png", got)
	}
}

func TestServeImage_ThumbIsSmallerAndBounded(t *testing.T) {
	dataURI := pngDataURI(t, 400, 300)

	full := serveRecipeImage(t, dataURI, "", nil)
	thumb := serveRecipeImage(t, dataURI, "?size=thumb", nil)

	if thumb.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", thumb.Code)
	}

	decoded, _, err := image.Decode(bytes.NewReader(thumb.Body.Bytes()))
	if err != nil {
		t.Fatalf("decoding thumbnail: %v", err)
	}
	bounds := decoded.Bounds()
	if bounds.Dx() != thumbMaxEdge {
		t.Errorf("got width %d, want %d for a landscape source", bounds.Dx(), thumbMaxEdge)
	}
	if bounds.Dy() != 150 {
		t.Errorf("got height %d, want 150 (aspect ratio preserved)", bounds.Dy())
	}
	if thumb.Body.Len() >= full.Body.Len() {
		t.Errorf("thumbnail (%d bytes) should be smaller than the original (%d bytes)",
			thumb.Body.Len(), full.Body.Len())
	}
}

func TestServeImage_ThumbAndFullHaveDifferentETags(t *testing.T) {
	dataURI := pngDataURI(t, 400, 300)

	full := serveRecipeImage(t, dataURI, "", nil).Header().Get("ETag")
	thumb := serveRecipeImage(t, dataURI, "?size=thumb", nil).Header().Get("ETag")

	if full == thumb {
		t.Errorf("both variants returned ETag %s; a client would serve the wrong bytes", full)
	}
}

func TestServeImage_MatchingETagReturns304(t *testing.T) {
	dataURI := pngDataURI(t, 400, 300)
	etag := serveRecipeImage(t, dataURI, "", nil).Header().Get("ETag")

	tests := []struct {
		name         string
		ifNoneMatch  string
		wantNotModif bool
	}{
		{"exact match", etag, true},
		{"weak validator", "W/" + etag, true},
		{"wildcard", "*", true},
		{"within a list", `"other", ` + etag, true},
		{"different etag", `"nope"`, false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := serveRecipeImage(t, dataURI, "", map[string]string{
				"If-None-Match": testCase.ifNoneMatch,
			})

			if testCase.wantNotModif {
				if recorder.Code != http.StatusNotModified {
					t.Fatalf("got %d, want 304", recorder.Code)
				}
				if recorder.Body.Len() != 0 {
					t.Errorf("a 304 must not carry a body, got %d bytes", recorder.Body.Len())
				}
				return
			}
			if recorder.Code != http.StatusOK {
				t.Fatalf("got %d, want 200", recorder.Code)
			}
		})
	}
}

func TestServeImage_UnknownSizeFallsBackToFull(t *testing.T) {
	dataURI := pngDataURI(t, 400, 300)

	full := serveRecipeImage(t, dataURI, "", nil)
	odd := serveRecipeImage(t, dataURI, "?size=banana", nil)

	if odd.Code != http.StatusOK {
		t.Fatalf("got %d, want 200", odd.Code)
	}
	if odd.Body.Len() != full.Body.Len() {
		t.Errorf("got %d bytes, want the full image's %d", odd.Body.Len(), full.Body.Len())
	}
}

func TestScaledBounds(t *testing.T) {
	tests := []struct {
		name                   string
		width, height, maxEdge int
		wantWidth, wantHeight  int
	}{
		{"landscape", 400, 300, 200, 200, 150},
		{"portrait", 300, 400, 200, 150, 200},
		{"square", 400, 400, 200, 200, 200},
		{"never upscales", 80, 60, 200, 80, 60},
		{"exactly at the limit", 200, 200, 200, 200, 200},
		{"extreme ratio keeps a pixel", 4000, 3, 200, 200, 1},
		{"degenerate", 0, 0, 200, 1, 1},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			width, height := scaledBounds(testCase.width, testCase.height, testCase.maxEdge)
			if width != testCase.wantWidth || height != testCase.wantHeight {
				t.Errorf("got %dx%d, want %dx%d", width, height, testCase.wantWidth, testCase.wantHeight)
			}
		})
	}
}

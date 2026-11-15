package handlers

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type recipeImageReader interface {
	FindImageData(context.Context, string) (string, error)
}
type ImageHandler struct {
	recipeRepo recipeImageReader
}

func NewImageHandler(recipeRepo recipeImageReader) *ImageHandler {
	return &ImageHandler{recipeRepo: recipeRepo}
}

func (handler *ImageHandler) ServeRecipeImage(w http.ResponseWriter, r *http.Request) {
	imageData, err := handler.recipeRepo.FindImageData(r.Context(), chi.URLParam(r, "id"))
	if err != nil || imageData == "" {
		writeJSONError(w, http.StatusNotFound, "recipe image not found")
		return
	}

	imageBytes, ok := decodeDataURI(imageData)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "recipe image not found")
		return
	}

	mimeType, ok := detectImageContentType(imageBytes)
	if !ok {
		writeJSONError(w, http.StatusNotFound, "recipe image not found")
		return
	}

	variant := parseImageVariant(r.URL.Query().Get("size"))
	etag := imageETag(imageBytes, variant)
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "private, max-age=86400")

	if etagMatches(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}

	if variant == imageVariantThumb {
		if thumbBytes, thumbType, rendered := thumbnailFor(etag, imageBytes); rendered {
			imageBytes, mimeType = thumbBytes, thumbType
		}
	}

	w.Header().Set("Content-Type", mimeType)
	_, _ = w.Write(imageBytes)
}

package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"strings"
	"sync"

	"golang.org/x/image/draw"

	// Decoder registrations for image.Decode. JPEG and PNG are also used for
	// encoding below; GIF and WebP are decode-only and re-encode as JPEG.
	_ "image/gif"

	_ "golang.org/x/image/webp"
)

// thumbMaxEdge is the longest edge, in pixels, of a `?size=thumb` variant.
// Sized for a recipe row on a phone or a watch, not for a hero image.
const thumbMaxEdge = 200

// thumbJPEGQuality trades a little fidelity for a much smaller payload; these
// are only ever drawn at list-row size.
const thumbJPEGQuality = 80

type imageVariant string

const (
	imageVariantFull  imageVariant = "full"
	imageVariantThumb imageVariant = "thumb"
)

// parseImageVariant maps the `size` query parameter onto a variant. Anything
// unrecognised falls back to the full image, so an old client or a typo still
// gets a usable response.
func parseImageVariant(size string) imageVariant {
	if strings.EqualFold(size, string(imageVariantThumb)) {
		return imageVariantThumb
	}
	return imageVariantFull
}

// imageETag derives a strong validator from the stored bytes and the variant,
// so the full image and its thumbnail never share a cache entry.
func imageETag(imageBytes []byte, variant imageVariant) string {
	sum := sha256.Sum256(imageBytes)
	return `"` + string(variant) + "-" + hex.EncodeToString(sum[:16]) + `"`
}

// etagMatches reports whether the request's If-None-Match header already covers
// etag, in which case the caller should answer 304.
func etagMatches(r *http.Request, etag string) bool {
	header := r.Header.Get("If-None-Match")
	if header == "" {
		return false
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		if candidate == "*" {
			return true
		}
		// A weak validator still identifies the same bytes here, since the tag
		// is a content hash.
		if strings.TrimPrefix(candidate, "W/") == etag {
			return true
		}
	}
	return false
}

// thumbnailCache memoises rendered thumbnails by ETag. Rendering is the only
// expensive part of serving an image, and a family hub has tens of recipes, so
// a small map is enough. When it fills it is dropped wholesale rather than
// evicted one entry at a time — crude, but predictable and allocation-free.
type thumbnailCache struct {
	mutex   sync.Mutex
	entries map[string]thumbnail
	limit   int
}

type thumbnail struct {
	bytes       []byte
	contentType string
}

var thumbnails = &thumbnailCache{entries: map[string]thumbnail{}, limit: 128}

func (cache *thumbnailCache) get(key string) (thumbnail, bool) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	entry, ok := cache.entries[key]
	return entry, ok
}

func (cache *thumbnailCache) put(key string, entry thumbnail) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	if len(cache.entries) >= cache.limit {
		clear(cache.entries)
	}
	cache.entries[key] = entry
}

// thumbnailFor renders (or returns a cached) downscaled copy of imageBytes.
// It reports ok=false when the image cannot be decoded, so callers can fall
// back to serving the original rather than 404-ing a perfectly good image.
func thumbnailFor(cacheKey string, imageBytes []byte) (thumbBytes []byte, contentType string, ok bool) {
	if entry, hit := thumbnails.get(cacheKey); hit {
		return entry.bytes, entry.contentType, true
	}

	thumbBytes, contentType, ok = renderThumbnail(imageBytes)
	if !ok {
		return nil, "", false
	}
	thumbnails.put(cacheKey, thumbnail{bytes: thumbBytes, contentType: contentType})
	return thumbBytes, contentType, true
}

// renderThumbnail decodes, downscales and re-encodes an image. PNGs stay PNGs
// so transparency survives; everything else (JPEG, GIF, WebP) becomes JPEG.
func renderThumbnail(imageBytes []byte) (thumbBytes []byte, contentType string, ok bool) {
	source, format, err := image.Decode(bytes.NewReader(imageBytes))
	if err != nil {
		return nil, "", false
	}

	bounds := source.Bounds()
	width, height := scaledBounds(bounds.Dx(), bounds.Dy(), thumbMaxEdge)

	scaled := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(scaled, scaled.Bounds(), source, bounds, draw.Over, nil)

	var buffer bytes.Buffer
	if format == "png" {
		if err := png.Encode(&buffer, scaled); err != nil {
			return nil, "", false
		}
		return buffer.Bytes(), "image/png", true
	}
	if err := jpeg.Encode(&buffer, scaled, &jpeg.Options{Quality: thumbJPEGQuality}); err != nil {
		return nil, "", false
	}
	return buffer.Bytes(), "image/jpeg", true
}

// scaledBounds fits width x height inside a maxEdge square, preserving aspect
// ratio and never upscaling. Both dimensions stay at least 1px.
func scaledBounds(width, height, maxEdge int) (int, int) {
	if width <= 0 || height <= 0 {
		return 1, 1
	}
	if width <= maxEdge && height <= maxEdge {
		return width, height
	}
	if width >= height {
		height = height * maxEdge / width
		width = maxEdge
	} else {
		width = width * maxEdge / height
		height = maxEdge
	}
	return max(width, 1), max(height, 1)
}

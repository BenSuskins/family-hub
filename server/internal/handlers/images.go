package handlers

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"
)

var allowedImageContentTypes = map[string]struct{}{
	"image/png":  {},
	"image/jpeg": {},
	"image/gif":  {},
	"image/webp": {},
}

// detectImageContentType sniffs imageBytes and returns the MIME type only if
// it is in the allow-list. The client-supplied Content-Type is deliberately
// ignored — otherwise an attacker can upload HTML/JS under an image route and
// have the browser execute it (stored XSS).
func detectImageContentType(imageBytes []byte) (string, bool) {
	contentType := http.DetectContentType(imageBytes)
	if _, ok := allowedImageContentTypes[contentType]; !ok {
		return "", false
	}
	return contentType, true
}

func encodeDataURI(contentType string, imageBytes []byte) string {
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(imageBytes)
}

// decodeDataURI parses a `data:<mime>;base64,<payload>` URI and returns the
// decoded payload bytes. The stored MIME is intentionally discarded — callers
// must re-validate via detectImageContentType before serving.
func decodeDataURI(dataURI string) ([]byte, bool) {
	withoutPrefix, ok := strings.CutPrefix(dataURI, "data:")
	if !ok {
		return nil, false
	}
	parts := strings.SplitN(withoutPrefix, ";base64,", 2)
	if len(parts) != 2 {
		return nil, false
	}
	imageBytes, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, false
	}
	return imageBytes, true
}

// readUploadedImage reads the named multipart form file, enforces the size
// limit, validates the MIME via byte-sniffing, and writes an HTTP error on
// failure. Returns ok=false if the caller should stop.
func readUploadedImage(w http.ResponseWriter, r *http.Request, field string, maxBytes int) (imageBytes []byte, contentType string, ok bool) {
	file, _, err := r.FormFile(field)
	if err != nil {
		http.Error(w, "Missing "+field+" file", http.StatusBadRequest)
		return nil, "", false
	}
	defer file.Close()

	imageBytes, err = io.ReadAll(io.LimitReader(file, int64(maxBytes)+1))
	if err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return nil, "", false
	}
	if len(imageBytes) > maxBytes {
		http.Error(w, "Image exceeds size limit", http.StatusBadRequest)
		return nil, "", false
	}

	contentType, ok = detectImageContentType(imageBytes)
	if !ok {
		http.Error(w, "Unsupported image format", http.StatusBadRequest)
		return nil, "", false
	}
	return imageBytes, contentType, true
}

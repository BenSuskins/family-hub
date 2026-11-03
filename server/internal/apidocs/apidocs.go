package apidocs

import (
	"embed"
	"net/http"
)

//go:embed openapi.json assets/swagger-ui-bundle.js assets/swagger-ui.css
var files embed.FS

func Specification() []byte {
	data, _ := files.ReadFile("openapi.json")
	return data
}

func SpecificationHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(Specification())
}

func ExplorerHandler(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Write([]byte(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Family Hub API</title><link rel="stylesheet" href="/api/docs/swagger-ui.css"></head>
<body><div id="swagger-ui"></div><script src="/api/docs/swagger-ui-bundle.js"></script>
<script>SwaggerUIBundle({url: "/api/openapi.json", dom_id: "#swagger-ui", validatorUrl: null, persistAuthorization: false});</script></body></html>`))
}

func JavaScriptHandler(writer http.ResponseWriter, _ *http.Request) {
	serveAsset(writer, "assets/swagger-ui-bundle.js", "text/javascript; charset=utf-8")
}

func StylesheetHandler(writer http.ResponseWriter, _ *http.Request) {
	serveAsset(writer, "assets/swagger-ui.css", "text/css; charset=utf-8")
}

func serveAsset(writer http.ResponseWriter, path, contentType string) {
	data, _ := files.ReadFile(path)
	writer.Header().Set("Content-Type", contentType)
	writer.Write(data)
}

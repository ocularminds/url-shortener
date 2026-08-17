package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed static/index.html static/app.js static/style.css
var staticAssets embed.FS

var staticFiles = newStaticFileHandler()

func newStaticFileHandler() http.Handler {
	root, err := fs.Sub(staticAssets, "static")
	if err != nil {
		panic("embedded static assets are unavailable")
	}
	return http.FileServer(http.FS(root))
}

func serveAsset(response http.ResponseWriter, request *http.Request, contentType string, cacheControl string) {
	response.Header().Set("Content-Type", contentType)
	response.Header().Set("Cache-Control", cacheControl)
	staticFiles.ServeHTTP(response, request)
}

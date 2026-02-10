package httpapi

import (
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

func OpenAPISpec() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		path := openapiPath()
		data, err := os.ReadFile(path)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "openapi spec not found"})
			return
		}
		w.Header().Set("Content-Type", "text/yaml")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}

func SwaggerUI() http.HandlerFunc {
	const html = `<!doctype html>
<html>
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>LRSolid API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css" />
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.ui = SwaggerUIBundle({
      url: '/docs/openapi.yaml',
      dom_id: '#swagger-ui',
      presets: [SwaggerUIBundle.presets.apis],
      layout: 'BaseLayout'
    });
  </script>
</body>
</html>`

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}
}

func openapiPath() string {
	_, file, _, _ := runtime.Caller(0)
	base := filepath.Dir(file)
	return filepath.Join(base, "..", "..", "docs", "openapi.yaml")
}

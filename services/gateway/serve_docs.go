package main

import (
	_ "embed"
	"net/http"
)

//go:embed docs/swagger.json
var swaggerJSON []byte

// swaggerUIHTML is a minimal Swagger UI page loading from the CDN.
// It points at the locally-served /docs/openapi.json for offline-safe spec access.
const swaggerUIHTML = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8"/>
  <meta name="viewport" content="width=device-width, initial-scale=1"/>
  <title>Fieldstone API Docs</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css"/>
</head>
<body>
<div id="swagger-ui"></div>
<script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
<script>
  SwaggerUIBundle({
    url: "/docs/openapi.json",
    dom_id: "#swagger-ui",
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
    layout: "BaseLayout",
    deepLinking: true,
  });
</script>
</body>
</html>`

// registerDocRoutes mounts the OpenAPI spec and Swagger UI on the router.
//
//   GET /docs           → interactive Swagger UI (loads spec from /docs/openapi.json)
//   GET /docs/openapi.json → raw OpenAPI 2.0 spec (JSON)
func registerDocRoutes(mux interface {
	Get(string, http.HandlerFunc)
}) {
	mux.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(swaggerUIHTML))
	})
	mux.Get("/docs/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Write(swaggerJSON)
	})
}

package lambdahandler

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
)

// OpenAPI specification - embedded at build time
// Place openapi.yaml in the same directory as this file or update path
const openapiSpec = `# OpenAPI spec will be embedded here
# For now, the spec is served from the repository root
# TODO: Copy openapi.yaml to internal/lambdahandler/ or use a different approach
`

// handleDocs serves the Swagger UI documentation page
func (h *Handler) handleDocs(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <meta name="description" content="Gimage API - AI-powered image generation and processing" />
  <title>Gimage API Documentation</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.10.0/swagger-ui.css" />
  <style>
    body {
      margin: 0;
      padding: 0;
    }
    .topbar {
      display: none;
    }
    .swagger-ui .info .title {
      font-size: 36px;
    }
    .swagger-ui .info hgroup.main {
      margin: 0 0 20px 0;
    }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.10.0/swagger-ui-bundle.js" crossorigin></script>
  <script src="https://unpkg.com/swagger-ui-dist@5.10.0/swagger-ui-standalone-preset.js" crossorigin></script>
  <script>
    window.onload = () => {
      window.ui = SwaggerUIBundle({
        url: './openapi.yaml',
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        plugins: [
          SwaggerUIBundle.plugins.DownloadUrl
        ],
        layout: "StandaloneLayout",
        defaultModelsExpandDepth: 1,
        defaultModelExpandDepth: 1,
        displayRequestDuration: true,
        filter: true,
        syntaxHighlight: {
          activate: true,
          theme: "monokai"
        }
      });
    };
  </script>
</body>
</html>`

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                "text/html; charset=utf-8",
			"Access-Control-Allow-Origin": "*",
			"Cache-Control":               "public, max-age=3600",
		},
		Body: html,
	}, nil
}

// handleOpenAPISpec serves the OpenAPI specification YAML file
func (h *Handler) handleOpenAPISpec(ctx context.Context) (events.APIGatewayProxyResponse, error) {
	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                "application/yaml",
			"Access-Control-Allow-Origin": "*",
			"Cache-Control":               "public, max-age=3600",
		},
		Body: openapiSpec,
	}, nil
}

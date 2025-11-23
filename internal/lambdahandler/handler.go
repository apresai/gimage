package lambdahandler

import (
	"context"
	"fmt"

	"github.com/aws/aws-lambda-go/events"
	"github.com/apresai/gimage/internal/observability"
)

// Handler is the main Lambda handler
type Handler struct {
	s3Client *S3Client
	log      *observability.VerboseLogger
}

// NewHandler creates a new Lambda handler
func NewHandler() *Handler {
	return &Handler{
		log: observability.NewVerboseLogger(observability.ComponentLambda),
	}
}

// Handle processes an API Gateway proxy request
func (h *Handler) Handle(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Log the request
	h.log.Info("Received %s request to %s", req.HTTPMethod, req.Path)

	// Handle OPTIONS requests for CORS preflight
	if req.HTTPMethod == "OPTIONS" {
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers:    corsHeaders(),
		}, nil
	}

	// Initialize S3 client lazily
	if h.s3Client == nil {
		s3Client, err := NewS3Client(ctx)
		if err != nil {
			h.log.Error("Failed to create S3 client: %v", err)
			return errorResponse(500, fmt.Sprintf("Failed to initialize S3 client: %v", err)), nil
		}
		h.s3Client = s3Client
	}

	// Route the request
	routeKey := fmt.Sprintf("%s %s", req.HTTPMethod, req.Path)

	switch routeKey {
	case "POST /generate":
		return h.handleGenerate(ctx, []byte(req.Body))
	case "POST /resize":
		return h.handleResize(ctx, []byte(req.Body))
	case "POST /scale":
		return h.handleScale(ctx, []byte(req.Body))
	case "POST /crop":
		return h.handleCrop(ctx, []byte(req.Body))
	case "POST /compress":
		return h.handleCompress(ctx, []byte(req.Body))
	case "POST /convert":
		return h.handleConvert(ctx, []byte(req.Body))
	case "POST /batch":
		return h.handleBatch(ctx, []byte(req.Body))
	case "GET /health":
		return h.handleHealth(ctx)
	case "GET /docs":
		return h.handleDocs(ctx)
	case "GET /openapi.yaml":
		return h.handleOpenAPISpec(ctx)
	default:
		return errorResponse(404, fmt.Sprintf("Route not found: %s", routeKey)), nil
	}
}

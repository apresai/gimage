package lambdahandler

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHandler(t *testing.T) {
	h := NewHandler()
	require.NotNil(t, h)
}

func TestHandleOptions(t *testing.T) {
	h := NewHandler()
	req := events.APIGatewayProxyRequest{
		HTTPMethod: "OPTIONS",
		Path:       "/generate",
	}

	resp, err := h.Handle(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "*", resp.Headers["Access-Control-Allow-Origin"])
	assert.NotEmpty(t, resp.Headers["Access-Control-Allow-Methods"])
}

func TestHandleUnknownRoute(t *testing.T) {
	h := NewHandler()
	// Set S3_BUCKET so S3 client init doesn't fail before routing
	t.Setenv("S3_BUCKET", "test-bucket")

	req := events.APIGatewayProxyRequest{
		HTTPMethod: "GET",
		Path:       "/unknown",
	}

	resp, err := h.Handle(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)

	var errResp ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(resp.Body), &errResp))
	assert.Contains(t, errResp.Message, "Route not found")
}

func TestHandleRouting(t *testing.T) {
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"health endpoint", "GET", "/health"},
		{"generate endpoint", "POST", "/generate"},
		{"resize endpoint", "POST", "/resize"},
		{"docs endpoint", "GET", "/docs"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler()
			t.Setenv("S3_BUCKET", "test-bucket")

			req := events.APIGatewayProxyRequest{
				HTTPMethod: tt.method,
				Path:       tt.path,
			}

			resp, err := h.Handle(context.Background(), req)
			require.NoError(t, err)
			// Should NOT return 404 - routes are recognized
			// May return 500 due to missing AWS config/S3, which is expected
			assert.NotEqual(t, 404, resp.StatusCode, "route %s %s should be recognized", tt.method, tt.path)
		})
	}
}

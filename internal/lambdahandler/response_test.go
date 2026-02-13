package lambdahandler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccessResponse(t *testing.T) {
	resp := successResponse(200, map[string]string{"status": "ok"})

	assert.Equal(t, 200, resp.StatusCode)
	assert.Contains(t, resp.Body, `"status"`)
	assert.Contains(t, resp.Body, `"ok"`)

	// Verify CORS headers are present
	assert.Equal(t, "application/json", resp.Headers["Content-Type"])
	assert.Equal(t, "*", resp.Headers["Access-Control-Allow-Origin"])
}

func TestErrorResponse(t *testing.T) {
	tests := []struct {
		name           string
		code           int
		message        string
		expectedError  string
		expectedInBody string
	}{
		{
			name:           "bad request",
			code:           400,
			message:        "bad input",
			expectedError:  "Bad Request",
			expectedInBody: "bad input",
		},
		{
			name:           "not found",
			code:           404,
			message:        "not found",
			expectedError:  "Not Found",
			expectedInBody: "not found",
		},
		{
			name:           "server error",
			code:           500,
			message:        "server error",
			expectedError:  "Internal Server Error",
			expectedInBody: "server error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := errorResponse(tt.code, tt.message)

			assert.Equal(t, tt.code, resp.StatusCode)

			var errResp ErrorResponse
			require.NoError(t, json.Unmarshal([]byte(resp.Body), &errResp))
			assert.Equal(t, tt.expectedError, errResp.Error)
			assert.Contains(t, errResp.Message, tt.expectedInBody)
			assert.Equal(t, tt.code, errResp.Code)
		})
	}
}

func TestCorsHeaders(t *testing.T) {
	headers := corsHeaders()

	assert.Equal(t, "application/json", headers["Content-Type"])
	assert.Equal(t, "*", headers["Access-Control-Allow-Origin"])
	assert.NotEmpty(t, headers["Access-Control-Allow-Methods"])
	assert.NotEmpty(t, headers["Access-Control-Allow-Headers"])
}

func TestHttpStatusText(t *testing.T) {
	tests := []struct {
		code     int
		expected string
	}{
		{200, "OK"},
		{400, "Bad Request"},
		{404, "Not Found"},
		{500, "Internal Server Error"},
		{503, "Service Unavailable"},
		{999, "HTTP 999"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, httpStatusText(tt.code))
		})
	}
}

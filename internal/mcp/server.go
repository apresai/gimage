package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/apresai/gimage/internal/config"
	"github.com/apresai/gimage/internal/observability"
)

// MCPServer implements the Model Context Protocol server
type MCPServer struct {
	name    string
	version string
	config  *config.Config
	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	tools   map[string]Tool
	prompts map[string]Prompt
	verbose bool
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(name, version string, cfg *config.Config, verbose bool) *MCPServer {
	// Initialize structured logging
	observability.Initialize(verbose)

	return &MCPServer{
		name:    name,
		version: version,
		config:  cfg,
		stdin:   os.Stdin,
		stdout:  os.Stdout,
		stderr:  os.Stderr,
		tools:   make(map[string]Tool),
		prompts: make(map[string]Prompt),
		verbose: verbose,
	}
}

// RegisterTool adds a tool to the server
func (s *MCPServer) RegisterTool(tool Tool) {
	s.tools[tool.Name] = tool
	logger := observability.Logger(context.Background())
	logger.Debug().
		Str("component", "mcp-server").
		Str("tool", tool.Name).
		Msg("Registered tool")
}

// GetTool returns a tool by name, or nil if not found
func (s *MCPServer) GetTool(name string) *Tool {
	if tool, exists := s.tools[name]; exists {
		return &tool
	}
	return nil
}

// Start begins listening for MCP protocol messages
func (s *MCPServer) Start(ctx context.Context) error {
	logger := observability.LoggerWithComponent(ctx, "mcp-server")

	logger.Info().
		Str("protocol_version", ProtocolVersion).
		Int("tools_count", len(s.tools)).
		Msg("MCP server starting")

	scanner := bufio.NewScanner(s.stdin)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Server shutting down (context cancelled)")
			return ctx.Err()
		default:
		}

		line := scanner.Bytes()

		logger.Debug().
			Str("raw_request", string(line)).
			Msg("Received request")

		var request JSONRPCRequest
		if err := json.Unmarshal(line, &request); err != nil {
			logger.Error().
				Err(err).
				Str("raw_request", string(line)).
				Msg("Failed to parse request")
			continue
		}

		// Generate request ID for tracing
		requestID := observability.GenerateRequestID()
		requestCtx := observability.WithRequestID(ctx, requestID)
		reqLogger := observability.LoggerWithComponent(requestCtx, "mcp-server")

		// CRITICAL: Detect notifications vs requests
		// Notifications have NO id field and must NOT receive responses
		if request.ID == nil {
			reqLogger.Debug().
				Str("method", request.Method).
				Msg("Received notification (no response will be sent)")
			// Handle notification but DO NOT send response
			s.handleNotification(requestCtx, &request)
			continue
		}

		reqLogger.Debug().
			Str("method", request.Method).
			Interface("id", request.ID).
			Msg("Received request")

		// This is a request (has ID), send a response
		response := s.HandleRequest(requestCtx, &request)

		responseBytes, err := json.Marshal(response)
		if err != nil {
			reqLogger.Error().
				Err(err).
				Msg("Failed to marshal response")
			continue
		}

		reqLogger.Debug().
			Str("response", string(responseBytes)).
			Msg("Sending response")

		fmt.Fprintln(s.stdout, string(responseBytes))
	}

	return scanner.Err()
}

// handleNotification processes MCP notifications (messages with no ID that expect no response)
func (s *MCPServer) handleNotification(ctx context.Context, req *JSONRPCRequest) {
	logger := observability.LoggerWithComponent(ctx, "mcp-server")

	// According to MCP spec, notifications are fire-and-forget
	// We log them but take no action
	logger.Info().
		Str("method", req.Method).
		Msg("Notification received (no response sent)")
	// No response is sent for notifications
}

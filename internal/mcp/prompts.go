package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/apresai/gimage/internal/observability"
)

// Prompt represents an MCP prompt template that teaches LLMs how to use tools
type Prompt struct {
	Name        string           // Unique identifier for the prompt
	Title       string           // Human-readable title
	Description string           // Brief description of what this prompt teaches
	Arguments   []PromptArgument // Arguments that can customize the prompt
	Template    string           // Message template with {{variable}} substitution
}

// PromptArgument defines an argument that can be passed to customize a prompt
type PromptArgument struct {
	Name        string // Argument name
	Description string // What this argument is for
	Required    bool   // Whether this argument is required
}

// RegisterPrompt adds a prompt to the server
func (s *MCPServer) RegisterPrompt(prompt Prompt) {
	if s.prompts == nil {
		s.prompts = make(map[string]Prompt)
	}
	s.prompts[prompt.Name] = prompt

	logger := observability.Logger(context.Background())
	logger.Debug().
		Str("component", "mcp-server").
		Str("prompt", prompt.Name).
		Msg("Registered prompt")
}

// GetPrompt retrieves a prompt and optionally substitutes arguments
func (s *MCPServer) GetPrompt(name string, arguments map[string]string) (string, error) {
	prompt, exists := s.prompts[name]
	if !exists {
		return "", fmt.Errorf("prompt not found: %s", name)
	}

	// Validate required arguments
	for _, arg := range prompt.Arguments {
		if arg.Required {
			if _, ok := arguments[arg.Name]; !ok {
				return "", fmt.Errorf("missing required argument: %s", arg.Name)
			}
		}
	}

	// Substitute variables in template
	result := prompt.Template
	for key, value := range arguments {
		placeholder := fmt.Sprintf("{{%s}}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// Remove any remaining placeholders (optional arguments not provided)
	// Replace {{variable}} with "" for cleaner output
	for _, arg := range prompt.Arguments {
		if !arg.Required {
			placeholder := fmt.Sprintf("{{%s}}", arg.Name)
			result = strings.ReplaceAll(result, placeholder, "")
		}
	}

	return result, nil
}

package tools

import (
	"testing"

	"github.com/apresai/gimage/internal/mcp"
)

func TestRegisterListModelsTool(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil, false)

	// Register the tool
	RegisterListModelsTool(server)

	// Verify tool is registered
	tool := server.GetTool("list_models")
	if tool == nil {
		t.Fatal("list_models tool not registered")
	}

	// Verify tool properties
	if tool.Name != "list_models" {
		t.Errorf("Expected name 'list_models', got '%s'", tool.Name)
	}

	if tool.Description == "" {
		t.Error("Tool description is empty")
	}

	if tool.Handler == nil {
		t.Error("Tool handler is nil")
	}
}

func TestListModelsTool_InputSchema(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil, false)
	RegisterListModelsTool(server)

	tool := server.GetTool("list_models")
	if tool == nil {
		t.Fatal("list_models tool not registered")
	}

	schema := tool.InputSchema

	// list_models takes no arguments
	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties field is not a map")
	}

	// Should have empty properties (no arguments)
	if len(properties) != 0 {
		t.Errorf("Expected 0 properties (no arguments), got %d", len(properties))
	}
}

func TestListModelsTool_ResponseStructure(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil, false)
	RegisterListModelsTool(server)

	tool := server.GetTool("list_models")
	if tool == nil {
		t.Fatal("list_models tool not registered")
	}

	// Call the handler with empty args
	result, err := tool.Handler(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	// Verify result structure
	if result == nil {
		t.Fatal("Result is nil")
	}

	// Check for required top-level fields (new Provider system)
	requiredFields := []string{"providers", "total", "configured", "default_provider", "pricing_note", "recommendations"}
	for _, field := range requiredFields {
		if _, exists := result[field]; !exists {
			t.Errorf("Required field '%s' missing from result", field)
		}
	}

	// Verify providers is an array
	providers, ok := result["providers"].([]map[string]interface{})
	if !ok {
		t.Fatal("providers field is not an array of maps")
	}

	// Should have at least some providers
	if len(providers) == 0 {
		t.Error("No providers returned (expected at least some providers)")
	}

	// Verify total matches length
	total, ok := result["total"].(int)
	if !ok {
		t.Fatal("total field is not an integer")
	}

	if total != len(providers) {
		t.Errorf("total (%d) does not match providers length (%d)", total, len(providers))
	}
}

func TestListModelsTool_ProviderStructure(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil, false)
	RegisterListModelsTool(server)

	tool := server.GetTool("list_models")
	if tool == nil {
		t.Fatal("list_models tool not registered")
	}

	result, err := tool.Handler(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	providers, ok := result["providers"].([]map[string]interface{})
	if !ok {
		t.Fatal("providers field is not an array of maps")
	}

	if len(providers) == 0 {
		t.Fatal("No providers to test")
	}

	// Check first provider structure
	provider := providers[0]
	requiredProviderFields := []string{
		"provider_id", "name", "api", "model_id", "description",
		"available", "missing_credentials",
		"pricing", "pricing_summary",
		"supports_styles", "supports_negative_prompt", "supports_seed",
		"max_prompt_length",
	}

	for _, field := range requiredProviderFields {
		if _, exists := provider[field]; !exists {
			t.Errorf("Required provider field '%s' missing", field)
		}
	}
}

func TestListModelsTool_PricingStructure(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil, false)
	RegisterListModelsTool(server)

	tool := server.GetTool("list_models")
	if tool == nil {
		t.Fatal("list_models tool not registered")
	}

	result, err := tool.Handler(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	providers, ok := result["providers"].([]map[string]interface{})
	if !ok {
		t.Fatal("providers field is not an array of maps")
	}

	if len(providers) == 0 {
		t.Fatal("No providers to test")
	}

	// Check first provider's pricing structure
	provider := providers[0]
	pricing, ok := provider["pricing"].(map[string]interface{})
	if !ok {
		t.Fatal("pricing field is not a map")
	}

	// Required pricing fields
	requiredPricingFields := []string{"currency", "free_tier"}
	for _, field := range requiredPricingFields {
		if _, exists := pricing[field]; !exists {
			t.Errorf("Required pricing field '%s' missing", field)
		}
	}
}

func TestListModelsTool_ConfiguredCount(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil, false)
	RegisterListModelsTool(server)

	tool := server.GetTool("list_models")
	if tool == nil {
		t.Fatal("list_models tool not registered")
	}

	result, err := tool.Handler(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	configured, ok := result["configured"].(int)
	if !ok {
		t.Fatal("configured field is not an integer")
	}

	// configured should be 0 or positive
	if configured < 0 {
		t.Errorf("configured count is negative: %d", configured)
	}

	// Count providers marked as available
	providers, ok := result["providers"].([]map[string]interface{})
	if !ok {
		t.Fatal("providers field is not an array of maps")
	}

	availableCount := 0
	for _, p := range providers {
		if available, ok := p["available"].(bool); ok && available {
			availableCount++
		}
	}

	if configured != availableCount {
		t.Errorf("configured count (%d) doesn't match available providers (%d)", configured, availableCount)
	}
}

func TestListModelsTool_DefaultProviderStructure(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil, false)
	RegisterListModelsTool(server)

	tool := server.GetTool("list_models")
	if tool == nil {
		t.Fatal("list_models tool not registered")
	}

	result, err := tool.Handler(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	defaultProvider, ok := result["default_provider"].(map[string]interface{})
	if !ok {
		t.Fatal("default_provider field is not a map")
	}

	// Check for default provider fields
	requiredDefaultFields := []string{"provider_id", "name", "pricing_summary"}
	for _, field := range requiredDefaultFields {
		if _, exists := defaultProvider[field]; !exists {
			t.Errorf("Required default_provider field '%s' missing", field)
		}
	}
}

func TestListModelsTool_RecommendationsStructure(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil, false)
	RegisterListModelsTool(server)

	tool := server.GetTool("list_models")
	if tool == nil {
		t.Fatal("list_models tool not registered")
	}

	result, err := tool.Handler(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	recommendations, ok := result["recommendations"].(map[string]interface{})
	if !ok {
		t.Fatal("recommendations field is not a map")
	}

	// Check for recommendation categories (updated for Provider system)
	requiredRecommendations := []string{"budget_users", "paid_users", "aws_users"}
	for _, field := range requiredRecommendations {
		if _, exists := recommendations[field]; !exists {
			t.Errorf("Required recommendation '%s' missing", field)
		}
	}

	// Verify they are strings
	for _, field := range requiredRecommendations {
		if _, ok := recommendations[field].(string); !ok {
			t.Errorf("Recommendation '%s' is not a string", field)
		}
	}
}

func TestListModelsTool_ProviderAvailability(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil, false)
	RegisterListModelsTool(server)

	tool := server.GetTool("list_models")
	if tool == nil {
		t.Fatal("list_models tool not registered")
	}

	result, err := tool.Handler(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	providers, ok := result["providers"].([]map[string]interface{})
	if !ok {
		t.Fatal("providers field is not an array of maps")
	}

	// Check that each provider has an availability status
	for i, provider := range providers {
		available, exists := provider["available"]
		if !exists {
			t.Errorf("Provider %d missing 'available' field", i)
			continue
		}

		// Should be a boolean
		if _, ok := available.(bool); !ok {
			t.Errorf("Provider %d 'available' field is not a boolean", i)
		}

		// Check missing_credentials field
		missingCreds, exists := provider["missing_credentials"]
		if !exists {
			t.Errorf("Provider %d missing 'missing_credentials' field", i)
			continue
		}

		// missing_credentials should be a string array
		if _, ok := missingCreds.([]string); !ok {
			t.Errorf("Provider %d 'missing_credentials' field is not a string array", i)
		}
	}
}

func TestListModelsTool_PricingNote(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil, false)
	RegisterListModelsTool(server)

	tool := server.GetTool("list_models")
	if tool == nil {
		t.Fatal("list_models tool not registered")
	}

	result, err := tool.Handler(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	pricingNote, ok := result["pricing_note"].(string)
	if !ok {
		t.Fatal("pricing_note is not a string")
	}

	if pricingNote == "" {
		t.Error("pricing_note is empty")
	}

	// Should mention USD and providers
	if len(pricingNote) < 50 {
		t.Error("pricing_note seems too short to be informative")
	}
}

func TestListModelsTool_APITypes(t *testing.T) {
	server := mcp.NewMCPServer("test", "1.0.0", nil, false)
	RegisterListModelsTool(server)

	tool := server.GetTool("list_models")
	if tool == nil {
		t.Fatal("list_models tool not registered")
	}

	result, err := tool.Handler(map[string]interface{}{})
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}

	providers, ok := result["providers"].([]map[string]interface{})
	if !ok {
		t.Fatal("providers field is not an array of maps")
	}

	// Track which APIs are present
	apiTypes := make(map[string]bool)
	for _, provider := range providers {
		api, ok := provider["api"].(string)
		if !ok {
			t.Error("Provider api field is not a string")
			continue
		}
		apiTypes[api] = true
	}

	// Should have at least one API type
	if len(apiTypes) == 0 {
		t.Error("No API types found in providers")
	}

	// Valid API types
	validAPIs := map[string]bool{"gemini": true, "vertex": true, "bedrock": true, "grok": true}
	for api := range apiTypes {
		if !validAPIs[api] {
			t.Errorf("Invalid API type: %s", api)
		}
	}
}

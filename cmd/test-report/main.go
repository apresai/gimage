package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// MCP Tool represents an MCP tool with its test status
type MCPTool struct {
	Name             string
	Exposed          bool
	UnitTests        bool
	IntegrationTests bool
	E2ETests         string // "yes", "no", or "N/A"
}

func main() {
	printHeader()
	tools := checkTools()
	printToolsTable(tools)
	printSummary(tools)
	runTests()
	runCLIE2ETests()
	checkE2ETests()
	printFooter()
}

func printHeader() {
	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                                               ║")
	fmt.Println("║                    📊  GIMAGE TEST COVERAGE REPORT  📊                        ║")
	fmt.Println("║                                                                               ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func printFooter() {
	fmt.Println("╔═══════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                            REPORT COMPLETE                                    ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func checkTools() []MCPTool {
	toolNames := []string{
		"generate",
		"resize",
		"scale",
		"crop",
		"compress",
		"convert",
		"batch",
		"models",
	}

	tools := make([]MCPTool, len(toolNames))

	for i, name := range toolNames {
		toolFile := filepath.Join("internal", "mcp", "tools", name+".go")
		testFile := filepath.Join("internal", "mcp", "tools", name+"_test.go")

		tool := MCPTool{
			Name:      name,
			Exposed:   fileExists(toolFile),
			UnitTests: fileExists(testFile),
		}

		// Check for integration tests (basic check)
		integrationPattern := filepath.Join("test", "integration", "*_test.go")
		if matches, err := filepath.Glob(integrationPattern); err == nil && len(matches) > 0 {
			tool.IntegrationTests = checkIntegrationTestsForTool(name, matches)
		}

		// E2E tests only for generate
		if name == "generate" {
			e2eFile := filepath.Join("test", "integration", "generate_e2e_test.go")
			if fileExists(e2eFile) {
				tool.E2ETests = "✅"
			} else {
				tool.E2ETests = "⚠️  MISSING"
			}
		} else {
			tool.E2ETests = "N/A"
		}

		tools[i] = tool
	}

	return tools
}

func checkIntegrationTestsForTool(toolName string, files []string) bool {
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		if strings.Contains(string(content), toolName) {
			return true
		}
	}
	return false
}

func printToolsTable(tools []MCPTool) {
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ MCP TOOLS STATUS                                                                │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────────────┤")
	fmt.Println("│ Tool              │ Exposed │ Unit Tests │ Integration │ E2E Tests             │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────────────┤")

	for _, tool := range tools {
		exposed := "❌"
		if tool.Exposed {
			exposed = "✅"
		}

		unitTests := "❌"
		if tool.UnitTests {
			unitTests = "✅"
		}

		integration := "❌"
		if tool.IntegrationTests {
			integration = "✅"
		}

		fmt.Printf("│ %-17s │ %-7s │ %-10s │ %-11s │ %-21s │\n",
			tool.Name, exposed, unitTests, integration, tool.E2ETests)
	}

	fmt.Println("└─────────────────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
}

func printSummary(tools []MCPTool) {
	totalTools := len(tools)
	toolsWithUnitTests := 0
	toolsWithIntegration := 0

	for _, tool := range tools {
		if tool.UnitTests {
			toolsWithUnitTests++
		}
		if tool.IntegrationTests {
			toolsWithIntegration++
		}
	}

	unitCoverage := 0
	integrationCoverage := 0
	if totalTools > 0 {
		unitCoverage = (toolsWithUnitTests * 100) / totalTools
		integrationCoverage = (toolsWithIntegration * 100) / totalTools
	}

	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ COVERAGE SUMMARY                                                                │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────────────┤")
	fmt.Printf("│ Total MCP Tools:           %-50d │\n", totalTools)
	fmt.Printf("│ Tools with Unit Tests:     %-40s (%d%%) │\n",
		fmt.Sprintf("%d/%d", toolsWithUnitTests, totalTools), unitCoverage)
	fmt.Printf("│ Tools with Integration:    %-40s (%d%%) │\n",
		fmt.Sprintf("%d/%d", toolsWithIntegration, totalTools), integrationCoverage)
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────────┘")
	fmt.Println()
}

func runTests() {
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ RUNNING UNIT TESTS                                                              │")
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────────┘")
	fmt.Println()

	cmd := exec.Command("go", "test", "-v", "-race", "./internal/...", "./test/...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Some tests failed\n")
	}

	fmt.Println()
}

func runCLIE2ETests() {
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ RUNNING CLI E2E TESTS (FREE - no API calls)                                    │")
	fmt.Println("└─────────────────────────────────────────────────────────────────────────────────┘")
	fmt.Println()

	cliE2EFile := filepath.Join("test", "integration", "cli_e2e_test.go")
	if !fileExists(cliE2EFile) {
		fmt.Println("⚠️  CLI E2E test file not found, skipping...")
		fmt.Println()
		return
	}

	// Build the binary first
	fmt.Println("Building binary for CLI E2E tests...")
	buildCmd := exec.Command("make", "build")
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr
	if err := buildCmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to build binary for CLI E2E tests\n")
		fmt.Println()
		return
	}

	// Run CLI E2E tests
	cmd := exec.Command("go", "test", "-v", "-tags=e2e", "./test/integration/cli_e2e_test.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: CLI E2E tests failed\n")
	}

	fmt.Println()
}

func checkE2ETests() {
	fmt.Println("┌─────────────────────────────────────────────────────────────────────────────────┐")
	fmt.Println("│ GENERATE IMAGE E2E TESTS (requires API credentials)                            │")
	fmt.Println("├─────────────────────────────────────────────────────────────────────────────────┤")

	e2eFile := filepath.Join("test", "integration", "generate_e2e_test.go")

	if fileExists(e2eFile) {
		fmt.Println("│ ✅  E2E test file exists: test/integration/generate_e2e_test.go                │")
		fmt.Println("│                                                                                 │")
		fmt.Println("│ ⚠️  These tests require real API credentials and may cost money                │")
		fmt.Println("│     - Gemini: ~$0.045-$0.24 per test, depending on model/resolution            │")
		fmt.Println("│     - Vertex: Gemini 3.1 Flash tiered pricing via generateContent              │")
		fmt.Println("│     - Grok: ~$0.02-$0.07 per test                                              │")
		fmt.Println("│                                                                                 │")
		fmt.Println("│ APIs Tested:                                                                    │")

		content, err := os.ReadFile(e2eFile)
		if err == nil {
			contentStr := string(content)

			if strings.Contains(contentStr, "Gemini") {
				fmt.Println("│   ✅  Gemini API                                                               │")
			} else {
				fmt.Println("│   ❌  Gemini API                                                               │")
			}

			if strings.Contains(contentStr, "Vertex") {
				fmt.Println("│   ✅  Vertex AI                                                                │")
			} else {
				fmt.Println("│   ❌  Vertex AI                                                                │")
			}

			if strings.Contains(contentStr, "Grok") {
				fmt.Println("│   ✅  xAI Grok                                                                 │")
			} else {
				fmt.Println("│   ❌  xAI Grok                                                                 │")
			}
		}
		fmt.Println("└─────────────────────────────────────────────────────────────────────────────────┘")
		fmt.Println()

		// Run the E2E tests
		fmt.Println("┌─────────────────────────────────────────────────────────────────────────────────┐")
		fmt.Println("│ RUNNING GENERATE IMAGE E2E TESTS (costs money!)                                │")
		fmt.Println("└─────────────────────────────────────────────────────────────────────────────────┘")
		fmt.Println()

		cmd := exec.Command("go", "test", "-v", "-tags=e2e", "./test/integration/generate_e2e_test.go")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: E2E tests failed (this may be due to missing credentials)\n")
		}
		fmt.Println()
	} else {
		fmt.Println("│ ❌  E2E test file NOT FOUND: test/integration/generate_e2e_test.go             │")
		fmt.Println("│                                                                                 │")
		fmt.Println("│ Recommendation: Create E2E tests for:                                          │")
		fmt.Println("│   - Gemini API real image generation                                           │")
		fmt.Println("│   - Vertex AI real image generation                                            │")
		fmt.Println("│   - xAI Grok real image generation                                             │")
		fmt.Println("└─────────────────────────────────────────────────────────────────────────────────┘")
		fmt.Println()
	}
}

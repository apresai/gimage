package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type TestResults struct {
	Name            string
	Passed          int
	Failed          int
	Total           int
	GeneratedImages []string
}

func main() {
	// Check for --cli-coverage flag
	if len(os.Args) == 3 && os.Args[1] == "--cli-coverage" {
		cli := parseTestLog(os.Args[2], "")
		if cli.Total > 0 {
			coverage := (cli.Passed * 100) / cli.Total
			fmt.Println("CLI & Config Packages (E2E Test Coverage):")
			fmt.Printf("  %d%% pass rate (%d/%d CLI E2E tests passed)\n", coverage, cli.Passed, cli.Total)
		}
		os.Exit(0)
	}

	if len(os.Args) < 4 {
		fmt.Println("Usage: test-summary <unit-log> <cli-e2e-log> <generate-e2e-log>")
		os.Exit(1)
	}

	unitLog := os.Args[1]
	cliLog := os.Args[2]
	generateLog := os.Args[3]

	unit := parseTestLog(unitLog, "Unit Tests")
	cli := parseTestLog(cliLog, "CLI E2E Tests (resize, scale, crop)")
	generate := parseTestLog(generateLog, "Generate Image E2E Tests (Gemini, Vertex, Bedrock, Grok)")

	// Print test results
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("                              📊 TEST RESULTS")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	printTestResult(unit, "")
	fmt.Println()
	printTestResult(cli, "(FREE - no API costs)")
	fmt.Println()
	printTestResult(generate, "(~$0.34 API costs)")
	fmt.Println()

	// Print generated images from all test runs
	allImages := append(append(unit.GeneratedImages, cli.GeneratedImages...), generate.GeneratedImages...)
	if len(allImages) > 0 {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("                         Generated Images (%d)\n", len(allImages))
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println()
		for _, img := range allImages {
			fmt.Printf("  %s\n", img)
		}
		fmt.Println()
	}

	// Print exit code (0 if all passed, 1 if any failed)
	totalFailed := unit.Failed + cli.Failed + generate.Failed
	if totalFailed > 0 {
		os.Exit(1)
	}
	os.Exit(0)
}

func parseTestLog(logPath, name string) TestResults {
	result := TestResults{Name: name}

	file, err := os.Open(logPath)
	if err != nil {
		// File doesn't exist or can't be read
		return result
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "--- PASS:") {
			result.Passed++
		} else if strings.HasPrefix(line, "--- FAIL:") {
			result.Failed++
		}
		// Extract generated image paths from test logs
		if idx := strings.Index(line, "GENERATED_IMAGE:"); idx >= 0 {
			imgPath := strings.TrimSpace(line[idx+len("GENERATED_IMAGE:"):])
			// Strip trailing metadata like "(1234 bytes, png, 512x512)"
			if parenIdx := strings.Index(imgPath, " ("); parenIdx >= 0 {
				imgPath = imgPath[:parenIdx]
			}
			if imgPath != "" {
				result.GeneratedImages = append(result.GeneratedImages, imgPath)
			}
		}
	}

	result.Total = result.Passed + result.Failed
	return result
}

func printTestResult(r TestResults, note string) {
	fmt.Printf("%s:\n", r.Name)
	if r.Total == 0 {
		fmt.Println("  ⚠️  NO TESTS RUN")
		return
	}

	if r.Failed == 0 {
		if note != "" {
			fmt.Printf("  ✅ PASSED: %d/%d tests %s\n", r.Passed, r.Total, note)
		} else {
			fmt.Printf("  ✅ PASSED: %d/%d tests\n", r.Passed, r.Total)
		}
	} else {
		fmt.Printf("  ❌ FAILED: %d/%d tests (%d passed) %s\n", r.Failed, r.Total, r.Passed, note)
	}
}

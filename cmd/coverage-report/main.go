package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type CoverageFile struct {
	Path      string
	Coverage  float64
	Category  string
	IsTested  bool
	Functions []FunctionCoverage
}

type FunctionCoverage struct {
	Name     string
	Line     int
	Coverage float64
}

type GeneratedImage struct {
	Path     string
	Provider string
	Size     string
	Format   string
	Base64   string
}

type CoverageReport struct {
	TotalCoverage    float64
	CorePackages     []CoverageFile
	CLIPackages      []CoverageFile
	UnitTestCoverage float64
	E2ECoverage      float64
	GeneratedImages  []GeneratedImage
	ImagePrompt      string
}

func runCommand(cmdStr string) (string, error) {
	parts := strings.Fields(cmdStr)
	cmd := exec.Command(parts[0], parts[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	return out.String(), err
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: coverage-report <coverage.out> [image-dir]")
		os.Exit(1)
	}

	coverageFile := os.Args[1]
	imageDir := "test/output/e2e"
	if len(os.Args) >= 3 {
		imageDir = os.Args[2]
	}

	report := parseCoverageFile(coverageFile)
	report.GeneratedImages = scanImages(imageDir)
	report.ImagePrompt = extractE2EPrompt()
	generateHTML(report)
}

func parseCoverageFile(path string) CoverageReport {
	// Run go tool cover -func to get function-level coverage
	cmd := fmt.Sprintf("go tool cover -func=%s", path)
	output, err := runCommand(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running go tool cover: %v\n", err)
		os.Exit(1)
	}

	fileMap := make(map[string]*CoverageFile)
	funcRegex := regexp.MustCompile(`^([^:]+):(\d+):\s+(\S+)\s+(\d+\.\d+)%`)

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "total:") {
			continue
		}

		matches := funcRegex.FindStringSubmatch(line)
		if len(matches) == 5 {
			filePath := matches[1]
			functionName := matches[3]
			var coverage float64
			fmt.Sscanf(matches[4], "%f", &coverage)

			if _, exists := fileMap[filePath]; !exists {
				fileMap[filePath] = &CoverageFile{
					Path:      filePath,
					Category:  categorizeFile(filePath),
					IsTested:  coverage > 0,
					Functions: []FunctionCoverage{},
				}
			}

			var lineNum int
			fmt.Sscanf(matches[2], "%d", &lineNum)

			fileMap[filePath].Functions = append(fileMap[filePath].Functions, FunctionCoverage{
				Name:     functionName,
				Line:     lineNum,
				Coverage: coverage,
			})
		}
	}

	// Calculate file-level coverage
	for _, file := range fileMap {
		if len(file.Functions) > 0 {
			total := 0.0
			for _, fn := range file.Functions {
				total += fn.Coverage
			}
			file.Coverage = total / float64(len(file.Functions))
			file.IsTested = file.Coverage > 0
		}
	}

	// Separate into core and CLI packages
	var corePackages, cliPackages []CoverageFile
	totalCoverage := 0.0
	totalFiles := 0
	unitTestedCount := 0
	unitTestedCoverage := 0.0

	for _, file := range fileMap {
		if file.Category == "core" {
			corePackages = append(corePackages, *file)
			if file.IsTested {
				unitTestedCount++
				unitTestedCoverage += file.Coverage
			}
		} else {
			cliPackages = append(cliPackages, *file)
		}
		totalCoverage += file.Coverage
		totalFiles++
	}

	// Sort by coverage (highest first)
	sort.Slice(corePackages, func(i, j int) bool {
		return corePackages[i].Coverage > corePackages[j].Coverage
	})
	sort.Slice(cliPackages, func(i, j int) bool {
		return cliPackages[i].Coverage > cliPackages[j].Coverage
	})

	avgCoverage := 0.0
	if totalFiles > 0 {
		avgCoverage = totalCoverage / float64(totalFiles)
	}

	avgUnitCoverage := 0.0
	if unitTestedCount > 0 {
		avgUnitCoverage = unitTestedCoverage / float64(unitTestedCount)
	}

	return CoverageReport{
		TotalCoverage:    avgCoverage,
		CorePackages:     corePackages,
		CLIPackages:      cliPackages,
		UnitTestCoverage: avgUnitCoverage,
		E2ECoverage:      0.0, // CLI packages are E2E tested
	}
}

func categorizeFile(path string) string {
	if strings.Contains(path, "/cmd/") ||
		strings.Contains(path, "/internal/cli/") ||
		strings.Contains(path, "/internal/config/") {
		return "cli"
	}
	return "core"
}

// scanImages scans a directory for generated test images and returns them as base64-embedded structs
func scanImages(dir string) []GeneratedImage {
	var images []GeneratedImage

	entries, err := os.ReadDir(dir)
	if err != nil {
		// Directory doesn't exist or can't be read - not an error
		return images
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" {
			continue
		}

		fullPath := filepath.Join(dir, name)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Extract provider from filename (e.g., "e2e_gemini_flash_1.png" -> "Gemini Flash")
		provider := extractProviderFromFilename(name)

		// Determine format for data URI
		format := strings.TrimPrefix(ext, ".")
		if format == "jpg" {
			format = "jpeg"
		}

		// Format file size
		sizeStr := formatFileSize(info.Size())

		images = append(images, GeneratedImage{
			Path:     fullPath,
			Provider: provider,
			Size:     sizeStr,
			Format:   format,
			Base64:   base64.StdEncoding.EncodeToString(data),
		})
	}

	// Sort by provider name for consistent ordering
	sort.Slice(images, func(i, j int) bool {
		return images[i].Provider < images[j].Provider
	})

	return images
}

// extractE2EPrompt reads the shared prompt from the E2E test file
func extractE2EPrompt() string {
	paths := []string{
		"test/integration/generate_e2e_test.go",
		filepath.Join("..", "..", "test", "integration", "generate_e2e_test.go"),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		re := regexp.MustCompile(`e2ePrompt\s*=\s*"([^"]+)"`)
		matches := re.FindSubmatch(data)
		if len(matches) >= 2 {
			return string(matches[1])
		}
	}

	return "Same prompt across all models for visual comparison"
}

// extractProviderFromFilename extracts a display-friendly provider name from an image filename
func extractProviderFromFilename(name string) string {
	// Remove extension
	name = strings.TrimSuffix(name, filepath.Ext(name))
	name = strings.ToLower(name)

	// Map filename patterns to display names
	patterns := []struct {
		contains string
		display  string
	}{
		{"gemini_flash", "Gemini Flash"},
		{"gemini_3_pro", "Gemini 3 Pro"},
		{"gemini3pro", "Gemini 3 Pro"},
		{"gemini", "Gemini"},
		{"vertex_sdk", "Vertex AI SDK"},
		{"vertex", "Vertex AI"},
		{"imagen", "Imagen"},
		{"bedrock", "Bedrock Nova Canvas"},
		{"nova", "Nova Canvas"},
		{"grok_imagine_pro", "Grok Imagine Pro"},
		{"grok_pro", "Grok Imagine Pro"},
		{"grok", "Grok Imagine"},
	}

	for _, p := range patterns {
		if strings.Contains(name, p.contains) {
			return p.display
		}
	}

	// Fallback: clean up the filename
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	// Capitalize first letter of each word
	words := strings.Fields(name)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// formatFileSize returns a human-readable file size string
func formatFileSize(size int64) string {
	switch {
	case size >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	case size >= 1024:
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

func generateHTML(report CoverageReport) {
	tmpl := `<!DOCTYPE html>
<html>
<head>
	<meta charset="utf-8">
	<title>Gimage Coverage Report</title>
	<style>
		:root {
			--bg-primary: #ffffff;
			--bg-secondary: #f8f9fa;
			--text-primary: #212529;
			--text-secondary: #6c757d;
			--border: #dee2e6;
			--success: #28a745;
			--warning: #ffc107;
			--danger: #dc3545;
			--info: #17a2b8;
		}

		@media (prefers-color-scheme: dark) {
			:root {
				--bg-primary: #1a1d23;
				--bg-secondary: #24272e;
				--text-primary: #e9ecef;
				--text-secondary: #adb5bd;
				--border: #495057;
			}
		}

		* {
			margin: 0;
			padding: 0;
			box-sizing: border-box;
		}

		body {
			font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
			background: var(--bg-primary);
			color: var(--text-primary);
			line-height: 1.6;
			padding: 2rem;
		}

		.container {
			max-width: 1200px;
			margin: 0 auto;
		}

		h1 {
			font-size: 2.5rem;
			margin-bottom: 0.5rem;
			color: var(--text-primary);
		}

		.subtitle {
			color: var(--text-secondary);
			margin-bottom: 2rem;
			font-size: 1.1rem;
		}

		.summary {
			display: grid;
			grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
			gap: 1.5rem;
			margin-bottom: 3rem;
		}

		.summary-card {
			background: var(--bg-secondary);
			border: 1px solid var(--border);
			border-radius: 8px;
			padding: 1.5rem;
		}

		.summary-card h3 {
			font-size: 0.9rem;
			text-transform: uppercase;
			letter-spacing: 0.5px;
			color: var(--text-secondary);
			margin-bottom: 0.5rem;
		}

		.summary-card .value {
			font-size: 2.5rem;
			font-weight: bold;
			margin-bottom: 0.25rem;
		}

		.summary-card .description {
			font-size: 0.9rem;
			color: var(--text-secondary);
		}

		.coverage-high { color: var(--success); }
		.coverage-medium { color: var(--warning); }
		.coverage-low { color: var(--danger); }
		.coverage-none { color: var(--text-secondary); }

		.section {
			margin-bottom: 3rem;
		}

		.section h2 {
			font-size: 1.5rem;
			margin-bottom: 1rem;
			padding-bottom: 0.5rem;
			border-bottom: 2px solid var(--border);
		}

		.file-list {
			background: var(--bg-secondary);
			border: 1px solid var(--border);
			border-radius: 8px;
			overflow: hidden;
		}

		.file-item {
			padding: 1rem 1.5rem;
			border-bottom: 1px solid var(--border);
			display: flex;
			justify-content: space-between;
			align-items: center;
			transition: background 0.2s;
		}

		.file-item:last-child {
			border-bottom: none;
		}

		.file-item:hover {
			background: var(--bg-primary);
		}

		.file-path {
			font-family: "SF Mono", Monaco, "Cascadia Code", monospace;
			font-size: 0.9rem;
			flex: 1;
		}

		.file-coverage {
			font-size: 1.1rem;
			font-weight: bold;
			min-width: 80px;
			text-align: right;
		}

		.badge {
			display: inline-block;
			padding: 0.25rem 0.75rem;
			border-radius: 12px;
			font-size: 0.85rem;
			font-weight: 600;
			margin-left: 1rem;
		}

		.badge-success {
			background: var(--success);
			color: white;
		}

		.badge-info {
			background: var(--info);
			color: white;
		}

		.badge-secondary {
			background: var(--text-secondary);
			color: white;
		}

		.note {
			background: var(--bg-secondary);
			border-left: 4px solid var(--info);
			padding: 1rem 1.5rem;
			margin: 1rem 0;
			border-radius: 4px;
		}

		.note-title {
			font-weight: bold;
			margin-bottom: 0.5rem;
		}

		.prompt-box {
			background: var(--bg-secondary);
			border: 1px solid var(--border);
			border-radius: 8px;
			padding: 1.5rem;
			margin: 1rem 0;
		}

		.prompt-label {
			font-size: 0.75rem;
			text-transform: uppercase;
			letter-spacing: 1px;
			color: var(--text-secondary);
			margin-bottom: 0.5rem;
		}

		.prompt-text {
			font-size: 1.15rem;
			font-style: italic;
			color: var(--text-primary);
			line-height: 1.6;
		}

		.prompt-note {
			font-size: 0.85rem;
			color: var(--text-secondary);
			margin-top: 0.5rem;
		}

		.image-grid {
			display: grid;
			grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
			gap: 1.5rem;
			margin-top: 1rem;
		}

		.image-card {
			background: var(--bg-secondary);
			border: 1px solid var(--border);
			border-radius: 8px;
			overflow: hidden;
			transition: box-shadow 0.2s;
		}

		.image-card:hover {
			box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
		}

		.image-card img {
			width: 100%;
			height: 250px;
			object-fit: cover;
			display: block;
		}

		.image-label {
			padding: 0.75rem 1rem 0.25rem;
			font-weight: 600;
			font-size: 0.95rem;
		}

		.image-meta {
			padding: 0 1rem 0.75rem;
			font-size: 0.8rem;
			color: var(--text-secondary);
		}

		footer {
			margin-top: 4rem;
			padding-top: 2rem;
			border-top: 1px solid var(--border);
			text-align: center;
			color: var(--text-secondary);
			font-size: 0.9rem;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>Gimage Coverage Report</h1>
		<p class="subtitle">Comprehensive test coverage analysis</p>

		<div class="summary">
			<div class="summary-card">
				<h3>Total Coverage</h3>
				<div class="value {{coverageClass .TotalCoverage}}">{{printf "%.1f" .TotalCoverage}}%</div>
				<div class="description">All packages</div>
			</div>

			<div class="summary-card">
				<h3>Unit Test Coverage</h3>
				<div class="value {{coverageClass .UnitTestCoverage}}">{{printf "%.1f" .UnitTestCoverage}}%</div>
				<div class="description">Core packages</div>
			</div>

			<div class="summary-card">
				<h3>Core Packages</h3>
				<div class="value coverage-high">{{len .CorePackages}}</div>
				<div class="description">Unit tested</div>
			</div>

			<div class="summary-card">
				<h3>CLI Packages</h3>
				<div class="value coverage-none">{{len .CLIPackages}}</div>
				<div class="description">E2E tested</div>
			</div>
		</div>

		{{if .GeneratedImages}}
		<div class="section">
			<h2>Generated Test Images <span class="badge badge-info">E2E</span></h2>
			<div class="prompt-box">
				<div class="prompt-label">Prompt</div>
				<div class="prompt-text">"{{.ImagePrompt}}"</div>
				<div class="prompt-note">Same prompt sent to all providers for visual comparison</div>
			</div>

			<div class="image-grid">
				{{range .GeneratedImages}}
				<div class="image-card">
					<img src="data:image/{{.Format}};base64,{{.Base64}}" alt="{{.Provider}} generated image" />
					<div class="image-label">{{.Provider}}</div>
					<div class="image-meta">{{.Size}}</div>
				</div>
				{{end}}
			</div>
		</div>
		{{end}}

		<div class="section">
			<h2>Core Packages <span class="badge badge-success">Unit Tested</span></h2>
			<div class="note">
				<div class="note-title">These packages have comprehensive unit test coverage</div>
				Includes: imaging operations, image generation clients, MCP tools, and shared models
			</div>

			<div class="file-list">
				{{range .CorePackages}}
				<div class="file-item">
					<div class="file-path">{{.Path}}</div>
					<div class="file-coverage {{coverageClass .Coverage}}">{{printf "%.1f" .Coverage}}%</div>
				</div>
				{{end}}
			</div>
		</div>

		<div class="section">
			<h2>CLI & Config Packages <span class="badge badge-info">E2E Tested</span></h2>
			<div class="note">
				<div class="note-title">These packages are tested via End-to-End tests</div>
				CLI commands and configuration are tested by running the actual binary in test/integration/cli_e2e_test.go.
				0% unit test coverage is expected and correct for these packages.
			</div>

			<div class="file-list">
				{{range .CLIPackages}}
				<div class="file-item">
					<div class="file-path">{{.Path}}</div>
					<div class="file-coverage coverage-none">{{printf "%.1f" .Coverage}}% <span class="badge badge-secondary">E2E</span></div>
				</div>
				{{end}}
			</div>
		</div>

		<footer>
			<p>Generated by gimage coverage-report tool</p>
			<p>For detailed line-by-line coverage, see <a href="coverage.html" style="color: var(--info);">coverage.html</a></p>
		</footer>
	</div>
</body>
</html>`

	funcMap := template.FuncMap{
		"coverageClass": func(coverage float64) string {
			if coverage >= 80.0 {
				return "coverage-high"
			} else if coverage >= 50.0 {
				return "coverage-medium"
			} else if coverage > 0.0 {
				return "coverage-low"
			}
			return "coverage-none"
		},
	}

	t := template.Must(template.New("report").Funcs(funcMap).Parse(tmpl))

	output, err := os.Create("coverage-report.html")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer output.Close()

	if err := t.Execute(output, report); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating report: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Coverage report generated: coverage-report.html")
	fmt.Printf("  Total Coverage: %.1f%%\n", report.TotalCoverage)
	fmt.Printf("  Unit Test Coverage: %.1f%%\n", report.UnitTestCoverage)
	fmt.Printf("  Core Packages: %d files\n", len(report.CorePackages))
	fmt.Printf("  CLI Packages: %d files (E2E tested)\n", len(report.CLIPackages))
	if len(report.GeneratedImages) > 0 {
		fmt.Printf("  Generated Images: %d (embedded in report)\n", len(report.GeneratedImages))
	}
}

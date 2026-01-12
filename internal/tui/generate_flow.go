// Package tui provides Terminal User Interface components for gimage.
package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/apresai/gimage/internal/generate"
	"github.com/apresai/gimage/internal/logging"
	"github.com/apresai/gimage/pkg/models"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// GenerateStep represents a step in the image generation workflow
type GenerateStep int

const (
	StepPrompt   GenerateStep = iota
	StepProvider              // Select provider (was StepModel)
	StepSize
	StepStyle
	StepAdvanced // NEW: Advanced options (negative prompt, seed, aspect ratio, etc.)
	StepOutput
	StepCommand // Show command preview before generating
	StepProgress
	StepProviderRetry // Select a different provider if first one fails
	StepResult
)

// Provider selection item
type providerOption struct {
	id         string // e.g., "gemini/flash-2.5"
	name       string // e.g., "Gemini 2.5 Flash (via Gemini API)"
	api        string // e.g., "gemini", "vertex", "bedrock"
	model      string // e.g., "gemini-2.5-flash-image"
	cost       string // formatted cost string
	free       bool
	configured bool     // whether authentication is set up
	missing    []string // missing credentials
}

// Size selection item
type sizeOption struct {
	size   string
	label  string
	aspect string
}

// Style selection item
type styleOption struct {
	value string
	label string
	desc  string
}

// Aspect ratio option for picker
type aspectRatioOption struct {
	value string
	label string
}

// Image size option for Gemini 3 Pro
type imageSizeOption struct {
	value string
	label string
	desc  string
}

// Output format option
type outputFormatOption struct {
	value string
	label string
}

// GenerateFlowModel handles the multi-step image generation flow
type GenerateFlowModel struct {
	currentStep GenerateStep
	width       int
	height      int

	// Step 1: Prompt input
	promptTextarea textarea.Model

	// Step 2: Provider selection
	providers        []providerOption
	selectedProvider int
	providerRegistry *generate.ProviderRegistry

	// Step 3: Size selection
	sizes        []sizeOption
	selectedSize int
	customWidth  textinput.Model
	customHeight textinput.Model
	useCustom    bool

	// Step 4: Style selection
	styles        []styleOption
	selectedStyle int

	// Step 5: Advanced options (NEW)
	negativePromptInput  textinput.Model
	seedInput            textinput.Model
	aspectRatios         []aspectRatioOption
	selectedAspectRatio  int
	imageSizes           []imageSizeOption
	selectedImageSize    int
	outputFormats        []outputFormatOption
	selectedOutputFormat int
	cfgScaleInput        textinput.Model
	countInput           textinput.Model
	advancedFocusIndex   int // Track which field is focused (0-6)

	// Step 6: Output path
	outputInput textinput.Model

	// Step 7: Command preview
	commandStr string

	// Step 8: Progress
	progressBar progress.Model
	progressMsg string
	generating  bool

	// Step 9: Result
	resultPath     string
	resultSize     int64
	generationTime time.Duration
	err            error

	// Provider retry state
	retryProviders          []providerOption
	selectedRetryProvider   int
	customProviderInput     textinput.Model
	showCustomProviderInput bool
	lastGenerationError     string
	providerRetryInput      textinput.Model

	// Error context for retry
	errorContext struct {
		prompt string
		model  string
		size   string
		style  string
		output string
	}
	showErrorDetails bool

	// Navigation state
	showHelp bool
}

// NewGenerateFlowModel creates a new generate flow model
func NewGenerateFlowModel() *GenerateFlowModel {
	// Initialize prompt textarea
	ta := textarea.New()
	ta.Placeholder = "Describe the image you want to generate..."
	ta.Focus()
	ta.CharLimit = 1000
	ta.SetWidth(70)
	ta.SetHeight(5)

	// Initialize custom size inputs
	widthInput := textinput.New()
	widthInput.Placeholder = "1024"
	widthInput.CharLimit = 4
	widthInput.Width = 10

	heightInput := textinput.New()
	heightInput.Placeholder = "1024"
	heightInput.CharLimit = 4
	heightInput.Width = 10

	// Initialize output path input
	outputInput := textinput.New()
	outputInput.Placeholder = "~/Desktop/gimage_output.png"
	outputInput.CharLimit = 256
	outputInput.Width = 60

	// Initialize progress bar
	prog := progress.New(progress.WithDefaultGradient())

	// Load available providers
	registry := generate.GetProviderRegistry()
	statuses := registry.GetAuthStatus()
	providerOpts := make([]providerOption, 0, len(statuses))

	for _, status := range statuses {
		p := status.Provider
		cost := "Variable"
		free := false

		if p.Pricing.FreeTier {
			cost = fmt.Sprintf("FREE (%s)", p.Pricing.FreeTierLimit)
			free = true
		} else if p.Pricing.CostPerImage != nil {
			cost = fmt.Sprintf("$%.4f/image", *p.Pricing.CostPerImage)
		}

		providerOpts = append(providerOpts, providerOption{
			id:         p.ID,
			name:       p.Name,
			api:        p.API,
			model:      p.ModelID,
			cost:       cost,
			free:       free,
			configured: status.Configured,
			missing:    status.Missing,
		})
	}

	// Size options
	sizeOpts := []sizeOption{
		{"1024x1024", "Square (1024x1024)", "1:1"},
		{"1792x1024", "Landscape (1792x1024)", "16:9"},
		{"1024x1792", "Portrait (1024x1792)", "9:16"},
		{"2048x2048", "Ultra HD (2048x2048)", "1:1"},
		{"custom", "Custom Size", ""},
	}

	// Style options
	styleOpts := []styleOption{
		{"", "None", "No specific style"},
		{"photorealistic", "Photorealistic", "Realistic photography style"},
		{"artistic", "Artistic", "Artistic and painterly style"},
		{"anime", "Anime", "Anime and manga style"},
	}

	// Initialize advanced options inputs
	negativePromptInput := textinput.New()
	negativePromptInput.Placeholder = "e.g., blurry, low quality, watermark"
	negativePromptInput.CharLimit = 500
	negativePromptInput.Width = 60

	seedInput := textinput.New()
	seedInput.Placeholder = "e.g., 12345 (for reproducibility)"
	seedInput.CharLimit = 20
	seedInput.Width = 30

	cfgScaleInput := textinput.New()
	cfgScaleInput.Placeholder = "7.0 (1.0-10.0, Bedrock only)"
	cfgScaleInput.CharLimit = 5
	cfgScaleInput.Width = 20

	countInput := textinput.New()
	countInput.Placeholder = "1 (1-5)"
	countInput.CharLimit = 2
	countInput.Width = 10

	// Aspect ratio options (for Gemini 3 Pro)
	aspectRatioOpts := []aspectRatioOption{
		{"", "Auto (from size)"},
		{"1:1", "Square (1:1)"},
		{"16:9", "Landscape (16:9)"},
		{"9:16", "Portrait (9:16)"},
		{"4:3", "Standard (4:3)"},
		{"3:4", "Portrait (3:4)"},
		{"3:2", "Photo (3:2)"},
		{"2:3", "Portrait Photo (2:3)"},
	}

	// Image size options (for Gemini 3 Pro native resolution)
	imageSizeOpts := []imageSizeOption{
		{"", "Default", "Use standard resolution"},
		{"1K", "1K (~1MP)", "Standard quality"},
		{"2K", "2K (~4MP)", "High quality"},
		{"4K", "4K (~16MP)", "Ultra quality ($0.24/image)"},
	}

	// Output format options
	outputFormatOpts := []outputFormatOption{
		{"", "Auto (PNG)"},
		{"png", "PNG (lossless)"},
		{"jpeg", "JPEG (smaller)"},
		{"webp", "WebP (modern)"},
	}

	// Initialize provider retry input
	providerRetryInput := textinput.New()
	providerRetryInput.Placeholder = "e.g., gemini/flash-2.5, vertex/imagen-4"
	providerRetryInput.CharLimit = 100
	providerRetryInput.Width = 70

	return &GenerateFlowModel{
		currentStep:      StepPrompt,
		promptTextarea:   ta,
		providers:        providerOpts,
		selectedProvider: 0,
		providerRegistry: registry,
		sizes:            sizeOpts,
		selectedSize:     0,
		customWidth:      widthInput,
		customHeight:     heightInput,
		styles:           styleOpts,
		selectedStyle:    0,
		// Advanced options
		negativePromptInput:  negativePromptInput,
		seedInput:            seedInput,
		aspectRatios:         aspectRatioOpts,
		selectedAspectRatio:  0,
		imageSizes:           imageSizeOpts,
		selectedImageSize:    0,
		outputFormats:        outputFormatOpts,
		selectedOutputFormat: 0,
		cfgScaleInput:        cfgScaleInput,
		countInput:           countInput,
		advancedFocusIndex:   0,
		// Other steps
		outputInput:           outputInput,
		progressBar:           prog,
		providerRetryInput:    providerRetryInput,
		retryProviders:        providerOpts,
		selectedRetryProvider: 0,
	}
}

// Init initializes the generate flow
func (m *GenerateFlowModel) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles messages for the generate flow
func (m *GenerateFlowModel) Update(msg tea.Msg) (*GenerateFlowModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global shortcuts
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
			return m, nil
		case "esc":
			if m.showHelp {
				m.showHelp = false
				return m, nil
			}
			// Go back to previous step or main menu
			if m.currentStep > StepPrompt {
				m.currentStep--
				// Reset focus for the previous step
				m.resetFocusForStep()
			} else {
				return m, Navigate(ScreenMainMenu)
			}
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case progressMsg:
		m.progressMsg = string(msg)
		return m, nil

	case generationCompleteMsg:
		m.generating = false
		m.resultPath = msg.path
		m.resultSize = msg.size
		m.generationTime = msg.duration
		m.err = msg.err
		m.lastGenerationError = ""
		if msg.err != nil {
			m.lastGenerationError = msg.err.Error()
			// Go to provider retry step if generation failed
			m.currentStep = StepProviderRetry
			// Log the failure
			logger := logging.GetLogger()
			logger.LogError("Generation failed with provider %s: %v", m.providers[m.selectedProvider].id, msg.err)
		} else {
			m.currentStep = StepResult
		}
		return m, nil
	}

	// Delegate to step-specific handlers
	switch m.currentStep {
	case StepPrompt:
		return m.updatePromptStep(msg)
	case StepProvider:
		return m.updateProviderStep(msg)
	case StepSize:
		return m.updateSizeStep(msg)
	case StepStyle:
		return m.updateStyleStep(msg)
	case StepAdvanced:
		return m.updateAdvancedStep(msg)
	case StepOutput:
		return m.updateOutputStep(msg)
	case StepCommand:
		return m.updateCommandStep(msg)
	case StepProgress:
		// Update progress bar - convert the model back to progress.Model
		var progModel tea.Model
		progModel, cmd = m.progressBar.Update(msg)
		m.progressBar = progModel.(progress.Model)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	case StepProviderRetry:
		return m.updateProviderRetryStep(msg)
	case StepResult:
		return m.updateResultStep(msg)
	}

	return m, tea.Batch(cmds...)
}

// View renders the generate flow
func (m *GenerateFlowModel) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	switch m.currentStep {
	case StepPrompt:
		return m.viewPromptStep()
	case StepProvider:
		return m.viewProviderStep()
	case StepSize:
		return m.viewSizeStep()
	case StepStyle:
		return m.viewStyleStep()
	case StepAdvanced:
		return m.viewAdvancedStep()
	case StepOutput:
		return m.viewOutputStep()
	case StepCommand:
		return m.viewCommandStep()
	case StepProgress:
		return m.viewProgressStep()
	case StepProviderRetry:
		return m.viewProviderRetryStep()
	case StepResult:
		return m.viewResultStep()
	default:
		return "Unknown step"
	}
}

// resetFocusForStep resets focus when going back to a step
func (m *GenerateFlowModel) resetFocusForStep() {
	switch m.currentStep {
	case StepPrompt:
		m.promptTextarea.Focus()
	case StepAdvanced:
		m.advancedFocusIndex = 0
		m.negativePromptInput.Focus()
	case StepOutput:
		m.outputInput.Focus()
	}
}

// Step 1: Prompt Input
func (m *GenerateFlowModel) updatePromptStep(msg tea.Msg) (*GenerateFlowModel, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			// Enter submits (move to next step) if prompt is not empty
			if len(strings.TrimSpace(m.promptTextarea.Value())) > 0 {
				m.currentStep = StepProvider
				return m, nil
			}
		case "shift+enter":
			// Shift+Enter inserts a newline
			currentValue := m.promptTextarea.Value()
			m.promptTextarea.SetValue(currentValue + "\n")
			return m, nil
		}
	}

	// For all other keys, let the textarea handle it
	m.promptTextarea, cmd = m.promptTextarea.Update(msg)
	return m, cmd
}

func (m *GenerateFlowModel) viewPromptStep() string {
	charCount := len(m.promptTextarea.Value())
	charLimit := m.promptTextarea.CharLimit

	content := TitleStyle.Render("Generate Image - Step 1/8") + "\n\n" +
		SubtitleStyle.Render("Describe the image you want to generate") + "\n\n" +
		m.promptTextarea.View() + "\n\n" +
		MutedStyle.Render(fmt.Sprintf("Characters: %d/%d", charCount, charLimit)) + "\n\n" +
		HelpStyle.Render("Enter: Continue • Shift+Enter: New line • Esc: Cancel • ?: Help")

	box := FocusedBoxStyle.Width(76).Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// Step 2: Model Selection
func (m *GenerateFlowModel) updateProviderStep(msg tea.Msg) (*GenerateFlowModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.selectedProvider > 0 {
				m.selectedProvider--
			}
		case "down", "j":
			if m.selectedProvider < len(m.providers)-1 {
				m.selectedProvider++
			}
		case "enter", " ":
			// Check if provider is configured
			provider := m.providers[m.selectedProvider]
			if !provider.configured {
				m.err = fmt.Errorf("provider '%s' is not configured\nMissing: %s\nRun: gimage auth setup %s",
					provider.id, strings.Join(provider.missing, ", "), provider.id)
				return m, nil
			}
			// Move to next step
			m.currentStep = StepSize
			return m, nil
		}
	}
	return m, nil
}

func (m *GenerateFlowModel) viewProviderStep() string {
	var items []string

	for i, provider := range m.providers {
		var style lipgloss.Style
		cursor := "  "
		if i == m.selectedProvider {
			style = SelectedMenuItemStyle
			cursor = "> "
		} else {
			style = MenuItemStyle
		}

		// Show badges
		badges := ""
		if provider.free {
			badges += SuccessStyle.Render(" [FREE]")
		}
		if !provider.configured {
			badges += ErrorStyle.Render(" [NOT CONFIGURED]")
		}

		// Format the provider info
		title := style.Render(cursor + provider.name + badges)
		desc := MutedStyle.Render("  Provider: " + provider.id + " | API: " + provider.api)
		cost := MutedStyle.Render("  Cost: " + provider.cost)

		items = append(items, title+"\n"+desc+"\n"+cost)
	}

	content := TitleStyle.Render("Generate Image - Step 2/8") + "\n\n" +
		SubtitleStyle.Render("Select Provider") + "\n\n" +
		strings.Join(items, "\n\n") + "\n\n" +
		HelpStyle.Render("↑/↓: Navigate • Enter: Select • Esc: Back")

	box := FocusedBoxStyle.Width(76).Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// Step 3: Size Selection
func (m *GenerateFlowModel) updateSizeStep(msg tea.Msg) (*GenerateFlowModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// If custom size is active, handle input
		if m.useCustom {
			switch keyMsg.String() {
			case "tab":
				// Toggle focus between width and height
				if m.customWidth.Focused() {
					m.customWidth.Blur()
					m.customHeight.Focus()
				} else {
					m.customHeight.Blur()
					m.customWidth.Focus()
				}
				return m, nil
			case "enter":
				// Validate and move to next step
				if m.customWidth.Value() != "" && m.customHeight.Value() != "" {
					m.currentStep = StepStyle
					return m, nil
				}
			case "esc":
				m.useCustom = false
				m.customWidth.Blur()
				m.customHeight.Blur()
				return m, nil
			}

			// Update inputs
			if m.customWidth.Focused() {
				m.customWidth, cmd = m.customWidth.Update(msg)
				cmds = append(cmds, cmd)
			} else {
				m.customHeight, cmd = m.customHeight.Update(msg)
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Normal navigation
		switch keyMsg.String() {
		case "up", "k":
			if m.selectedSize > 0 {
				m.selectedSize--
			}
		case "down", "j":
			if m.selectedSize < len(m.sizes)-1 {
				m.selectedSize++
			}
		case "enter", " ":
			// Check if custom size is selected
			if m.sizes[m.selectedSize].size == "custom" {
				m.useCustom = true
				m.customWidth.Focus()
				return m, textinput.Blink
			}
			m.currentStep = StepStyle
			return m, nil
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *GenerateFlowModel) viewSizeStep() string {
	var items []string

	if !m.useCustom {
		for i, size := range m.sizes {
			var style lipgloss.Style
			cursor := "  "
			if i == m.selectedSize {
				style = SelectedMenuItemStyle
				cursor = "> "
			} else {
				style = MenuItemStyle
			}

			title := style.Render(cursor + size.label)
			desc := ""
			if size.aspect != "" {
				desc = MutedStyle.Render("  Aspect ratio: " + size.aspect)
			}

			if desc != "" {
				items = append(items, title+"\n"+desc)
			} else {
				items = append(items, title)
			}
		}

		content := TitleStyle.Render("Generate Image - Step 3/8") + "\n\n" +
			SubtitleStyle.Render("Select Image Size") + "\n\n" +
			strings.Join(items, "\n\n") + "\n\n" +
			HelpStyle.Render("↑/↓: Navigate • Enter: Select • Esc: Back")

		box := FocusedBoxStyle.Width(76).Render(content)

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			box,
		)
	}

	// Custom size input view
	content := TitleStyle.Render("Generate Image - Step 3/8") + "\n\n" +
		SubtitleStyle.Render("Enter Custom Size") + "\n\n" +
		FormatKeyValue("Width", m.customWidth.View()) + "\n\n" +
		FormatKeyValue("Height", m.customHeight.View()) + "\n\n" +
		WarningStyle.Render("Note: Max size is 2048x2048 for most models") + "\n\n" +
		HelpStyle.Render("Tab: Switch field • Enter: Continue • Esc: Back")

	box := FocusedBoxStyle.Width(76).Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// Step 4: Style Selection
func (m *GenerateFlowModel) updateStyleStep(msg tea.Msg) (*GenerateFlowModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.selectedStyle > 0 {
				m.selectedStyle--
			}
		case "down", "j":
			if m.selectedStyle < len(m.styles)-1 {
				m.selectedStyle++
			}
		case "enter", " ":
			// Go to Advanced Options step
			m.currentStep = StepAdvanced
			m.advancedFocusIndex = 0
			m.negativePromptInput.Focus()
			return m, textinput.Blink
		}
	}
	return m, nil
}

func (m *GenerateFlowModel) viewStyleStep() string {
	var items []string

	for i, style := range m.styles {
		var itemStyle lipgloss.Style
		cursor := "  "
		if i == m.selectedStyle {
			itemStyle = SelectedMenuItemStyle
			cursor = "> "
		} else {
			itemStyle = MenuItemStyle
		}

		title := itemStyle.Render(cursor + style.label)
		desc := MutedStyle.Render("  " + style.desc)

		items = append(items, title+"\n"+desc)
	}

	content := TitleStyle.Render("Generate Image - Step 4/8") + "\n\n" +
		SubtitleStyle.Render("Select Image Style (Optional)") + "\n\n" +
		strings.Join(items, "\n\n") + "\n\n" +
		HelpStyle.Render("↑/↓: Navigate • Enter: Select • Esc: Back")

	box := FocusedBoxStyle.Width(76).Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// Step 5: Advanced Options
func (m *GenerateFlowModel) updateAdvancedStep(msg tea.Msg) (*GenerateFlowModel, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "tab", "down", "j":
			// Move to next field
			m.blurAllAdvancedInputs()
			m.advancedFocusIndex = (m.advancedFocusIndex + 1) % 7
			m.focusAdvancedInput()
			return m, textinput.Blink
		case "shift+tab", "up", "k":
			// Move to previous field
			m.blurAllAdvancedInputs()
			m.advancedFocusIndex = (m.advancedFocusIndex - 1 + 7) % 7
			m.focusAdvancedInput()
			return m, textinput.Blink
		case "enter":
			// Move to output step
			m.blurAllAdvancedInputs()
			m.currentStep = StepOutput
			m.outputInput.Focus()
			return m, textinput.Blink
		case "left", "h":
			// Navigate picker options left
			if m.advancedFocusIndex == 2 && m.selectedAspectRatio > 0 {
				m.selectedAspectRatio--
			} else if m.advancedFocusIndex == 3 && m.selectedImageSize > 0 {
				m.selectedImageSize--
			} else if m.advancedFocusIndex == 4 && m.selectedOutputFormat > 0 {
				m.selectedOutputFormat--
			}
			return m, nil
		case "right", "l":
			// Navigate picker options right
			if m.advancedFocusIndex == 2 && m.selectedAspectRatio < len(m.aspectRatios)-1 {
				m.selectedAspectRatio++
			} else if m.advancedFocusIndex == 3 && m.selectedImageSize < len(m.imageSizes)-1 {
				m.selectedImageSize++
			} else if m.advancedFocusIndex == 4 && m.selectedOutputFormat < len(m.outputFormats)-1 {
				m.selectedOutputFormat++
			}
			return m, nil
		}
	}

	// Update the focused input
	switch m.advancedFocusIndex {
	case 0:
		m.negativePromptInput, cmd = m.negativePromptInput.Update(msg)
	case 1:
		m.seedInput, cmd = m.seedInput.Update(msg)
	case 5:
		m.cfgScaleInput, cmd = m.cfgScaleInput.Update(msg)
	case 6:
		m.countInput, cmd = m.countInput.Update(msg)
	}

	return m, cmd
}

func (m *GenerateFlowModel) blurAllAdvancedInputs() {
	m.negativePromptInput.Blur()
	m.seedInput.Blur()
	m.cfgScaleInput.Blur()
	m.countInput.Blur()
}

func (m *GenerateFlowModel) focusAdvancedInput() {
	switch m.advancedFocusIndex {
	case 0:
		m.negativePromptInput.Focus()
	case 1:
		m.seedInput.Focus()
	case 5:
		m.cfgScaleInput.Focus()
	case 6:
		m.countInput.Focus()
		// 2, 3, 4 are pickers - no focus needed
	}
}

func (m *GenerateFlowModel) viewAdvancedStep() string {
	// Get provider info to show relevant options
	provider := m.providers[m.selectedProvider]
	isGemini3Pro := strings.Contains(provider.model, "gemini-3")
	isBedrock := provider.api == "bedrock"
	isVertex := provider.api == "vertex"

	var content string
	content = TitleStyle.Render("Generate Image - Step 5/8") + "\n\n" +
		SubtitleStyle.Render("Advanced Options (Optional)") + "\n\n"

	// Negative Prompt (all providers)
	focusIndicator := "  "
	if m.advancedFocusIndex == 0 {
		focusIndicator = "> "
	}
	content += focusIndicator + FormatKeyValue("Negative Prompt", m.negativePromptInput.View()) + "\n"
	content += MutedStyle.Render("    Describe what you DON'T want in the image") + "\n\n"

	// Seed (all providers)
	focusIndicator = "  "
	if m.advancedFocusIndex == 1 {
		focusIndicator = "> "
	}
	content += focusIndicator + FormatKeyValue("Seed", m.seedInput.View()) + "\n"
	content += MutedStyle.Render("    Use same seed for reproducible results") + "\n\n"

	// Aspect Ratio (Gemini 3 Pro only)
	focusIndicator = "  "
	if m.advancedFocusIndex == 2 {
		focusIndicator = "> "
	}
	aspectRatioValue := m.aspectRatios[m.selectedAspectRatio].label
	if isGemini3Pro {
		content += focusIndicator + FormatKeyValue("Aspect Ratio", aspectRatioValue) + "\n"
		content += MutedStyle.Render("    ←/→ to change (Gemini 3 Pro only)") + "\n\n"
	} else {
		content += MutedStyle.Render("  Aspect Ratio: (N/A - Gemini 3 Pro only)") + "\n\n"
	}

	// Image Size (Gemini 3 Pro native upscaling)
	focusIndicator = "  "
	if m.advancedFocusIndex == 3 {
		focusIndicator = "> "
	}
	imageSizeValue := m.imageSizes[m.selectedImageSize].label
	if isGemini3Pro {
		content += focusIndicator + FormatKeyValue("Native Resolution", imageSizeValue) + "\n"
		content += MutedStyle.Render("    ←/→ to change: 1K, 2K, or 4K") + "\n\n"
	} else {
		content += MutedStyle.Render("  Native Resolution: (N/A - Gemini 3 Pro only)") + "\n\n"
	}

	// Output Format (Vertex AI only)
	focusIndicator = "  "
	if m.advancedFocusIndex == 4 {
		focusIndicator = "> "
	}
	outputFormatValue := m.outputFormats[m.selectedOutputFormat].label
	if isVertex {
		content += focusIndicator + FormatKeyValue("Output Format", outputFormatValue) + "\n"
		content += MutedStyle.Render("    ←/→ to change (Vertex AI only)") + "\n\n"
	} else {
		content += MutedStyle.Render("  Output Format: (N/A - Vertex AI only)") + "\n\n"
	}

	// CFG Scale (Bedrock only)
	focusIndicator = "  "
	if m.advancedFocusIndex == 5 {
		focusIndicator = "> "
	}
	if isBedrock {
		content += focusIndicator + FormatKeyValue("CFG Scale", m.cfgScaleInput.View()) + "\n"
		content += MutedStyle.Render("    1.0-10.0: higher = follows prompt more (Bedrock only)") + "\n\n"
	} else {
		content += MutedStyle.Render("  CFG Scale: (N/A - Bedrock only)") + "\n\n"
	}

	// Number of Images (all providers)
	focusIndicator = "  "
	if m.advancedFocusIndex == 6 {
		focusIndicator = "> "
	}
	content += focusIndicator + FormatKeyValue("Count", m.countInput.View()) + "\n"
	content += MutedStyle.Render("    Number of images to generate (1-5)") + "\n\n"

	content += HelpStyle.Render("Tab/↑↓: Navigate • ←→: Change picker • Enter: Continue • Esc: Back")

	box := FocusedBoxStyle.Width(76).Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// Step 6: Output Path
func (m *GenerateFlowModel) updateOutputStep(msg tea.Msg) (*GenerateFlowModel, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			// Build and show command preview
			m.buildCommand()
			m.currentStep = StepCommand
			return m, nil
		}
	}

	m.outputInput, cmd = m.outputInput.Update(msg)
	return m, cmd
}

func (m *GenerateFlowModel) viewOutputStep() string {
	// Set default path if empty
	if m.outputInput.Value() == "" {
		home, _ := os.UserHomeDir()
		timestamp := time.Now().Format("20060102_150405")
		defaultPath := filepath.Join(home, "Desktop", fmt.Sprintf("gimage_%s.png", timestamp))
		m.outputInput.SetValue(defaultPath)
	}

	content := TitleStyle.Render("Generate Image - Step 6/8") + "\n\n" +
		SubtitleStyle.Render("Specify Output Path") + "\n\n" +
		"Output file: " + m.outputInput.View() + "\n\n" +
		MutedStyle.Render("The image will be saved to this location") + "\n\n" +
		HelpStyle.Render("Enter: Generate • Esc: Back")

	box := FocusedBoxStyle.Width(76).Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// Step 7: Command Preview
func (m *GenerateFlowModel) buildCommand() {
	// Build the size string
	var size string
	if m.useCustom {
		size = fmt.Sprintf("%sx%s", m.customWidth.Value(), m.customHeight.Value())
	} else {
		size = m.sizes[m.selectedSize].size
	}

	// Quote the prompt properly for shell
	prompt := strings.ReplaceAll(m.promptTextarea.Value(), "\"", "\\\"")

	// Build command
	cmdParts := []string{
		fmt.Sprintf("gimage generate \"%s\"", prompt),
		fmt.Sprintf("--provider %s", m.providers[m.selectedProvider].id),
		fmt.Sprintf("--size %s", size),
	}

	// Add style if not "None"
	if m.styles[m.selectedStyle].value != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--style %s", m.styles[m.selectedStyle].value))
	}

	// Add advanced options
	if m.negativePromptInput.Value() != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--negative \"%s\"", m.negativePromptInput.Value()))
	}
	if m.seedInput.Value() != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--seed %s", m.seedInput.Value()))
	}
	if m.aspectRatios[m.selectedAspectRatio].value != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--aspect-ratio %s", m.aspectRatios[m.selectedAspectRatio].value))
	}
	if m.imageSizes[m.selectedImageSize].value != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--image-size %s", m.imageSizes[m.selectedImageSize].value))
	}
	if m.outputFormats[m.selectedOutputFormat].value != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--output-format %s", m.outputFormats[m.selectedOutputFormat].value))
	}
	if m.cfgScaleInput.Value() != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--cfg-scale %s", m.cfgScaleInput.Value()))
	}
	if m.countInput.Value() != "" && m.countInput.Value() != "1" {
		cmdParts = append(cmdParts, fmt.Sprintf("--count %s", m.countInput.Value()))
	}

	// Add output path
	cmdParts = append(cmdParts, fmt.Sprintf("--output %s", m.outputInput.Value()))

	m.commandStr = strings.Join(cmdParts, " \\\n  ")
}

func (m *GenerateFlowModel) updateCommandStep(msg tea.Msg) (*GenerateFlowModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			// Proceed to generation
			return m, m.startGeneration()
		case "c":
			// Copy command to clipboard (note: might not work in all terminals)
			// For now, we'll just show the command and let the user copy it manually
			return m, nil
		}
	}
	return m, nil
}

func (m *GenerateFlowModel) viewCommandStep() string {
	// Display provider info
	provider := m.providers[m.selectedProvider]
	apiDisplay := strings.ToUpper(provider.api)

	content := TitleStyle.Render("Generate Image - Step 7/8") + "\n\n" +
		SubtitleStyle.Render("Verify Your Command") + "\n\n" +
		FormatKeyValue("Provider", provider.name) + "\n" +
		FormatKeyValue("API", apiDisplay) + "\n" +
		FormatKeyValue("Model", provider.model) + "\n\n" +
		MutedStyle.Render("Equivalent CLI Command:") + "\n\n" +
		CodeBlockStyle.Render(m.commandStr) + "\n\n" +
		WarningStyle.Render("You can copy this command and run it manually:") + "\n" +
		MutedStyle.Render("  $ "+strings.ReplaceAll(m.commandStr, "\n", "\n  $ ")) + "\n\n" +
		HelpStyle.Render("Enter: Generate • Esc: Back")

	box := FocusedBoxStyle.Width(76).Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// Step 7: Progress
func (m *GenerateFlowModel) viewProgressStep() string {
	spinner := SpinnerFrames[int(time.Now().Unix())%len(SpinnerFrames)]

	content := TitleStyle.Render("Generate Image - Step 8/8") + "\n\n" +
		SubtitleStyle.Render("Generating...") + "\n\n" +
		SuccessStyle.Render(spinner) + " " + m.progressMsg + "\n\n" +
		m.progressBar.View() + "\n\n" +
		MutedStyle.Render("Please wait while your image is being generated...")

	box := FocusedBoxStyle.Width(76).Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// Step 7: Result
func (m *GenerateFlowModel) updateResultStep(msg tea.Msg) (*GenerateFlowModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "r":
			// Retry - restore previous settings and go back to step 1
			if m.err != nil {
				m.promptTextarea.SetValue(m.errorContext.prompt)
				m.err = nil
				m.showErrorDetails = false
				m.currentStep = StepPrompt
				m.promptTextarea.Focus()
				return m, textarea.Blink
			}
		case "d":
			// Toggle error details
			if m.err != nil {
				m.showErrorDetails = !m.showErrorDetails
			}
		case "g":
			// Generate another image - reset to step 1
			return NewGenerateFlowModel(), textarea.Blink
		case "m":
			// Go to main menu
			return m, Navigate(ScreenMainMenu)
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *GenerateFlowModel) viewResultStep() string {
	if m.err != nil {
		var content string

		if m.showErrorDetails {
			// Show detailed error information
			content = TitleStyle.Render("Generation Failed - Error Details") + "\n\n" +
				ErrorStyle.Render("Error: "+m.err.Error()) + "\n\n" +
				SubtitleStyle.Render("Generation Parameters") + "\n\n" +
				FormatKeyValue("Prompt", truncateString(m.errorContext.prompt, 60)) + "\n" +
				FormatKeyValue("Model", m.errorContext.model) + "\n" +
				FormatKeyValue("Size", m.errorContext.size) + "\n" +
				FormatKeyValue("Style", m.errorContext.style) + "\n" +
				FormatKeyValue("Output", m.errorContext.output) + "\n\n" +
				WarningStyle.Render("Troubleshooting Tips:") + "\n" +
				"• Check your API credentials in Settings\n" +
				"• Verify the model name is correct\n" +
				"• Try a different model or size\n" +
				"• Check your internet connection\n\n" +
				HelpStyle.Render("r: Retry with same settings • d: Hide details • g: New image • m: Main menu")
		} else {
			// Show simple error message
			content = TitleStyle.Render("Generation Failed") + "\n\n" +
				ErrorStyle.Render("Error: "+m.err.Error()) + "\n\n" +
				SubtitleStyle.Render("Command that failed:") + "\n\n" +
				FormatKeyValue("Model", m.errorContext.model) + "\n" +
				FormatKeyValue("Size", m.errorContext.size) + "\n\n" +
				HelpStyle.Render("r: Retry with same settings • d: Show details • g: New image • m: Main menu")
		}

		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorError).
			Padding(1, 2).
			Width(80).
			Render(content)

		return lipgloss.Place(
			m.width,
			m.height,
			lipgloss.Center,
			lipgloss.Center,
			box,
		)
	}

	content := TitleStyle.Render("Generation Complete!") + "\n\n" +
		SuccessStyle.Render("✓ Image generated successfully") + "\n\n" +
		FormatKeyValue("Saved to", m.resultPath) + "\n" +
		FormatKeyValue("File size", FormatFileSize(m.resultSize)) + "\n" +
		FormatKeyValue("Time taken", m.generationTime.String()) + "\n\n" +
		HelpStyle.Render("g: Generate another • m: Main menu • q: Quit")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSuccess).
		Padding(1, 2).
		Width(76).
		Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// renderHelp renders the help screen
func (m *GenerateFlowModel) renderHelp() string {
	helpContent := TitleStyle.Render("Keyboard Shortcuts") + "\n\n" +
		FormatKeyValue("↑/k, ↓/j", "Navigate options") + "\n" +
		FormatKeyValue("Enter/Space", "Select option") + "\n" +
		FormatKeyValue("Enter", "Submit prompt (continue to next step)") + "\n" +
		FormatKeyValue("Shift+Enter", "New line in prompt") + "\n" +
		FormatKeyValue("Tab", "Switch input fields") + "\n" +
		FormatKeyValue("Esc", "Go back") + "\n" +
		FormatKeyValue("?", "Toggle help") + "\n" +
		FormatKeyValue("Ctrl+C", "Quit") + "\n\n" +
		SubtitleStyle.Render("Generation Workflow") + "\n\n" +
		"1. Enter a detailed prompt\n" +
		"2. Choose an AI provider (Gemini is free!)\n" +
		"3. Select image size\n" +
		"4. Pick a style (optional)\n" +
		"5. Advanced options (negative prompt, seed, etc.)\n" +
		"6. Specify output path\n" +
		"7. Preview command\n" +
		"8. Watch the progress\n\n" +
		HelpStyle.Render("Press Esc to close this help")

	box := FocusedBoxStyle.Width(70).Render(helpContent)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

// Step 8: Model Retry
func (m *GenerateFlowModel) updateProviderRetryStep(msg tea.Msg) (*GenerateFlowModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			// Navigate up in provider list
			if m.selectedRetryProvider > 0 {
				m.selectedRetryProvider--
			}
			return m, nil

		case "down", "j":
			// Navigate down in provider list
			if m.selectedRetryProvider < len(m.retryProviders) {
				m.selectedRetryProvider++
			}
			return m, nil

		case "enter":
			if m.selectedRetryProvider < len(m.retryProviders) {
				// Selected a provider from the grid
				selectedProvider := m.retryProviders[m.selectedRetryProvider]

				// Find the index in the main providers list
				for i, p := range m.providers {
					if p.id == selectedProvider.id {
						m.selectedProvider = i
						break
					}
				}

				// Log the retry
				logger := logging.GetLogger()
				logger.LogInfo("User selected provider %s for retry after previous failure", selectedProvider.name)

				// Rebuild command and go to generation
				m.buildCommand()
				m.currentStep = StepCommand
				return m, nil
			} else if m.showCustomProviderInput {
				// User entered a custom provider ID
				customProviderID := m.providerRetryInput.Value()
				if customProviderID != "" {
					// Try to resolve the provider
					if provider, err := m.providerRegistry.ResolveProvider(customProviderID); err == nil {
						// Check if provider is configured
						hasAuth, _, _ := m.providerRegistry.CheckAuth(provider)
						if !hasAuth {
							m.err = fmt.Errorf("provider '%s' is not configured", provider.ID)
							return m, nil
						}

						// Create a temporary provider option
						cost := "Variable"
						free := false
						if provider.Pricing.FreeTier {
							cost = fmt.Sprintf("FREE (%s)", provider.Pricing.FreeTierLimit)
							free = true
						} else if provider.Pricing.CostPerImage != nil {
							cost = fmt.Sprintf("$%.4f/image", *provider.Pricing.CostPerImage)
						}

						tempProvider := providerOption{
							id:         provider.ID,
							name:       provider.Name,
							api:        provider.API,
							model:      provider.ModelID,
							cost:       cost,
							free:       free,
							configured: hasAuth,
						}

						// Add to providers list if not already there
						found := false
						for i, p := range m.providers {
							if p.id == provider.ID {
								found = true
								m.selectedProvider = i
								break
							}
						}

						if !found {
							m.providers = append(m.providers, tempProvider)
							m.selectedProvider = len(m.providers) - 1
						}

						// Log the custom provider selection
						logger := logging.GetLogger()
						logger.LogInfo("User selected custom provider: %s", customProviderID)

						// Rebuild command and go to generation
						m.buildCommand()
						m.currentStep = StepCommand
						return m, nil
					} else {
						logger := logging.GetLogger()
						logger.LogError("Failed to resolve custom provider: %s", customProviderID)
					}
				}
			}

		case "c":
			// Show custom model input
			m.showCustomProviderInput = !m.showCustomProviderInput
			if m.showCustomProviderInput {
				m.providerRetryInput.Focus()
				m.providerRetryInput.SetValue("")
			}
			return m, nil

		case "esc":
			// Go back to result step
			m.currentStep = StepResult
			return m, nil
		}
	}

	// If custom input is active, let the input handle it
	if m.showCustomProviderInput {
		var cmd tea.Cmd
		m.providerRetryInput, cmd = m.providerRetryInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m *GenerateFlowModel) viewProviderRetryStep() string {
	// Build provider grid
	gridContent := m.renderProviderGrid()

	content := TitleStyle.Render("Generation Failed - Select Another Provider") + "\n\n" +
		ErrorStyle.Render("✗ "+m.lastGenerationError) + "\n\n" +
		SubtitleStyle.Render("Choose a different provider to retry:") + "\n\n" +
		gridContent + "\n\n"

	if m.showCustomProviderInput {
		content += SubtitleStyle.Render("Or enter a custom provider ID:") + "\n" +
			"Provider: " + m.providerRetryInput.View() + "\n\n" +
			MutedStyle.Render("(press Enter to submit, Esc to cancel)") + "\n\n"
	} else {
		content += HelpStyle.Render("↑/k: Up • ↓/j: Down • Enter: Select • c: Custom Provider • Esc: Back")
	}

	box := FocusedBoxStyle.Width(80).Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		box,
	)
}

func (m *GenerateFlowModel) renderProviderGrid() string {
	// Create a table showing providers with their details
	rows := []string{}

	// Add column headers
	headerRow := fmt.Sprintf(
		"%-4s | %-25s | %-12s | %-10s | %s",
		"  #", "Provider", "Price", "Auth", "API",
	)
	rows = append(rows, HeaderStyle.Render(headerRow))
	rows = append(rows, strings.Repeat("─", 80))

	// Add provider rows
	for i, provider := range m.retryProviders {
		// Auth status
		authStatus := "✗"
		if provider.configured {
			authStatus = "✓"
		}

		// Mark selected row
		prefix := "  "
		if i == m.selectedRetryProvider && !m.showCustomProviderInput {
			prefix = "→ "
		}

		// Format provider info
		displayName := provider.name
		if len(displayName) > 25 {
			displayName = displayName[:22] + "..."
		}

		rowStr := fmt.Sprintf(
			"%s%d | %-25s | %-12s | %-10s | %s",
			prefix,
			i+1,
			displayName,
			provider.cost,
			authStatus,
			provider.api,
		)

		rows = append(rows, rowStr)
	}

	return strings.Join(rows, "\n")
}

// Helper function to resolve custom model names
func resolveCustomModelName(input string) (string, error) {
	registry := generate.GetProviderRegistry()

	// Try to resolve as provider ID or alias
	provider, err := registry.ResolveProvider(input)
	if err == nil {
		return provider.ModelID, nil
	}

	// Try alias resolution
	if resolvedName := generate.ResolveModelName(input); resolvedName != input {
		if provider, err := registry.ResolveProvider(resolvedName); err == nil {
			return provider.ModelID, nil
		}
	}

	return "", fmt.Errorf("unknown model: %s", input)
}

// buildCLICommand builds the equivalent CLI command for logging and reproducibility
func (m *GenerateFlowModel) buildCLICommand(prompt string, model string, size string, style string, output string) string {
	// Quote the prompt properly for shell
	quotedPrompt := strings.ReplaceAll(prompt, "\"", "\\\"")

	// Build command
	cmdParts := []string{
		fmt.Sprintf("gimage generate \"%s\"", quotedPrompt),
		fmt.Sprintf("--model %s", model),
		fmt.Sprintf("--size %s", size),
	}

	// Add style if not "None"
	if style != "" {
		cmdParts = append(cmdParts, fmt.Sprintf("--style %s", style))
	}

	// Add output path
	cmdParts = append(cmdParts, fmt.Sprintf("--output %s", output))

	return strings.Join(cmdParts, " \\\n  ")
}

// HeaderStyle for table headers
var HeaderStyle = lipgloss.NewStyle().
	Foreground(ColorSecondary).
	Bold(true)

// startGeneration begins the image generation process
func (m *GenerateFlowModel) startGeneration() tea.Cmd {
	m.currentStep = StepProgress
	m.generating = true
	m.progressMsg = "Initializing..."

	return tea.Batch(
		m.tickProgress(),
		m.generateImageCmd(),
	)
}

// tickProgress updates the progress display
func (m *GenerateFlowModel) tickProgress() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// generateImageCmd performs the actual image generation
func (m *GenerateFlowModel) generateImageCmd() tea.Cmd {
	return func() tea.Msg {
		startTime := time.Now()
		logger := logging.GetLogger()

		// Build the size string
		var size string
		if m.useCustom {
			size = fmt.Sprintf("%sx%s", m.customWidth.Value(), m.customHeight.Value())
		} else {
			size = m.sizes[m.selectedSize].size
		}

		// Get selected provider
		provider := m.providers[m.selectedProvider]
		logger.LogDebug("TUI: Selected provider index=%d, id=%q, name=%q",
			m.selectedProvider, provider.id, provider.name)

		// Parse advanced options
		var seed int64
		if m.seedInput.Value() != "" {
			fmt.Sscanf(m.seedInput.Value(), "%d", &seed)
		}

		var cfgScale float64
		if m.cfgScaleInput.Value() != "" {
			fmt.Sscanf(m.cfgScaleInput.Value(), "%f", &cfgScale)
		}

		var count int
		if m.countInput.Value() != "" {
			fmt.Sscanf(m.countInput.Value(), "%d", &count)
		}

		// Build options with advanced parameters
		options := models.GenerateOptions{
			Model:          provider.model,
			Size:           size,
			Style:          m.styles[m.selectedStyle].value,
			NegativePrompt: m.negativePromptInput.Value(),
			Seed:           seed,
			AspectRatio:    m.aspectRatios[m.selectedAspectRatio].value,
			ImageSize:      m.imageSizes[m.selectedImageSize].value,
			OutputFormat:   m.outputFormats[m.selectedOutputFormat].value,
			CfgScale:       cfgScale,
			NumberOfImages: count,
		}

		logger.LogDebug("TUI: Options built - Provider=%q, Model=%q, Size=%q, Style=%q, Seed=%d",
			provider.id, options.Model, options.Size, options.Style, seed)

		// Save error context for potential retry
		m.errorContext.prompt = m.promptTextarea.Value()
		m.errorContext.model = provider.name
		m.errorContext.size = size
		m.errorContext.style = m.styles[m.selectedStyle].label
		m.errorContext.output = m.outputInput.Value()

		// Build equivalent CLI command for logging and reproducibility
		cliCommand := fmt.Sprintf("gimage generate \"%s\" --provider %s --size %s",
			strings.ReplaceAll(m.promptTextarea.Value(), "\"", "\\\""), provider.id, size)
		if options.Style != "" {
			cliCommand += fmt.Sprintf(" --style %s", options.Style)
		}
		cliCommand += fmt.Sprintf(" --output %s", m.outputInput.Value())

		// Log generation start
		logger.LogGenerateStart(
			m.promptTextarea.Value(),
			provider.model,
			provider.api,
			size,
			options.Style,
			m.outputInput.Value(),
		)
		logger.LogGenerateCommand(cliCommand)

		// Send progress updates
		go func() {
			time.Sleep(500 * time.Millisecond)
			// Note: In real implementation, we'd have proper progress channels
		}()

		// Use the provider system to generate
		ctx := context.Background()
		logger.LogDebug("TUI: Using provider %q to generate", provider.id)

		// Create client using the provider
		client, err := m.providerRegistry.CreateClient(provider.id)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to create client: %v", err)
			logger.LogError("%s", errMsg)
			return generationCompleteMsg{err: fmt.Errorf("%s", errMsg)}
		}
		defer client.Close()

		// Generate the image
		results, err := client.GenerateImage(ctx, m.promptTextarea.Value(), options)
		if err != nil {
			errMsg := fmt.Sprintf("Generation failed: %v", err)
			logger.LogError("%s", errMsg)
			logger.LogErrorContext("Generation Error", err, map[string]string{
				"model":  options.Model,
				"size":   size,
				"prompt": m.promptTextarea.Value(),
			})
			return generationCompleteMsg{err: fmt.Errorf("%s", errMsg)}
		}

		logger.LogInfo("Image generated successfully")

		// Save the generated image(s) to disk
		var (
			outputPath string // Final path for the primary image (for TUI display)
			saveErr    error
		)

		if len(results) == 0 {
			saveErr = fmt.Errorf("no images returned by provider")
		} else {
			// Determine base output path
			baseOutputPath := m.outputInput.Value()
			if baseOutputPath == "" {
				baseOutputPath = generate.GenerateOutputPath(results[0].Format)
			}

			// Save all images
			for i, img := range results {
				currentOutputPath := baseOutputPath
				if len(results) > 1 {
					// For multiple images, append _N before the extension
					ext := strings.ToLower(img.Format)
					if !strings.HasPrefix(ext, ".") {
						ext = "." + ext
					}
					base := strings.TrimSuffix(baseOutputPath, filepath.Ext(baseOutputPath))
					currentOutputPath = fmt.Sprintf("%s_%d%s", base, i+1, ext)
				}

				if err := generate.SaveImage(img, currentOutputPath); err != nil {
					saveErr = fmt.Errorf("failed to save image %d: %w", i+1, err)
					break
				}

				// The first image saved is considered the primary for TUI display
				if i == 0 {
					outputPath = currentOutputPath
				}
			}
		}

		if saveErr != nil {
			errMsg := fmt.Sprintf("failed to save image(s): %v", saveErr)
			logger.LogError("%s", errMsg)
			logger.LogErrorContext("Image Save Error", saveErr, map[string]string{
				"output_path": outputPath, // May be empty if save failed early
				"model":       options.Model,
			})
			return generationCompleteMsg{err: fmt.Errorf("%s", errMsg)}
		}

		// Get file info
		fileInfo, err := os.Stat(outputPath)
		if err != nil {
			errMsg := fmt.Sprintf("image saved but couldn't stat file: %v", err)
			logger.LogError("%s", errMsg)
			return generationCompleteMsg{
				path:     outputPath,
				duration: time.Since(startTime),
				err:      fmt.Errorf("%s", errMsg),
			}
		}

		// Log successful completion
		duration := time.Since(startTime)
		logger.LogGenerateComplete(true, outputPath, fileInfo.Size(), duration, "")
		logger.LogInfo("Image generation completed successfully in %s", duration.String())

		return generationCompleteMsg{
			path:     outputPath,
			size:     fileInfo.Size(),
			duration: duration,
		}
	}
}

// Custom messages
type tickMsg time.Time
type progressMsg string

type generationCompleteMsg struct {
	path     string
	size     int64
	duration time.Duration
	err      error
}

// truncateString truncates a string to maxLen characters, adding "..." if truncated
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

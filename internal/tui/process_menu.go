// Package tui provides Terminal User Interface components for gimage.
package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/apresai/gimage/internal/imaging"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// ProcessStep represents a step in the image processing workflow
type ProcessStep int

const (
	ProcessStepSelectFile ProcessStep = iota
	ProcessStepSelectOperation
	ProcessStepConfigure
	ProcessStepProcessing
	ProcessStepResult
)

// ProcessOperation represents available image processing operations
type ProcessOperation int

const (
	OpResize ProcessOperation = iota
	OpScale
	OpCrop
	OpCompress
	OpConvert
)

// ProcessMenuModel handles the image processing workflow
type ProcessMenuModel struct {
	currentStep ProcessStep
	width       int
	height      int

	// File selection
	filePicker   *FilePicker
	selectedFile FileInfo
	fileError    error

	// Operation selection
	operations      []string
	selectedOp      int
	currentOp       ProcessOperation

	// Configuration inputs
	widthInput      textinput.Model
	heightInput     textinput.Model
	qualityInput    textinput.Model
	formatInput     textinput.Model
	scaleInput      textinput.Model
	cropXInput      textinput.Model
	cropYInput      textinput.Model
	cropWInput      textinput.Model
	cropHInput      textinput.Model
	outputInput     textinput.Model
	focusedInput    int
	totalInputs     int

	// Processing state
	processing   bool
	progressMsg  string
	resultPath   string
	resultSize   int64
	processTime  time.Duration
	err          error
	showHelp     bool
}

// NewProcessMenuModel creates a new process menu model
func NewProcessMenuModel() *ProcessMenuModel {
	// Initialize file picker for Desktop directory
	home, _ := os.UserHomeDir()
	desktopPath := filepath.Join(home, "Desktop")
	fp, _ := NewFilePicker(desktopPath)
	if fp != nil {
		// Filter for image files
		fp.SetFilter([]string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".tiff", ".webp"})
		fp.Refresh()
	}

	return &ProcessMenuModel{
		currentStep: ProcessStepSelectFile,
		filePicker:  fp,
		operations: []string{
			"Resize - Change image dimensions",
			"Scale - Resize by factor (preserves aspect ratio)",
			"Crop - Extract a region from the image",
			"Compress - Reduce file size",
			"Convert - Change image format",
		},
		widthInput:   createNumberInput("1024", 5),
		heightInput:  createNumberInput("1024", 5),
		qualityInput: createNumberInput("85", 3),
		formatInput:  createTextInput("png", 10),
		scaleInput:   createTextInput("0.5", 10),
		cropXInput:   createNumberInput("0", 5),
		cropYInput:   createNumberInput("0", 5),
		cropWInput:   createNumberInput("512", 5),
		cropHInput:   createNumberInput("512", 5),
		outputInput:  createTextInput("", 60),
	}
}

func createNumberInput(placeholder string, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 10
	ti.SetWidth(width)
	return ti
}

func createTextInput(placeholder string, width int) textinput.Model {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.CharLimit = 256
	ti.SetWidth(width)
	return ti
}

// Init initializes the process menu
func (m *ProcessMenuModel) Init() tea.Cmd {
	return nil
}

// Update handles messages for the process menu
func (m *ProcessMenuModel) Update(msg tea.Msg) (*ProcessMenuModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
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
			if m.currentStep > ProcessStepSelectFile {
				m.currentStep--
				return m, nil
			}
			return m, Navigate(ScreenMainMenu)
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case processCompleteMsg:
		m.processing = false
		m.resultPath = msg.path
		m.resultSize = msg.size
		m.processTime = msg.duration
		m.err = msg.err
		m.currentStep = ProcessStepResult
		return m, nil
	}

	switch m.currentStep {
	case ProcessStepSelectFile:
		return m.updateSelectFile(msg)
	case ProcessStepSelectOperation:
		return m.updateSelectOperation(msg)
	case ProcessStepConfigure:
		return m.updateConfigure(msg)
	case ProcessStepResult:
		return m.updateResult(msg)
	}

	return m, nil
}

// View renders the process menu
func (m *ProcessMenuModel) View() string {
	if m.showHelp {
		return m.renderHelp()
	}

	switch m.currentStep {
	case ProcessStepSelectFile:
		return m.viewSelectFile()
	case ProcessStepSelectOperation:
		return m.viewSelectOperation()
	case ProcessStepConfigure:
		return m.viewConfigure()
	case ProcessStepProcessing:
		return m.viewProcessing()
	case ProcessStepResult:
		return m.viewResult()
	}

	return "Unknown step"
}

// Step 1: File Selection
func (m *ProcessMenuModel) updateSelectFile(msg tea.Msg) (*ProcessMenuModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			if m.filePicker != nil && m.filePicker.Count() > 0 {
				file, err := m.filePicker.GetFile(m.selectedOp)
				if err == nil {
					m.selectedFile = file
					m.currentStep = ProcessStepSelectOperation
				} else {
					m.fileError = err
				}
			}
			return m, nil
		case "up", "k":
			if m.selectedOp > 0 {
				m.selectedOp--
			}
		case "down", "j":
			if m.filePicker != nil && m.selectedOp < m.filePicker.Count()-1 {
				m.selectedOp++
			}
		}
	}
	return m, nil
}

func (m *ProcessMenuModel) viewSelectFile() string {
	if m.filePicker == nil || m.filePicker.Count() == 0 {
		content := TitleStyle.Render("Process Image - Step 1/4") + "\n\n" +
			ErrorStyle.Render("No images found in Desktop folder") + "\n\n" +
			MutedStyle.Render("Place some images in ~/Desktop and try again") + "\n\n" +
			HelpStyle.Render("Esc: Back to main menu")

		box := FocusedBoxStyle.Width(76).Render(content)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}

	files := m.filePicker.ListFiles()
	var items []string
	for i, file := range files {
		cursor := "  "
		style := MenuItemStyle
		if i == m.selectedOp {
			cursor = "> "
			style = SelectedMenuItemStyle
		}
		items = append(items, style.Render(cursor+file))
	}

	content := TitleStyle.Render("Process Image - Step 1/4") + "\n\n" +
		SubtitleStyle.Render("Select an image to process") + "\n\n" +
		strings.Join(items[:min(len(items), 10)], "\n") + "\n\n" +
		MutedStyle.Render(fmt.Sprintf("Showing %d images from %s", m.filePicker.Count(), m.filePicker.GetDirectory())) + "\n\n" +
		HelpStyle.Render("↑/↓: Navigate • Enter: Select • Esc: Back")

	box := FocusedBoxStyle.Width(90).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// Step 2: Operation Selection
func (m *ProcessMenuModel) updateSelectOperation(msg tea.Msg) (*ProcessMenuModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.selectedOp > 0 {
				m.selectedOp--
			}
		case "down", "j":
			if m.selectedOp < len(m.operations)-1 {
				m.selectedOp++
			}
		case "enter", "space":
			m.currentOp = ProcessOperation(m.selectedOp)
			m.currentStep = ProcessStepConfigure
			m.focusedInput = 0
			m.widthInput.Focus()
			return m, textinput.Blink
		}
	}
	return m, nil
}

func (m *ProcessMenuModel) viewSelectOperation() string {
	var items []string
	for i, op := range m.operations {
		cursor := "  "
		style := MenuItemStyle
		if i == m.selectedOp {
			cursor = "> "
			style = SelectedMenuItemStyle
		}
		items = append(items, style.Render(cursor+op))
	}

	content := TitleStyle.Render("Process Image - Step 2/4") + "\n\n" +
		SubtitleStyle.Render("Select Operation") + "\n\n" +
		FormatKeyValue("File", m.selectedFile.Name) + "\n" +
		FormatKeyValue("Size", FormatImageInfo(m.selectedFile)) + "\n\n" +
		strings.Join(items, "\n\n") + "\n\n" +
		HelpStyle.Render("↑/↓: Navigate • Enter: Select • Esc: Back")

	box := FocusedBoxStyle.Width(76).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// Step 3: Configure
func (m *ProcessMenuModel) updateConfigure(msg tea.Msg) (*ProcessMenuModel, tea.Cmd) {
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "tab", "shift+tab":
			return m.handleTabNavigation(keyMsg.String()), nil
		case "enter":
			return m, m.startProcessing()
		}
	}

	// Update the focused input
	switch m.currentOp {
	case OpResize:
		if m.focusedInput == 0 {
			m.widthInput, cmd = m.widthInput.Update(msg)
		} else if m.focusedInput == 1 {
			m.heightInput, cmd = m.heightInput.Update(msg)
		} else {
			m.outputInput, cmd = m.outputInput.Update(msg)
		}
	case OpScale:
		if m.focusedInput == 0 {
			m.scaleInput, cmd = m.scaleInput.Update(msg)
		} else {
			m.outputInput, cmd = m.outputInput.Update(msg)
		}
	case OpCrop:
		inputs := []*textinput.Model{&m.cropXInput, &m.cropYInput, &m.cropWInput, &m.cropHInput, &m.outputInput}
		if m.focusedInput < len(inputs) {
			*inputs[m.focusedInput], cmd = inputs[m.focusedInput].Update(msg)
		}
	case OpCompress:
		if m.focusedInput == 0 {
			m.qualityInput, cmd = m.qualityInput.Update(msg)
		} else {
			m.outputInput, cmd = m.outputInput.Update(msg)
		}
	case OpConvert:
		if m.focusedInput == 0 {
			m.formatInput, cmd = m.formatInput.Update(msg)
		} else {
			m.outputInput, cmd = m.outputInput.Update(msg)
		}
	}

	return m, cmd
}

func (m *ProcessMenuModel) handleTabNavigation(key string) *ProcessMenuModel {
	// Determine total inputs based on operation
	switch m.currentOp {
	case OpResize:
		m.totalInputs = 3 // width, height, output
	case OpScale:
		m.totalInputs = 2 // scale, output
	case OpCrop:
		m.totalInputs = 5 // x, y, width, height, output
	case OpCompress:
		m.totalInputs = 2 // quality, output
	case OpConvert:
		m.totalInputs = 2 // format, output
	}

	// Blur current input
	m.blurAllInputs()

	// Move focus
	if key == "tab" {
		m.focusedInput = (m.focusedInput + 1) % m.totalInputs
	} else {
		m.focusedInput = (m.focusedInput - 1 + m.totalInputs) % m.totalInputs
	}

	// Focus new input
	m.focusInput(m.focusedInput)

	return m
}

func (m *ProcessMenuModel) blurAllInputs() {
	m.widthInput.Blur()
	m.heightInput.Blur()
	m.scaleInput.Blur()
	m.qualityInput.Blur()
	m.formatInput.Blur()
	m.cropXInput.Blur()
	m.cropYInput.Blur()
	m.cropWInput.Blur()
	m.cropHInput.Blur()
	m.outputInput.Blur()
}

func (m *ProcessMenuModel) focusInput(index int) {
	switch m.currentOp {
	case OpResize:
		if index == 0 {
			m.widthInput.Focus()
		} else if index == 1 {
			m.heightInput.Focus()
		} else {
			m.outputInput.Focus()
		}
	case OpScale:
		if index == 0 {
			m.scaleInput.Focus()
		} else {
			m.outputInput.Focus()
		}
	case OpCrop:
		inputs := []*textinput.Model{&m.cropXInput, &m.cropYInput, &m.cropWInput, &m.cropHInput, &m.outputInput}
		if index < len(inputs) {
			inputs[index].Focus()
		}
	case OpCompress:
		if index == 0 {
			m.qualityInput.Focus()
		} else {
			m.outputInput.Focus()
		}
	case OpConvert:
		if index == 0 {
			m.formatInput.Focus()
		} else {
			m.outputInput.Focus()
		}
	}
}

func (m *ProcessMenuModel) viewConfigure() string {
	// Set default output path if empty
	if m.outputInput.Value() == "" {
		dir := filepath.Dir(m.selectedFile.Path)
		ext := filepath.Ext(m.selectedFile.Name)
		nameNoExt := strings.TrimSuffix(m.selectedFile.Name, ext)
		defaultPath := filepath.Join(dir, nameNoExt+"_processed"+ext)
		m.outputInput.SetValue(defaultPath)
	}

	var configContent string
	switch m.currentOp {
	case OpResize:
		configContent = FormatKeyValue("Target Width", m.widthInput.View()) + "\n\n" +
			FormatKeyValue("Target Height", m.heightInput.View()) + "\n\n" +
			FormatKeyValue("Output", m.outputInput.View()) + "\n\n" +
			WarningStyle.Render("Note: Aspect ratio may change")
	case OpScale:
		configContent = FormatKeyValue("Scale Factor", m.scaleInput.View()) + "\n\n" +
			MutedStyle.Render("Examples: 0.5 = half size, 2.0 = double size") + "\n\n" +
			FormatKeyValue("Output", m.outputInput.View())
	case OpCrop:
		configContent = FormatKeyValue("X Position", m.cropXInput.View()) + "\n\n" +
			FormatKeyValue("Y Position", m.cropYInput.View()) + "\n\n" +
			FormatKeyValue("Width", m.cropWInput.View()) + "\n\n" +
			FormatKeyValue("Height", m.cropHInput.View()) + "\n\n" +
			FormatKeyValue("Output", m.outputInput.View())
	case OpCompress:
		configContent = FormatKeyValue("Quality (1-100)", m.qualityInput.View()) + "\n\n" +
			MutedStyle.Render("85-95 recommended for photos, 75-85 for web") + "\n\n" +
			FormatKeyValue("Output", m.outputInput.View())
	case OpConvert:
		configContent = FormatKeyValue("Target Format", m.formatInput.View()) + "\n\n" +
			MutedStyle.Render("Options: png, jpg, webp, gif, tiff, bmp") + "\n\n" +
			FormatKeyValue("Output", m.outputInput.View())
	}

	content := TitleStyle.Render("Process Image - Step 3/4") + "\n\n" +
		SubtitleStyle.Render("Configure "+m.operations[m.currentOp]) + "\n\n" +
		configContent + "\n\n" +
		HelpStyle.Render("Tab: Next field • Enter: Process • Esc: Back")

	box := FocusedBoxStyle.Width(90).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// Step 4: Processing
func (m *ProcessMenuModel) viewProcessing() string {
	spinner := SpinnerFrames[int(time.Now().Unix())%len(SpinnerFrames)]

	content := TitleStyle.Render("Process Image - Step 4/4") + "\n\n" +
		SubtitleStyle.Render("Processing...") + "\n\n" +
		SuccessStyle.Render(spinner) + " " + m.progressMsg + "\n\n" +
		MutedStyle.Render("Please wait while your image is being processed...")

	box := FocusedBoxStyle.Width(76).Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

// Step 5: Result
func (m *ProcessMenuModel) updateResult(msg tea.Msg) (*ProcessMenuModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "p":
			// Process another image
			return NewProcessMenuModel(), nil
		case "m":
			return m, Navigate(ScreenMainMenu)
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *ProcessMenuModel) viewResult() string {
	if m.err != nil {
		content := TitleStyle.Render("Processing Failed") + "\n\n" +
			ErrorStyle.Render("Error: "+m.err.Error()) + "\n\n" +
			HelpStyle.Render("p: Try another • m: Main menu • q: Quit")

		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorError).
			Padding(1, 2).
			Width(76).
			Render(content)

		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}

	content := TitleStyle.Render("Processing Complete!") + "\n\n" +
		SuccessStyle.Render("✓ Image processed successfully") + "\n\n" +
		FormatKeyValue("Saved to", m.resultPath) + "\n" +
		FormatKeyValue("File size", FormatFileSize(m.resultSize)) + "\n" +
		FormatKeyValue("Time taken", m.processTime.String()) + "\n\n" +
		HelpStyle.Render("p: Process another • m: Main menu • q: Quit")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSuccess).
		Padding(1, 2).
		Width(76).
		Render(content)

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *ProcessMenuModel) renderHelp() string {
	helpContent := TitleStyle.Render("Image Processing Help") + "\n\n" +
		SubtitleStyle.Render("Available Operations") + "\n\n" +
		FormatKeyValue("Resize", "Change width and height (may distort)") + "\n" +
		FormatKeyValue("Scale", "Resize by factor (preserves aspect)") + "\n" +
		FormatKeyValue("Crop", "Extract a rectangular region") + "\n" +
		FormatKeyValue("Compress", "Reduce file size (quality 1-100)") + "\n" +
		FormatKeyValue("Convert", "Change format (PNG/JPG/WebP/etc)") + "\n\n" +
		SubtitleStyle.Render("Keyboard Shortcuts") + "\n\n" +
		FormatKeyValue("↑/↓, k/j", "Navigate") + "\n" +
		FormatKeyValue("Enter", "Confirm/Next") + "\n" +
		FormatKeyValue("Tab/Shift+Tab", "Switch input fields") + "\n" +
		FormatKeyValue("Esc", "Go back") + "\n" +
		FormatKeyValue("?", "Toggle help") + "\n\n" +
		HelpStyle.Render("Press Esc to close")

	box := FocusedBoxStyle.Width(70).Render(helpContent)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
}

func (m *ProcessMenuModel) startProcessing() tea.Cmd {
	m.currentStep = ProcessStepProcessing
	m.processing = true
	m.progressMsg = "Processing image..."

	return m.processImageCmd()
}

func (m *ProcessMenuModel) processImageCmd() tea.Cmd {
	return func() tea.Msg {
		startTime := time.Now()
		ctx := context.Background()

		var err error
		outputPath := m.outputInput.Value()

		switch m.currentOp {
		case OpResize:
			width, _ := strconv.Atoi(m.widthInput.Value())
			height, _ := strconv.Atoi(m.heightInput.Value())
			err = imaging.ResizeImage(ctx, m.selectedFile.Path, outputPath, width, height)

		case OpScale:
			factor, _ := strconv.ParseFloat(m.scaleInput.Value(), 64)
			err = imaging.ScaleImage(ctx, m.selectedFile.Path, outputPath, factor)

		case OpCrop:
			x, _ := strconv.Atoi(m.cropXInput.Value())
			y, _ := strconv.Atoi(m.cropYInput.Value())
			width, _ := strconv.Atoi(m.cropWInput.Value())
			height, _ := strconv.Atoi(m.cropHInput.Value())
			err = imaging.CropImage(ctx, m.selectedFile.Path, outputPath, x, y, width, height)

		case OpCompress:
			quality, _ := strconv.Atoi(m.qualityInput.Value())
			err = imaging.CompressImage(ctx, m.selectedFile.Path, outputPath, quality)

		case OpConvert:
			// Convert uses ConvertImageFile which auto-detects format from extension
			err = imaging.ConvertImageFile(ctx, m.selectedFile.Path, outputPath)
		}

		if err != nil {
			return processCompleteMsg{err: err}
		}

		fileInfo, err := os.Stat(outputPath)
		if err != nil {
			return processCompleteMsg{
				path:     outputPath,
				duration: time.Since(startTime),
				err:      fmt.Errorf("processed but couldn't stat file: %w", err),
			}
		}

		return processCompleteMsg{
			path:     outputPath,
			size:     fileInfo.Size(),
			duration: time.Since(startTime),
		}
	}
}

type processCompleteMsg struct {
	path     string
	size     int64
	duration time.Duration
	err      error
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

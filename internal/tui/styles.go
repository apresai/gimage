// Package tui provides Terminal User Interface components for gimage.
package tui

import (
	"charm.land/lipgloss/v2"
)

// Color palette
var (
	ColorPrimary   = lipgloss.Color("#7B61FF") // Purple
	ColorSecondary = lipgloss.Color("#00D9FF") // Cyan
	ColorSuccess   = lipgloss.Color("#00FF88") // Green
	ColorWarning   = lipgloss.Color("#FFB454") // Orange
	ColorError     = lipgloss.Color("#FF6B6B") // Red
	ColorMuted     = lipgloss.Color("#6C757D") // Gray
	ColorBorder    = lipgloss.Color("#444444") // Dark gray
	ColorFocused   = lipgloss.Color("#7B61FF") // Purple (same as primary)
)

// Common styles
var (
	// TitleStyle for large headings
	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true).
			Padding(0, 1)

	// SubtitleStyle for smaller headings
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Padding(0, 1)

	// MenuItemStyle for unselected menu items
	MenuItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Padding(0, 2)

	// SelectedMenuItemStyle for selected menu items
	SelectedMenuItemStyle = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				Padding(0, 2).
				Background(lipgloss.Color("#2A2A2A"))

	// BoxStyle for bordered containers
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	// FocusedBoxStyle for focused bordered containers
	FocusedBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorFocused).
			Padding(1, 2)

	// SuccessStyle for success messages
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	// ErrorStyle for error messages
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	// WarningStyle for warning messages
	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	// MutedStyle for less important text
	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// HelpStyle for help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true).
			Padding(1, 0)

	// CodeBlockStyle for code blocks
	CodeBlockStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1e1e1e")).
			Foreground(lipgloss.Color("#00FF88")).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorMuted)

	// StatusBarStyle for the bottom status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#333333")).
			Padding(0, 1)
)

// ASCII Art for the logo
const LogoASCII = `
   _____ _
  / ____(_)
 | |  __ _ _ __ ___   __ _  __ _  ___
 | | |_ | | '_ ' _ \ / _' |/ _' |/ _ \
 | |__| | | | | | | | (_| | (_| |  __/
  \_____|_|_| |_| |_|\__,_|\__, |\___|
                            __/ |
                           |___/
`

// SmallerLogoASCII for compact spaces
const SmallerLogoASCII = `
   ___ _
  / _ (_)_ __  __ _ __ _ ___
 | (_) | | '  \/ _' / _' / -_)
  \___/|_|_|_|_\__,_\__, \___|
                     |___/
`

// SpinnerFrames for loading animations
var SpinnerFrames = []string{
	"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏",
}

// ProgressBarStyle creates a styled progress bar
func ProgressBarStyle(percent float64, width int) string {
	filledWidth := int(float64(width) * (percent / 100.0))
	if filledWidth > width {
		filledWidth = width
	}
	if filledWidth < 0 {
		filledWidth = 0
	}

	filled := lipgloss.NewStyle().
		Foreground(ColorSuccess).
		Render(string(repeatRune('█', filledWidth)))

	empty := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Render(string(repeatRune('░', width-filledWidth)))

	return filled + empty
}

// repeatRune creates a string of repeated runes
func repeatRune(r rune, count int) []rune {
	result := make([]rune, count)
	for i := range result {
		result[i] = r
	}
	return result
}

// FormatKeyValue formats a key-value pair with styling
func FormatKeyValue(key, value string) string {
	keyStyle := lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true)
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	return keyStyle.Render(key+":") + " " + valueStyle.Render(value)
}

// FormatList formats a list of items with bullets
func FormatList(items []string) string {
	result := ""
	bullet := lipgloss.NewStyle().Foreground(ColorPrimary).Render("•")
	for _, item := range items {
		result += bullet + " " + item + "\n"
	}
	return result
}

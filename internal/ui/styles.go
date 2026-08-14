package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	colorPrimary  = lipgloss.Color("#7C9EF0") // soft blue
	colorAccent   = lipgloss.Color("#F0A47C") // soft orange
	colorMuted    = lipgloss.Color("#666688")
	colorError    = lipgloss.Color("#F07C7C")
	colorSuccess  = lipgloss.Color("#7CF09C")
	colorFile     = lipgloss.Color("#B0B0CC")
	colorBorder   = lipgloss.Color("#444466")
	colorSelected = lipgloss.Color("#cd0fc1")
	colorHeaderBg = lipgloss.Color("#1A1A2E")

	// Base styles
	StyleNormal = lipgloss.NewStyle()

	StylePath = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	StyleSelected = lipgloss.NewStyle().
			Background(colorSelected).
			Foreground(lipgloss.Color("#EEEEFF")).
			Bold(true)

	StyleSelector = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true)
	StyleBrandPrimary = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true)
	StyleBrandSecondary = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#5865F2")).
				Bold(true)

	StyleDirName = lipgloss.NewStyle().
			Foreground(colorPrimary)

	StyleFileName = lipgloss.NewStyle().
			Foreground(colorFile)

	StyleNumber = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	StyleMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	StyleError = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	StyleSuccess = lipgloss.NewStyle().
			Foreground(colorSuccess)

	StyleHeader = lipgloss.NewStyle().
			Background(colorHeaderBg).
			Foreground(colorPrimary).
			Bold(true).
			Padding(0, 1)

	StyleStatusBar = lipgloss.NewStyle().
			Foreground(colorMuted)

	StyleHintKey = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFE66D")).
			Bold(true)

	StyleConfirmBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2).
			Margin(1, 0)

	StyleInputPrompt = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	StyleDetail = lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true)

	StyleVimBadge = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Bold(true)

	StyleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder)

	// StyleParentCrumb: greyed-out non-interactive ancestor directory lines
	// shown above the tree root when display.parent_depth > 0.
	StyleParentCrumb = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#3A3A5A")).
				Italic(true)

	// StyleRootDir: the root directory label rendered just above tree nodes.
	// Slightly brighter than crumbs but still non-interactive.
	StyleRootDir = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#555577"))

	// StyleClipboard: warm yellow — used for clipboard bar and [copy]/[cut] tags.
	StyleClipboard = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F0E07C")).
			Bold(true)

	// StylePreview: left-bordered panel for the file/image preview column.
	StylePreview = lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	// StylePreviewLabel: muted label text used in the file-info panel.
	StylePreviewLabel = lipgloss.NewStyle().
				Foreground(colorMuted)

	// StylePreviewTitle: accent title line at the top of the preview panel.
	StylePreviewTitle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)
)

// ConfigureTheme applies a complete semantic palette. Terminal mode omits
// explicit colors so the terminal's normal foreground and background inherit.
func ConfigureTheme(colors map[string]string, terminal bool) {
	StyleNormal = themedStyle(lipgloss.NewStyle(), colors, terminal, "foreground", "")
	StylePath = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "primary", "#7C9EF0")
	StyleSelected = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "selected_foreground", "#EEEEFF")
	StyleSelected = themedBackground(StyleSelected, colors, terminal, "selected_background", "#cd0fc1")
	StyleDirName = themedStyle(lipgloss.NewStyle(), colors, terminal, "primary", "#7C9EF0")
	StyleFileName = themedStyle(lipgloss.NewStyle(), colors, terminal, "file", "#B0B0CC")
	StyleNumber = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "accent", "#F0A47C")
	StyleMuted = themedStyle(lipgloss.NewStyle(), colors, terminal, "muted", "#666688")
	StyleError = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "error", "#F07C7C")
	StyleSuccess = themedStyle(lipgloss.NewStyle(), colors, terminal, "success", "#7CF09C")
	StyleHeader = themedBackground(lipgloss.NewStyle().Bold(true).Padding(0, 1), colors, terminal, "header_background", "#1A1A2E")
	StyleHeader = themedStyle(StyleHeader, colors, terminal, "primary", "#7C9EF0")
	StyleStatusBar = themedStyle(lipgloss.NewStyle(), colors, terminal, "muted", "#666688")
	StyleHintKey = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "hint_key", "#FFE66D")
	StyleConfirmBox = themedStyle(lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).Margin(1, 0), colors, terminal, "foreground", "")
	StyleConfirmBox = themedBorder(StyleConfirmBox, colors, terminal, "accent", "#F0A47C")
	StyleInputPrompt = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "accent", "#F0A47C")
	StyleDetail = themedStyle(lipgloss.NewStyle().Italic(true), colors, terminal, "muted", "#666688")
	StyleVimBadge = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "success", "#7CF09C")
	StyleBorder = themedBorder(lipgloss.NewStyle().Border(lipgloss.RoundedBorder()), colors, terminal, "border", "#444466")
	StyleParentCrumb = themedStyle(lipgloss.NewStyle().Italic(true), colors, terminal, "parent_crumb", "#3A3A5A")
	StyleRootDir = themedStyle(lipgloss.NewStyle(), colors, terminal, "root_directory", "#555577")
	StyleClipboard = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "clipboard", "#F0E07C")
	StylePreview = themedBorder(lipgloss.NewStyle().BorderLeft(true).BorderStyle(lipgloss.NormalBorder()).Padding(0, 1), colors, terminal, "border", "#444466")
	StylePreviewLabel = themedStyle(lipgloss.NewStyle(), colors, terminal, "muted", "#666688")
	StylePreviewTitle = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "accent", "#F0A47C")
	StyleSelector = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "selector", "#FFFFFF")
	StyleBrandPrimary = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "brand_primary", "#FFFFFF")
	StyleBrandSecondary = themedStyle(lipgloss.NewStyle().Bold(true), colors, terminal, "brand_secondary", "#5865F2")
}

func themedStyle(style lipgloss.Style, colors map[string]string, terminal bool, key, fallback string) lipgloss.Style {
	if value, ok := themedValue(colors, terminal, key, fallback); ok {
		return style.Foreground(lipgloss.Color(value))
	}
	return style
}

func themedBackground(style lipgloss.Style, colors map[string]string, terminal bool, key, fallback string) lipgloss.Style {
	if value, ok := themedValue(colors, terminal, key, fallback); ok {
		return style.Background(lipgloss.Color(value))
	}
	return style
}

func themedBorder(style lipgloss.Style, colors map[string]string, terminal bool, key, fallback string) lipgloss.Style {
	if value, ok := themedValue(colors, terminal, key, fallback); ok {
		return style.BorderForeground(lipgloss.Color(value))
	}
	return style
}

func themedValue(colors map[string]string, terminal bool, key, fallback string) (string, bool) {
	if value := colors[key]; value != "" {
		return value, true
	}
	if terminal {
		return "", false
	}
	return fallback, fallback != ""
}

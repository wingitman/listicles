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
	colorSelected = lipgloss.Color("#2A2A4A")
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
)

package tui

import "github.com/charmbracelet/lipgloss"

// Color palette — dark cyberpunk / hacker aesthetic.
var (
	colorBg        = lipgloss.Color("#0a0e14")
	colorFg        = lipgloss.Color("#c5c8c6")
	colorDim       = lipgloss.Color("#5c6370")
	colorAccent    = lipgloss.Color("#00e5ff")
	colorGreen     = lipgloss.Color("#00e676")
	colorRed       = lipgloss.Color("#ff1744")
	colorYellow    = lipgloss.Color("#ffd740")
	colorOrange    = lipgloss.Color("#ff9100")
	colorPurple    = lipgloss.Color("#b388ff")
	colorPink      = lipgloss.Color("#ff80ab")
	colorBorder    = lipgloss.Color("#1e2a3a")
	colorHighlight = lipgloss.Color("#1a2332")
	colorSelected  = lipgloss.Color("#12293d")
)

// Layout styles.
var (
	styleApp = lipgloss.NewStyle().
			Background(colorBg).
			Foreground(colorFg)

	styleTitle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			Padding(0, 1)

	styleTitleBar = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(colorBorder).
			Padding(0, 1).
			MarginBottom(1)

	styleStatusBar = lipgloss.NewStyle().
			Foreground(colorDim).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(colorBorder).
			Padding(0, 1)

	styleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	// Table styles.
	styleTableHeader = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true).
				Padding(0, 1)

	styleTableRow = lipgloss.NewStyle().
			Foreground(colorFg).
			Padding(0, 1)

	styleTableSelected = lipgloss.NewStyle().
				Foreground(colorAccent).
				Background(colorSelected).
				Bold(true).
				Padding(0, 1)

	// Status indicators.
	styleUp = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	styleDown = lipgloss.NewStyle().
			Foreground(colorRed)

	styleSpoofed = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	// Interactive elements.
	styleKey = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	styleKeyDesc = lipgloss.NewStyle().
			Foreground(colorDim)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorGreen).
			Bold(true)

	styleError = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	styleWarning = lipgloss.NewStyle().
			Foreground(colorYellow).
			Bold(true)

	// Section label.
	styleLabel = lipgloss.NewStyle().
			Foreground(colorPurple).
			Bold(true)

	stylePrompt = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	// Vendor picker.
	styleVendorName = lipgloss.NewStyle().
			Foreground(colorFg).
			Bold(true)

	styleVendorOUI = lipgloss.NewStyle().
			Foreground(colorDim)
)

// Helper to render a key hint like "[R] Random".
func keyHint(key, desc string) string {
	return styleKey.Render("["+key+"]") + " " + styleKeyDesc.Render(desc)
}

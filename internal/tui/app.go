package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/A3-N/macsetgo/internal/network"
)

// Page identifies the currently active view.
type Page int

const (
	PageDashboard Page = iota
	PageActions
	PageProfiles
	PageHistory
	PageVendorPicker
	PageDaemon
	PageHelp
)

// tickMsg triggers periodic interface refresh.
type tickMsg time.Time

// refreshMsg signals that interface data should be reloaded.
type refreshMsg struct{}

// interfacesMsg carries the result of an interface scan.
type interfacesMsg struct {
	ifaces []network.Interface
}

// statusMsg displays a temporary status message.
type statusMsg struct {
	text    string
	isError bool
}

// navigateMsg requests page navigation.
type navigateMsg struct {
	page Page
}

// App is the root Bubbletea model.
type App struct {
	page        Page
	prevPage    Page
	width       int
	height      int
	interfaces  []network.Interface
	statusText  string
	statusError bool

	// Child models.
	dashboard   DashboardModel
	actions     ActionsModel
	profiles    ProfilesModel
	history     HistoryModel
	vendorPick  VendorPickerModel
	daemonView  DaemonModel
}

// NewApp creates and initialises the root app model.
func NewApp() App {
	app := App{
		page: PageDashboard,
	}
	app.dashboard = NewDashboardModel()
	app.actions = NewActionsModel()
	app.profiles = NewProfilesModel()
	app.history = NewHistoryModel()
	app.vendorPick = NewVendorPickerModel()
	app.daemonView = NewDaemonModel()
	return app
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		refreshInterfaces(),
		a.tick(),
	)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Propagate to all children.
		a.dashboard.SetSize(msg.Width, msg.Height-4)
		a.actions.SetSize(msg.Width, msg.Height-4)
		a.profiles.SetSize(msg.Width, msg.Height-4)
		a.history.SetSize(msg.Width, msg.Height-4)
		a.vendorPick.SetSize(msg.Width, msg.Height-4)
		a.daemonView.SetSize(msg.Width, msg.Height-4)

	case tea.KeyMsg:
		// Global keybindings (always active unless an input is focused).
		switch msg.String() {
		case "ctrl+c", "q":
			if a.page == PageDashboard {
				return a, tea.Quit
			}
			// Otherwise go back to dashboard.
			a.page = PageDashboard
			return a, nil

		case "?":
			if a.page == PageHelp {
				a.page = a.prevPage
			} else {
				a.prevPage = a.page
				a.page = PageHelp
			}
			return a, nil

		case "esc":
			if a.page != PageDashboard {
				a.page = PageDashboard
				return a, nil
			}
		}

	case interfacesMsg:
		a.interfaces = msg.ifaces

	case tickMsg:
		cmds = append(cmds, refreshInterfaces(), a.tick())

	case refreshMsg:
		cmds = append(cmds, refreshInterfaces())

	case statusMsg:
		a.statusText = msg.text
		a.statusError = msg.isError

	case navigateMsg:
		a.prevPage = a.page
		a.page = msg.page
	}

	// Delegate to active page.
	var cmd tea.Cmd
	switch a.page {
	case PageDashboard:
		a.dashboard.SetInterfaces(a.interfaces)
		a.dashboard, cmd = a.dashboard.Update(msg)
		cmds = append(cmds, cmd)

		// Check if dashboard wants to navigate to actions.
		if a.dashboard.selectedAction {
			a.dashboard.selectedAction = false
			if sel := a.dashboard.SelectedInterface(); sel != nil {
				a.actions.SetInterface(sel)
				a.prevPage = a.page
				a.page = PageActions
			}
		}
		// Check nav requests from dashboard.
		if a.dashboard.navRequest != nil {
			a.prevPage = a.page
			a.page = *a.dashboard.navRequest
			a.dashboard.navRequest = nil
			// Initialize the target page.
			switch a.page {
			case PageProfiles:
				a.profiles.Refresh()
			case PageHistory:
				a.history.Refresh()
			case PageDaemon:
				a.daemonView.Refresh()
			}
		}

	case PageActions:
		a.actions, cmd = a.actions.Update(msg)
		cmds = append(cmds, cmd)
		if a.actions.done {
			a.actions.done = false
			a.page = PageDashboard
			cmds = append(cmds, refreshInterfaces())
		}
		if a.actions.wantVendorPicker {
			a.actions.wantVendorPicker = false
			a.prevPage = a.page
			a.page = PageVendorPicker
			a.vendorPick.Reset()
		}

	case PageProfiles:
		a.profiles.SetInterfaces(a.interfaces)
		a.profiles, cmd = a.profiles.Update(msg)
		cmds = append(cmds, cmd)
		if a.profiles.done {
			a.profiles.done = false
			a.page = PageDashboard
			cmds = append(cmds, refreshInterfaces())
		}

	case PageHistory:
		a.history, cmd = a.history.Update(msg)
		cmds = append(cmds, cmd)
		if a.history.done {
			a.history.done = false
			a.page = PageDashboard
		}

	case PageVendorPicker:
		a.vendorPick, cmd = a.vendorPick.Update(msg)
		cmds = append(cmds, cmd)
		if a.vendorPick.done {
			a.vendorPick.done = false
			if a.vendorPick.selected != "" {
				a.actions.ApplyVendor(a.vendorPick.selected)
				a.page = PageActions
				// Trigger the vendor apply action.
				cmds = append(cmds, a.actions.DoVendorApply())
			} else {
				a.page = PageActions
			}
		}

	case PageDaemon:
		a.daemonView, cmd = a.daemonView.Update(msg)
		cmds = append(cmds, cmd)
		if a.daemonView.done {
			a.daemonView.done = false
			a.page = PageDashboard
		}

	case PageHelp:
		// Any key goes back.
		if _, ok := msg.(tea.KeyMsg); ok {
			a.page = a.prevPage
		}
	}

	return a, tea.Batch(cmds...)
}

func (a App) View() string {
	if a.width == 0 {
		return "Initializing..."
	}

	// Title bar.
	title := styleTitleBar.Width(a.width - 2).Render(
		styleTitle.Render("macsetgo") + styleKeyDesc.Render("  github.com/A3-N/macsetgo"),
	)

	// Page content.
	var content string
	switch a.page {
	case PageDashboard:
		content = a.dashboard.View()
	case PageActions:
		content = a.actions.View()
	case PageProfiles:
		content = a.profiles.View()
	case PageHistory:
		content = a.history.View()
	case PageVendorPicker:
		content = a.vendorPick.View()
	case PageDaemon:
		content = a.daemonView.View()
	case PageHelp:
		content = renderHelp(a.width)
	}

	// Status bar.
	statusStyle := styleStatusBar.Width(a.width - 2)
	var status string
	if a.statusText != "" {
		if a.statusError {
			status = styleError.Render("✗ " + a.statusText)
		} else {
			status = styleSuccess.Render("✓ " + a.statusText)
		}
	} else {
		status = styleKeyDesc.Render("Press ? for help")
	}

	statusBar := statusStyle.Render(status)

	return lipgloss.JoinVertical(lipgloss.Left, title, content, statusBar)
}

// refreshInterfaces loads the current interface list and returns them as a message.
func refreshInterfaces() tea.Cmd {
	return func() tea.Msg {
		ifaces, err := network.ListInterfaces()
		if err != nil {
			return statusMsg{text: err.Error(), isError: true}
		}
		return interfacesMsg{ifaces: ifaces}
	}
}

func (a App) tick() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// renderHelp renders the help overlay.
func renderHelp(width int) string {
	bold := lipgloss.NewStyle().Foreground(colorFg).Bold(true)
	section := styleLabel
	dim := styleKeyDesc
	sep := dim.Render(strings.Repeat("─", 40))

	var b strings.Builder

	b.WriteString("\n")
	b.WriteString("  " + section.Render("GLOBAL") + "\n\n")
	b.WriteString("  " + keyHint("?", "Toggle help") + "\n")
	b.WriteString("  " + keyHint("Esc", "Back to dashboard") + "\n")
	b.WriteString("  " + keyHint("q", "Quit") + "  " + dim.Render("(from dashboard)") + "\n")
	b.WriteString("  " + keyHint("Ctrl+C", "Force quit") + "\n")

	b.WriteString("\n  " + sep + "\n\n")

	b.WriteString("  " + section.Render("DASHBOARD") + "\n\n")
	b.WriteString("  " + keyHint("j/k", "") + bold.Render("Navigate adapters") + "\n")
	b.WriteString("  " + keyHint("Enter", "") + bold.Render("Open actions") + "  " + dim.Render("for selected adapter") + "\n")
	b.WriteString("  " + keyHint("P", "") + bold.Render("Profiles") + "  " + dim.Render("save/load MAC configs") + "\n")
	b.WriteString("  " + keyHint("H", "") + bold.Render("History") + "  " + dim.Render("view all MAC changes") + "\n")
	b.WriteString("  " + keyHint("D", "") + bold.Render("Daemon") + "  " + dim.Render("auto-change settings") + "\n")

	b.WriteString("\n  " + sep + "\n\n")

	b.WriteString("  " + section.Render("ACTIONS") + "\n\n")
	b.WriteString("  " + keyHint("R", "") + bold.Render("Randomize") + "  " + dim.Render("unicast locally-administered") + "\n")
	b.WriteString("  " + keyHint("V", "") + bold.Render("Vendor random") + "  " + dim.Render("match a real vendor OUI") + "\n")
	b.WriteString("  " + keyHint("M", "") + bold.Render("Manual MAC") + "  " + dim.Render("type a specific address") + "\n")
	b.WriteString("  " + keyHint("P", "") + bold.Render("Restore permanent") + "  " + dim.Render("factory MAC") + "\n")
	b.WriteString("  " + keyHint("A", "") + bold.Render("Randomize all") + "  " + dim.Render("every active adapter") + "\n")

	b.WriteString("\n  " + sep + "\n\n")

	b.WriteString("  " + section.Render("HISTORY") + "\n\n")
	b.WriteString("  " + keyHint("R", "") + bold.Render("Restore") + "  " + dim.Render("re-apply the new MAC") + "\n")
	b.WriteString("  " + keyHint("O", "") + bold.Render("Revert") + "  " + dim.Render("apply the old MAC") + "\n")

	return b.String()
}

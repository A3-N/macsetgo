package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/A3-N/macsetgo/internal/config"
	"github.com/A3-N/macsetgo/internal/daemon"
)

// DaemonModel manages the auto-change daemon.
type DaemonModel struct {
	status     string
	profiles   []config.Profile
	cfg        config.Config
	cursor     int
	width      int
	height     int
	done       bool
	statusText string
	statusErr  bool
}

func NewDaemonModel() DaemonModel {
	return DaemonModel{}
}

func (d *DaemonModel) SetSize(w, h int) {
	d.width = w
	d.height = h
}

func (d *DaemonModel) Refresh() {
	d.status = daemon.Status()
	d.profiles, _ = config.ListProfiles()
	d.cfg, _ = config.LoadConfig()
	d.cursor = 0
	d.statusText = ""
	d.statusErr = false
}

func (d DaemonModel) Update(msg tea.Msg) (DaemonModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "i":
			return d.installDaemon()
		case "u":
			return d.uninstallDaemon()
		case "up", "k":
			if d.cursor > 0 {
				d.cursor--
			}
		case "down", "j":
			if d.cursor < len(d.profiles)-1 {
				d.cursor++
			}
		case "enter":
			// Set daemon profile.
			if d.cursor >= 0 && d.cursor < len(d.profiles) {
				return d.setDaemonProfile(d.profiles[d.cursor].Name)
			}
		case "esc", "q":
			d.done = true
		}
	}
	return d, nil
}

func (d DaemonModel) installDaemon() (DaemonModel, tea.Cmd) {
	if d.cfg.DaemonProfile == "" {
		d.statusText = "Select a profile first (Enter on a profile below)"
		d.statusErr = true
		return d, nil
	}

	if err := daemon.Install(); err != nil {
		d.statusText = fmt.Sprintf("Install failed: %v", err)
		d.statusErr = true
	} else {
		d.statusText = "Daemon installed and started"
		d.statusErr = false
		d.status = daemon.Status()
	}
	return d, nil
}

func (d DaemonModel) uninstallDaemon() (DaemonModel, tea.Cmd) {
	if err := daemon.Uninstall(); err != nil {
		d.statusText = fmt.Sprintf("Uninstall failed: %v", err)
		d.statusErr = true
	} else {
		d.statusText = "Daemon uninstalled"
		d.statusErr = false
		d.status = daemon.Status()
	}
	return d, nil
}

func (d DaemonModel) setDaemonProfile(name string) (DaemonModel, tea.Cmd) {
	d.cfg.DaemonProfile = name
	if err := config.SaveConfig(d.cfg); err != nil {
		d.statusText = fmt.Sprintf("Config save failed: %v", err)
		d.statusErr = true
	} else {
		d.statusText = fmt.Sprintf("Daemon profile set to %q", name)
		d.statusErr = false
	}
	return d, nil
}

func (d DaemonModel) View() string {
	var b strings.Builder

	b.WriteString("  " + styleLabel.Render("Auto-Change Daemon") + "\n\n")

	// Current status.
	var statusStyle = styleDown
	statusLabel := d.status
	switch {
	case strings.Contains(d.status, "running"):
		statusStyle = styleUp
	case strings.Contains(d.status, "installed"):
		statusStyle = styleWarning
	}

	b.WriteString(fmt.Sprintf("  %s %s\n",
		styleKeyDesc.Render("Status:"),
		statusStyle.Render(statusLabel),
	))

	profileName := d.cfg.DaemonProfile
	if profileName == "" {
		profileName = "(none)"
	}
	b.WriteString(fmt.Sprintf("  %s %s\n",
		styleKeyDesc.Render("Profile:"),
		styleAccent.Render(profileName),
	))

	b.WriteString(fmt.Sprintf("  %s %ds\n",
		styleKeyDesc.Render("Poll interval:"),
		d.cfg.DaemonPollInterval,
	))

	b.WriteString(fmt.Sprintf("  %s %v\n",
		styleKeyDesc.Render("Match by port:"),
		d.cfg.MatchByPortName,
	))

	b.WriteString("\n")

	// Profile selection.
	b.WriteString("  " + styleLabel.Render("Select daemon profile:") + "\n\n")

	if len(d.profiles) == 0 {
		b.WriteString("  " + styleKeyDesc.Render("No profiles saved. Create one from the Profiles page first.") + "\n")
	} else {
		for i, profile := range d.profiles {
			cursor := "  "
			if i == d.cursor {
				cursor = styleAccent.Render("▸ ")
			}

			active := ""
			if profile.Name == d.cfg.DaemonProfile {
				active = styleSuccess.Render(" ●")
			}

			b.WriteString(fmt.Sprintf("  %s%-20s %s%s\n",
				cursor,
				styleVendorName.Render(profile.Name),
				styleKeyDesc.Render(fmt.Sprintf("%d ifaces", len(profile.Entries))),
				active,
			))
		}
	}

	// Status message.
	if d.statusText != "" {
		b.WriteString("\n")
		if d.statusErr {
			b.WriteString("  " + styleError.Render("✗ "+d.statusText))
		} else {
			b.WriteString("  " + styleSuccess.Render("✓ "+d.statusText))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s  %s  %s  %s",
		keyHint("Enter", "Set profile"),
		keyHint("I", "Install"),
		keyHint("U", "Uninstall"),
		keyHint("Esc", "Back"),
	))

	return b.String()
}

package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/A3-N/macsetgo/internal/config"
	"github.com/A3-N/macsetgo/internal/network"
)

// ProfilesModel manages saved MAC profiles.
type ProfilesModel struct {
	profiles   []config.Profile
	interfaces []network.Interface
	cursor     int
	width      int
	height     int
	done       bool

	// Save mode.
	saving    bool
	nameInput string

	// Status.
	statusText string
	statusErr  bool
}

func NewProfilesModel() ProfilesModel {
	return ProfilesModel{}
}

func (p *ProfilesModel) SetSize(w, h int) {
	p.width = w
	p.height = h
}

func (p *ProfilesModel) SetInterfaces(ifaces []network.Interface) {
	p.interfaces = ifaces
}

func (p *ProfilesModel) Refresh() {
	profiles, _ := config.ListProfiles()
	p.profiles = profiles
	p.cursor = 0
	p.saving = false
	p.nameInput = ""
	p.statusText = ""
	p.statusErr = false
}

func (p ProfilesModel) Update(msg tea.Msg) (ProfilesModel, tea.Cmd) {
	if p.saving {
		return p.updateSaveInput(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
			}
		case "down", "j":
			if p.cursor < len(p.profiles)-1 {
				p.cursor++
			}
		case "enter":
			if p.cursor >= 0 && p.cursor < len(p.profiles) {
				return p.applyProfile(p.profiles[p.cursor])
			}
		case "n":
			p.saving = true
			p.nameInput = ""
		case "x", "delete":
			if p.cursor >= 0 && p.cursor < len(p.profiles) {
				return p.deleteProfile(p.profiles[p.cursor].Name)
			}
		case "esc", "q":
			p.done = true
		}
	}
	return p, nil
}

func (p ProfilesModel) updateSaveInput(msg tea.Msg) (ProfilesModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if p.nameInput != "" {
				return p.saveCurrentProfile()
			}
		case "backspace":
			if len(p.nameInput) > 0 {
				p.nameInput = p.nameInput[:len(p.nameInput)-1]
			}
		case "esc":
			p.saving = false
		default:
			if len(msg.String()) == 1 {
				ch := msg.String()[0]
				if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
					(ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
					p.nameInput += string(ch)
				}
			}
		}
	}
	return p, nil
}

func (p ProfilesModel) saveCurrentProfile() (ProfilesModel, tea.Cmd) {
	entries := make(map[string]string)
	for _, iface := range p.interfaces {
		if iface.CurrentMAC != "" {
			entries[iface.HardwarePort] = iface.CurrentMAC
		}
	}

	profile := config.Profile{
		Name:      p.nameInput,
		CreatedAt: time.Now(),
		Entries:   entries,
	}

	if err := config.SaveProfile(profile); err != nil {
		p.statusText = fmt.Sprintf("Save failed: %v", err)
		p.statusErr = true
	} else {
		p.statusText = fmt.Sprintf("Profile %q saved (%d interfaces)", p.nameInput, len(entries))
		p.statusErr = false
	}

	p.saving = false
	p.Refresh()
	return p, nil
}

func (p ProfilesModel) applyProfile(profile config.Profile) (ProfilesModel, tea.Cmd) {
	ifaces, err := network.ListInterfaces()
	if err != nil {
		p.statusText = fmt.Sprintf("Failed: %v", err)
		p.statusErr = true
		return p, nil
	}

	applied := 0
	for _, iface := range ifaces {
		mac, ok := profile.Entries[iface.HardwarePort]
		if !ok {
			continue
		}
		oldMAC := iface.CurrentMAC
		if err := network.SetMAC(&iface, mac); err != nil {
			continue
		}
		_ = config.LogChange(config.HistoryEntry{
			Interface: iface.Name,
			PortName:  iface.HardwarePort,
			OldMAC:    oldMAC,
			NewMAC:    mac,
			Method:    fmt.Sprintf("profile:%s", profile.Name),
		})
		applied++
	}

	p.statusText = fmt.Sprintf("Profile %q applied to %d interfaces", profile.Name, applied)
	p.statusErr = false
	return p, nil
}

func (p ProfilesModel) deleteProfile(name string) (ProfilesModel, tea.Cmd) {
	if err := config.DeleteProfile(name); err != nil {
		p.statusText = fmt.Sprintf("Delete failed: %v", err)
		p.statusErr = true
	} else {
		p.statusText = fmt.Sprintf("Profile %q deleted", name)
		p.statusErr = false
	}
	p.Refresh()
	return p, nil
}

func (p ProfilesModel) View() string {
	var b strings.Builder

	b.WriteString("  " + styleLabel.Render("Profiles") + "\n\n")

	// Save mode.
	if p.saving {
		b.WriteString("  " + stylePrompt.Render("Profile name:") + "\n\n")
		inputStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
		cursorStyle := lipgloss.NewStyle().Foreground(colorFg).Bold(true)
		display := "  " + inputStyle.Render(p.nameInput) + cursorStyle.Render("█")
		b.WriteString(display)
		b.WriteString("\n\n")
		b.WriteString("  " + keyHint("Enter", "Save") + "  " + keyHint("Esc", "Cancel"))
		return b.String()
	}

	if len(p.profiles) == 0 {
		b.WriteString("  " + styleKeyDesc.Render("No saved profiles"))
		b.WriteString("\n\n")
	} else {
		for i, profile := range p.profiles {
			cursor := "  "
			if i == p.cursor {
				cursor = styleAccent.Render("▸ ")
			}

			age := time.Since(profile.CreatedAt)
			var ageStr string
			if age < 24*time.Hour {
				ageStr = fmt.Sprintf("%.0fh ago", age.Hours())
			} else {
				ageStr = profile.CreatedAt.Format("2006-01-02")
			}

			b.WriteString(fmt.Sprintf("  %s%-20s %s  %s\n",
				cursor,
				styleVendorName.Render(profile.Name),
				styleKeyDesc.Render(fmt.Sprintf("%d ifaces", len(profile.Entries))),
				styleKeyDesc.Render(ageStr),
			))

			// Show interface MACs if selected.
			if i == p.cursor {
				for port, mac := range profile.Entries {
					b.WriteString(fmt.Sprintf("      %s → %s\n",
						styleKeyDesc.Render(port),
						lipgloss.NewStyle().Foreground(colorAccent).Render(mac),
					))
				}
			}
		}
	}

	b.WriteString("\n")

	// Status.
	if p.statusText != "" {
		if p.statusErr {
			b.WriteString("  " + styleError.Render("✗ "+p.statusText) + "\n\n")
		} else {
			b.WriteString("  " + styleSuccess.Render("✓ "+p.statusText) + "\n\n")
		}
	}

	// Footer.
	b.WriteString(fmt.Sprintf("  %s  %s  %s  %s",
		keyHint("Enter", "Apply"),
		keyHint("N", "New"),
		keyHint("X", "Delete"),
		keyHint("Esc", "Back"),
	))

	return b.String()
}

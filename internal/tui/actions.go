package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/A3-N/macsetgo/internal/config"
	"github.com/A3-N/macsetgo/internal/network"
	"github.com/A3-N/macsetgo/internal/oui"
)

type actionState int

const (
	actionMenu actionState = iota
	actionManualInput
	actionResult
)

// ActionsModel is the MAC action menu for a selected adapter.
type ActionsModel struct {
	iface            *network.Interface
	state            actionState
	cursor           int
	width            int
	height           int
	done             bool
	wantVendorPicker bool
	vendorName       string

	// Manual input state.
	inputBuffer string

	// Result state.
	resultText string
	resultErr  bool
}

type actionItem struct {
	key   string
	label string
	desc  string
}

var menuItems = []actionItem{
	{"r", "Randomize", "Generate and apply a random unicast MAC"},
	{"v", "Vendor Random", "Random MAC with a specific vendor's OUI prefix"},
	{"m", "Manual", "Enter a specific MAC address"},
	{"p", "Restore Permanent", "Reset to factory MAC address"},
	{"a", "Randomize All", "Randomize MAC on all active adapters"},
}

func NewActionsModel() ActionsModel {
	return ActionsModel{}
}

func (a *ActionsModel) SetSize(w, h int) {
	a.width = w
	a.height = h
}

func (a *ActionsModel) SetInterface(iface *network.Interface) {
	a.iface = iface
	a.state = actionMenu
	a.cursor = 0
	a.inputBuffer = ""
	a.resultText = ""
	a.resultErr = false
	a.vendorName = ""
}

func (a *ActionsModel) ApplyVendor(vendor string) {
	a.vendorName = vendor
}

func (a *ActionsModel) DoVendorApply() tea.Cmd {
	return func() tea.Msg {
		return vendorApplyMsg{}
	}
}

type vendorApplyMsg struct{}

func (a ActionsModel) Update(msg tea.Msg) (ActionsModel, tea.Cmd) {
	switch a.state {
	case actionMenu:
		return a.updateMenu(msg)
	case actionManualInput:
		return a.updateManualInput(msg)
	case actionResult:
		return a.updateResult(msg)
	}
	return a, nil
}

func (a ActionsModel) updateMenu(msg tea.Msg) (ActionsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if a.cursor > 0 {
				a.cursor--
			}
		case "down", "j":
			if a.cursor < len(menuItems)-1 {
				a.cursor++
			}
		case "enter":
			return a.executeAction(menuItems[a.cursor].key)
		case "r":
			return a.executeAction("r")
		case "v":
			a.wantVendorPicker = true
			return a, nil
		case "m":
			a.state = actionManualInput
			a.inputBuffer = ""
			return a, nil
		case "p":
			return a.executeAction("p")
		case "a":
			return a.executeAction("a")
		case "esc", "q":
			a.done = true
			return a, nil
		}

	case vendorApplyMsg:
		return a.executeAction("v")
	}
	return a, nil
}

func (a ActionsModel) updateManualInput(msg tea.Msg) (ActionsModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			if a.inputBuffer != "" {
				return a.applyManualMAC()
			}
		case "backspace":
			if len(a.inputBuffer) > 0 {
				a.inputBuffer = a.inputBuffer[:len(a.inputBuffer)-1]
			}
		case "esc":
			a.state = actionMenu
			return a, nil
		default:
			if len(msg.String()) == 1 {
				ch := msg.String()[0]
				// Allow hex digits and colons.
				if (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') || ch == ':' {
					a.inputBuffer += string(ch)
					// Auto-insert colons.
					clean := strings.ReplaceAll(a.inputBuffer, ":", "")
					if len(clean) > 0 && len(clean)%2 == 0 && len(clean) < 12 && !strings.HasSuffix(a.inputBuffer, ":") {
						a.inputBuffer += ":"
					}
				}
			}
		}
	}
	return a, nil
}

func (a ActionsModel) updateResult(msg tea.Msg) (ActionsModel, tea.Cmd) {
	if _, ok := msg.(tea.KeyMsg); ok {
		a.done = true
	}
	return a, nil
}

func (a ActionsModel) executeAction(key string) (ActionsModel, tea.Cmd) {
	if a.iface == nil {
		a.state = actionResult
		a.resultText = "No interface selected"
		a.resultErr = true
		return a, nil
	}

	switch key {
	case "r":
		return a.doRandomize()
	case "v":
		return a.doVendorRandom()
	case "p":
		return a.doRestore()
	case "a":
		return a.doRandomizeAll()
	}
	return a, nil
}

func (a ActionsModel) doRandomize() (ActionsModel, tea.Cmd) {
	oldMAC := a.iface.CurrentMAC
	newMAC, err := network.GenerateRandomMAC()
	if err != nil {
		a.state = actionResult
		a.resultText = fmt.Sprintf("Failed to generate MAC: %v", err)
		a.resultErr = true
		return a, nil
	}

	if err := network.SetMAC(a.iface, newMAC); err != nil {
		a.state = actionResult
		a.resultText = fmt.Sprintf("Failed to set MAC: %v", err)
		a.resultErr = true
		return a, nil
	}

	_ = config.LogChange(config.HistoryEntry{
		Interface: a.iface.Name,
		PortName:  a.iface.HardwarePort,
		OldMAC:    oldMAC,
		NewMAC:    newMAC,
		Method:    "random",
	})

	a.state = actionResult
	a.resultText = fmt.Sprintf("MAC changed: %s → %s", oldMAC, newMAC)
	a.resultErr = false
	return a, nil
}

func (a ActionsModel) doVendorRandom() (ActionsModel, tea.Cmd) {
	if a.vendorName == "" {
		a.state = actionResult
		a.resultText = "No vendor selected"
		a.resultErr = true
		return a, nil
	}

	oldMAC := a.iface.CurrentMAC
	newMAC, err := oui.RandomMACForVendor(a.vendorName)
	if err != nil {
		a.state = actionResult
		a.resultText = fmt.Sprintf("Failed to generate vendor MAC: %v", err)
		a.resultErr = true
		return a, nil
	}

	if err := network.SetMAC(a.iface, newMAC); err != nil {
		a.state = actionResult
		a.resultText = fmt.Sprintf("Failed to set MAC: %v", err)
		a.resultErr = true
		return a, nil
	}

	_ = config.LogChange(config.HistoryEntry{
		Interface: a.iface.Name,
		PortName:  a.iface.HardwarePort,
		OldMAC:    oldMAC,
		NewMAC:    newMAC,
		Method:    fmt.Sprintf("vendor:%s", a.vendorName),
	})

	a.state = actionResult
	a.resultText = fmt.Sprintf("MAC changed (%s): %s → %s", a.vendorName, oldMAC, newMAC)
	a.resultErr = false
	return a, nil
}

func (a ActionsModel) applyManualMAC() (ActionsModel, tea.Cmd) {
	mac := strings.ToLower(strings.TrimSpace(a.inputBuffer))
	if err := network.ValidateMAC(mac); err != nil {
		a.state = actionResult
		a.resultText = fmt.Sprintf("Invalid MAC: %v", err)
		a.resultErr = true
		return a, nil
	}

	if network.IsMulticast(mac) {
		a.state = actionResult
		a.resultText = "Warning: multicast MAC may not work. Set anyway? Press any key to go back."
		a.resultErr = true
		// For simplicity, we still set it — just warn.
	}

	oldMAC := a.iface.CurrentMAC
	if err := network.SetMAC(a.iface, mac); err != nil {
		a.state = actionResult
		a.resultText = fmt.Sprintf("Failed to set MAC: %v", err)
		a.resultErr = true
		return a, nil
	}

	_ = config.LogChange(config.HistoryEntry{
		Interface: a.iface.Name,
		PortName:  a.iface.HardwarePort,
		OldMAC:    oldMAC,
		NewMAC:    mac,
		Method:    "manual",
	})

	a.state = actionResult
	a.resultText = fmt.Sprintf("MAC changed: %s → %s", oldMAC, mac)
	a.resultErr = false
	return a, nil
}

func (a ActionsModel) doRestore() (ActionsModel, tea.Cmd) {
	oldMAC := a.iface.CurrentMAC
	if err := network.RestorePermanentMAC(a.iface); err != nil {
		a.state = actionResult
		a.resultText = fmt.Sprintf("Failed to restore: %v", err)
		a.resultErr = true
		return a, nil
	}

	_ = config.LogChange(config.HistoryEntry{
		Interface: a.iface.Name,
		PortName:  a.iface.HardwarePort,
		OldMAC:    oldMAC,
		NewMAC:    a.iface.PermanentMAC,
		Method:    "restore",
	})

	a.state = actionResult
	a.resultText = fmt.Sprintf("MAC restored: %s → %s", oldMAC, a.iface.PermanentMAC)
	a.resultErr = false
	return a, nil
}

func (a ActionsModel) doRandomizeAll() (ActionsModel, tea.Cmd) {
	ifaces, err := network.ListInterfaces()
	if err != nil {
		a.state = actionResult
		a.resultText = fmt.Sprintf("Failed to list interfaces: %v", err)
		a.resultErr = true
		return a, nil
	}

	var results []string
	for _, iface := range ifaces {
		if !iface.IsUp {
			continue
		}
		oldMAC := iface.CurrentMAC
		newMAC, err := network.GenerateRandomMAC()
		if err != nil {
			results = append(results, fmt.Sprintf("  %s: failed to generate (%v)", iface.Name, err))
			continue
		}
		if err := network.SetMAC(&iface, newMAC); err != nil {
			results = append(results, fmt.Sprintf("  %s: failed to set (%v)", iface.Name, err))
			continue
		}
		_ = config.LogChange(config.HistoryEntry{
			Interface: iface.Name,
			PortName:  iface.HardwarePort,
			OldMAC:    oldMAC,
			NewMAC:    newMAC,
			Method:    "random-all",
		})
		results = append(results, fmt.Sprintf("  %s: %s → %s", iface.Name, oldMAC, newMAC))
	}

	a.state = actionResult
	if len(results) == 0 {
		a.resultText = "No active interfaces to randomize"
		a.resultErr = true
	} else {
		a.resultText = "Batch randomize:\n" + strings.Join(results, "\n")
		a.resultErr = false
	}
	return a, nil
}

func (a ActionsModel) View() string {
	if a.iface == nil {
		return "No interface selected"
	}

	var b strings.Builder

	// Interface header.
	header := fmt.Sprintf("  %s  %s  %s",
		styleLabel.Render(a.iface.Name),
		styleKeyDesc.Render(a.iface.HardwarePort),
		styleKeyDesc.Render("("+a.iface.CurrentMAC+")"),
	)
	b.WriteString(header)
	b.WriteString("\n\n")

	switch a.state {
	case actionMenu:
		for i, item := range menuItems {
			cursor := "  "
			if i == a.cursor {
				cursor = styleAccent.Render("▸ ")
			}
			b.WriteString(fmt.Sprintf("  %s%s  %s\n",
				cursor,
				styleKey.Render("["+item.key+"]")+" "+lipgloss.NewStyle().Foreground(colorFg).Render(item.label),
				styleKeyDesc.Render(item.desc),
			))
		}
		b.WriteString("\n")
		b.WriteString("  " + keyHint("Esc", "Back"))

	case actionManualInput:
		b.WriteString("  " + stylePrompt.Render("Enter MAC address:") + "\n\n")

		// Render input inline: show typed chars in accent color, cursor block, then dim placeholder for remaining.
		inputStyle := lipgloss.NewStyle().Foreground(colorAccent).Bold(true)
		cursorStyle := lipgloss.NewStyle().Foreground(colorFg).Bold(true)
		placeholderStyle := lipgloss.NewStyle().Foreground(colorDim)

		// Build the display: typed text + cursor + remaining placeholder
		placeholder := "aa:bb:cc:dd:ee:ff"
		display := inputStyle.Render(a.inputBuffer) + cursorStyle.Render("█")
		if len(a.inputBuffer) < len(placeholder) {
			display += placeholderStyle.Render(placeholder[len(a.inputBuffer):])
		}
		b.WriteString("  " + display)
		b.WriteString("\n\n")
		b.WriteString("  " + keyHint("Enter", "Apply") + "  " + keyHint("Esc", "Cancel"))

	case actionResult:
		if a.resultErr {
			b.WriteString("  " + styleError.Render("✗ "+a.resultText))
		} else {
			b.WriteString("  " + styleSuccess.Render("✓ "+a.resultText))
		}
		b.WriteString("\n\n")
		b.WriteString("  " + styleKeyDesc.Render("Press any key to continue"))
	}

	return b.String()
}

var styleAccent = lipgloss.NewStyle().Foreground(colorAccent)

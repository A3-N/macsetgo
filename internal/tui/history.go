package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/A3-N/macsetgo/internal/config"
	"github.com/A3-N/macsetgo/internal/network"
)

// HistoryModel displays the change history log with restore support.
type HistoryModel struct {
	entries    []config.HistoryEntry
	cursor     int
	width      int
	height     int
	done       bool
	statusText string
	statusErr  bool
}

func NewHistoryModel() HistoryModel {
	return HistoryModel{}
}

func (h *HistoryModel) SetSize(w, h2 int) {
	h.width = w
	h.height = h2
}

func (h *HistoryModel) Refresh() {
	entries, _ := config.GetHistory(100)
	h.entries = entries
	h.cursor = 0
	h.statusText = ""
	h.statusErr = false
}

func (h HistoryModel) Update(msg tea.Msg) (HistoryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if h.cursor > 0 {
				h.cursor--
			}
		case "down", "j":
			if h.cursor < len(h.entries)-1 {
				h.cursor++
			}
		case "r":
			// Restore the new MAC from the selected history entry.
			if h.cursor >= 0 && h.cursor < len(h.entries) {
				return h.restoreEntry(h.entries[h.cursor])
			}
		case "o":
			// Restore the OLD MAC from the selected history entry (revert).
			if h.cursor >= 0 && h.cursor < len(h.entries) {
				return h.revertEntry(h.entries[h.cursor])
			}
		case "esc", "q":
			h.done = true
		}
	}
	return h, nil
}

func (h HistoryModel) restoreEntry(entry config.HistoryEntry) (HistoryModel, tea.Cmd) {
	if entry.NewMAC == "" || entry.Interface == "" {
		h.statusText = "Invalid history entry"
		h.statusErr = true
		return h, nil
	}

	iface, err := network.GetInterface(entry.Interface)
	if err != nil {
		h.statusText = fmt.Sprintf("Interface %s not found", entry.Interface)
		h.statusErr = true
		return h, nil
	}

	oldMAC := iface.CurrentMAC
	if err := network.SetMAC(iface, entry.NewMAC); err != nil {
		h.statusText = fmt.Sprintf("Failed: %v", err)
		h.statusErr = true
		return h, nil
	}

	_ = config.LogChange(config.HistoryEntry{
		Interface: iface.Name,
		PortName:  iface.HardwarePort,
		OldMAC:    oldMAC,
		NewMAC:    entry.NewMAC,
		Method:    "history-restore",
	})

	h.statusText = fmt.Sprintf("Restored %s on %s: %s → %s", entry.NewMAC, entry.Interface, oldMAC, entry.NewMAC)
	h.statusErr = false
	return h, nil
}

func (h HistoryModel) revertEntry(entry config.HistoryEntry) (HistoryModel, tea.Cmd) {
	if entry.OldMAC == "" || entry.Interface == "" {
		h.statusText = "Invalid history entry"
		h.statusErr = true
		return h, nil
	}

	iface, err := network.GetInterface(entry.Interface)
	if err != nil {
		h.statusText = fmt.Sprintf("Interface %s not found", entry.Interface)
		h.statusErr = true
		return h, nil
	}

	currentMAC := iface.CurrentMAC
	if err := network.SetMAC(iface, entry.OldMAC); err != nil {
		h.statusText = fmt.Sprintf("Failed: %v", err)
		h.statusErr = true
		return h, nil
	}

	_ = config.LogChange(config.HistoryEntry{
		Interface: iface.Name,
		PortName:  iface.HardwarePort,
		OldMAC:    currentMAC,
		NewMAC:    entry.OldMAC,
		Method:    "history-revert",
	})

	h.statusText = fmt.Sprintf("Reverted %s on %s: %s → %s", entry.OldMAC, entry.Interface, currentMAC, entry.OldMAC)
	h.statusErr = false
	return h, nil
}

func (h HistoryModel) View() string {
	var b strings.Builder

	b.WriteString("  " + styleLabel.Render("Change History") + "\n\n")

	if len(h.entries) == 0 {
		b.WriteString("  " + styleKeyDesc.Render("No history yet"))
		b.WriteString("\n\n")
		b.WriteString("  " + keyHint("Esc", "Back"))
		return b.String()
	}

	// Header.
	header := fmt.Sprintf("  %-20s %-8s %-14s %-19s %-19s %s",
		"TIME", "IFACE", "TYPE", "OLD MAC", "NEW MAC", "METHOD",
	)
	b.WriteString(styleTableHeader.Render(header))
	b.WriteString("\n")

	sepWidth := h.width - 6
	if sepWidth < 90 {
		sepWidth = 90
	}
	sep := "  " + strings.Repeat("─", sepWidth)
	b.WriteString(lipgloss.NewStyle().Foreground(colorBorder).Render(sep))
	b.WriteString("\n")

	// Visible rows.
	maxVisible := h.height - 10
	if maxVisible < 5 {
		maxVisible = 5
	}

	start := 0
	if h.cursor >= maxVisible {
		start = h.cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(h.entries) {
		end = len(h.entries)
	}

	for i := start; i < end; i++ {
		entry := h.entries[i]
		isSelected := i == h.cursor

		timeStr := entry.Timestamp.Format("2006-01-02 15:04:05")
		portName := entry.PortName
		if len(portName) > 14 {
			portName = portName[:11] + "..."
		}

		row := fmt.Sprintf("  %-20s %-8s %-14s %-19s %-19s %s",
			timeStr,
			entry.Interface,
			portName,
			entry.OldMAC,
			entry.NewMAC,
			entry.Method,
		)

		if isSelected {
			// Full-width highlight.
			rowWidth := h.width - 4
			if rowWidth < 90 {
				rowWidth = 90
			}
			if len(row) < rowWidth {
				row += strings.Repeat(" ", rowWidth-len(row))
			}
			row = lipgloss.NewStyle().
				Foreground(colorAccent).
				Background(colorSelected).
				Bold(true).
				Render(row)
		} else {
			row = styleTableRow.Render(row)
		}

		b.WriteString(row)
		b.WriteString("\n")
	}

	// Status.
	if h.statusText != "" {
		b.WriteString("\n")
		if h.statusErr {
			b.WriteString("  " + styleError.Render("✗ "+h.statusText))
		} else {
			b.WriteString("  " + styleSuccess.Render("✓ "+h.statusText))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %d entries  %s  %s  %s",
		len(h.entries),
		keyHint("R", "Restore MAC"),
		keyHint("O", "Revert to old"),
		keyHint("Esc", "Back"),
	))

	return b.String()
}

package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/A3-N/macsetgo/internal/oui"
)

// VendorPickerModel lets the user search and select a vendor for OUI-based MAC generation.
type VendorPickerModel struct {
	vendors   []oui.Vendor
	filtered  []oui.Vendor
	cursor    int
	filter    string
	width     int
	height    int
	done      bool
	selected  string // Selected vendor name, empty if cancelled.
}

func NewVendorPickerModel() VendorPickerModel {
	vendors := oui.ListVendors()
	return VendorPickerModel{
		vendors:  vendors,
		filtered: vendors,
	}
}

func (v *VendorPickerModel) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *VendorPickerModel) Reset() {
	v.filter = ""
	v.cursor = 0
	v.selected = ""
	v.done = false
	v.filtered = v.vendors
}

func (v VendorPickerModel) Update(msg tea.Msg) (VendorPickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "ctrl+p":
			if v.cursor > 0 {
				v.cursor--
			}
		case "down", "ctrl+n":
			if v.cursor < len(v.filtered)-1 {
				v.cursor++
			}
		case "enter":
			if v.cursor >= 0 && v.cursor < len(v.filtered) {
				v.selected = v.filtered[v.cursor].Name
				v.done = true
			}
		case "esc":
			v.selected = ""
			v.done = true
		case "backspace":
			if len(v.filter) > 0 {
				v.filter = v.filter[:len(v.filter)-1]
				v.applyFilter()
			}
		default:
			if len(msg.String()) == 1 {
				v.filter += msg.String()
				v.applyFilter()
			}
		}
	}
	return v, nil
}

func (v *VendorPickerModel) applyFilter() {
	if v.filter == "" {
		v.filtered = v.vendors
	} else {
		lower := strings.ToLower(v.filter)
		v.filtered = nil
		for _, vendor := range v.vendors {
			if strings.Contains(strings.ToLower(vendor.Name), lower) {
				v.filtered = append(v.filtered, vendor)
			}
		}
	}
	v.cursor = 0
}

func (v VendorPickerModel) View() string {
	var b strings.Builder

	b.WriteString("  " + styleLabel.Render("Select Vendor") + "\n\n")

	// Search bar — inline style, no border box.
	searchIcon := stylePrompt.Render("⌕ Search: ")
	if v.filter == "" {
		searchText := styleKeyDesc.Render("type to filter...") + styleAccent.Render("█")
		b.WriteString("  " + searchIcon + searchText)
	} else {
		searchText := styleAccent.Render(v.filter) + styleKeyDesc.Render("█")
		b.WriteString("  " + searchIcon + searchText)
	}
	b.WriteString("\n\n")

	if len(v.filtered) == 0 {
		b.WriteString("  " + styleKeyDesc.Render("No vendors match"))
		return b.String()
	}

	// Show vendors — limit visible rows.
	maxVisible := v.height - 8
	if maxVisible < 5 {
		maxVisible = 5
	}

	start := 0
	if v.cursor >= maxVisible {
		start = v.cursor - maxVisible + 1
	}
	end := start + maxVisible
	if end > len(v.filtered) {
		end = len(v.filtered)
	}

	for i := start; i < end; i++ {
		vendor := v.filtered[i]
		cursor := "  "
		if i == v.cursor {
			cursor = styleAccent.Render("▸ ")
		}

		ouiPreview := ""
		if len(vendor.OUIs) > 0 {
			ouiPreview = vendor.OUIs[0]
			if len(vendor.OUIs) > 1 {
				ouiPreview += fmt.Sprintf(" (+%d more)", len(vendor.OUIs)-1)
			}
		}

		b.WriteString(fmt.Sprintf("  %s%-20s %s\n",
			cursor,
			styleVendorName.Render(vendor.Name),
			styleVendorOUI.Render(ouiPreview),
		))
	}

	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  %s  %d/%d vendors",
		keyHint("Enter", "Select")+"  "+keyHint("Esc", "Cancel"),
		len(v.filtered), len(v.vendors),
	))

	return b.String()
}

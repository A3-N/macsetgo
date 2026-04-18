package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/A3-N/macsetgo/internal/network"
	"github.com/A3-N/macsetgo/internal/oui"
)

// DashboardModel is the main adapter list view.
type DashboardModel struct {
	interfaces     []network.Interface
	cursor         int
	width          int
	height         int
	selectedAction bool
	navRequest     *Page
}

func NewDashboardModel() DashboardModel {
	return DashboardModel{}
}

func (d *DashboardModel) SetSize(w, h int) {
	d.width = w
	d.height = h
}

func (d *DashboardModel) SetInterfaces(ifaces []network.Interface) {
	d.interfaces = ifaces
	if d.cursor >= len(ifaces) && len(ifaces) > 0 {
		d.cursor = len(ifaces) - 1
	}
}

func (d *DashboardModel) SelectedInterface() *network.Interface {
	if d.cursor >= 0 && d.cursor < len(d.interfaces) {
		iface := d.interfaces[d.cursor]
		return &iface
	}
	return nil
}

func (d DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if d.cursor > 0 {
				d.cursor--
			}
		case "down", "j":
			if d.cursor < len(d.interfaces)-1 {
				d.cursor++
			}
		case "enter":
			d.selectedAction = true

		case "p":
			page := PageProfiles
			d.navRequest = &page
		case "h":
			page := PageHistory
			d.navRequest = &page
		case "d":
			page := PageDaemon
			d.navRequest = &page
		}
	}
	return d, nil
}

func (d DashboardModel) View() string {
	if len(d.interfaces) == 0 {
		return lipgloss.NewStyle().Padding(1, 2).Render(
			styleWarning.Render("No network interfaces detected"),
		)
	}

	var b strings.Builder

	// Column widths.
	colSt := 4
	colIface := 9
	colType := 12
	colCurrent := 19
	colPermanent := 19
	colVendor := 16
	colStatus := 10

	sepWidth := d.width - 6
	if sepWidth < 80 {
		sepWidth = 80
	}

	// Header.
	header := fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s %-*s %-*s",
		colSt, "ST",
		colIface, "IFACE",
		colType, "TYPE",
		colCurrent, "CURRENT MAC",
		colPermanent, "PERMANENT MAC",
		colVendor, "VENDOR",
		colStatus, "STATUS",
	)
	// Pad header to full width.
	headerWidth := lipgloss.Width(header)
	if headerWidth < d.width-4 {
		header += strings.Repeat(" ", d.width-4-headerWidth)
	}
	b.WriteString(styleTableHeader.Render(header))
	b.WriteString("\n")

	// Separator.
	sep := "  " + strings.Repeat("─", sepWidth)
	b.WriteString(lipgloss.NewStyle().Foreground(colorBorder).Render(sep))
	b.WriteString("\n")

	// Rows.
	rowWidth := d.width - 4
	if rowWidth < 80 {
		rowWidth = 80
	}

	for i, iface := range d.interfaces {
		isSelected := i == d.cursor

		// Status indicator.
		var stIcon string
		if iface.IsUp {
			stIcon = "●"
		} else {
			stIcon = "○"
		}

		// Current MAC.
		currentMAC := iface.CurrentMAC
		if currentMAC == "" {
			currentMAC = "—"
		}

		permanentMAC := iface.PermanentMAC
		if permanentMAC == "" {
			permanentMAC = "—"
		}

		// Vendor lookup.
		vendor := ""
		if iface.CurrentMAC != "" {
			vendor = oui.LookupVendor(iface.CurrentMAC)
			if vendor == "Unknown" {
				vendor = ""
			}
		}

		// Status text (plain).
		var statusPlain string
		if iface.IsSpoofed {
			statusPlain = "SPOOFED"
		} else if iface.IsUp {
			statusPlain = "NATIVE"
		} else {
			statusPlain = "DOWN"
		}

		ifaceType := iface.ShortType()

		// Build plain text row with fixed-width columns.
		plainRow := fmt.Sprintf("  %s %-*s %-*s %-*s %-*s %-*s %s",
			stIcon+"   ",
			colIface, trunc(iface.Name, colIface-1),
			colType, trunc(ifaceType, colType-1),
			colCurrent, currentMAC,
			colPermanent, permanentMAC,
			colVendor, trunc(vendor, colVendor-1),
			statusPlain,
		)

		// Pad to full width.
		plainRowWidth := lipgloss.Width(plainRow)
		if plainRowWidth < rowWidth {
			plainRow += strings.Repeat(" ", rowWidth-plainRowWidth)
		}

		// Now apply colors to the whole row.
		if isSelected {
			// Selected row: cyan text on dark blue background, full width.
			row := styleTableSelected.Render(plainRow)
			b.WriteString(row)
		} else {
			// Normal row: colorize individual parts.
			var stStyled string
			if iface.IsUp {
				stStyled = styleUp.Render(stIcon) + "   "
			} else {
				stStyled = styleDown.Render(stIcon) + "   "
			}

			ifaceNamePadded := fmt.Sprintf("%-*s", colIface, trunc(iface.Name, colIface-1))

			typePadded := fmt.Sprintf("%-*s", colType, trunc(ifaceType, colType-1))
			typeStyled := typePadded
			if iface.IsUSB {
				typeStyled = styleOrange.Render(typePadded)
			}

			currentMACPadded := fmt.Sprintf("%-*s", colCurrent, currentMAC)
			permanentMACPadded := fmt.Sprintf("%-*s", colPermanent, permanentMAC)
			vendorPadded := fmt.Sprintf("%-*s", colVendor, trunc(vendor, colVendor-1))

			var statusStyled string
			if iface.IsSpoofed {
				statusStyled = styleSpoofed.Render(statusPlain)
			} else if iface.IsUp {
				statusStyled = styleUp.Render(statusPlain)
			} else {
				statusStyled = styleDown.Render(statusPlain)
			}

			styledRow := fmt.Sprintf("  %s %s %s %s %s %s %s",
				stStyled,
				ifaceNamePadded,
				typeStyled,
				currentMACPadded,
				permanentMACPadded,
				vendorPadded,
				statusStyled,
			)
			b.WriteString(styleTableRow.Render(styledRow))
		}
		b.WriteString("\n")
	}

	// Footer with key hints.
	b.WriteString("\n")
	footer := fmt.Sprintf("  %s  %s  %s  %s  %s",
		keyHint("Enter", "Actions"),
		keyHint("P", "Profiles"),
		keyHint("H", "History"),
		keyHint("D", "Daemon"),
		keyHint("?", "Help"),
	)
	b.WriteString(footer)

	// Detail panel for selected interface.
	if sel := d.SelectedInterface(); sel != nil {
		detail := network.GetInterfaceDetail(sel.Name)

		b.WriteString("\n\n")
		detailSep := "  " + strings.Repeat("─", sepWidth)
		b.WriteString(lipgloss.NewStyle().Foreground(colorBorder).Render(detailSep))
		b.WriteString("\n")
		b.WriteString("  " + styleLabel.Render(sel.Name+" — "+sel.HardwarePort) + "\n\n")

		// Row 1: IP info
		col1 := 24
		valStyle := lipgloss.NewStyle().Foreground(colorFg)
		if detail.IPv4 != "" {
			b.WriteString(fmt.Sprintf("  %-*s %s\n",
				col1, styleKeyDesc.Render("IPv4:"),
				valStyle.Render(detail.IPv4+"/"+detail.Netmask),
			))
		}
		if detail.IPv6 != "" {
			b.WriteString(fmt.Sprintf("  %-*s %s\n",
				col1, styleKeyDesc.Render("IPv6:"),
				valStyle.Render(detail.IPv6),
			))
		}
		if detail.IPv4 == "" && detail.IPv6 == "" {
			b.WriteString(fmt.Sprintf("  %-*s %s\n",
				col1, styleKeyDesc.Render("IP:"),
				styleDown.Render("no address assigned"),
			))
		}

		// Row 2: Network info (NAC-relevant)
		if detail.Gateway != "" {
			b.WriteString(fmt.Sprintf("  %-*s %s\n",
				col1, styleKeyDesc.Render("Gateway:"),
				valStyle.Render(detail.Gateway),
			))
		}
		if len(detail.DNS) > 0 {
			dnsStr := strings.Join(detail.DNS, ", ")
			b.WriteString(fmt.Sprintf("  %-*s %s\n",
				col1, styleKeyDesc.Render("DNS:"),
				valStyle.Render(dnsStr),
			))
		}
		if detail.DHCPServer != "" {
			dhcpLine := detail.DHCPServer
			if detail.LeaseTime != "" {
				dhcpLine += "  " + styleKeyDesc.Render("lease:") + " " + detail.LeaseTime
			}
			b.WriteString(fmt.Sprintf("  %-*s %s\n",
				col1, styleKeyDesc.Render("DHCP Server:"),
				valStyle.Render(dhcpLine),
			))
		}

		// 802.1X status.
		if detail.Dot1XStatus != "none" {
			var dot1xStyled string
			switch detail.Dot1XStatus {
			case "active":
				dot1xStyled = styleWarning.Render("● ACTIVE")
			case "configured", "configured (system)":
				dot1xStyled = styleKeyDesc.Render("configured")
			default:
				dot1xStyled = valStyle.Render(detail.Dot1XStatus)
			}
			if detail.Dot1XMethod != "" {
				dot1xStyled += "  " + styleKeyDesc.Render("method:") + " " + valStyle.Render(detail.Dot1XMethod)
			}
			b.WriteString(fmt.Sprintf("  %-*s %s\n",
				col1, styleKeyDesc.Render("802.1X:"),
				dot1xStyled,
			))
		}

		// Row 3: Media / MTU
		if detail.Media != "" {
			b.WriteString(fmt.Sprintf("  %-*s %s\n",
				col1, styleKeyDesc.Render("Media:"),
				valStyle.Render(detail.Media),
			))
		}
		if detail.MTU != "" {
			b.WriteString(fmt.Sprintf("  %-*s %s\n",
				col1, styleKeyDesc.Render("MTU:"),
				valStyle.Render(detail.MTU),
			))
		}

		// Row 4: Traffic stats
		if detail.PktsIn != "" {
			b.WriteString("\n")
			b.WriteString(fmt.Sprintf("  %-*s %s %s    %s %s\n",
				col1, styleKeyDesc.Render("Packets:"),
				styleUp.Render("▼"),
				valStyle.Render(detail.PktsIn+" in"),
				styleAccent.Render("▲"),
				valStyle.Render(detail.PktsOut+" out"),
			))
			b.WriteString(fmt.Sprintf("  %-*s %s %s    %s %s\n",
				col1, styleKeyDesc.Render("Data:"),
				styleUp.Render("▼"),
				valStyle.Render(detail.BytesIn+" in"),
				styleAccent.Render("▲"),
				valStyle.Render(detail.BytesOut+" out"),
			))
			if detail.ErrsIn != "0" || detail.ErrsOut != "0" {
				b.WriteString(fmt.Sprintf("  %-*s %s in / %s out\n",
					col1, styleWarning.Render("Errors:"),
					detail.ErrsIn, detail.ErrsOut,
				))
			}
		}
	}

	return b.String()
}

// styleOrange for USB adapter badge.
var styleOrange = lipgloss.NewStyle().Foreground(colorOrange)

func trunc(s string, length int) string {
	runes := []rune(s)
	if len(runes) > length {
		return string(runes[:length-1]) + "…"
	}
	return s
}

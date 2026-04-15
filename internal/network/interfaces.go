package network

import (
	"fmt"
	"os/exec"
	"strings"
)

// ListInterfaces discovers all network interfaces on the system by parsing
// the output of `networksetup -listallhardwareports`. It cross-references
// ifconfig for current MAC and status, and networksetup for permanent MAC.
func ListInterfaces() ([]Interface, error) {
	out, err := exec.Command("networksetup", "-listallhardwareports").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("networksetup: %w", err)
	}

	ifaces := parseHardwarePorts(string(out))

	// Enrich each interface with current MAC, status, and permanent MAC.
	for i := range ifaces {
		ifaces[i].CurrentMAC = getCurrentMAC(ifaces[i].Name)
		ifaces[i].PermanentMAC = getPermanentMAC(ifaces[i].Name)
		ifaces[i].IsUp = isInterfaceUp(ifaces[i].Name)
		ifaces[i].Type = classifyType(ifaces[i].HardwarePort)
		ifaces[i].IsUSB = ifaces[i].Type == TypeUSBEthernet
		ifaces[i].IsSpoofed = ifaces[i].CurrentMAC != "" &&
			ifaces[i].PermanentMAC != "" &&
			!strings.EqualFold(ifaces[i].CurrentMAC, ifaces[i].PermanentMAC)
	}

	return ifaces, nil
}

// GetInterface returns a single interface by device name (e.g. "en0").
func GetInterface(name string) (*Interface, error) {
	ifaces, err := ListInterfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Name == name {
			return &iface, nil
		}
	}
	return nil, fmt.Errorf("interface %q not found", name)
}

// parseHardwarePorts parses the output of `networksetup -listallhardwareports`.
// Each entry looks like:
//
//	Hardware Port: Wi-Fi
//	Device: en0
//	Ethernet Address: aa:bb:cc:dd:ee:ff
func parseHardwarePorts(output string) []Interface {
	var ifaces []Interface
	var current Interface
	inEntry := false

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "Hardware Port:") {
			if inEntry && current.Name != "" {
				ifaces = append(ifaces, current)
			}
			current = Interface{}
			current.HardwarePort = strings.TrimPrefix(line, "Hardware Port: ")
			inEntry = true
		} else if strings.HasPrefix(line, "Device:") {
			current.Name = strings.TrimPrefix(line, "Device: ")
		} else if strings.HasPrefix(line, "Ethernet Address:") {
			// This is the "permanent" MAC from networksetup listing, but we use
			// the more reliable networksetup -getmacaddress for the actual value.
		}
	}
	if inEntry && current.Name != "" {
		ifaces = append(ifaces, current)
	}

	return ifaces
}

// classifyType maps a hardware port name to a normalized InterfaceType.
func classifyType(hardwarePort string) InterfaceType {
	hp := strings.ToLower(hardwarePort)
	switch {
	case hp == "wi-fi":
		return TypeWiFi
	case strings.Contains(hp, "usb"):
		return TypeUSBEthernet
	case strings.Contains(hp, "thunderbolt"):
		return TypeThunderbolt
	case strings.Contains(hp, "bluetooth"):
		return TypeBluetooth
	case strings.Contains(hp, "firewire"):
		return TypeFirewire
	case strings.Contains(hp, "ethernet"):
		return TypeEthernet
	default:
		return TypeOther
	}
}

// getCurrentMAC retrieves the current (possibly spoofed) MAC from ifconfig.
func getCurrentMAC(device string) string {
	out, err := exec.Command("ifconfig", device).CombinedOutput()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "ether ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}
	return ""
}

// getPermanentMAC retrieves the hardware/permanent MAC via networksetup.
func getPermanentMAC(device string) string {
	out, err := exec.Command("networksetup", "-getmacaddress", device).CombinedOutput()
	if err != nil {
		return ""
	}
	// Output: "Ethernet Address: aa:bb:cc:dd:ee:ff (Hardware Port: Wi-Fi)"
	fields := strings.Fields(string(out))
	for _, f := range fields {
		if strings.Count(f, ":") == 5 && len(f) == 17 {
			return strings.ToLower(f)
		}
	}
	return ""
}

// isInterfaceUp checks ifconfig status flags for the UP flag.
func isInterfaceUp(device string) bool {
	out, err := exec.Command("ifconfig", device).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "status: active")
}

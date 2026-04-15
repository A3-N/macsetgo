package network

// InterfaceType represents the kind of network interface.
type InterfaceType string

const (
	TypeWiFi              InterfaceType = "Wi-Fi"
	TypeEthernet          InterfaceType = "Ethernet"
	TypeUSBEthernet       InterfaceType = "USB Ethernet"
	TypeThunderbolt       InterfaceType = "Thunderbolt"
	TypeBluetooth         InterfaceType = "Bluetooth"
	TypeFirewire          InterfaceType = "FireWire"
	TypeOther             InterfaceType = "Other"
)

// Interface represents a macOS network interface with all relevant metadata.
type Interface struct {
	Name         string        // Device name, e.g. "en0"
	HardwarePort string       // Human-readable name from networksetup, e.g. "Wi-Fi", "USB 10/100/1000 LAN"
	Type         InterfaceType // Classified type
	CurrentMAC   string        // Current MAC address (may differ from permanent if spoofed)
	PermanentMAC string        // Factory/permanent MAC address
	IsUp         bool          // Whether the interface is currently active
	IsUSB        bool          // Whether this is an external USB adapter
	IsSpoofed    bool          // Whether current MAC differs from permanent
}

// ShortType returns a compact label for the interface type.
func (iface *Interface) ShortType() string {
	switch iface.Type {
	case TypeWiFi:
		return "Wi-Fi"
	case TypeUSBEthernet:
		return "USB Eth"
	case TypeThunderbolt:
		return "TB Eth"
	case TypeBluetooth:
		return "BT"
	case TypeFirewire:
		return "FW"
	case TypeEthernet:
		return "Ethernet"
	default:
		return "Other"
	}
}

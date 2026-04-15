package network

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// InterfaceDetail holds live metadata for a network interface.
type InterfaceDetail struct {
	IPv4       string   // e.g. "192.168.1.156"
	Netmask    string   // e.g. "255.255.255.0"
	Broadcast  string   // e.g. "192.168.1.255"
	IPv6       string   // Primary IPv6 address
	Gateway    string   // Default gateway IP
	DNS        []string // DNS server IPs
	DHCPServer string   // DHCP server that issued the lease
	LeaseTime  string   // DHCP lease duration
	MTU        string   // e.g. "1500"
	Media      string   // e.g. "autoselect", "1000baseT"
	Flags      string   // e.g. "UP,BROADCAST,RUNNING"
	PktsIn     string   // Packets received
	PktsOut    string   // Packets sent
	BytesIn    string   // Bytes received
	BytesOut   string   // Bytes sent
	ErrsIn     string   // Input errors
	ErrsOut    string   // Output errors
	Colls      string   // Collisions
	// 802.1X
	Dot1XStatus string // "authenticated", "authenticating", "configured", "none"
	Dot1XMethod string // EAP method if available (e.g. "PEAP", "TLS")
}

// GetInterfaceDetail fetches live metadata for a given interface.
func GetInterfaceDetail(device string) InterfaceDetail {
	var d InterfaceDetail

	// Parse ifconfig output.
	out, err := exec.Command("ifconfig", device).CombinedOutput()
	if err == nil {
		d.parseIfconfig(string(out))
	}

	// Parse netstat for packet/byte counters.
	out, err = exec.Command("netstat", "-bI", device).CombinedOutput()
	if err == nil {
		d.parseNetstat(string(out), device)
	}

	// Gateway — find the default route for this interface.
	out, err = exec.Command("route", "-n", "get", "default").CombinedOutput()
	if err == nil {
		d.parseGateway(string(out), device)
	}

	// DNS servers.
	out, err = exec.Command("scutil", "--dns").CombinedOutput()
	if err == nil {
		d.parseDNS(string(out))
	}

	// DHCP lease info.
	out, err = exec.Command("ipconfig", "getpacket", device).CombinedOutput()
	if err == nil {
		d.parseDHCP(string(out))
	}

	// 802.1X detection.
	d.detect8021X(device)

	return d
}

// detect8021X checks for 802.1X authentication state on the interface.
func (d *InterfaceDetail) detect8021X(device string) {
	d.Dot1XStatus = "none"

	// Check if eapolclient is actively running for this interface.
	out, err := exec.Command("ps", "aux").CombinedOutput()
	if err == nil {
		for _, line := range strings.Split(string(out), "\n") {
			if strings.Contains(line, "eapolclient") && strings.Contains(line, device) {
				d.Dot1XStatus = "active"
				// Try to extract EAP method from args.
				for _, method := range []string{"PEAP", "TLS", "TTLS", "EAP-FAST", "LEAP", "MD5"} {
					if strings.Contains(strings.ToUpper(line), method) {
						d.Dot1XMethod = method
						break
					}
				}
				break
			}
		}
	}

	// If not actively running, check if 802.1X is configured.
	if d.Dot1XStatus == "none" {
		configPath := "/Library/Preferences/SystemConfiguration/com.apple.network.eapolclient.configuration.plist"
		if _, err := os.Stat(configPath); err == nil {
			// Config exists — check if this interface has an entry.
			out, err := exec.Command("defaults", "read", configPath).CombinedOutput()
			if err == nil && strings.Contains(string(out), device) {
				d.Dot1XStatus = "configured"
			} else if err == nil && len(string(out)) > 10 {
				// Config exists but may not reference device by name — still noteworthy.
				d.Dot1XStatus = "configured (system)"
			}
		}
	}
}

func (d *InterfaceDetail) parseGateway(output, device string) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "gateway:") {
			d.Gateway = strings.TrimSpace(strings.TrimPrefix(line, "gateway:"))
		}
	}
}

func (d *InterfaceDetail) parseDNS(output string) {
	seen := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "nameserver[") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				ip := strings.TrimSpace(parts[1])
				if !seen[ip] {
					seen[ip] = true
					d.DNS = append(d.DNS, ip)
				}
			}
		}
	}
}

func (d *InterfaceDetail) parseDHCP(output string) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		// "server_identifier (ip): 192.168.1.1"
		if strings.HasPrefix(line, "server_identifier") {
			if idx := strings.LastIndex(line, ":"); idx != -1 {
				d.DHCPServer = strings.TrimSpace(line[idx+1:])
			}
		}

		// "lease_time (uint32): 0xa8c0"
		if strings.HasPrefix(line, "lease_time") {
			if idx := strings.LastIndex(line, ":"); idx != -1 {
				hexVal := strings.TrimSpace(line[idx+1:])
				hexVal = strings.TrimPrefix(hexVal, "0x")
				var secs int64
				if _, err := fmt.Sscanf(hexVal, "%x", &secs); err == nil {
					hours := secs / 3600
					mins := (secs % 3600) / 60
					if hours > 0 {
						d.LeaseTime = fmt.Sprintf("%dh %dm", hours, mins)
					} else {
						d.LeaseTime = fmt.Sprintf("%dm", mins)
					}
				}
			}
		}
	}
}

func (d *InterfaceDetail) parseIfconfig(output string) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)

		// Flags and MTU from first line: "en0: flags=8863<UP,BROADCAST,...> mtu 1500"
		if strings.Contains(line, "flags=") && strings.Contains(line, "<") {
			// Extract flags.
			if start := strings.Index(line, "<"); start != -1 {
				if end := strings.Index(line, ">"); end != -1 && end > start {
					d.Flags = line[start+1 : end]
				}
			}
			// Extract MTU.
			if idx := strings.Index(line, "mtu "); idx != -1 {
				d.MTU = strings.Fields(line[idx:])[1]
			}
		}

		// IPv4: "inet 192.168.1.156 netmask 0xffffff00 broadcast 192.168.1.255"
		if strings.HasPrefix(line, "inet ") && !strings.HasPrefix(line, "inet6") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				d.IPv4 = fields[1]
			}
			for i, f := range fields {
				if f == "netmask" && i+1 < len(fields) {
					d.Netmask = hexMaskToDecimal(fields[i+1])
				}
				if f == "broadcast" && i+1 < len(fields) {
					d.Broadcast = fields[i+1]
				}
			}
		}

		// IPv6: "inet6 fe80::... prefixlen 64"
		if strings.HasPrefix(line, "inet6 ") && d.IPv6 == "" {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				// Clean up scope ID (e.g. "fe80::1%en0" -> "fe80::1")
				addr := fields[1]
				if idx := strings.Index(addr, "%"); idx != -1 {
					addr = addr[:idx]
				}
				d.IPv6 = addr
			}
		}

		// Media: "media: autoselect (1000baseT <full-duplex>)"
		if strings.HasPrefix(line, "media:") {
			d.Media = strings.TrimPrefix(line, "media: ")
		}
	}
}

func (d *InterfaceDetail) parseNetstat(output, device string) {
	// netstat -bI output format:
	// Name  Mtu  Network  Address  Ipkts  Ierrs  Ibytes  Opkts  Oerrs  Obytes  Coll
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}
		// Match the first line for this device that has <Link#> network.
		if fields[0] == device && strings.Contains(fields[2], "Link#") {
			d.PktsIn = formatNumber(fields[4])
			d.ErrsIn = fields[5]
			d.BytesIn = formatBytes(fields[6])
			d.PktsOut = fields[7]
			d.ErrsOut = fields[8]
			d.BytesOut = formatBytes(fields[9])
			d.Colls = fields[10]
			break
		}
	}
}

// hexMaskToDecimal converts "0xffffff00" to "255.255.255.0".
func hexMaskToDecimal(hex string) string {
	hex = strings.TrimPrefix(hex, "0x")
	if len(hex) != 8 {
		return hex
	}
	var octets [4]byte
	for i := 0; i < 4; i++ {
		var b byte
		_, _ = fmt.Sscanf(hex[i*2:i*2+2], "%x", &b)
		octets[i] = b
	}
	return fmt.Sprintf("%d.%d.%d.%d", octets[0], octets[1], octets[2], octets[3])
}

// formatBytes converts a byte count string to human-readable.
func formatBytes(s string) string {
	var n float64
	_, err := fmt.Sscanf(s, "%f", &n)
	if err != nil {
		return s
	}
	switch {
	case n >= 1e12:
		return fmt.Sprintf("%.1f TB", n/1e12)
	case n >= 1e9:
		return fmt.Sprintf("%.1f GB", n/1e9)
	case n >= 1e6:
		return fmt.Sprintf("%.1f MB", n/1e6)
	case n >= 1e3:
		return fmt.Sprintf("%.1f KB", n/1e3)
	default:
		return fmt.Sprintf("%.0f B", n)
	}
}

// formatNumber adds comma separators to large numbers.
func formatNumber(s string) string {
	var n int64
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil {
		return s
	}
	if n < 1000 {
		return s
	}
	// Simple comma formatting.
	str := fmt.Sprintf("%d", n)
	var result []byte
	for i, c := range str {
		if i > 0 && (len(str)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

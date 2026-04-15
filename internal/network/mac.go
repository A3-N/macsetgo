package network

import (
	"crypto/rand"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var macRegex = regexp.MustCompile(`^([0-9a-fA-F]{2}:){5}[0-9a-fA-F]{2}$`)

// GenerateRandomMAC generates a random unicast, locally-administered MAC address.
// The first octet is forced to have bit 1 (multicast) cleared and bit 2 (local) set.
func GenerateRandomMAC() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	// Clear multicast bit (bit 0 of first octet) — unicast
	buf[0] &= 0xFE
	// Set locally administered bit (bit 1 of first octet)
	buf[0] |= 0x02
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		buf[0], buf[1], buf[2], buf[3], buf[4], buf[5]), nil
}

// GenerateVendorMAC generates a random MAC with the given OUI prefix (first 3 bytes).
// The oui parameter should be in the format "aa:bb:cc".
func GenerateVendorMAC(oui string) (string, error) {
	parts := strings.Split(strings.ToLower(oui), ":")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid OUI format %q, expected xx:xx:xx", oui)
	}

	buf := make([]byte, 3)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("crypto/rand: %w", err)
	}
	return fmt.Sprintf("%s:%s:%s:%02x:%02x:%02x",
		parts[0], parts[1], parts[2], buf[0], buf[1], buf[2]), nil
}

// SetMAC sets the MAC address on the given interface.
// For Wi-Fi: powers off Wi-Fi, disassociates, brings interface down, sets MAC, brings up, powers on.
// For others: brings interface down, sets MAC, brings back up.
func SetMAC(iface *Interface, mac string) error {
	if err := ValidateMAC(mac); err != nil {
		return err
	}

	if iface.Type == TypeWiFi {
		return setMACWiFi(iface, mac)
	}
	return setMACGeneric(iface, mac)
}

// setMACWiFi handles Wi-Fi interfaces which need a full power cycle on modern macOS.
func setMACWiFi(iface *Interface, mac string) error {
	// 1. Disassociate from current network.
	_ = exec.Command(
		"/System/Library/PrivateFrameworks/Apple80211.framework/Resources/airport",
		"-z",
	).Run()

	// 2. Power off Wi-Fi.
	_ = exec.Command("networksetup", "-setairportpower", iface.Name, "off").Run()

	// 3. Small delay to let the hardware settle.
	time.Sleep(500 * time.Millisecond)

	// 4. Set the MAC address (try multiple approaches).
	var setErr error
	out, err := exec.Command("ifconfig", iface.Name, "ether", mac).CombinedOutput()
	if err != nil {
		// Fallback: bring interface down, set, bring up.
		_ = exec.Command("ifconfig", iface.Name, "down").Run()
		time.Sleep(200 * time.Millisecond)
		out, err = exec.Command("ifconfig", iface.Name, "ether", mac).CombinedOutput()
		if err != nil {
			setErr = fmt.Errorf("ifconfig: %s (%w)", strings.TrimSpace(string(out)), err)
		}
		_ = exec.Command("ifconfig", iface.Name, "up").Run()
	}

	// 5. Power Wi-Fi back on.
	_ = exec.Command("networksetup", "-setairportpower", iface.Name, "on").Run()

	// 6. Detect new hardware to trigger reconnection.
	_ = exec.Command("networksetup", "-detectnewhardware").Run()

	return setErr
}

// setMACGeneric handles Ethernet/USB/Thunderbolt interfaces.
func setMACGeneric(iface *Interface, mac string) error {
	// Try setting directly first (works on most interfaces).
	out, err := exec.Command("ifconfig", iface.Name, "ether", mac).CombinedOutput()
	if err == nil {
		return nil
	}

	// If direct set failed, try the down/set/up cycle as fallback.
	_ = exec.Command("ifconfig", iface.Name, "down").Run()
	time.Sleep(200 * time.Millisecond)

	out, err = exec.Command("ifconfig", iface.Name, "ether", mac).CombinedOutput()

	_ = exec.Command("ifconfig", iface.Name, "up").Run()

	if err != nil {
		return fmt.Errorf("ifconfig: %s (%w)", strings.TrimSpace(string(out)), err)
	}

	return nil
}

// RestorePermanentMAC resets the interface to its factory MAC address.
func RestorePermanentMAC(iface *Interface) error {
	if iface.PermanentMAC == "" {
		return fmt.Errorf("no permanent MAC known for %s", iface.Name)
	}
	return SetMAC(iface, iface.PermanentMAC)
}

// ValidateMAC checks that a MAC string is in valid format (aa:bb:cc:dd:ee:ff).
func ValidateMAC(mac string) error {
	if !macRegex.MatchString(mac) {
		return fmt.Errorf("invalid MAC format %q, expected xx:xx:xx:xx:xx:xx", mac)
	}
	return nil
}

// IsMulticast returns true if the MAC address is a multicast address
// (bit 0 of the first octet is set).
func IsMulticast(mac string) bool {
	parts := strings.Split(mac, ":")
	if len(parts) < 1 {
		return false
	}
	var firstByte byte
	_, err := fmt.Sscanf(parts[0], "%x", &firstByte)
	if err != nil {
		return false
	}
	return firstByte&0x01 != 0
}

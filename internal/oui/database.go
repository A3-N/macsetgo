package oui

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// ouiToVendor is a reverse lookup map built at init time.
var ouiToVendor map[string]string

func init() {
	ouiToVendor = make(map[string]string)
	for _, v := range vendors {
		for _, oui := range v.OUIs {
			ouiToVendor[strings.ToLower(oui)] = v.Name
		}
	}
}

// LookupVendor returns the vendor name for a given MAC address, or "Unknown".
func LookupVendor(mac string) string {
	mac = strings.ToLower(mac)
	if len(mac) < 8 {
		return "Unknown"
	}
	oui := mac[:8] // "xx:xx:xx"
	if name, ok := ouiToVendor[oui]; ok {
		return name
	}
	return "Unknown"
}

// ListVendors returns all vendors sorted alphabetically.
func ListVendors() []Vendor {
	sorted := make([]Vendor, len(vendors))
	copy(sorted, vendors)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	return sorted
}

// RandomMACForVendor generates a random MAC address using a random OUI
// from the specified vendor.
func RandomMACForVendor(vendorName string) (string, error) {
	for _, v := range vendors {
		if strings.EqualFold(v.Name, vendorName) {
			if len(v.OUIs) == 0 {
				return "", fmt.Errorf("vendor %q has no OUI entries", vendorName)
			}
			// Pick a random OUI from this vendor.
			idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(v.OUIs))))
			if err != nil {
				return "", fmt.Errorf("random selection: %w", err)
			}
			oui := v.OUIs[idx.Int64()]

			// Generate random last 3 bytes.
			buf := make([]byte, 3)
			if _, err := rand.Read(buf); err != nil {
				return "", fmt.Errorf("crypto/rand: %w", err)
			}
			return fmt.Sprintf("%s:%02x:%02x:%02x", oui, buf[0], buf[1], buf[2]), nil
		}
	}
	return "", fmt.Errorf("vendor %q not found", vendorName)
}

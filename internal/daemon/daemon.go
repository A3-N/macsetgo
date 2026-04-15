package daemon

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/A3-N/macsetgo/internal/config"
	"github.com/A3-N/macsetgo/internal/network"
)

// Run starts the daemon polling loop. It monitors for new network interfaces
// and auto-applies a saved profile when one appears.
func Run() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if cfg.DaemonProfile == "" {
		return fmt.Errorf("no daemon profile configured. Set one with: macsetgo --daemon configure <profile>")
	}

	profile, err := config.LoadProfile(cfg.DaemonProfile)
	if err != nil {
		return fmt.Errorf("load profile %q: %w", cfg.DaemonProfile, err)
	}

	interval := time.Duration(cfg.DaemonPollInterval) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second
	}

	log.Printf("macsetgo daemon started — profile=%q poll=%s matchByPort=%v",
		cfg.DaemonProfile, interval, cfg.MatchByPortName)

	// Track known interfaces to detect new ones.
	known := make(map[string]bool)

	// Populate initial known set.
	if ifaces, err := network.ListInterfaces(); err == nil {
		for _, iface := range ifaces {
			key := interfaceKey(iface, cfg.MatchByPortName)
			known[key] = true
		}
	}

	// Handle graceful shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			log.Println("macsetgo daemon shutting down")
			return nil

		case <-ticker.C:
			ifaces, err := network.ListInterfaces()
			if err != nil {
				log.Printf("error listing interfaces: %v", err)
				continue
			}

			currentKeys := make(map[string]bool)
			for _, iface := range ifaces {
				key := interfaceKey(iface, cfg.MatchByPortName)
				currentKeys[key] = true

				if known[key] {
					continue
				}

				// New interface detected!
				log.Printf("new interface detected: %s (%s)", iface.Name, iface.HardwarePort)

				// Check if the profile has a MAC for this interface.
				var targetMAC string
				if cfg.MatchByPortName {
					targetMAC = profile.Entries[iface.HardwarePort]
				} else {
					targetMAC = profile.Entries[iface.Name]
				}

				if targetMAC == "" {
					log.Printf("  no profile entry for %s, skipping", key)
					known[key] = true
					continue
				}

				// Apply the MAC.
				log.Printf("  applying MAC %s to %s (%s)", targetMAC, iface.Name, iface.HardwarePort)
				if err := network.SetMAC(&iface, targetMAC); err != nil {
					log.Printf("  error setting MAC: %v", err)
				} else {
					log.Printf("  MAC changed successfully")

					// Log to history.
					_ = config.LogChange(config.HistoryEntry{
						Interface: iface.Name,
						PortName:  iface.HardwarePort,
						OldMAC:    iface.CurrentMAC,
						NewMAC:    targetMAC,
						Method:    fmt.Sprintf("daemon:profile:%s", cfg.DaemonProfile),
					})
				}

				known[key] = true
			}

			// Remove stale entries (interfaces that were unplugged).
			for key := range known {
				if !currentKeys[key] {
					delete(known, key)
				}
			}
		}
	}
}

func interfaceKey(iface network.Interface, byPort bool) string {
	if byPort {
		return strings.ToLower(iface.HardwarePort)
	}
	return iface.Name
}

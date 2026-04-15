package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/A3-N/macsetgo/internal/config"
	"github.com/A3-N/macsetgo/internal/daemon"
	"github.com/A3-N/macsetgo/internal/network"
	"github.com/A3-N/macsetgo/internal/oui"
	"github.com/A3-N/macsetgo/internal/tui"
)

func main() {
	// macOS only.
	if runtime.GOOS != "darwin" {
		fatal("macsetgo is only available on macOS")
	}

	// CLI mode — parse flags before root check so --help works without sudo.
	showFlag := flag.Bool("s", false, "Show interface info")
	randomFlag := flag.Bool("r", false, "Randomize MAC address")
	manualFlag := flag.String("m", "", "Set a specific MAC address")
	permanentFlag := flag.Bool("p", false, "Restore permanent MAC")
	vendorFlag := flag.String("vendor", "", "Vendor name for OUI-based random MAC (use with -r)")
	profileCmd := flag.String("profile", "", "Profile command: list, save <name>, load <name>, delete <name>")
	daemonCmd := flag.String("daemon", "", "Daemon command: run, install, uninstall, status, configure <profile>")
	historyFlag := flag.Bool("history", false, "Show change history")

	flag.Usage = func() {
		cyan := "\033[36m"
		bold := "\033[1m"
		dim := "\033[2m"
		yellow := "\033[33m"
		red := "\033[31m"
		reset := "\033[0m"
		green := "\033[32m"

		fmt.Fprintf(os.Stderr, "\n  %smacsetgo%s %sgithub.com/A3-N/macsetgo%s\n", cyan+bold, reset, dim, reset)
		fmt.Fprintf(os.Stderr, "  %sRequires root privileges%s\n\n", red+bold, reset)

		fmt.Fprintf(os.Stderr, "  %sUSAGE%s\n", yellow+bold, reset)
		fmt.Fprintf(os.Stderr, "    macsetgo %s[options] [interface]%s\n\n", dim, reset)

		fmt.Fprintf(os.Stderr, "  %sFLAGS%s\n", yellow+bold, reset)
		fmt.Fprintf(os.Stderr, "    %s-r%s              %sRandomize MAC address%s\n", cyan+bold, reset, reset, reset)
		fmt.Fprintf(os.Stderr, "    %s-s%s              %sShow interface info%s\n", cyan+bold, reset, reset, reset)
		fmt.Fprintf(os.Stderr, "    %s-m%s %s<mac>%s        %sSet a specific MAC address%s\n", cyan+bold, reset, dim, reset, reset, reset)
		fmt.Fprintf(os.Stderr, "    %s-p%s              %sRestore permanent MAC%s\n", cyan+bold, reset, reset, reset)
		fmt.Fprintf(os.Stderr, "    %s--vendor%s %s<name>%s  %sVendor OUI prefix%s %s(use with -r)%s\n", cyan+bold, reset, dim, reset, reset, reset, dim, reset)
		fmt.Fprintf(os.Stderr, "    %s--profile%s %s<cmd>%s  %slist, save, load, delete%s\n", cyan+bold, reset, dim, reset, reset, reset)
		fmt.Fprintf(os.Stderr, "    %s--daemon%s %s<cmd>%s   %srun, install, uninstall, status, configure%s\n", cyan+bold, reset, dim, reset, reset, reset)
		fmt.Fprintf(os.Stderr, "    %s--history%s        %sShow change history%s\n\n", cyan+bold, reset, reset, reset)

		fmt.Fprintf(os.Stderr, "  %sEXAMPLES%s\n", yellow+bold, reset)
		fmt.Fprintf(os.Stderr, "    %smacsetgo%s                              %s# Launch TUI%s\n", green, reset, dim, reset)
		fmt.Fprintf(os.Stderr, "    %smacsetgo -r en0%s                       %s# Random MAC%s\n", green, reset, dim, reset)
		fmt.Fprintf(os.Stderr, "    %smacsetgo -r --vendor Apple en0%s        %s# Random Apple MAC%s\n", green, reset, dim, reset)
		fmt.Fprintf(os.Stderr, "    %smacsetgo -m aa:bb:cc:dd:ee:ff en0%s     %s# Set specific MAC%s\n", green, reset, dim, reset)
		fmt.Fprintf(os.Stderr, "    %smacsetgo -p en0%s                       %s# Restore permanent%s\n", green, reset, dim, reset)
		fmt.Fprintf(os.Stderr, "    %smacsetgo -s en0%s                       %s# Show interface info%s\n", green, reset, dim, reset)
		fmt.Fprintf(os.Stderr, "    %smacsetgo --profile list%s               %s# List profiles%s\n", green, reset, dim, reset)
		fmt.Fprintf(os.Stderr, "    %smacsetgo --daemon status%s              %s# Daemon status%s\n\n", green, reset, dim, reset)
	}

	flag.Parse()

	// Root check — required for all operations except --help.
	if os.Geteuid() != 0 {
		fatal("macsetgo requires root privileges\n  Usage: sudo macsetgo [options]")
	}

	// If no args (after flag parsing), launch TUI.
	if len(os.Args) == 1 {
		launchTUI()
		return
	}

	switch {

	case *showFlag:
		ifaceName := flag.Arg(0)
		if ifaceName == "" {
			fatal("specify an interface: macsetgo -s <interface>")
		}
		cliShow(ifaceName)

	case *randomFlag:
		ifaceName := flag.Arg(0)
		if ifaceName == "" {
			fatal("specify an interface: macsetgo -r <interface>")
		}
		cliRandom(ifaceName, *vendorFlag)

	case *manualFlag != "":
		ifaceName := flag.Arg(0)
		if ifaceName == "" {
			fatal("specify an interface: macsetgo -m <mac> <interface>")
		}
		cliManual(ifaceName, *manualFlag)

	case *permanentFlag:
		ifaceName := flag.Arg(0)
		if ifaceName == "" {
			fatal("specify an interface: macsetgo -p <interface>")
		}
		cliPermanent(ifaceName)

	case *profileCmd != "":
		cliProfile(*profileCmd, flag.Args())

	case *daemonCmd != "":
		cliDaemon(*daemonCmd, flag.Args())

	case *historyFlag:
		cliHistory()

	default:
		// If there's an arg but no flag, try showing that interface.
		if flag.NArg() > 0 {
			cliShow(flag.Arg(0))
		} else {
			launchTUI()
		}
	}
}

func launchTUI() {
	app := tui.NewApp()
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fatal("TUI error: %v", err)
	}
}

func cliShow(ifaceName string) {
	iface, err := network.GetInterface(ifaceName)
	if err != nil {
		fatal("%v", err)
	}

	vendor := oui.LookupVendor(iface.CurrentMAC)
	spoofed := ""
	if iface.IsSpoofed {
		spoofed = " (SPOOFED)"
	}

	fmt.Printf("Interface:      %s\n", iface.Name)
	fmt.Printf("Hardware Port:  %s\n", iface.HardwarePort)
	fmt.Printf("Type:           %s\n", iface.Type)
	fmt.Printf("Status:         %s\n", boolStatus(iface.IsUp))
	fmt.Printf("Current MAC:    %s%s\n", iface.CurrentMAC, spoofed)
	fmt.Printf("Permanent MAC:  %s\n", iface.PermanentMAC)
	if vendor != "Unknown" {
		fmt.Printf("Vendor:         %s\n", vendor)
	}
	if iface.IsUSB {
		fmt.Printf("USB Adapter:    yes\n")
	}
}

func cliRandom(ifaceName, vendor string) {
	iface, err := network.GetInterface(ifaceName)
	if err != nil {
		fatal("%v", err)
	}

	oldMAC := iface.CurrentMAC

	var newMAC string
	var method string
	if vendor != "" {
		newMAC, err = oui.RandomMACForVendor(vendor)
		method = fmt.Sprintf("vendor:%s", vendor)
	} else {
		newMAC, err = network.GenerateRandomMAC()
		method = "random"
	}
	if err != nil {
		fatal("generate MAC: %v", err)
	}

	if err := network.SetMAC(iface, newMAC); err != nil {
		fatal("set MAC: %v", err)
	}

	_ = config.LogChange(config.HistoryEntry{
		Interface: iface.Name,
		PortName:  iface.HardwarePort,
		OldMAC:    oldMAC,
		NewMAC:    newMAC,
		Method:    method,
	})

	fmt.Printf("Permanent MAC:  %s\n", iface.PermanentMAC)
	fmt.Printf("Old MAC:        %s\n", oldMAC)
	fmt.Printf("New MAC:        %s\n", newMAC)
	if vendor != "" {
		fmt.Printf("Vendor:         %s\n", vendor)
	}
}

func cliManual(ifaceName, mac string) {
	iface, err := network.GetInterface(ifaceName)
	if err != nil {
		fatal("%v", err)
	}

	if network.IsMulticast(mac) {
		fmt.Fprintf(os.Stderr, "WARNING: MAC address is multicast — setting it may not work\n")
	}

	oldMAC := iface.CurrentMAC
	if err := network.SetMAC(iface, mac); err != nil {
		fatal("set MAC: %v", err)
	}

	_ = config.LogChange(config.HistoryEntry{
		Interface: iface.Name,
		PortName:  iface.HardwarePort,
		OldMAC:    oldMAC,
		NewMAC:    mac,
		Method:    "manual",
	})

	fmt.Printf("Permanent MAC:  %s\n", iface.PermanentMAC)
	fmt.Printf("Old MAC:        %s\n", oldMAC)
	fmt.Printf("New MAC:        %s\n", mac)
}

func cliPermanent(ifaceName string) {
	iface, err := network.GetInterface(ifaceName)
	if err != nil {
		fatal("%v", err)
	}

	oldMAC := iface.CurrentMAC
	if err := network.RestorePermanentMAC(iface); err != nil {
		fatal("restore: %v", err)
	}

	_ = config.LogChange(config.HistoryEntry{
		Interface: iface.Name,
		PortName:  iface.HardwarePort,
		OldMAC:    oldMAC,
		NewMAC:    iface.PermanentMAC,
		Method:    "restore",
	})

	fmt.Printf("Permanent MAC:  %s\n", iface.PermanentMAC)
	fmt.Printf("Old MAC:        %s\n", oldMAC)
	fmt.Printf("New MAC:        %s\n", iface.PermanentMAC)
}

func cliProfile(cmd string, args []string) {
	switch cmd {
	case "list":
		profiles, err := config.ListProfiles()
		if err != nil {
			fatal("list profiles: %v", err)
		}
		if len(profiles) == 0 {
			fmt.Println("No saved profiles")
			return
		}
		for _, p := range profiles {
			fmt.Printf("%-20s  %d interfaces  %s\n", p.Name, len(p.Entries), p.CreatedAt.Format("2006-01-02 15:04"))
			for port, mac := range p.Entries {
				fmt.Printf("  %-20s → %s\n", port, mac)
			}
		}

	case "save":
		if len(args) == 0 {
			fatal("specify a profile name: macsetgo --profile save <name>")
		}
		name := args[0]
		ifaces, err := network.ListInterfaces()
		if err != nil {
			fatal("list interfaces: %v", err)
		}
		entries := make(map[string]string)
		for _, iface := range ifaces {
			if iface.CurrentMAC != "" {
				entries[iface.HardwarePort] = iface.CurrentMAC
			}
		}
		if err := config.SaveProfile(config.Profile{
			Name:    name,
			Entries: entries,
		}); err != nil {
			fatal("save profile: %v", err)
		}
		fmt.Printf("Profile %q saved (%d interfaces)\n", name, len(entries))

	case "load":
		if len(args) == 0 {
			fatal("specify a profile name: macsetgo --profile load <name>")
		}
		name := args[0]
		profile, err := config.LoadProfile(name)
		if err != nil {
			fatal("load profile: %v", err)
		}
		ifaces, err := network.ListInterfaces()
		if err != nil {
			fatal("list interfaces: %v", err)
		}
		applied := 0
		for _, iface := range ifaces {
			mac, ok := profile.Entries[iface.HardwarePort]
			if !ok {
				continue
			}
			oldMAC := iface.CurrentMAC
			if err := network.SetMAC(&iface, mac); err != nil {
				fmt.Fprintf(os.Stderr, "  %s: failed (%v)\n", iface.Name, err)
				continue
			}
			_ = config.LogChange(config.HistoryEntry{
				Interface: iface.Name,
				PortName:  iface.HardwarePort,
				OldMAC:    oldMAC,
				NewMAC:    mac,
				Method:    fmt.Sprintf("profile:%s", name),
			})
			fmt.Printf("  %s (%s): %s → %s\n", iface.Name, iface.HardwarePort, oldMAC, mac)
			applied++
		}
		fmt.Printf("Applied profile %q to %d interfaces\n", name, applied)

	case "delete":
		if len(args) == 0 {
			fatal("specify a profile name: macsetgo --profile delete <name>")
		}
		if err := config.DeleteProfile(args[0]); err != nil {
			fatal("delete profile: %v", err)
		}
		fmt.Printf("Profile %q deleted\n", args[0])

	default:
		fatal("unknown profile command: %s (use: list, save, load, delete)", cmd)
	}
}

func cliDaemon(cmd string, args []string) {
	switch cmd {
	case "run":
		if err := daemon.Run(); err != nil {
			fatal("daemon: %v", err)
		}

	case "install":
		if err := daemon.Install(); err != nil {
			fatal("install daemon: %v", err)
		}
		fmt.Println("Daemon installed and started")

	case "uninstall":
		if err := daemon.Uninstall(); err != nil {
			fatal("uninstall daemon: %v", err)
		}
		fmt.Println("Daemon uninstalled")

	case "status":
		fmt.Printf("Daemon: %s\n", daemon.Status())
		cfg, _ := config.LoadConfig()
		if cfg.DaemonProfile != "" {
			fmt.Printf("Profile: %s\n", cfg.DaemonProfile)
		}
		fmt.Printf("Poll interval: %ds\n", cfg.DaemonPollInterval)

	case "configure":
		if len(args) == 0 {
			fatal("specify a profile: macsetgo --daemon configure <profile>")
		}
		profileName := args[0]
		// Verify the profile exists.
		if _, err := config.LoadProfile(profileName); err != nil {
			fatal("profile %q not found", profileName)
		}
		cfg, err := config.LoadConfig()
		if err != nil {
			fatal("load config: %v", err)
		}
		cfg.DaemonProfile = profileName
		if err := config.SaveConfig(cfg); err != nil {
			fatal("save config: %v", err)
		}
		fmt.Printf("Daemon profile set to %q\n", profileName)

	default:
		fatal("unknown daemon command: %s (use: run, install, uninstall, status, configure)", cmd)
	}
}

func cliHistory() {
	entries, err := config.GetHistory(20)
	if err != nil {
		fatal("history: %v", err)
	}
	if len(entries) == 0 {
		fmt.Println("No history")
		return
	}

	fmt.Printf("%-20s %-8s %-19s %-19s %s\n", "TIME", "IFACE", "OLD MAC", "NEW MAC", "METHOD")
	fmt.Println(strings.Repeat("─", 85))
	for _, e := range entries {
		fmt.Printf("%-20s %-8s %-19s %-19s %s\n",
			e.Timestamp.Format("2006-01-02 15:04:05"),
			e.Interface,
			e.OldMAC,
			e.NewMAC,
			e.Method,
		)
	}
}

func boolStatus(b bool) string {
	if b {
		return "active"
	}
	return "inactive"
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}

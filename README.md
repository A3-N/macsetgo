# macsetgo

A full-featured MAC address manager for macOS — TUI + CLI with profiles, vendor spoofing, history, and an auto-change daemon.

Built in Go with [Bubbletea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss).

> [!IMPORTANT]
> Requires root privileges to run.

![main](img/main.png)

## Features

- **Interactive TUI** — Navigate adapters, change MACs, manage profiles with keyboard shortcuts
- **Adapter Detection** — Auto-detects Wi-Fi, Ethernet, USB Ethernet (external dongles), Thunderbolt, Bluetooth
- **MAC Operations** — Random (unicast-safe), vendor-specific OUI, manual, restore permanent
- **Vendor Spoofing** — Generate MACs matching 25+ vendors (Apple, Samsung, Intel, Cisco, etc.)
- **Profiles** — Save/load named MAC configurations across interfaces
- **History** — Full log of all MAC changes with timestamps, with restore/revert support
- **Auto-Change Daemon** — Automatically apply a saved profile when a new adapter is plugged in (via launchd)
- **NAC Intelligence** — Shows IP, gateway, DNS, DHCP server, lease time, 802.1X status, packet counters
- **CLI Mode** — Scriptable flag-based interface for automation

## Installation

### From Source

```sh
git clone https://github.com/A3-N/macsetgo.git
cd macsetgo
go build -ldflags="-s -w" -o macsetgo .
cp macsetgo /usr/local/bin/
```

Or use the Makefile:

```sh
make install
```

## Usage

### TUI Mode (Interactive)

```sh
macsetgo
```

Navigate with arrow keys, press `?` for help.

### CLI Mode

```sh
# Show interface info
macsetgo -s en0

# Randomize MAC
macsetgo -r en0

# Randomize with a specific vendor OUI
macsetgo -r -vendor Apple en0

# Set a specific MAC
macsetgo -m aa:bb:cc:dd:ee:ff en0

# Restore permanent/factory MAC
macsetgo -p en0
```

### Profiles

```sh
# Save current MAC state as a named profile
macsetgo -profile save coffee-shop

# List all profiles
macsetgo -profile list

# Apply a saved profile
macsetgo -profile load coffee-shop

# Delete a profile
macsetgo -profile delete coffee-shop
```

### Auto-Change Daemon

```sh
# Set which profile to auto-apply on new adapters
macsetgo -daemon configure coffee-shop

# Install the daemon (starts monitoring immediately)
macsetgo -daemon install

# Check daemon status
macsetgo -daemon status

# Remove the daemon
macsetgo -daemon uninstall
```

### History

```sh
macsetgo -history
```

## TUI Key Bindings

| Key | Action |
|-----|--------|
| `j/k` | Navigate adapter list |
| `Enter` | Open actions for selected adapter |
| `p` | Profiles |
| `h` | History |
| `d` | Daemon settings |
| `?` | Help |
| `Esc` | Back |
| `q` | Quit (from dashboard) |

### Actions Menu

| Key | Action |
|-----|--------|
| `r` | Randomize MAC |
| `v` | Random MAC with vendor OUI |
| `m` | Set manual MAC |
| `p` | Restore permanent MAC |
| `a` | Randomize all active adapters |

### History

| Key | Action |
|-----|--------|
| `r` | Restore (re-apply the new MAC) |
| `o` | Revert (apply the old MAC) |

## Config

All data stored in `~/.config/macsetgo/`:

```
~/.config/macsetgo/
  config.json      # App settings (daemon profile, poll interval)
  profiles.json    # Saved MAC profiles
  history.json     # Change history log
  daemon.log       # Daemon output log
```

## Platform Support

| Platform | Architecture | Status |
|----------|-------------|--------|
| macOS Sequoia (15.x) | Apple Silicon (arm64) | Tested |
| macOS Sonoma (14.x) | Apple Silicon (arm64) | Supported |
| macOS Ventura (13.x) | Apple Silicon / Intel (amd64) | Supported |
| macOS Monterey (12.x) | Apple Silicon / Intel (amd64) | Supported |
| macOS Big Sur (11.x) | Intel (amd64) | Supported |

> [!NOTE]
> macOS only. Relies on `ifconfig`, `networksetup`, `netstat`, `route`, `scutil`, and `ipconfig` — all included in macOS by default. Wi-Fi MAC changes on Apple Silicon require a full power cycle of the Wi-Fi adapter, which macsetgo handles automatically.

## Requirements

- macOS 11.0+ (Big Sur or later)
- Root privileges
- Go 1.21+ (build only)

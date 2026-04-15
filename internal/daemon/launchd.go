package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	plistLabel = "com.macsetgo.daemon"
	plistDir   = "/Library/LaunchDaemons"
	plistName  = "com.macsetgo.daemon.plist"
)

var plistTemplate = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
        <string>--daemon</string>
        <string>run</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>{{.LogPath}}</string>
    <key>StandardErrorPath</key>
    <string>{{.LogPath}}</string>
</dict>
</plist>
`))

type plistData struct {
	Label      string
	BinaryPath string
	LogPath    string
}

func plistPath() string {
	return filepath.Join(plistDir, plistName)
}

// Install creates the LaunchDaemon plist and loads it.
func Install() error {
	// Find the macsetgo binary path.
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find executable path: %w", err)
	}

	// Resolve symlinks.
	binaryPath, err = filepath.EvalSymlinks(binaryPath)
	if err != nil {
		return fmt.Errorf("resolve symlinks: %w", err)
	}

	// Determine log path.
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("home directory: %w", err)
	}
	logDir := filepath.Join(home, ".config", "macsetgo")
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}
	logPath := filepath.Join(logDir, "daemon.log")

	// Generate plist content.
	data := plistData{
		Label:      plistLabel,
		BinaryPath: binaryPath,
		LogPath:    logPath,
	}

	var buf strings.Builder
	if err := plistTemplate.Execute(&buf, data); err != nil {
		return fmt.Errorf("generate plist: %w", err)
	}

	// Write plist.
	if err := os.WriteFile(plistPath(), []byte(buf.String()), 0644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	// Load the daemon.
	out, err := exec.Command("launchctl", "load", plistPath()).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl load: %s (%w)", strings.TrimSpace(string(out)), err)
	}

	return nil
}

// Uninstall unloads and removes the LaunchDaemon plist.
func Uninstall() error {
	// Unload (ignore errors if not loaded).
	_ = exec.Command("launchctl", "unload", plistPath()).Run()

	if err := os.Remove(plistPath()); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}
	return nil
}

// Status returns a human-readable status of the daemon.
func Status() string {
	// Check if plist exists.
	if _, err := os.Stat(plistPath()); os.IsNotExist(err) {
		return "not installed"
	}

	// Check if loaded.
	out, err := exec.Command("launchctl", "list", plistLabel).CombinedOutput()
	if err != nil {
		return "installed (not running)"
	}

	output := string(out)
	if strings.Contains(output, plistLabel) {
		return "installed (running)"
	}

	return "installed (unknown state)"
}

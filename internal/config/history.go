package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	historyFile     = "history.json"
	maxHistorySize  = 1000
	historyTrimTo   = 500
)

// HistoryEntry records a single MAC address change event.
type HistoryEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Interface string    `json:"interface"`     // Device name, e.g. "en0"
	PortName  string    `json:"port_name"`     // Hardware port, e.g. "Wi-Fi"
	OldMAC    string    `json:"old_mac"`
	NewMAC    string    `json:"new_mac"`
	Method    string    `json:"method"`        // "random", "vendor:Apple", "manual", "profile:work", "restore"
}

type historyStore struct {
	Entries []HistoryEntry `json:"entries"`
}

func historyPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, historyFile), nil
}

func loadHistory() (historyStore, error) {
	path, err := historyPath()
	if err != nil {
		return historyStore{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return historyStore{}, nil
		}
		return historyStore{}, fmt.Errorf("read history: %w", err)
	}

	var store historyStore
	if err := json.Unmarshal(data, &store); err != nil {
		return historyStore{}, fmt.Errorf("parse history: %w", err)
	}
	return store, nil
}

func saveHistory(store historyStore) error {
	path, err := historyPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal history: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// LogChange appends a new entry to the history log.
// Auto-rotates when exceeding maxHistorySize.
func LogChange(entry HistoryEntry) error {
	store, err := loadHistory()
	if err != nil {
		return err
	}

	entry.Timestamp = time.Now()
	store.Entries = append(store.Entries, entry)

	// Trim if too large — keep the most recent entries.
	if len(store.Entries) > maxHistorySize {
		store.Entries = store.Entries[len(store.Entries)-historyTrimTo:]
	}

	return saveHistory(store)
}

// GetHistory returns the last `limit` history entries (most recent first).
// If limit <= 0, returns all entries.
func GetHistory(limit int) ([]HistoryEntry, error) {
	store, err := loadHistory()
	if err != nil {
		return nil, err
	}

	entries := store.Entries

	// Reverse to show most recent first.
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	if limit > 0 && limit < len(entries) {
		entries = entries[:limit]
	}

	return entries, nil
}

package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const profilesFile = "profiles.json"

// Profile stores a named MAC configuration for one or more interfaces.
type Profile struct {
	Name      string            `json:"name"`
	CreatedAt time.Time         `json:"created_at"`
	// Entries maps hardware port name (e.g. "Wi-Fi", "USB 10/100/1000 LAN") to MAC address.
	// We use port name instead of device name since device names (enX) can change across replugs.
	Entries map[string]string   `json:"entries"`
}

// profileStore is the on-disk format.
type profileStore struct {
	Profiles []Profile `json:"profiles"`
}

func profilesPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, profilesFile), nil
}

func loadStore() (profileStore, error) {
	path, err := profilesPath()
	if err != nil {
		return profileStore{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return profileStore{}, nil
		}
		return profileStore{}, fmt.Errorf("read profiles: %w", err)
	}

	var store profileStore
	if err := json.Unmarshal(data, &store); err != nil {
		return profileStore{}, fmt.Errorf("parse profiles: %w", err)
	}
	return store, nil
}

func saveStore(store profileStore) error {
	path, err := profilesPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal profiles: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// SaveProfile saves or overwrites a named profile.
func SaveProfile(profile Profile) error {
	store, err := loadStore()
	if err != nil {
		return err
	}

	// Replace if exists, otherwise append.
	found := false
	for i, p := range store.Profiles {
		if p.Name == profile.Name {
			store.Profiles[i] = profile
			found = true
			break
		}
	}
	if !found {
		store.Profiles = append(store.Profiles, profile)
	}

	return saveStore(store)
}

// LoadProfile retrieves a profile by name.
func LoadProfile(name string) (*Profile, error) {
	store, err := loadStore()
	if err != nil {
		return nil, err
	}
	for _, p := range store.Profiles {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("profile %q not found", name)
}

// ListProfiles returns all profiles sorted by name.
func ListProfiles() ([]Profile, error) {
	store, err := loadStore()
	if err != nil {
		return nil, err
	}
	sort.Slice(store.Profiles, func(i, j int) bool {
		return store.Profiles[i].Name < store.Profiles[j].Name
	})
	return store.Profiles, nil
}

// DeleteProfile removes a profile by name.
func DeleteProfile(name string) error {
	store, err := loadStore()
	if err != nil {
		return err
	}

	filtered := store.Profiles[:0]
	found := false
	for _, p := range store.Profiles {
		if p.Name == name {
			found = true
			continue
		}
		filtered = append(filtered, p)
	}
	if !found {
		return fmt.Errorf("profile %q not found", name)
	}

	store.Profiles = filtered
	return saveStore(store)
}

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds operator identity + paths.
type Config struct {
	MyCall string `json:"my_call"`
	MyGrid string `json:"my_grid"`
	DBPath string `json:"db_path"`
}

// Dir returns the config dir, creating it if needed.
func Dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		home, herr := os.UserHomeDir()
		if herr != nil {
			return "", herr
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "hamlog")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// DefaultDBPath returns the default sqlite path.
func DefaultDBPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "hamlog")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "hamlog.db"), nil
}

// Path returns config.json path.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads config, returning empty Config if missing.
func Load() (Config, error) {
	var c Config
	p, err := Path()
	if err != nil {
		return c, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			db, derr := DefaultDBPath()
			if derr != nil {
				return c, nil
			}
			c.DBPath = db
			return c, nil
		}
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	if c.DBPath == "" {
		db, err := DefaultDBPath()
		if err == nil {
			c.DBPath = db
		}
	}
	return c, nil
}

// Save writes config.
func (c Config) Save() error {
	p, err := Path()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o600)
}

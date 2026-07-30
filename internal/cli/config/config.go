package config 

import (
    "encoding/json"
    "os"
    "path/filepath"
)

type Config struct {
    Host  string `json:"host"`
    Token string `json:"token"`
}

func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".devflow", "config.json"), nil
}

// Load reads the config file. Returns empty Config if not found
func Load() (Config, error) {
	path, err := configPath()
	if err != nil {
		return Config{}, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{Host: "http://localhost:8080"}, nil
	}
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	return cfg, json.Unmarshal(data, &cfg)
}

// Save writes the config file, creating the directory if needed
func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// ClearToken removes the JWT from disk (logout)
func ClearToken() error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	cfg.Token = ""
	return Save(cfg)
}

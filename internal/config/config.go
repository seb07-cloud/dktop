package config

import (
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

type Config struct {
	RefreshRate   int      `yaml:"refresh_rate"`   // in milliseconds
	DefaultView   string   `yaml:"default_view"`   // containers, images
	AutostartList []string `yaml:"autostart_list"` // container names/IDs to autostart
	LogLines      int      `yaml:"log_lines"`      // number of log lines to show
}

var DefaultConfig = Config{
	RefreshRate:   1000,
	DefaultView:   "containers",
	AutostartList: []string{},
	LogLines:      100,
}

// GetConfigDir returns the platform-specific config directory
func GetConfigDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		// Use %APPDATA% on Windows (typically C:\Users\username\AppData\Roaming)
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "dktop"), nil
	default:
		// Use ~/.config on Unix-like systems (macOS, Linux)
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", "dktop"), nil
	}
}

func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "config.yaml"), nil
}

func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return &DefaultConfig, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return &DefaultConfig, nil
		}
		return nil, err
	}

	cfg := DefaultConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Save() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func (c *Config) AddAutostart(containerID string) {
	for _, id := range c.AutostartList {
		if id == containerID {
			return // Already in list
		}
	}
	c.AutostartList = append(c.AutostartList, containerID)
}

func (c *Config) RemoveAutostart(containerID string) {
	for i, id := range c.AutostartList {
		if id == containerID {
			c.AutostartList = append(c.AutostartList[:i], c.AutostartList[i+1:]...)
			return
		}
	}
}

func (c *Config) IsAutostart(containerID string) bool {
	for _, id := range c.AutostartList {
		if id == containerID {
			return true
		}
	}
	return false
}

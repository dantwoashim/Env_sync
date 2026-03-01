// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package config

import (
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

// CurrentConfigVersion is the latest config schema version.
const CurrentConfigVersion = 2

// VersionedConfig wraps Config with its schema version.
type VersionedConfig struct {
	ConfigVersion int `toml:"config_version"`
	Config
}

// LoadConfig reads and migrates the config from the standard path.
func LoadConfig() (*Config, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := Default()
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	// Try versioned config first
	var vc VersionedConfig
	if err := toml.Unmarshal(data, &vc); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	cfg := &vc.Config

	// Migrate if needed
	if vc.ConfigVersion < CurrentConfigVersion {
		if err := migrate(cfg, vc.ConfigVersion); err != nil {
			return nil, fmt.Errorf("migrating config: %w", err)
		}
		// Save migrated config
		if err := SaveConfig(cfg); err != nil {
			return nil, fmt.Errorf("saving migrated config: %w", err)
		}
	}

	return cfg, nil
}

// SaveConfig writes the config to the standard path.
func SaveConfig(cfg *Config) error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}

	vc := VersionedConfig{
		ConfigVersion: CurrentConfigVersion,
		Config:        *cfg,
	}

	data, err := toml.Marshal(vc)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}

// migrate upgrades config from oldVersion to CurrentConfigVersion.
func migrate(cfg *Config, fromVersion int) error {
	// v0/v1 → v2: add holepunch settings
	if fromVersion < 2 {
		if cfg.Network.HolePunchTimeoutMs == 0 {
			cfg.Network.HolePunchTimeoutMs = 5000
		}
		cfg.Network.HolePunchEnabled = true
		if cfg.Sync.MergeStrategy == "" {
			cfg.Sync.MergeStrategy = "interactive"
		}
		if cfg.Sync.MaxVersions == 0 {
			cfg.Sync.MaxVersions = 10
		}
	}

	return nil
}

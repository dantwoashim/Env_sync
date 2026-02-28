package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ConfigFilePath returns the full path to the config TOML file.
func ConfigFilePath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// PeersDir returns the directory for peer registry files.
func PeersDir(teamID string) (string, error) {
	dataDir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "teams", teamID, "peers"), nil
}

// TeamFilePath returns the path for a team's metadata file.
func TeamFilePath(teamID string) (string, error) {
	dataDir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "teams", teamID, "team.toml"), nil
}

// StoreDir returns the version store directory for a project.
func StoreDir(projectHash string) (string, error) {
	dataDir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "store", projectHash), nil
}

// AuditLogPath returns the path to the audit log file.
func AuditLogPath() (string, error) {
	dataDir, err := DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "audit.jsonl"), nil
}

// ProjectConfigPath returns the path to the per-project config file.
func ProjectConfigPath() string {
	return ".envsync.toml"
}

// FindProjectConfig searches up from cwd to find .envsync.toml.
func FindProjectConfig() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, ".envsync.toml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf(".envsync.toml not found (run 'envsync init' to create one)")
}

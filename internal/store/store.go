// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/envsync/envsync/internal/config"
	"github.com/envsync/envsync/internal/crypto"
)

// Store manages encrypted .env version history.
type Store struct {
	maxVersions int
	baseDir     string
}

// VersionInfo describes a stored version.
type VersionInfo struct {
	Sequence  int
	Timestamp time.Time
	FilePath  string
	SizeBytes int64
}

// New creates a new Store with the given maximum version count.
func New(maxVersions int) (*Store, error) {
	dataDir, err := config.DataDir()
	if err != nil {
		return nil, err
	}

	return &Store{
		maxVersions: maxVersions,
		baseDir:     filepath.Join(dataDir, "store"),
	}, nil
}

// projectDir returns the directory for a specific project.
func (s *Store) projectDir(projectHash string) string {
	return filepath.Join(s.baseDir, projectHash)
}

// Save encrypts and saves a .env file as a new version.
func (s *Store) Save(projectHash string, content []byte, sequence int, encryptionKey [32]byte) error {
	dir := s.projectDir(projectHash)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating project store dir: %w", err)
	}

	// Encrypt the content
	encrypted, err := crypto.Encrypt(content, encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypting version: %w", err)
	}

	// Write file: {sequence}_{timestamp}.enc
	timestamp := time.Now().UTC().Format("20060102T150405Z")
	filename := fmt.Sprintf("%06d_%s.enc", sequence, timestamp)
	filePath := filepath.Join(dir, filename)

	if err := os.WriteFile(filePath, encrypted, 0600); err != nil {
		return fmt.Errorf("writing version file: %w", err)
	}

	// Rotate old versions
	return s.rotate(projectHash)
}

// Restore decrypts and returns a specific version.
func (s *Store) Restore(projectHash string, sequence int, encryptionKey [32]byte) ([]byte, error) {
	versions, err := s.List(projectHash)
	if err != nil {
		return nil, err
	}

	for _, v := range versions {
		if v.Sequence == sequence {
			data, err := os.ReadFile(v.FilePath)
			if err != nil {
				return nil, fmt.Errorf("reading version file: %w", err)
			}

			plaintext, err := crypto.Decrypt(data, encryptionKey)
			if err != nil {
				return nil, fmt.Errorf("decrypting version: %w", err)
			}

			return plaintext, nil
		}
	}

	return nil, fmt.Errorf("version %d not found", sequence)
}

// List returns all stored versions for a project, newest first.
func (s *Store) List(projectHash string) ([]VersionInfo, error) {
	dir := s.projectDir(projectHash)

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading version store: %w", err)
	}

	var versions []VersionInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".enc") {
			continue
		}

		info, err := parseVersionFilename(entry.Name())
		if err != nil {
			continue // Skip malformed files
		}
		info.FilePath = filepath.Join(dir, entry.Name())

		fileInfo, err := entry.Info()
		if err == nil {
			info.SizeBytes = fileInfo.Size()
		}

		versions = append(versions, info)
	}

	// Sort by sequence descending (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Sequence > versions[j].Sequence
	})

	return versions, nil
}

// Latest returns the most recent version, or nil if none exist.
func (s *Store) Latest(projectHash string) (*VersionInfo, error) {
	versions, err := s.List(projectHash)
	if err != nil {
		return nil, err
	}
	if len(versions) == 0 {
		return nil, nil
	}
	return &versions[0], nil
}

// rotate removes old versions beyond the max count.
func (s *Store) rotate(projectHash string) error {
	versions, err := s.List(projectHash)
	if err != nil {
		return err
	}

	if len(versions) <= s.maxVersions {
		return nil
	}

	// Remove oldest versions
	for _, v := range versions[s.maxVersions:] {
		if err := os.Remove(v.FilePath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing old version: %w", err)
		}
	}

	return nil
}

// parseVersionFilename parses a filename like "000042_20260228T134500Z.enc".
func parseVersionFilename(name string) (VersionInfo, error) {
	name = strings.TrimSuffix(name, ".enc")
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return VersionInfo{}, fmt.Errorf("malformed version filename: %s", name)
	}

	seq, err := strconv.Atoi(parts[0])
	if err != nil {
		return VersionInfo{}, fmt.Errorf("invalid sequence number: %w", err)
	}

	ts, err := time.Parse("20060102T150405Z", parts[1])
	if err != nil {
		return VersionInfo{}, fmt.Errorf("invalid timestamp: %w", err)
	}

	return VersionInfo{
		Sequence:  seq,
		Timestamp: ts,
	}, nil
}

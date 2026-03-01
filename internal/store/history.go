// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package store

import (
	"fmt"
	"sort"
)

// History provides version listing, comparison, and restoration metadata.
type History struct {
	store *Store
}

// NewHistory creates a History viewer backed by the given store.
func NewHistory(s *Store) *History {
	return &History{store: s}
}

// ListVersions returns all versions for a project, newest first.
func (h *History) ListVersions(projectHash string) ([]VersionInfo, error) {
	versions, err := h.store.List(projectHash)
	if err != nil {
		return nil, err
	}

	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Sequence > versions[j].Sequence
	})

	return versions, nil
}

// LatestVersion returns the most recent version, or nil if none.
func (h *History) LatestVersion(projectHash string) (*VersionInfo, error) {
	return h.store.Latest(projectHash)
}

// VersionCount returns how many versions are stored for a project.
func (h *History) VersionCount(projectHash string) (int, error) {
	versions, err := h.store.List(projectHash)
	if err != nil {
		return 0, err
	}
	return len(versions), nil
}

// VersionDiff describes changes between two versions.
type VersionDiff struct {
	FromSeq int
	ToSeq   int
	Added   int
	Removed int
	Changed int
}

// Compare computes the difference between two versions.
func (h *History) Compare(projectHash string, fromSeq, toSeq int, key [32]byte) (*VersionDiff, error) {
	fromData, err := h.store.Restore(projectHash, fromSeq, key)
	if err != nil {
		return nil, fmt.Errorf("reading version %d: %w", fromSeq, err)
	}

	toData, err := h.store.Restore(projectHash, toSeq, key)
	if err != nil {
		return nil, fmt.Errorf("reading version %d: %w", toSeq, err)
	}

	fromVars := countVariables(fromData)
	toVars := countVariables(toData)

	diff := &VersionDiff{
		FromSeq: fromSeq,
		ToSeq:   toSeq,
	}

	for k := range toVars {
		if _, exists := fromVars[k]; !exists {
			diff.Added++
		} else if fromVars[k] != toVars[k] {
			diff.Changed++
		}
	}

	for k := range fromVars {
		if _, exists := toVars[k]; !exists {
			diff.Removed++
		}
	}

	return diff, nil
}

// countVariables parses raw .env bytes into key=value pairs.
func countVariables(data []byte) map[string]string {
	vars := make(map[string]string)
	s := string(data)
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			start = i + 1

			// Trim whitespace
			j := 0
			for j < len(line) && (line[j] == ' ' || line[j] == '\t') {
				j++
			}
			line = line[j:]

			if line == "" || line[0] == '#' {
				continue
			}

			for k := 0; k < len(line); k++ {
				if line[k] == '=' {
					key := line[:k]
					val := line[k+1:]
					vars[key] = val
					break
				}
			}
		}
	}
	return vars
}

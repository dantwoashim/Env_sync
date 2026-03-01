// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package audit

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/envsync/envsync/internal/config"
)

// EventType identifies an audit event.
type EventType string

const (
	EventPush             EventType = "push"
	EventPull             EventType = "pull"
	EventInvite           EventType = "invite"
	EventJoin             EventType = "join"
	EventRevoke           EventType = "revoke"
	EventConflictResolved EventType = "conflict_resolved"
	EventBackup           EventType = "backup"
	EventRestore          EventType = "restore"
)

// Entry is a single audit log entry.
type Entry struct {
	Timestamp   time.Time `json:"timestamp"`
	Event       EventType `json:"event"`
	Peer        string    `json:"peer,omitempty"`
	File        string    `json:"file,omitempty"`
	VarsChanged int       `json:"vars_changed,omitempty"`
	Method      string    `json:"method,omitempty"`
	Details     string    `json:"details,omitempty"`
}

// Logger is an append-only audit log.
type Logger struct {
	mu   sync.Mutex
	path string
}

// NewLogger creates a new audit logger.
func NewLogger() (*Logger, error) {
	dataDir, err := config.DataDir()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}

	return &Logger{
		path: filepath.Join(dataDir, "audit.jsonl"),
	}, nil
}

// Log appends an event to the audit log.
func (l *Logger) Log(entry Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling audit entry: %w", err)
	}

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("opening audit log: %w", err)
	}
	defer f.Close()

	_, err = f.Write(append(data, '\n'))
	return err
}

// Read returns all audit entries, newest first.
func (l *Logger) Read(limit int) ([]Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := os.ReadFile(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var all []Entry
	for _, line := range splitLines(data) {
		if len(line) == 0 {
			continue
		}
		var entry Entry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		all = append(all, entry)
	}

	// Reverse (newest first)
	for i, j := 0, len(all)-1; i < j; i, j = i+1, j-1 {
		all[i], all[j] = all[j], all[i]
	}

	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}

	return all, nil
}

// FilterByPeer returns entries for a specific peer.
func (l *Logger) FilterByPeer(peer string, limit int) ([]Entry, error) {
	all, err := l.Read(0)
	if err != nil {
		return nil, err
	}

	var filtered []Entry
	for _, e := range all {
		if e.Peer == peer {
			filtered = append(filtered, e)
			if limit > 0 && len(filtered) >= limit {
				break
			}
		}
	}

	return filtered, nil
}

// FilterByEvent returns entries of a specific event type.
func (l *Logger) FilterByEvent(event EventType, limit int) ([]Entry, error) {
	all, err := l.Read(0)
	if err != nil {
		return nil, err
	}

	var filtered []Entry
	for _, e := range all {
		if e.Event == event {
			filtered = append(filtered, e)
			if limit > 0 && len(filtered) >= limit {
				break
			}
		}
	}

	return filtered, nil
}

func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}

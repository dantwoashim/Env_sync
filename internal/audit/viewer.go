// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package audit

import (
	"fmt"
	"strings"
	"time"
)

// FilterOptions defines criteria for filtering audit entries.
type FilterOptions struct {
	Peer  string
	Event string
	Since time.Time
	Until time.Time
	LastN int
}

// FilterEntries returns entries matching the given criteria.
func FilterEntries(entries []Entry, opts FilterOptions) []Entry {
	var result []Entry

	for _, e := range entries {
		if opts.Peer != "" && e.Peer != opts.Peer {
			continue
		}
		if opts.Event != "" && string(e.Event) != opts.Event {
			continue
		}
		if !opts.Since.IsZero() && e.Timestamp.Before(opts.Since) {
			continue
		}
		if !opts.Until.IsZero() && e.Timestamp.After(opts.Until) {
			continue
		}
		result = append(result, e)
	}

	if opts.LastN > 0 && len(result) > opts.LastN {
		result = result[len(result)-opts.LastN:]
	}

	return result
}

// FormatEntry returns a single-line formatted display of an audit entry.
func FormatEntry(e Entry) string {
	ts := e.Timestamp.Format("2006-01-02 15:04:05")
	icon := eventIcon(e.Event)
	peer := e.Peer
	if peer == "" {
		peer = "(unknown)"
	}

	detail := ""
	if e.File != "" {
		detail = fmt.Sprintf(" (%s", e.File)
		if e.VarsChanged > 0 {
			detail += fmt.Sprintf(", %d vars", e.VarsChanged)
		}
		detail += ")"
	}

	method := ""
	if e.Method != "" {
		method = fmt.Sprintf(" via %s", e.Method)
	}

	return fmt.Sprintf("%s %s %s %s%s%s", ts, icon, e.Event, peer, detail, method)
}

// FormatTable returns a formatted table of audit entries.
func FormatTable(entries []Entry) string {
	if len(entries) == 0 {
		return "  No audit entries found.\n"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  %-19s  %-2s  %-10s  %-20s  %s\n",
		"Timestamp", "", "Event", "Peer", "Details"))
	sb.WriteString(fmt.Sprintf("  %-19s  %-2s  %-10s  %-20s  %s\n",
		strings.Repeat("─", 19), "──", strings.Repeat("─", 10), strings.Repeat("─", 20), strings.Repeat("─", 20)))

	for _, e := range entries {
		ts := e.Timestamp.Format("2006-01-02 15:04:05")
		icon := eventIcon(e.Event)
		peer := e.Peer
		if peer == "" {
			peer = "(unknown)"
		}

		detail := e.File
		if e.VarsChanged > 0 {
			detail += fmt.Sprintf(" (%d vars)", e.VarsChanged)
		}
		if e.Method != "" {
			detail += " via " + e.Method
		}

		sb.WriteString(fmt.Sprintf("  %-19s  %s  %-10s  %-20s  %s\n",
			ts, icon, e.Event, peer, detail))
	}

	return sb.String()
}

func eventIcon(event EventType) string {
	switch event {
	case EventPush:
		return "⬆"
	case EventPull:
		return "⬇"
	case EventInvite:
		return "📨"
	case EventJoin:
		return "🤝"
	case EventRevoke:
		return "🚫"
	case EventConflictResolved:
		return "⚡"
	case EventBackup:
		return "💾"
	case EventRestore:
		return "♻"
	default:
		return "·"
	}
}

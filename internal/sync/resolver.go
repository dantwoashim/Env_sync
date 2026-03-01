// Copyright (c) EnvSync Contributors. SPDX-License-Identifier: MIT

package sync

import (
	"fmt"

	"github.com/envsync/envsync/internal/envfile"
	"github.com/envsync/envsync/internal/ui"
)

// ResolveConflicts handles the conflict resolution flow.
// base = last synced, ours = local, theirs = incoming.
func ResolveConflicts(base, ours, theirs *envfile.EnvFile) (*envfile.EnvFile, error) {
	result := envfile.ThreeWayMerge(base, ours, theirs)

	if !result.HasConflicts() {
		return result.Merged, nil
	}

	// Auto-resolve trivial conflicts (whitespace-only differences)
	var realConflicts []envfile.Conflict
	for _, c := range result.Conflicts {
		if normalizeWhitespace(c.OurValue) == normalizeWhitespace(c.TheirValue) {
			// Whitespace-only → keep ours silently
			result.Merged.Set(c.Key, c.OurValue)
			result.AutoMerged++
		} else {
			realConflicts = append(realConflicts, c)
		}
	}

	if len(realConflicts) == 0 {
		return result.Merged, nil
	}

	// Build conflict items for the TUI
	items := make([]ui.ConflictItem, len(realConflicts))
	for i, c := range realConflicts {
		items[i] = ui.ConflictItem{
			Key:        c.Key,
			BaseValue:  c.BaseValue,
			OurValue:   c.OurValue,
			TheirValue: c.TheirValue,
		}
	}

	// Show summary
	ui.Header("Merge Conflicts")
	ui.Line(fmt.Sprintf("%d auto-merged, %d conflicts need resolution",
		result.AutoMerged, len(realConflicts)))
	ui.Blank()

	// Launch interactive TUI
	tuiResult := ui.RunMergeTUI(items)

	if tuiResult.Aborted {
		return nil, fmt.Errorf("merge aborted by user")
	}

	// Apply decisions
	for _, item := range tuiResult.Conflicts {
		switch item.Decision {
		case ui.MergeAccept:
			result.Merged.Set(item.Key, item.TheirValue)
		case ui.MergeReject:
			result.Merged.Set(item.Key, item.OurValue)
		case ui.MergeEdit:
			result.Merged.Set(item.Key, item.EditValue)
		case ui.MergeSkip:
			// Keep ours (default from merge)
		}
	}

	return result.Merged, nil
}

// normalizeWhitespace trims and collapses whitespace for comparison.
func normalizeWhitespace(s string) string {
	result := make([]byte, 0, len(s))
	inSpace := false
	for _, b := range []byte(s) {
		if b == ' ' || b == '\t' || b == '\r' || b == '\n' {
			if !inSpace {
				result = append(result, ' ')
				inSpace = true
			}
		} else {
			result = append(result, b)
			inSpace = false
		}
	}
	return string(result)
}

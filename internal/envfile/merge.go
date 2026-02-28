package envfile

// MergeResult is the result of a three-way merge.
type MergeResult struct {
	// Merged is the resulting env file after auto-merge.
	Merged *EnvFile

	// Conflicts are variables that were modified on both sides.
	Conflicts []Conflict

	// AutoMerged is the count of variables automatically merged.
	AutoMerged int
}

// Conflict represents a variable modified by both local and remote.
type Conflict struct {
	Key       string
	BaseValue string // value in common ancestor
	OurValue  string // local value
	TheirValue string // remote value
}

// HasConflicts returns true if there are unresolved conflicts.
func (r *MergeResult) HasConflicts() bool {
	return len(r.Conflicts) > 0
}

// ThreeWayMerge performs a three-way merge between base, ours, and theirs.
// base = last synced version, ours = local file, theirs = incoming from peer.
func ThreeWayMerge(base, ours, theirs *EnvFile) *MergeResult {
	result := &MergeResult{
		Merged: &EnvFile{},
	}

	// Build lookup maps
	baseMap := envToMap(base)
	ourMap := envToMap(ours)
	theirMap := envToMap(theirs)

	// Collect all keys
	allKeys := make(map[string]bool)
	for k := range baseMap {
		allKeys[k] = true
	}
	for k := range ourMap {
		allKeys[k] = true
	}
	for k := range theirMap {
		allKeys[k] = true
	}

	for key := range allKeys {
		baseVal, inBase := baseMap[key]
		ourVal, inOurs := ourMap[key]
		theirVal, inTheirs := theirMap[key]

		switch {
		// Both sides unchanged from base
		case inBase && inOurs && inTheirs && ourVal == baseVal && theirVal == baseVal:
			result.Merged.Set(key, baseVal)
			result.AutoMerged++

		// Only we changed
		case inBase && inOurs && inTheirs && ourVal != baseVal && theirVal == baseVal:
			result.Merged.Set(key, ourVal)
			result.AutoMerged++

		// Only they changed
		case inBase && inOurs && inTheirs && ourVal == baseVal && theirVal != baseVal:
			result.Merged.Set(key, theirVal)
			result.AutoMerged++

		// Both changed to same value
		case inBase && inOurs && inTheirs && ourVal == theirVal:
			result.Merged.Set(key, ourVal)
			result.AutoMerged++

		// Both changed to different values — CONFLICT
		case inBase && inOurs && inTheirs && ourVal != theirVal:
			result.Conflicts = append(result.Conflicts, Conflict{
				Key:        key,
				BaseValue:  baseVal,
				OurValue:   ourVal,
				TheirValue: theirVal,
			})
			// Default to ours until resolved
			result.Merged.Set(key, ourVal)

		// We added, they didn't
		case !inBase && inOurs && !inTheirs:
			result.Merged.Set(key, ourVal)
			result.AutoMerged++

		// They added, we didn't
		case !inBase && !inOurs && inTheirs:
			result.Merged.Set(key, theirVal)
			result.AutoMerged++

		// Both added same key — potential conflict
		case !inBase && inOurs && inTheirs:
			if ourVal == theirVal {
				result.Merged.Set(key, ourVal)
				result.AutoMerged++
			} else {
				result.Conflicts = append(result.Conflicts, Conflict{
					Key:        key,
					BaseValue:  "",
					OurValue:   ourVal,
					TheirValue: theirVal,
				})
				result.Merged.Set(key, ourVal)
			}

		// We deleted, they kept unchanged
		case inBase && !inOurs && inTheirs && theirVal == baseVal:
			// Honor our deletion
			result.AutoMerged++

		// They deleted, we kept unchanged
		case inBase && inOurs && !inTheirs && ourVal == baseVal:
			// Honor their deletion
			result.AutoMerged++

		// One deleted, other modified — conflict
		case inBase && !inOurs && inTheirs && theirVal != baseVal:
			result.Conflicts = append(result.Conflicts, Conflict{
				Key:        key,
				BaseValue:  baseVal,
				OurValue:   "(deleted)",
				TheirValue: theirVal,
			})

		case inBase && inOurs && !inTheirs && ourVal != baseVal:
			result.Conflicts = append(result.Conflicts, Conflict{
				Key:        key,
				BaseValue:  baseVal,
				OurValue:   ourVal,
				TheirValue: "(deleted)",
			})

		// Both deleted
		case inBase && !inOurs && !inTheirs:
			// Both agreed to delete
			result.AutoMerged++

		// Only exists on our side (not in base or theirs)
		case !inBase && inOurs:
			result.Merged.Set(key, ourVal)
			result.AutoMerged++
		}
	}

	return result
}

// envToMap converts an EnvFile to a simple key→value map.
func envToMap(ef *EnvFile) map[string]string {
	m := make(map[string]string)
	if ef == nil {
		return m
	}
	for _, entry := range ef.Entries {
		if entry.Key != "" {
			m[entry.Key] = entry.Value
		}
	}
	return m
}

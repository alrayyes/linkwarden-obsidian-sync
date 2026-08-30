// Package state persists the set of links this tool already has notes
// for, between runs.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// SyncState is the on-disk marker: every Linkwarden link ID this tool has
// a note for, mapped to that note's current filename. The filename is
// tracked, not just the ID, because it changes when a link's name (and so
// its slug) does — reconciliation needs to know the *old* filename to
// remove before writing the new one, and there's no way to recompute that
// from the ID alone.
type SyncState struct {
	Links map[int]string `json:"links"`
}

// Load reads the sync marker from path. A missing file is not an error —
// it means no sync has happened yet — and returns the zero SyncState.
func Load(path string) (SyncState, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is the operator-configured state file location, not attacker input
	if os.IsNotExist(err) {
		return SyncState{}, nil
	}
	if err != nil {
		return SyncState{}, fmt.Errorf("reading %s: %w", path, err)
	}

	var s SyncState
	if err := json.Unmarshal(data, &s); err != nil {
		return SyncState{}, fmt.Errorf("parsing %s: %w", path, err)
	}

	return s, nil
}

// Save writes the sync marker to path, creating its parent directory if
// needed.
func Save(path string, s SyncState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling state: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}

	return nil
}

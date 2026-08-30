// Package state persists the on-disk last-synced marker between runs.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/alrayyes/linkwarden-obsidian-sync/internal/linkwarden"
)

// SyncState is the on-disk marker: the newest createdAt timestamp already
// synced, and the IDs of every link that shares that exact timestamp. The ID
// set exists because Linkwarden's createdAt has only second resolution — two
// links saved in the same second are otherwise indistinguishable by time
// alone, and without tracking IDs a rerun would either duplicate one of them
// or silently drop the other depending on API ordering.
type SyncState struct {
	LastSyncedAt time.Time `json:"lastSyncedAt"`
	SeenAtLast   []int     `json:"seenAtLast"`
}

// Load reads the sync marker from path. A missing file is not an error — it
// means no sync has happened yet — and returns the zero SyncState.
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

// SeenAtLastSet returns SeenAtLast as a set, for an O(1) membership check
// against a link's ID at the exact-timestamp boundary.
func (s SyncState) SeenAtLastSet() map[int]bool {
	set := make(map[int]bool, len(s.SeenAtLast))
	for _, id := range s.SeenAtLast {
		set[id] = true
	}

	return set
}

// Next computes the marker to persist after a successful sync, given the
// links that were just fetched (newest-first). If nothing new came in, the
// existing state is returned unchanged.
func Next(current SyncState, freshNewestFirst []linkwarden.Link) SyncState {
	if len(freshNewestFirst) == 0 {
		return current
	}

	newest := freshNewestFirst[0].CreatedAt
	var seen []int
	for _, l := range freshNewestFirst {
		if l.CreatedAt.Equal(newest) {
			seen = append(seen, l.ID)
		}
	}

	return SyncState{LastSyncedAt: newest, SeenAtLast: seen}
}

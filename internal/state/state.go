// Package state persists the on-disk last-synced marker between runs.
package state

import (
	"encoding/json"
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

func Load(path string) (SyncState, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return SyncState{}, nil
	}
	if err != nil {
		return SyncState{}, err
	}

	var s SyncState
	if err := json.Unmarshal(data, &s); err != nil {
		return SyncState{}, err
	}
	return s, nil
}

func Save(path string, s SyncState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

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

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// syncState is the on-disk marker: the newest createdAt timestamp already
// synced, and the IDs of every link that shares that exact timestamp. The ID
// set exists because Linkwarden's createdAt has only second resolution — two
// links saved in the same second are otherwise indistinguishable by time
// alone, and without tracking IDs a rerun would either duplicate one of them
// or silently drop the other depending on API ordering.
type syncState struct {
	LastSyncedAt time.Time `json:"lastSyncedAt"`
	SeenAtLast   []int     `json:"seenAtLast"`
}

func loadState(path string) (syncState, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return syncState{}, nil
	}
	if err != nil {
		return syncState{}, err
	}

	var s syncState
	if err := json.Unmarshal(data, &s); err != nil {
		return syncState{}, err
	}
	return s, nil
}

func saveState(path string, s syncState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (s syncState) seenAtLastSet() map[int]bool {
	set := make(map[int]bool, len(s.SeenAtLast))
	for _, id := range s.SeenAtLast {
		set[id] = true
	}
	return set
}

// nextState computes the marker to persist after a successful sync, given
// the links that were just fetched (newest-first). If nothing new came in,
// the existing state is returned unchanged.
func nextState(current syncState, freshNewestFirst []link) syncState {
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

	return syncState{LastSyncedAt: newest, SeenAtLast: seen}
}

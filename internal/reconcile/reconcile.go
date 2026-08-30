// Package reconcile makes a vault directory's notes mirror Linkwarden's
// current set of links: adding a note for a new link, overwriting one for
// a link that already had a note, and removing a note for a link that's
// gone.
package reconcile

import (
	"fmt"

	"github.com/alrayyes/linkwarden-obsidian-sync/internal/linkwarden"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/note"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/state"
)

// Result summarizes what one reconciliation run did.
type Result struct {
	Added, Updated, Removed int
}

// Notes makes notesDir's *.md files mirror links exactly. For every link
// it writes (or overwrites — this tool doesn't diff content first, so an
// unchanged link still counts as Updated) its note, removing the old file
// first if the link's name changed since prev and so its filename's slug
// did too. Any link present in prev but not in links has its note
// removed. Returns the state to persist for the next run.
func Notes(notesDir string, links []linkwarden.Link, prev state.SyncState) (state.SyncState, Result, error) {
	next := state.SyncState{Links: make(map[int]string, len(links))}
	var result Result
	seen := make(map[int]bool, len(links))

	for _, l := range links {
		seen[l.ID] = true

		filename, added, err := writeOne(notesDir, l, prev)
		if err != nil {
			return state.SyncState{}, Result{}, err
		}

		next.Links[l.ID] = filename
		if added {
			result.Added++
		} else {
			result.Updated++
		}
	}

	for id, filename := range prev.Links {
		if seen[id] {
			continue
		}
		if err := note.Remove(notesDir, filename); err != nil {
			return state.SyncState{}, Result{}, fmt.Errorf("removing note for link %d: %w", id, err)
		}
		result.Removed++
	}

	return next, result, nil
}

// writeOne writes l's note, removing its previous file first if the link's
// name changed since prev and so its filename's slug did too. added
// reports whether l is new to prev, for the caller's Result tally.
func writeOne(notesDir string, l linkwarden.Link, prev state.SyncState) (filename string, added bool, err error) {
	filename = note.Filename(l)
	oldFilename, existedBefore := prev.Links[l.ID]

	if existedBefore && oldFilename != filename {
		if err := note.Remove(notesDir, oldFilename); err != nil {
			return "", false, fmt.Errorf("removing renamed note for link %d: %w", l.ID, err)
		}
	}

	if _, err := note.Write(notesDir, l); err != nil {
		return "", false, fmt.Errorf("writing note for link %d: %w", l.ID, err)
	}

	return filename, !existedBefore, nil
}

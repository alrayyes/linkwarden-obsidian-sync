package reconcile_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alrayyes/linkwarden-obsidian-sync/internal/linkwarden"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/reconcile"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/state"
	"github.com/stretchr/testify/require"
)

func link(id int, name, description string) linkwarden.Link {
	return linkwarden.Link{ID: id, Name: name, URL: "https://example.com", Description: description, CreatedAt: time.Now()}
}

func TestNotesAddsNewLinks(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	links := []linkwarden.Link{link(1, "First", ""), link(2, "Second", "")}

	next, result, err := reconcile.Notes(dir, links, state.SyncState{})
	require.NoError(t, err)
	require.Equal(t, reconcile.Result{Added: 2}, result)
	require.Len(t, next.Links, 2)

	for id, filename := range next.Links {
		_, err := os.Stat(filepath.Join(dir, filename))
		require.NoError(t, err, "note for link %d should exist on disk", id)
	}
}

func TestNotesUpdatesContentForAnExistingLink(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	prevState, _, err := reconcile.Notes(dir, []linkwarden.Link{link(1, "First", "old description")}, state.SyncState{})
	require.NoError(t, err)

	next, result, err := reconcile.Notes(dir, []linkwarden.Link{link(1, "First", "new description")}, prevState)
	require.NoError(t, err)
	require.Equal(t, reconcile.Result{Updated: 1}, result)

	data, err := os.ReadFile(filepath.Join(dir, next.Links[1])) //nolint:gosec // test's own t.TempDir() fixture
	require.NoError(t, err)
	require.Contains(t, string(data), "new description")
	require.NotContains(t, string(data), "old description")
}

func TestNotesRenamesWhenTheSlugChanges(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	prevState, _, err := reconcile.Notes(dir, []linkwarden.Link{link(1, "Old Title", "")}, state.SyncState{})
	require.NoError(t, err)
	oldFilename := prevState.Links[1]

	next, result, err := reconcile.Notes(dir, []linkwarden.Link{link(1, "New Title", "")}, prevState)
	require.NoError(t, err)
	require.Equal(t, reconcile.Result{Updated: 1}, result)

	newFilename := next.Links[1]
	require.NotEqual(t, oldFilename, newFilename)

	_, err = os.Stat(filepath.Join(dir, oldFilename))
	require.True(t, os.IsNotExist(err), "old-slug file should be gone")

	_, err = os.Stat(filepath.Join(dir, newFilename))
	require.NoError(t, err, "new-slug file should exist")
}

func TestNotesRemovesALinkNoLongerPresent(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	prevState, _, err := reconcile.Notes(dir, []linkwarden.Link{link(1, "Keep", ""), link(2, "Drop", "")}, state.SyncState{})
	require.NoError(t, err)
	droppedFilename := prevState.Links[2]

	next, result, err := reconcile.Notes(dir, []linkwarden.Link{link(1, "Keep", "")}, prevState)
	require.NoError(t, err)
	require.Equal(t, reconcile.Result{Updated: 1, Removed: 1}, result)
	require.Len(t, next.Links, 1)
	require.NotContains(t, next.Links, 2)

	_, err = os.Stat(filepath.Join(dir, droppedFilename))
	require.True(t, os.IsNotExist(err))
}

func TestNotesEmptyLinkSetRemovesEverything(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	prevState, _, err := reconcile.Notes(dir, []linkwarden.Link{link(1, "Only", "")}, state.SyncState{})
	require.NoError(t, err)

	next, result, err := reconcile.Notes(dir, nil, prevState)
	require.NoError(t, err)
	require.Equal(t, reconcile.Result{Removed: 1}, result)
	require.Empty(t, next.Links)
}

func TestNotesUnchangedContentStillCountsAsUpdated(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	links := []linkwarden.Link{link(1, "Stable", "same every time")}
	prevState, _, err := reconcile.Notes(dir, links, state.SyncState{})
	require.NoError(t, err)

	_, result, err := reconcile.Notes(dir, links, prevState)
	require.NoError(t, err)
	require.Equal(t, reconcile.Result{Updated: 1}, result, "every fetched link that already existed counts as updated, even if content didn't change — this tool doesn't diff content before overwriting")
}

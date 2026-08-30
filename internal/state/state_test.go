package state_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/alrayyes/linkwarden-obsidian-sync/internal/state"
	"github.com/stretchr/testify/require"
)

func TestLoadMissingFileReturnsZeroState(t *testing.T) {
	t.Parallel()

	got, err := state.Load(filepath.Join(t.TempDir(), "does-not-exist.json"))
	require.NoError(t, err)
	require.Empty(t, got.Links)
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "state.json")
	want := state.SyncState{Links: map[int]string{1: "one.md", 2: "two.md"}}

	require.NoError(t, state.Save(path, want))

	got, err := state.Load(path)
	require.NoError(t, err)
	require.Equal(t, want.Links, got.Links)
}

func TestLoadMalformedFileIsAnError(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "state.json")
	require.NoError(t, os.WriteFile(path, []byte("not json"), 0o600))

	_, err := state.Load(path)
	require.Error(t, err)
}

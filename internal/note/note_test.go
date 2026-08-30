package note_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alrayyes/linkwarden-obsidian-sync/internal/linkwarden"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/note"
	"github.com/stretchr/testify/require"
)

func TestSlugify(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		id   int
		want string
	}{
		{"A Real Link Title", 1, "A Real Link Title"},
		{"", 42, "link-42"},
		{"!!!", 42, "link-42"},
		{"Weird/Chars: allowed?", 7, "WeirdChars allowed"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, c.want, note.Slugify(c.name, c.id))
		})
	}
}

func TestNoteMarkdown(t *testing.T) {
	t.Parallel()

	t.Run("escapes quotes", func(t *testing.T) {
		t.Parallel()

		l := linkwarden.Link{
			ID:          1,
			Name:        `A "quoted" title`,
			URL:         "https://example.com/a?b=c",
			Description: "a description",
			Tags:        []linkwarden.Tag{{Name: "go"}, {Name: "cli"}},
			Collection:  linkwarden.Collection{Name: "Reading"},
			CreatedAt:   time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC),
		}

		got := note.Markdown(l)

		for _, want := range []string{
			`url: "https://example.com/a?b=c"`,
			`collection: "Reading"`,
			"  - \"go\"",
			"  - \"cli\"",
			"created: 2026-08-23",
			"linkwarden_id: 1",
			`# A "quoted" title`,
			"<https://example.com/a?b=c>",
			"a description",
		} {
			require.Contains(t, got, want)
		}
	})

	t.Run("no tags", func(t *testing.T) {
		t.Parallel()

		l := linkwarden.Link{ID: 2, Name: "No tags", URL: "https://example.com", CreatedAt: time.Now()}
		require.Contains(t, note.Markdown(l), "tags: []")
	})
}

func TestWriteOverwritesAnExistingNote(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l := linkwarden.Link{ID: 1, Name: "First write", URL: "https://example.com", CreatedAt: time.Now()}

	path, err := note.Write(dir, l)
	require.NoError(t, err)

	// Simulate a human having edited the note, or an earlier sync writing
	// stale content, before a rerun.
	require.NoError(t, os.WriteFile(path, []byte("stale content\n"), 0o600))

	l.Description = "updated description"
	_, err = note.Write(dir, l)
	require.NoError(t, err)

	after, err := os.ReadFile(path) //nolint:gosec // path is this test's own t.TempDir() fixture, not user input
	require.NoError(t, err)
	require.Contains(t, string(after), "updated description")
	require.NotContains(t, string(after), "stale content")
}

func TestWriteCreatesDir(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "nested", "subdir")
	l := linkwarden.Link{ID: 1, Name: "x", URL: "https://example.com", CreatedAt: time.Now()}

	_, err := note.Write(dir, l)
	require.NoError(t, err)
}

func TestFilename(t *testing.T) {
	t.Parallel()

	require.Equal(t, "A Real Link Title.md", note.Filename(linkwarden.Link{ID: 1, Name: "A Real Link Title"}))
	require.Equal(t, "link-42.md", note.Filename(linkwarden.Link{ID: 42, Name: ""}))
}

func TestRemove(t *testing.T) {
	t.Parallel()

	t.Run("deletes an existing note", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		l := linkwarden.Link{ID: 1, Name: "x", URL: "https://example.com", CreatedAt: time.Now()}
		path, err := note.Write(dir, l)
		require.NoError(t, err)

		require.NoError(t, note.Remove(dir, filepath.Base(path)))

		_, err = os.Stat(path)
		require.True(t, os.IsNotExist(err))
	})

	t.Run("a note already gone is not an error", func(t *testing.T) {
		t.Parallel()

		require.NoError(t, note.Remove(t.TempDir(), "never-existed.md"))
	})
}

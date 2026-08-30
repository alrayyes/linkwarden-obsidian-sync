package note_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/alrayyes/linkwarden-obsidian-sync/internal/linkwarden"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/note"
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

		got := note.NoteMarkdown(l)

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
		require.Contains(t, note.NoteMarkdown(l), "tags: []")
	})
}

func TestWriteNoteSkipsExisting(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	l := linkwarden.Link{ID: 1, Name: "First write", URL: "https://example.com", CreatedAt: time.Now()}

	written, path, err := note.WriteNote(dir, l)
	require.NoError(t, err)
	require.True(t, written, "first call should write the note")

	original, err := os.ReadFile(path)
	require.NoError(t, err)

	// Simulate a human having started editing the note before a rerun.
	edited := string(original) + "\nmy own notes\n"
	require.NoError(t, os.WriteFile(path, []byte(edited), 0o644))

	written, _, err = note.WriteNote(dir, l)
	require.NoError(t, err)
	require.False(t, written, "second call must not overwrite an existing note")

	after, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, edited, string(after))
}

func TestWriteNoteCreatesDir(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "nested", "subdir")
	l := linkwarden.Link{ID: 1, Name: "x", URL: "https://example.com", CreatedAt: time.Now()}

	_, _, err := note.WriteNote(dir, l)
	require.NoError(t, err)
}

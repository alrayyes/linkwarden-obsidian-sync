package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSlugify(t *testing.T) {
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
		got := slugify(c.name, c.id)
		if got != c.want {
			t.Errorf("slugify(%q, %d) = %q, want %q", c.name, c.id, got, c.want)
		}
	}
}

func TestNoteMarkdownEscapesQuotes(t *testing.T) {
	l := link{
		ID:          1,
		Name:        `A "quoted" title`,
		URL:         "https://example.com/a?b=c",
		Description: "a description",
		Tags:        []tag{{Name: "go"}, {Name: "cli"}},
		Collection:  collection{Name: "Reading"},
		CreatedAt:   time.Date(2026, 8, 23, 0, 0, 0, 0, time.UTC),
	}

	got := noteMarkdown(l)

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
		if !strings.Contains(got, want) {
			t.Errorf("noteMarkdown() missing %q in:\n%s", want, got)
		}
	}
}

func TestNoteMarkdownNoTags(t *testing.T) {
	l := link{ID: 2, Name: "No tags", URL: "https://example.com", CreatedAt: time.Now()}
	got := noteMarkdown(l)
	if !strings.Contains(got, "tags: []") {
		t.Errorf("noteMarkdown() with no tags should contain 'tags: []', got:\n%s", got)
	}
}

func TestWriteNoteSkipsExisting(t *testing.T) {
	dir := t.TempDir()
	l := link{ID: 1, Name: "First write", URL: "https://example.com", CreatedAt: time.Now()}

	written, path, err := writeNote(dir, l)
	if err != nil {
		t.Fatalf("writeNote() error = %v", err)
	}
	if !written {
		t.Fatal("writeNote() first call: want written = true")
	}

	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written note: %v", err)
	}

	// Simulate a human having started editing the note before a rerun.
	edited := string(original) + "\nmy own notes\n"
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatalf("simulating edit: %v", err)
	}

	written, _, err = writeNote(dir, l)
	if err != nil {
		t.Fatalf("writeNote() second call error = %v", err)
	}
	if written {
		t.Fatal("writeNote() second call: want written = false (must not overwrite an existing note)")
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading note after rerun: %v", err)
	}
	if string(after) != edited {
		t.Fatal("writeNote() overwrote a note that already existed on disk")
	}
}

func TestWriteNoteCreatesDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "subdir")
	l := link{ID: 1, Name: "x", URL: "https://example.com", CreatedAt: time.Now()}

	if _, _, err := writeNote(dir, l); err != nil {
		t.Fatalf("writeNote() into a non-existent dir: %v", err)
	}
}

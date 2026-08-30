// Package note turns a Linkwarden link into an Obsidian note and writes it
// safely to disk.
package note

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alrayyes/linkwarden-obsidian-sync/internal/linkwarden"
)

var slugDisallowed = regexp.MustCompile(`[^a-zA-Z0-9 _-]+`)

// Slugify turns a link's name (or, failing that, its URL) into a filesystem-
// and Obsidian-safe filename. It never returns an empty string: a name that's
// all punctuation falls back to "link-<id>" so two links never collide on an
// empty slug.
func Slugify(name string, id int) string {
	cleaned := slugDisallowed.ReplaceAllString(name, "")
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return fmt.Sprintf("link-%d", id)
	}
	if len(cleaned) > 120 {
		cleaned = strings.TrimSpace(cleaned[:120])
	}

	return cleaned
}

// Markdown renders a link as an Obsidian note: YAML frontmatter the vault
// can query on (url, tags, collection, created), then the description as the
// body so the note reads as something rather than bare metadata.
func Markdown(l linkwarden.Link) string {
	var b strings.Builder

	b.WriteString("---\n")
	fmt.Fprintf(&b, "url: %s\n", yamlQuote(l.URL))
	fmt.Fprintf(&b, "collection: %s\n", yamlQuote(l.Collection.Name))
	if len(l.Tags) > 0 {
		b.WriteString("tags:\n")
		for _, t := range l.Tags {
			fmt.Fprintf(&b, "  - %s\n", yamlQuote(t.Name))
		}
	} else {
		b.WriteString("tags: []\n")
	}
	fmt.Fprintf(&b, "created: %s\n", l.CreatedAt.Format("2006-01-02"))
	fmt.Fprintf(&b, "linkwarden_id: %d\n", l.ID)
	b.WriteString("---\n\n")

	title := l.Name
	if title == "" {
		title = l.URL
	}
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, "<%s>\n", l.URL)

	if l.Description != "" {
		fmt.Fprintf(&b, "\n%s\n", l.Description)
	}

	return b.String()
}

// yamlQuote wraps a scalar in double quotes for YAML flow style, escaping the
// two characters that would otherwise break it. Link names and descriptions
// come from whatever a webpage's <title> says, which is arbitrary text a
// human never wrote for this file — don't trust it to be YAML-safe unquoted.
func yamlQuote(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)

	return `"` + s + `"`
}

// Filename is the .md file a link's note is written under.
func Filename(l linkwarden.Link) string {
	return Slugify(l.Name, l.ID) + ".md"
}

// Write renders l as markdown and writes it to dir, creating dir if
// needed. It always overwrites an existing file at that path:
// reconciliation's whole point is that the vault mirrors Linkwarden's
// current state, so a hand-edit to a synced note doesn't survive the next
// run — the same reasoning that makes Remove unconditional.
func Write(dir string, l linkwarden.Link) (path string, err error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", fmt.Errorf("creating %s: %w", dir, err)
	}

	path = filepath.Join(dir, Filename(l))
	if err := os.WriteFile(path, []byte(Markdown(l)), 0o600); err != nil {
		return "", fmt.Errorf("writing %s: %w", path, err)
	}

	return path, nil
}

// Remove deletes the note at dir/filename. A note that's already gone
// (removed by hand, say) isn't an error — the end state either way is
// "it's gone."
func Remove(dir, filename string) error {
	path := filepath.Join(dir, filename)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %s: %w", path, err)
	}

	return nil
}

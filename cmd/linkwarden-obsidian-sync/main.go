// Command linkwarden-obsidian-sync copies newly saved Linkwarden links into
// an Obsidian vault as notes, then pushes them to a dated branch for review.
//
// It's meant to run periodically and unattended (a systemd --user timer),
// with no LLM in the loop: everything it does is a deterministic API call,
// file write and git command.
package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"

	"github.com/alrayyes/linkwarden-obsidian-sync/internal/linkwarden"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/note"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/state"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/vault"
)

// version is set at build time via goreleaser's -X main.version ldflag,
// off the tag release-please creates. Unset in a `go build` or `go run`.
var version = "dev"

// errLinkwardenURLNotSet and errLinkwardenTokenNotSet are the two config
// errors loadConfig can return.
var (
	errLinkwardenURLNotSet   = errors.New("LINKWARDEN_URL is not set (e.g. https://linkwarden.example.com, no trailing slash)")
	errLinkwardenTokenNotSet = errors.New("LINKWARDEN_TOKEN is not set (Linkwarden → Settings → Access Tokens)")
)

func main() {
	log.Printf("linkwarden-obsidian-sync %s", version)

	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := run(cfg); err != nil {
		log.Fatal(err)
	}
}

// run does the actual sync: read state, fetch what's new, write notes, save
// state, then push. Split out of main so the top level stays a thin
// parse-config-and-report-the-error shell.
func run(cfg config) error {
	statePath := filepath.Join(cfg.stateDir, "last-synced.json")
	st, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("reading state at %s: %w", statePath, err)
	}

	client := linkwarden.NewClient(cfg.linkwardenURL, cfg.linkwardenToken)
	freshNewestFirst, err := client.FetchNewLinks(st.LastSyncedAt, st.SeenAtLastSet())
	if err != nil {
		return fmt.Errorf("fetching links: %w", err)
	}

	if len(freshNewestFirst) == 0 {
		log.Println("nothing new since last sync")

		return nil
	}

	notesDir := filepath.Join(cfg.vaultPath, cfg.vaultSubdir)
	written, err := writeNotes(notesDir, freshNewestFirst)
	if err != nil {
		return err
	}

	if err := state.Save(statePath, state.Next(st, freshNewestFirst)); err != nil {
		return fmt.Errorf("saving state to %s: %w", statePath, err)
	}

	if written == 0 {
		log.Println("no new notes written (all already existed)")

		return nil
	}

	if cfg.skipGit {
		log.Printf("wrote %d note(s), skipping git (LINKWARDEN_SYNC_SKIP_GIT set)", written)

		return nil
	}

	return pushToVault(cfg, written)
}

// writeNotes writes freshNewestFirst into notesDir in the order they were
// actually saved (oldest first), so a partial failure leaves the notes
// already on disk in a sensible sequence.
func writeNotes(notesDir string, freshNewestFirst []linkwarden.Link) (int, error) {
	written := 0
	for _, l := range slices.Backward(freshNewestFirst) {
		ok, path, err := note.WriteNote(notesDir, l)
		if err != nil {
			return written, fmt.Errorf("writing note for link %d: %w", l.ID, err)
		}
		if ok {
			written++
			log.Printf("wrote %s", path)
		} else {
			log.Printf("skipped %s (already exists)", path)
		}
	}

	return written, nil
}

// pushToVault commits the newly written notes and pushes them to a dated
// branch, reporting where to open the review pull request.
func pushToVault(cfg config, written int) error {
	branch, err := vault.CommitAndPush(cfg.vaultPath, cfg.vaultSubdir, written)
	if err != nil {
		return fmt.Errorf("git: %w", err)
	}
	if branch == "" {
		log.Println("wrote notes but git reported no changes to commit")

		return nil
	}

	fmt.Printf("wrote %d note(s) on branch %s\n", written, branch)
	fmt.Printf("open a pull request: https://git.higherlearning.eu/alrayyes/obsidian/compare/main...%s\n", branch)

	return nil
}

type config struct {
	linkwardenURL   string
	linkwardenToken string
	vaultPath       string
	vaultSubdir     string
	stateDir        string
	skipGit         bool
}

func loadConfig() (config, error) {
	c := config{
		vaultPath:   getenvDefault("VAULT_PATH", filepath.Join(os.Getenv("HOME"), "Documents", "obsidian")),
		vaultSubdir: getenvDefault("VAULT_SUBDIR", "Linkwarden"),
		stateDir:    getenvDefault("LINKWARDEN_SYNC_STATE_DIR", filepath.Join(os.Getenv("HOME"), ".local", "state", "linkwarden-obsidian-sync")),
		skipGit:     os.Getenv("LINKWARDEN_SYNC_SKIP_GIT") != "",
	}

	c.linkwardenURL = os.Getenv("LINKWARDEN_URL")
	if c.linkwardenURL == "" {
		return config{}, errLinkwardenURLNotSet
	}

	c.linkwardenToken = os.Getenv("LINKWARDEN_TOKEN")
	if c.linkwardenToken == "" {
		return config{}, errLinkwardenTokenNotSet
	}

	return c, nil
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

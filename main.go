// Command linkwarden-obsidian-sync copies newly saved Linkwarden links into
// an Obsidian vault as notes, then pushes them to a dated branch for review.
//
// It's meant to run periodically and unattended (a systemd --user timer),
// with no LLM in the loop: everything it does is a deterministic API call,
// file write and git command.
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	statePath := filepath.Join(cfg.stateDir, "last-synced.json")
	state, err := loadState(statePath)
	if err != nil {
		log.Fatalf("reading state at %s: %v", statePath, err)
	}

	client := newLinkwardenClient(cfg.linkwardenURL, cfg.linkwardenToken)
	freshNewestFirst, err := client.fetchNewLinks(state.LastSyncedAt, state.seenAtLastSet())
	if err != nil {
		log.Fatalf("fetching links: %v", err)
	}

	if len(freshNewestFirst) == 0 {
		log.Println("nothing new since last sync")
		return
	}

	notesDir := filepath.Join(cfg.vaultPath, cfg.vaultSubdir)

	written := 0
	for i := len(freshNewestFirst) - 1; i >= 0; i-- {
		l := freshNewestFirst[i]
		ok, path, err := writeNote(notesDir, l)
		if err != nil {
			log.Fatalf("writing note for link %d: %v", l.ID, err)
		}
		if ok {
			written++
			log.Printf("wrote %s", path)
		} else {
			log.Printf("skipped %s (already exists)", path)
		}
	}

	newState := nextState(state, freshNewestFirst)
	if err := saveState(statePath, newState); err != nil {
		log.Fatalf("saving state to %s: %v", statePath, err)
	}

	if written == 0 {
		log.Println("no new notes written (all already existed)")
		return
	}

	if cfg.skipGit {
		log.Printf("wrote %d note(s), skipping git (LINKWARDEN_SYNC_SKIP_GIT set)", written)
		return
	}

	branch, err := commitAndPush(cfg.vaultPath, cfg.vaultSubdir, written)
	if err != nil {
		log.Fatalf("git: %v", err)
	}
	if branch == "" {
		log.Println("wrote notes but git reported no changes to commit")
		return
	}

	fmt.Printf("wrote %d note(s) on branch %s\n", written, branch)
	fmt.Printf("open a pull request: https://git.higherlearning.eu/alrayyes/obsidian/compare/main...%s\n", branch)
}

// commitAndPush stages the vault subdirectory, commits on a dated branch and
// pushes it. It returns the branch name, or "" if there was nothing to
// commit. It shells out to git directly rather than a library — this only
// ever needs to run against the one local clone a human already has
// configured with working push credentials.
func commitAndPush(vaultPath, subdir string, count int) (string, error) {
	run := func(args ...string) (string, error) {
		cmd := exec.Command("git", args...)
		cmd.Dir = vaultPath
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git %v: %w\n%s", args, err, out)
		}
		return string(out), nil
	}

	branch := fmt.Sprintf("linkwarden-sync-%s", time.Now().Format("2006-01-02-150405"))
	if _, err := run("checkout", "-b", branch); err != nil {
		return "", err
	}

	if _, err := run("add", subdir); err != nil {
		return "", err
	}

	status, err := run("status", "--porcelain", "--", subdir)
	if err != nil {
		return "", err
	}
	if status == "" {
		_, _ = run("checkout", "-")
		_, _ = run("branch", "-D", branch)
		return "", nil
	}

	msg := fmt.Sprintf("docs(linkwarden): sync %d new link(s)", count)
	if _, err := run("commit", "-m", msg); err != nil {
		return "", err
	}
	if _, err := run("push", "-u", "origin", branch); err != nil {
		return "", err
	}

	return branch, nil
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
		return config{}, fmt.Errorf("LINKWARDEN_URL is not set (e.g. https://linkwarden.example.com, no trailing slash)")
	}

	c.linkwardenToken = os.Getenv("LINKWARDEN_TOKEN")
	if c.linkwardenToken == "" {
		return config{}, fmt.Errorf("LINKWARDEN_TOKEN is not set (Linkwarden → Settings → Access Tokens)")
	}

	return c, nil
}

func getenvDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

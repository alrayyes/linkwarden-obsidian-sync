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
	"path/filepath"
	"slices"

	"github.com/alrayyes/linkwarden-obsidian-sync/internal/config"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/linkwarden"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/note"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/state"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/vault"
	"github.com/spf13/cobra"
)

// version is set at build time via goreleaser's -X main.version ldflag,
// off the tag release-please creates. Unset in a `go build` or `go run`.
var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		log.Fatal(err)
	}
}

// newRootCmd builds the command tree: the root command runs the sync,
// `init` writes a template config file. Split out of main so tests can
// exercise it without going through os.Exit.
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "linkwarden-obsidian-sync",
		Short: "Copy newly saved Linkwarden links into an Obsidian vault as notes",
		// This tool's own errors are already formatted for a human to read
		// (log.Fatal-shaped, from run's own wrapping); cobra's usage dump on
		// every runtime failure would bury that under noise meant for a
		// misused flag, not a failed sync.
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(*cobra.Command, []string) {
			log.Printf("linkwarden-obsidian-sync %s", version)
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSync(cmd)
		},
	}
	root.PersistentFlags().String("config", "", "path to config file (default $XDG_CONFIG_HOME/linkwarden-obsidian-sync/config.toml)")
	root.Flags().Bool("skip-git", false, "write notes but skip the git commit/push step")

	root.AddCommand(newInitCmd())

	return root
}

func newInitCmd() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Write a template config file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			force, _ := cmd.Flags().GetBool("force")

			written, err := config.WriteTemplate(configPath, force)
			if err != nil {
				return fmt.Errorf("init: %w", err)
			}
			fmt.Printf("wrote %s\n", written)
			fmt.Println("edit it to set linkwarden_url and linkwarden_token, then run linkwarden-obsidian-sync")

			return nil
		},
	}
	initCmd.Flags().Bool("force", false, "overwrite an existing config file")

	return initCmd
}

// runSync loads config and runs the actual sync: read state, fetch what's
// new, write notes, save state, then push.
func runSync(cmd *cobra.Command) error {
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if cmd.Flags().Changed("skip-git") {
		cfg.SkipGit, _ = cmd.Flags().GetBool("skip-git")
	}

	statePath := filepath.Join(cfg.StateDir, "last-synced.json")
	st, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("reading state at %s: %w", statePath, err)
	}

	client := linkwarden.NewClient(cfg.LinkwardenURL, cfg.LinkwardenToken)
	freshNewestFirst, err := client.FetchNewLinks(st.LastSyncedAt, st.SeenAtLastSet())
	if err != nil {
		return fmt.Errorf("fetching links: %w", err)
	}

	if len(freshNewestFirst) == 0 {
		log.Println("nothing new since last sync")

		return nil
	}

	notesDir := filepath.Join(cfg.VaultPath, cfg.VaultSubdir)
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

	if cfg.SkipGit {
		log.Printf("wrote %d note(s), skipping git (--skip-git set)", written)

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
func pushToVault(cfg config.Config, written int) error {
	branch, err := vault.CommitAndPush(cfg.VaultPath, cfg.VaultSubdir, written)
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

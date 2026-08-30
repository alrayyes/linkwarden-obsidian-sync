// Command linkwarden-obsidian-sync copies newly saved Linkwarden links into
// an Obsidian vault as notes, then pushes them to a dated branch for review.
//
// It's meant to run periodically and unattended (a systemd --user timer),
// with no LLM in the loop: everything it does is a deterministic API call,
// file write and git command.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/alrayyes/linkwarden-obsidian-sync/internal/config"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/linkwarden"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/note"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/state"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/vault"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// linkwardenURLEnv and linkwardenTokenEnv are the two environment
// variables internal/config binds linkwarden_url/linkwarden_token to.
// Named here, not imported from internal/config, so that package doesn't
// need to export its env-binding names just for this first-run check.
const (
	linkwardenURLEnv   = "LINKWARDEN_URL"
	linkwardenTokenEnv = "LINKWARDEN_TOKEN"
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
	root.Flags().BoolP("yes", "y", false, "on a first, unconfigured run, write a template config file without prompting")

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
	cfg, ranInit, err := loadConfigOrOfferInit(cmd)
	if err != nil {
		return err
	}
	if ranInit {
		return nil
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

// loadConfigOrOfferInit loads the config, offering to write a template on
// a genuinely unconfigured run instead of just erroring (see offerInit).
// ranInit reports whether a template was written, in which case the
// caller should stop rather than try to sync with placeholder credentials.
func loadConfigOrOfferInit(cmd *cobra.Command) (config.Config, bool, error) {
	configPath, _ := cmd.Flags().GetString("config")
	loaded, loadErr := config.Load(configPath)
	if loadErr == nil {
		return loaded, false, nil
	}
	if !errors.Is(loadErr, config.ErrMissingRequired) {
		return config.Config{}, false, fmt.Errorf("config: %w", loadErr)
	}

	wrote, promptErr := offerInit(cmd, configPath)
	if promptErr != nil {
		return config.Config{}, false, fmt.Errorf("config: %w", promptErr)
	}
	if wrote {
		return config.Config{}, true, nil
	}

	return config.Config{}, false, fmt.Errorf("config: %w", loadErr)
}

// offerInit offers to write a template config file when the run is
// genuinely unconfigured — no config file at the resolved path, and
// neither LINKWARDEN_URL nor LINKWARDEN_TOKEN set — rather than leaving
// someone with nothing but a "missing required settings" error and no
// path forward. It reports whether it wrote a template, in which case the
// caller should stop: there's nothing left to sync with placeholder
// credentials.
//
// A config file that already exists (just missing a key, say) or an
// env var already set means someone has already tried to configure this,
// wrongly — that gets the plain error, not a wizard.
func offerInit(cmd *cobra.Command, configPath string) (wrote bool, err error) {
	resolved, err := config.ResolvePath(configPath)
	if err != nil {
		return false, fmt.Errorf("resolving config path: %w", err)
	}
	if _, statErr := os.Stat(resolved); statErr == nil {
		return false, nil
	}
	if os.Getenv(linkwardenURLEnv) != "" || os.Getenv(linkwardenTokenEnv) != "" {
		return false, nil
	}

	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		if !isTerminal(os.Stdin) || !isTerminal(os.Stdout) {
			fmt.Fprintf(os.Stderr, "no config found at %s and no %s/%s set; run `linkwarden-obsidian-sync init` to get started\n",
				resolved, linkwardenURLEnv, linkwardenTokenEnv)

			return false, nil
		}

		fmt.Printf("No config found at %s. Write a template now? [y/N] ", resolved)
		answer, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(answer)), "y") {
			return false, nil
		}
	}

	written, err := config.WriteTemplate(resolved, false)
	if err != nil {
		return false, fmt.Errorf("writing template: %w", err)
	}
	fmt.Printf("wrote %s\n", written)
	fmt.Println("edit it to set linkwarden_url and linkwarden_token, then run linkwarden-obsidian-sync again")

	return true, nil
}

func isTerminal(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
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

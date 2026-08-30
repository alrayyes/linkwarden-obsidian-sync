// Command linkwarden-obsidian-sync keeps a directory of Obsidian notes in
// sync with your saved Linkwarden links: one note per link, added,
// updated or removed to match Linkwarden's current state exactly.
//
// It's meant to run periodically and unattended (a systemd --user timer),
// with no LLM in the loop: everything it does is a deterministic API call
// and a file write. It doesn't touch git — reviewing and committing
// whatever changed is the operator's own workflow from here.
package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/alrayyes/linkwarden-obsidian-sync/internal/config"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/linkwarden"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/reconcile"
	"github.com/alrayyes/linkwarden-obsidian-sync/internal/state"
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
		Short: "Keep a directory of Obsidian notes in sync with your saved Linkwarden links",
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

// runSync loads config, fetches every link Linkwarden currently has, and
// reconciles the vault's notes to match — adding, updating and removing
// files as needed. State only advances once reconciliation actually
// succeeds: a failure partway through leaves it untouched, so a retry
// re-processes the same links instead of silently treating them as done.
func runSync(cmd *cobra.Command) error {
	cfg, ranInit, err := loadConfigOrOfferInit(cmd)
	if err != nil {
		return err
	}
	if ranInit {
		return nil
	}

	statePath := filepath.Join(cfg.StateDir, "last-synced.json")
	prevState, err := state.Load(statePath)
	if err != nil {
		return fmt.Errorf("reading state at %s: %w", statePath, err)
	}

	client := linkwarden.NewClient(cfg.LinkwardenURL, cfg.LinkwardenToken)
	links, err := client.FetchAllLinks()
	if err != nil {
		return fmt.Errorf("fetching links: %w", err)
	}

	notesDir := filepath.Join(cfg.VaultPath, cfg.VaultSubdir)
	newState, result, err := reconcile.Notes(notesDir, links, prevState)
	if err != nil {
		return fmt.Errorf("reconciling notes: %w", err)
	}

	if err := state.Save(statePath, newState); err != nil {
		return fmt.Errorf("saving state to %s: %w", statePath, err)
	}

	if result == (reconcile.Result{}) {
		log.Println("nothing changed")
	} else {
		log.Printf("synced %d link(s): %d added, %d updated, %d removed", len(links), result.Added, result.Updated, result.Removed)
	}

	return nil
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

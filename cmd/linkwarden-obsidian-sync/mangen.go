//go:build mangen

// Command linkwarden-obsidian-sync, built with -tags mangen, writes a man
// page per command to the directory named on the command line instead of
// running the sync — a build-time generator, never what release users
// install. See packaging/aur/PKGBUILD and .goreleaser.yaml's before.hooks
// for the two places that invoke it.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra/doc"
)

func main() {
	if len(os.Args) != 2 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: go run -tags mangen ./cmd/linkwarden-obsidian-sync <output-dir>")
		os.Exit(1)
	}
	outDir := os.Args[1]

	if err := os.MkdirAll(outDir, 0o750); err != nil { //nolint:gosec // outDir is this build-time-only tool's own os.Args[1], not untrusted input
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	header := &doc.GenManHeader{
		Title:   "LINKWARDEN-OBSIDIAN-SYNC",
		Section: "1",
		Source:  "linkwarden-obsidian-sync",
		Manual:  "linkwarden-obsidian-sync manual",
	}
	if err := doc.GenManTree(newRootCmd(), header, outDir); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Package vault commits and pushes newly written notes in an Obsidian vault
// checkout.
package vault

import (
	"fmt"
	"os/exec"
	"time"
)

// CommitAndPush stages the vault subdirectory, commits on a dated branch and
// pushes it. It returns the branch name, or "" if there was nothing to
// commit. It shells out to git directly rather than a library — this only
// ever needs to run against the one local clone a human already has
// configured with working push credentials.
func CommitAndPush(vaultPath, subdir string, count int) (string, error) {
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

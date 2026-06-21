package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// IsRepo reports whether the current working directory is inside a Git repository.
func IsRepo() bool {
	return exec.Command("git", "rev-parse", "--git-dir").Run() == nil
}

// StagedDiff returns the output of `git diff --cached`, trimmed of surrounding whitespace.
func StagedDiff() (string, error) {
	out, err := exec.Command("git", "diff", "--cached").Output()
	if err != nil {
		return "", fmt.Errorf("running git diff --cached: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// StageAll stages all changes in the working tree (equivalent to git add -A).
func StageAll() error {
	out, err := exec.Command("git", "add", "-A").CombinedOutput()
	if err != nil {
		return fmt.Errorf("git add -A failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

// Commit creates a commit with the given message.
func Commit(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Stdout = nil
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

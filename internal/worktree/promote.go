package worktree

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/knqyf263/gh-worktree/internal/git"
)

// PromoteToPR promotes a branch worktree to a PR worktree by updating its metadata.
func PromoteToPR(branchName string, prNumber int, prTitle string) error {
	// Get git root for config path
	gitRoot, err := git.GetRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}

	// Set worktree type to "pr"
	if err := git.SetConfig(gitRoot, fmt.Sprintf("branch.%s.gh-worktree-type", branchName), "pr"); err != nil {
		return fmt.Errorf("failed to set worktree type: %w", err)
	}

	// Set PR number
	if err := git.SetConfig(gitRoot, fmt.Sprintf("branch.%s.gh-worktree-pr-number", branchName), strconv.Itoa(prNumber)); err != nil {
		return fmt.Errorf("failed to set PR number: %w", err)
	}

	// Set PR title
	if err := git.SetConfig(gitRoot, fmt.Sprintf("branch.%s.gh-worktree-pr-title", branchName), prTitle); err != nil {
		return fmt.Errorf("failed to set PR title: %w", err)
	}

	return nil
}

// GetWorktreeType returns the type of the worktree for the given branch.
// Returns "pr", "branch", or "" if not set.
func GetWorktreeType(branchName string) (string, error) {
	gitRoot, err := git.GetRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get git root: %w", err)
	}

	worktreeType, err := git.GetConfig(gitRoot, fmt.Sprintf("branch.%s.gh-worktree-type", branchName))
	if err != nil {
		// If config doesn't exist, try to detect from PR number
		prNumber, err := git.GetConfig(gitRoot, fmt.Sprintf("branch.%s.gh-worktree-pr-number", branchName))
		if err == nil && prNumber != "" {
			return "pr", nil
		}
		return "", nil
	}
	return strings.TrimSpace(worktreeType), nil
}

// SetWorktreeType sets the worktree type metadata for a branch.
func SetWorktreeType(branchName string, worktreeType string) error {
	gitRoot, err := git.GetRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}

	return git.SetConfig(gitRoot, fmt.Sprintf("branch.%s.gh-worktree-type", branchName), worktreeType)
}

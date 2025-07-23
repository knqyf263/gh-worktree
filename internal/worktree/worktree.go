package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/knqyf263/gh-worktree/internal/git"
)

// Info represents information about a git worktree
type Info struct {
	Path     string
	Commit   string
	Branch   string
	PRNumber int
	Title    string
}

// List returns all configured worktrees
func List() ([]*Info, error) {
	gitRoot, err := git.GetRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get git root: %w", err)
	}

	cmd := exec.Command("git", "-C", gitRoot, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree list: %w", err)
	}

	var worktrees []*Info
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	var currentWorktree *Info
	for _, line := range lines {
		if line == "" {
			if currentWorktree != nil {
				worktrees = append(worktrees, currentWorktree)
				currentWorktree = nil
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			currentWorktree = &Info{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if strings.HasPrefix(line, "HEAD ") {
			if currentWorktree != nil {
				currentWorktree.Commit = strings.TrimPrefix(line, "HEAD ")
			}
		} else if strings.HasPrefix(line, "branch ") {
			if currentWorktree != nil {
				branchRef := strings.TrimPrefix(line, "branch ")
				// Extract branch name from refs/heads/branch-name
				if strings.HasPrefix(branchRef, "refs/heads/") {
					currentWorktree.Branch = strings.TrimPrefix(branchRef, "refs/heads/")
				} else {
					currentWorktree.Branch = branchRef
				}
			}
		}
	}

	// Add the last worktree if exists
	if currentWorktree != nil {
		worktrees = append(worktrees, currentWorktree)
	}

	return worktrees, nil
}

// ListPRWorktrees returns only PR worktrees
func ListPRWorktrees(repoName string) ([]*Info, error) {
	allWorktrees, err := List()
	if err != nil {
		return nil, err
	}

	gitRoot, err := git.GetRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get git root: %w", err)
	}

	parentDir := filepath.Dir(gitRoot)

	var prWorktrees []*Info
	for _, wt := range allWorktrees {
		// Check if this is a PR worktree based on naming pattern
		if strings.HasPrefix(filepath.Base(wt.Path), repoName+"-pr") {
			// Extract PR number from path
			baseName := filepath.Base(wt.Path)
			prPrefix := repoName + "-pr"
			if len(baseName) > len(prPrefix) {
				prNumberStr := baseName[len(prPrefix):]
				var prNumber int
				if n, err := fmt.Sscanf(prNumberStr, "%d", &prNumber); err == nil && n == 1 {
					wt.PRNumber = prNumber
					// Verify it's in the expected parent directory
					if filepath.Dir(wt.Path) == parentDir {
						// Try to get PR title from git config
						wt.Title = GetPRTitle(wt.Path, wt.Branch)
						prWorktrees = append(prWorktrees, wt)
					}
				}
			}
		}
	}

	return prWorktrees, nil
}

// GetPRTitle retrieves the PR title from git config
func GetPRTitle(worktreePath, branchName string) string {
	if branchName == "" {
		return ""
	}

	title, err := git.GetConfig(worktreePath, fmt.Sprintf("branch.%s.gh-worktree-pr-title", branchName))
	if err != nil {
		return ""
	}
	return title
}

// Remove removes a worktree
func Remove(worktreePath string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, worktreePath)

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// DeleteBranch deletes a git branch
func DeleteBranch(branchName string) error {
	cmd := exec.Command("git", "branch", "-D", branchName)
	return cmd.Run()
}

// GeneratePath generates the path for a PR worktree
func GeneratePath(repoName string, prNumber int) (string, error) {
	gitRoot, err := git.GetRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get git root: %w", err)
	}

	return filepath.Join(filepath.Dir(gitRoot), fmt.Sprintf("%s-pr%d", repoName, prNumber)), nil
}

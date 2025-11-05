package worktree

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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

	// Resolve symlinks in parent directory for comparison
	parentDir := filepath.Dir(gitRoot)
	parentDir, err = filepath.EvalSymlinks(parentDir)
	if err != nil {
		// If EvalSymlinks fails, use the original path
		parentDir = filepath.Dir(gitRoot)
	}

	var prWorktrees []*Info
	for _, wt := range allWorktrees {
		// Skip main worktree
		if wt.Path == gitRoot {
			continue
		}

		// Resolve symlinks in worktree path for comparison
		wtParentDir := filepath.Dir(wt.Path)
		wtParentDir, err = filepath.EvalSymlinks(wtParentDir)
		if err != nil {
			// If EvalSymlinks fails, use the original path
			wtParentDir = filepath.Dir(wt.Path)
		}

		// Skip if not in parent directory
		if wtParentDir != parentDir {
			continue
		}

		baseName := filepath.Base(wt.Path)

		// Check if this is a PR worktree based on naming pattern (repo-pr###)
		isPRByName := false
		if strings.HasPrefix(baseName, repoName+"-pr") {
			prPrefix := repoName + "-pr"
			if len(baseName) > len(prPrefix) {
				prNumberStr := baseName[len(prPrefix):]
				var prNumber int
				if n, err := fmt.Sscanf(prNumberStr, "%d", &prNumber); err == nil && n == 1 {
					wt.PRNumber = prNumber
					isPRByName = true
				}
			}
		}

		// Also check metadata for worktree type
		worktreeType, _ := GetWorktreeType(wt.Branch)
		isPRByMetadata := worktreeType == "pr"

		// Include if it's a PR worktree by either naming or metadata
		if isPRByName || isPRByMetadata {
			// Get PR title from git config
			wt.Title = GetPRTitle(wt.Path, wt.Branch)
			
			// If PR number not set yet, try to get it from git config
			if wt.PRNumber == 0 {
				prNumberStr, err := git.GetConfig(gitRoot, fmt.Sprintf("branch.%s.gh-worktree-pr-number", wt.Branch))
				if err == nil && prNumberStr != "" {
					if prNum, err := strconv.Atoi(strings.TrimSpace(prNumberStr)); err == nil {
						wt.PRNumber = prNum
					}
				}
			}
			
			prWorktrees = append(prWorktrees, wt)
		}
	}

	return prWorktrees, nil
}

// ListBranchWorktrees lists all branch worktrees (non-PR worktrees).
func ListBranchWorktrees(repoName string) ([]*Info, error) {
	allWorktrees, err := List()
	if err != nil {
		return nil, err
	}

	gitRoot, err := git.GetRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get git root: %w", err)
	}

	// Resolve symlinks in parent directory for comparison
	parentDir := filepath.Dir(gitRoot)
	parentDir, err = filepath.EvalSymlinks(parentDir)
	if err != nil {
		// If EvalSymlinks fails, use the original path
		parentDir = filepath.Dir(gitRoot)
	}

	var branchWorktrees []*Info
	for _, wt := range allWorktrees {
		// Skip main worktree
		if wt.Path == gitRoot {
			continue
		}

		baseName := filepath.Base(wt.Path)
		
		// Check if it's NOT a PR worktree (doesn't match repo-pr### pattern)
		if strings.HasPrefix(baseName, repoName+"-pr") {
			// Check if it's actually a PR worktree
			prPrefix := repoName + "-pr"
			if len(baseName) > len(prPrefix) {
				prNumberStr := baseName[len(prPrefix):]
				var prNumber int
				if n, err := fmt.Sscanf(prNumberStr, "%d", &prNumber); err == nil && n == 1 {
					// This is a PR worktree, skip it
					continue
				}
			}
		}

		// Resolve symlinks in worktree path for comparison
		wtParentDir := filepath.Dir(wt.Path)
		wtParentDir, err = filepath.EvalSymlinks(wtParentDir)
		if err != nil {
			// If EvalSymlinks fails, use the original path
			wtParentDir = filepath.Dir(wt.Path)
		}

		// Check if it starts with repo name and is in parent directory
		if strings.HasPrefix(baseName, repoName+"-") && wtParentDir == parentDir {
			// Check worktree type from git config
			worktreeType, _ := GetWorktreeType(wt.Branch)
			if worktreeType == "branch" || worktreeType == "" {
				branchWorktrees = append(branchWorktrees, wt)
			}
		}
	}

	return branchWorktrees, nil
}

// ListAllWorktrees lists all worktrees (PR and branch worktrees).
func ListAllWorktrees(repoName string) (prWorktrees []*Info, branchWorktrees []*Info, err error) {
	prWorktrees, err = ListPRWorktrees(repoName)
	if err != nil {
		return nil, nil, err
	}

	branchWorktrees, err = ListBranchWorktrees(repoName)
	if err != nil {
		return nil, nil, err
	}

	return prWorktrees, branchWorktrees, nil
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

// GeneratePathForBranch generates the path for a branch worktree.
// Format: ../repo-name-{branch-name}
// Slashes in branch names are replaced with dashes to avoid creating nested directories.
func GeneratePathForBranch(repoName string, branchName string) (string, error) {
	gitRoot, err := git.GetRoot()
	if err != nil {
		return "", fmt.Errorf("failed to get git root: %w", err)
	}

	// Replace slashes with dashes to avoid creating nested directories
	sanitizedBranchName := strings.ReplaceAll(branchName, "/", "-")

	return filepath.Join(filepath.Dir(gitRoot), fmt.Sprintf("%s-%s", repoName, sanitizedBranchName)), nil
}

// DetectWorktreeType detects the type of worktree based on its path.
// Returns "pr", "branch", or "main".
func DetectWorktreeType(path string) string {
	// Check if it's the main worktree by looking at git config
	gitRoot, err := git.GetRoot()
	if err == nil && path == gitRoot {
		return "main"
	}

	// Extract the last component of the path
	baseName := filepath.Base(path)
	
	// Check if it matches PR pattern: repo-pr123
	if strings.Contains(baseName, "-pr") && len(strings.Split(baseName, "-pr")) == 2 {
		prPart := strings.Split(baseName, "-pr")[1]
		if _, err := strconv.Atoi(prPart); err == nil {
			return "pr"
		}
	}
	
	return "branch"
}

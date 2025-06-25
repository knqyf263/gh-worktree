package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Remote represents a git remote
type Remote struct {
	Name string
	URL  string
}

// GetRemotes returns all configured git remotes
func GetRemotes() ([]*Remote, error) {
	cmd := exec.Command("git", "remote", "-v")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var remotes []*Remote
	seen := make(map[string]bool)
	
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if strings.Contains(line, "(fetch)") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[0]
				url := parts[1]
				if !seen[name] {
					remotes = append(remotes, &Remote{Name: name, URL: url})
					seen[name] = true
				}
			}
		}
	}

	return remotes, nil
}

// GetRoot returns the root directory of the main git repository
func GetRoot() (string, error) {
	// Get the main repository root by finding the git common directory
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git common dir: %w", err)
	}
	
	gitCommonDir := strings.TrimSpace(string(output))
	
	// If it's an absolute path, get its parent
	if filepath.IsAbs(gitCommonDir) {
		return filepath.Dir(gitCommonDir), nil
	}
	
	// If it's a relative path, resolve it from current directory
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}
	
	absGitDir := filepath.Join(currentDir, gitCommonDir)
	return filepath.Dir(absGitDir), nil
}

// BranchExists checks if a local branch exists
func BranchExists(branchName string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", fmt.Sprintf("refs/heads/%s", branchName))
	return cmd.Run() == nil
}

// GetBranchName returns the current branch name at the given path
func GetBranchName(worktreePath string) string {
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// ExecuteCommands runs a series of git commands
func ExecuteCommands(cmdQueue [][]string) error {
	for _, args := range cmdQueue {
		cmd := exec.Command("git", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to execute git %s: %w", strings.Join(args, " "), err)
		}
	}
	return nil
}

// GetConfig gets a git config value from a specific path
func GetConfig(path, key string) (string, error) {
	cmd := exec.Command("git", "-C", path, "config", "--local", key)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// SetConfig sets a git config value at a specific path
func SetConfig(path, key, value string) error {
	cmd := exec.Command("git", "-C", path, "config", key, value)
	return cmd.Run()
}
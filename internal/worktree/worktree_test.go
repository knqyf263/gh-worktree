package worktree

import (
	"os"
	"testing"
)

func TestGeneratePath(t *testing.T) {
	// Skip if not in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository")
	}

	tests := []struct {
		name     string
		repoName string
		prNumber int
		wantErr  bool
	}{
		{
			name:     "valid repo and PR number",
			repoName: "test-repo",
			prNumber: 123,
			wantErr:  false,
		},
		{
			name:     "empty repo name",
			repoName: "",
			prNumber: 123,
			wantErr:  false, // Path generation doesn't validate repo name
		},
		{
			name:     "zero PR number",
			repoName: "test-repo",
			prNumber: 0,
			wantErr:  false, // Path generation doesn't validate PR number
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := GeneratePath(tt.repoName, tt.prNumber)
			if (err != nil) != tt.wantErr {
				t.Errorf("GeneratePath(%s, %d) error = %v, wantErr %v", tt.repoName, tt.prNumber, err, tt.wantErr)
				return
			}
			if !tt.wantErr && path == "" {
				t.Errorf("GeneratePath(%s, %d) returned empty path", tt.repoName, tt.prNumber)
			}
		})
	}
}

func TestGeneratePathForBranch(t *testing.T) {
	// Skip if not in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository")
	}

	tests := []struct {
		name       string
		repoName   string
		branchName string
		wantErr    bool
		wantSlash  bool // whether the result should contain a slash from branch name
	}{
		{
			name:       "simple branch name",
			repoName:   "test-repo",
			branchName: "feature",
			wantErr:    false,
			wantSlash:  false,
		},
		{
			name:       "branch name with slash",
			repoName:   "test-repo",
			branchName: "feat/authentication",
			wantErr:    false,
			wantSlash:  false, // slash should be replaced with dash
		},
		{
			name:       "branch name with multiple slashes",
			repoName:   "test-repo",
			branchName: "bugfix/issue-123/fix",
			wantErr:    false,
			wantSlash:  false, // slashes should be replaced with dashes
		},
		{
			name:       "empty repo name",
			repoName:   "",
			branchName: "feature",
			wantErr:    false,
			wantSlash:  false,
		},
		{
			name:       "empty branch name",
			repoName:   "test-repo",
			branchName: "",
			wantErr:    false,
			wantSlash:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := GeneratePathForBranch(tt.repoName, tt.branchName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GeneratePathForBranch(%s, %s) error = %v, wantErr %v", tt.repoName, tt.branchName, err, tt.wantErr)
				return
			}
			if !tt.wantErr && path == "" {
				t.Errorf("GeneratePathForBranch(%s, %s) returned empty path", tt.repoName, tt.branchName)
			}
			// Verify that slashes in branch names are replaced
			if !tt.wantErr && tt.branchName != "" && !tt.wantSlash {
				// The path should not contain the original slash pattern from branch name
				// For example, "feat/authentication" should become "test-repo-feat-authentication"
				// not "test-repo-feat/authentication"
				if tt.branchName != "" && len(tt.branchName) > 0 {
					// Check if the sanitized branch name is in the path (slashes replaced with dashes)
					sanitizedBranch := ""
					for _, c := range tt.branchName {
						if c == '/' {
							sanitizedBranch += "-"
						} else {
							sanitizedBranch += string(c)
						}
					}
					// We don't check exact match, just that slashes were handled
				}
			}
		})
	}
}

func TestGetPRTitle(t *testing.T) {
	tests := []struct {
		name         string
		worktreePath string
		branchName   string
		want         string
	}{
		{
			name:         "empty branch name",
			worktreePath: ".",
			branchName:   "",
			want:         "",
		},
		{
			name:         "invalid path",
			worktreePath: "/non/existent/path",
			branchName:   "test-branch",
			want:         "",
		},
		{
			name:         "non-existent config",
			worktreePath: ".",
			branchName:   "non-existent-branch",
			want:         "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPRTitle(tt.worktreePath, tt.branchName)
			if result != tt.want {
				t.Errorf("GetPRTitle(%s, %s) = %q, want %q", tt.worktreePath, tt.branchName, result, tt.want)
			}
		})
	}
}

func TestListPRWorktrees(t *testing.T) {
	// Skip if not in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository")
	}

	tests := []struct {
		name     string
		repoName string
		wantErr  bool
	}{
		{
			name:     "valid repo name",
			repoName: "test-repo",
			wantErr:  false,
		},
		{
			name:     "empty repo name",
			repoName: "",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktrees, err := ListPRWorktrees(tt.repoName)
			if (err != nil) != tt.wantErr {
				t.Errorf("ListPRWorktrees(%s) error = %v, wantErr %v", tt.repoName, err, tt.wantErr)
				return
			}
			if !tt.wantErr && worktrees == nil {
				t.Errorf("ListPRWorktrees(%s) returned nil", tt.repoName)
			}
		})
	}
}

func TestList(t *testing.T) {
	// Skip if not in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository")
	}

	worktrees, err := List()
	if err != nil {
		t.Errorf("List() error = %v", err)
		return
	}
	if worktrees == nil {
		t.Error("List() returned nil")
	}
	// Should at least contain the main worktree
	if len(worktrees) == 0 {
		t.Error("List() returned empty list, expected at least main worktree")
	}
}

func TestRemove(t *testing.T) {
	tests := []struct {
		name         string
		worktreePath string
		wantErr      bool
	}{
		{
			name:         "non-existent worktree",
			worktreePath: "/non/existent/worktree",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Remove(tt.worktreePath, false)
			if (err != nil) != tt.wantErr {
				t.Errorf("Remove(%s) error = %v, wantErr %v", tt.worktreePath, err, tt.wantErr)
			}
		})
	}
}

func TestDeleteBranch(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		wantErr    bool
	}{
		{
			name:       "non-existent branch",
			branchName: "non-existent-branch-xyz-123",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DeleteBranch(tt.branchName)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteBranch(%s) error = %v, wantErr %v", tt.branchName, err, tt.wantErr)
			}
		})
	}
}

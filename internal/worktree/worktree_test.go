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
			err := Remove(tt.worktreePath)
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
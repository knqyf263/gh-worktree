package git

import (
	"os"
	"testing"
)

func TestGetBranchName(t *testing.T) {
	// Skip if not in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository")
	}

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "current directory",
			path:    ".",
			wantErr: false,
		},
		{
			name:    "non-existent directory",
			path:    "/non/existent/path",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBranchName(tt.path)
			if tt.wantErr && result != "" {
				t.Errorf("GetBranchName(%s) expected empty result for error case, got %s", tt.path, result)
			}
			if !tt.wantErr && result == "" {
				t.Errorf("GetBranchName(%s) expected non-empty result, got empty", tt.path)
			}
		})
	}
}

func TestBranchExists(t *testing.T) {
	// Skip if not in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository")
	}

	tests := []struct {
		name       string
		branchName string
		want       bool
	}{
		{
			name:       "non-existent branch",
			branchName: "non-existent-branch-xyz-123",
			want:       false,
		},
		{
			name:       "invalid branch name",
			branchName: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BranchExists(tt.branchName)
			if result != tt.want {
				t.Errorf("BranchExists(%s) = %v, want %v", tt.branchName, result, tt.want)
			}
		})
	}
}

func TestGetConfig(t *testing.T) {
	// Skip if not in a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository")
	}

	tests := []struct {
		name    string
		path    string
		key     string
		wantErr bool
	}{
		{
			name:    "non-existent config key",
			path:    ".",
			key:     "non.existent.key",
			wantErr: true,
		},
		{
			name:    "invalid path",
			path:    "/non/existent/path",
			key:     "user.name",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetConfig(tt.path, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetConfig(%s, %s) error = %v, wantErr %v", tt.path, tt.key, err, tt.wantErr)
			}
		})
	}
}

func TestSetConfig(t *testing.T) {
	// Skip if not in a git repository or if we can't write
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		t.Skip("Not in a git repository")
	}

	tests := []struct {
		name    string
		path    string
		key     string
		value   string
		wantErr bool
	}{
		{
			name:    "invalid path",
			path:    "/non/existent/path",
			key:     "test.key",
			value:   "test-value",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SetConfig(tt.path, tt.key, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("SetConfig(%s, %s, %s) error = %v, wantErr %v", tt.path, tt.key, tt.value, err, tt.wantErr)
			}
		})
	}
}

func TestExecuteCommands(t *testing.T) {
	tests := []struct {
		name     string
		commands [][]string
		wantErr  bool
	}{
		{
			name:     "empty command queue",
			commands: [][]string{},
			wantErr:  false,
		},
		{
			name: "invalid git command",
			commands: [][]string{
				{"invalid-command"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ExecuteCommands(tt.commands)
			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCommands() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

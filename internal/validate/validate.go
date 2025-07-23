package validate

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	// validBranchName matches valid git branch names
	validBranchName = regexp.MustCompile(`^[a-zA-Z0-9._/-]+$`)
	// validRepoName matches valid repository names
	validRepoName = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)
)

// SanitizeForGitConfig removes or escapes dangerous characters for git config values
func SanitizeForGitConfig(value string) string {
	// Remove null bytes, newlines, and other control characters
	value = strings.ReplaceAll(value, "\x00", "")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\t", " ")
	// Remove shell special characters that could be dangerous
	for _, char := range []string{";", "&", "|", "`", "$", "(", ")", "<", ">", "'", "\"", "\\"} {
		value = strings.ReplaceAll(value, char, "")
	}
	return strings.TrimSpace(value)
}

// BranchName checks if branch name is safe for use in commands
func BranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	if len(name) > 255 {
		return fmt.Errorf("branch name too long")
	}
	if !validBranchName.MatchString(name) {
		return fmt.Errorf("invalid branch name: contains unsafe characters")
	}
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("invalid branch name format")
	}
	return nil
}

// RepoName checks if repository name is safe for path construction
func RepoName(name string) error {
	if name == "" {
		return fmt.Errorf("repository name cannot be empty")
	}
	if len(name) > 100 {
		return fmt.Errorf("repository name too long")
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("repository name contains path traversal")
	}
	if !validRepoName.MatchString(name) {
		return fmt.Errorf("invalid repository name: contains unsafe characters")
	}
	return nil
}

// PRNumber checks if PR number is valid
func PRNumber(prNumber int) error {
	if prNumber <= 0 || prNumber > 999999 {
		return fmt.Errorf("invalid PR number: %d", prNumber)
	}
	return nil
}

// URL checks if URL is safe GitHub URL
func URL(urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsedURL.Scheme != "https" {
		return fmt.Errorf("only HTTPS URLs are allowed")
	}

	// Check for credentials in URL
	if parsedURL.User != nil {
		return fmt.Errorf("URL cannot contain credentials")
	}

	if parsedURL.Host != "github.com" {
		return fmt.Errorf("only github.com URLs are allowed")
	}

	return nil
}

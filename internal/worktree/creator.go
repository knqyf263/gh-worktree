package worktree

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/knqyf263/gh-worktree/internal/git"
	"github.com/knqyf263/gh-worktree/internal/github"
	"github.com/knqyf263/gh-worktree/internal/validate"
)

// CheckoutOptions represents options for creating a worktree
type CheckoutOptions struct {
	RecurseSubmodules bool
	Force             bool
	Detach            bool
	BranchName        string
}

// Creator handles worktree creation logic
type Creator struct {
	remotes []*git.Remote
	repo    repository.Repository
}

// NewCreator creates a new worktree creator
func NewCreator(repo repository.Repository) (*Creator, error) {
	remotes, err := git.GetRemotes()
	if err != nil {
		return nil, fmt.Errorf("failed to get remotes: %w", err)
	}

	return &Creator{
		remotes: remotes,
		repo:    repo,
	}, nil
}

// Create creates a new worktree for the given PR
func (c *Creator) Create(worktreePath string, pr *github.PullRequest, opts *CheckoutOptions) error {
	// Find base remote (origin or upstream)
	baseRemote := c.findBaseRemote()
	if baseRemote == nil {
		return fmt.Errorf("no suitable remote found")
	}

	// Determine if we have a head remote
	headRemote := baseRemote
	isCrossRepo := pr.Head.Repo.Owner.Login != c.repo.Owner
	if isCrossRepo {
		headRemote = c.findHeadRemote(pr)
	}

	branchName := pr.Head.Ref
	if opts.BranchName != "" {
		branchName = opts.BranchName
	}

	var cmdQueue [][]string

	if headRemote != nil {
		cmds, err := c.cmdsForExistingRemote(headRemote, pr, opts, worktreePath, branchName)
		if err != nil {
			return fmt.Errorf("failed to create commands for existing remote: %w", err)
		}
		cmdQueue = append(cmdQueue, cmds...)
	} else {
		cmds, err := c.cmdsForMissingRemote(pr, baseRemote, opts, worktreePath, branchName)
		if err != nil {
			return fmt.Errorf("failed to create commands for missing remote: %w", err)
		}
		cmdQueue = append(cmdQueue, cmds...)
	}

	if opts.RecurseSubmodules {
		cmdQueue = append(cmdQueue, []string{"submodule", "sync", "--recursive"})
		cmdQueue = append(cmdQueue, []string{"submodule", "update", "--init", "--recursive"})
	}

	err := git.ExecuteCommands(cmdQueue)
	if err != nil {
		return err
	}

	// Store PR metadata in worktree git config
	err = c.storePRMetadata(worktreePath, pr)
	if err != nil {
		return fmt.Errorf("failed to store PR metadata: %w", err)
	}

	return nil
}

func (c *Creator) findBaseRemote() *git.Remote {
	// Prefer upstream remote if it exists
	for _, remote := range c.remotes {
		if remote.Name == "upstream" {
			return remote
		}
	}
	
	// Fall back to origin if upstream doesn't exist
	for _, remote := range c.remotes {
		if remote.Name == "origin" {
			return remote
		}
	}
	
	// If neither upstream nor origin exist, use the first remote
	if len(c.remotes) > 0 {
		return c.remotes[0]
	}
	
	return nil
}

func (c *Creator) findHeadRemote(pr *github.PullRequest) *git.Remote {
	headRepoName := pr.Head.Repo.Name
	headOwner := pr.Head.Repo.Owner.Login
	
	for _, remote := range c.remotes {
		if strings.Contains(remote.URL, headOwner) && strings.Contains(remote.URL, headRepoName) {
			return remote
		}
	}
	
	return nil
}

func (c *Creator) cmdsForExistingRemote(remote *git.Remote, pr *github.PullRequest, opts *CheckoutOptions, worktreePath, branchName string) ([][]string, error) {
	// Validate inputs
	if err := validate.BranchName(pr.Head.Ref); err != nil {
		return nil, fmt.Errorf("invalid head ref: %w", err)
	}
	if err := validate.BranchName(branchName); err != nil {
		return nil, fmt.Errorf("invalid branch name: %w", err)
	}
	
	var cmds [][]string
	remoteBranch := fmt.Sprintf("%s/%s", remote.Name, pr.Head.Ref)

	refSpec := fmt.Sprintf("+refs/heads/%s:refs/remotes/%s", pr.Head.Ref, remoteBranch)
	if opts.Detach {
		refSpec = fmt.Sprintf("+refs/heads/%s", pr.Head.Ref)
	}

	cmds = append(cmds, []string{"fetch", remote.Name, refSpec, "--no-tags"})

	if opts.Detach {
		cmds = append(cmds, []string{"worktree", "add", "--detach", worktreePath, "FETCH_HEAD"})
	} else {
		if git.BranchExists(branchName) {
			if opts.Force {
				cmds = append(cmds, []string{"worktree", "add", "--force", worktreePath, branchName})
				cmds = append(cmds, []string{"-C", worktreePath, "reset", "--hard", fmt.Sprintf("refs/remotes/%s", remoteBranch)})
			} else {
				cmds = append(cmds, []string{"worktree", "add", worktreePath, branchName})
				cmds = append(cmds, []string{"-C", worktreePath, "merge", "--ff-only", fmt.Sprintf("refs/remotes/%s", remoteBranch)})
			}
		} else {
			cmds = append(cmds, []string{"worktree", "add", "-b", branchName, worktreePath, remoteBranch})
			// Set up tracking after creating the worktree
			cmds = append(cmds, []string{"-C", worktreePath, "config", fmt.Sprintf("branch.%s.remote", branchName), remote.Name})
			cmds = append(cmds, []string{"-C", worktreePath, "config", fmt.Sprintf("branch.%s.merge", branchName), fmt.Sprintf("refs/heads/%s", pr.Head.Ref)})
		}
	}

	return cmds, nil
}

func (c *Creator) cmdsForMissingRemote(pr *github.PullRequest, baseRemote *git.Remote, opts *CheckoutOptions, worktreePath, branchName string) ([][]string, error) {
	// Validate inputs
	if err := validate.PRNumber(pr.Number); err != nil {
		return nil, fmt.Errorf("invalid PR number: %w", err)
	}
	if err := validate.BranchName(branchName); err != nil {
		return nil, fmt.Errorf("invalid branch name: %w", err)
	}
	if err := validate.BranchName(pr.Head.Ref); err != nil {
		return nil, fmt.Errorf("invalid head ref: %w", err)
	}
	
	var cmds [][]string
	ref := fmt.Sprintf("refs/pull/%d/head", pr.Number)

	if opts.Detach {
		cmds = append(cmds, []string{"fetch", baseRemote.Name, ref, "--no-tags"})
		cmds = append(cmds, []string{"worktree", "add", "--detach", worktreePath, "FETCH_HEAD"})
		return cmds, nil
	}

	fetchCmd := []string{"fetch", baseRemote.Name, fmt.Sprintf("%s:%s", ref, branchName), "--no-tags"}
	if opts.Force {
		fetchCmd = append(fetchCmd, "--force")
	}
	cmds = append(cmds, fetchCmd)
	
	cmds = append(cmds, []string{"worktree", "add", worktreePath, branchName})

	// Configure remote settings for the new worktree
	remoteName := baseRemote.Name
	mergeRef := ref
	if pr.MaintainerCanModify && pr.Head.Repo.Name != "" {
		// Validate GitHub URL components before constructing URL
		if err := validate.RepoName(pr.Head.Repo.Name); err != nil {
			return nil, fmt.Errorf("invalid head repo name: %w", err)
		}
		if err := validate.RepoName(pr.Head.Repo.Owner.Login); err != nil {
			return nil, fmt.Errorf("invalid head repo owner: %w", err)
		}
		
		// If maintainer can modify, set up for push to head repository
		pushRemote := fmt.Sprintf("https://github.com/%s/%s", pr.Head.Repo.Owner.Login, pr.Head.Repo.Name)
		if err := validate.URL(pushRemote); err != nil {
			return nil, fmt.Errorf("invalid push remote URL: %w", err)
		}
		mergeRef = fmt.Sprintf("refs/heads/%s", pr.Head.Ref)
		cmds = append(cmds, []string{"-C", worktreePath, "config", fmt.Sprintf("branch.%s.pushRemote", branchName), pushRemote})
	}

	cmds = append(cmds, []string{"-C", worktreePath, "config", fmt.Sprintf("branch.%s.remote", branchName), remoteName})
	cmds = append(cmds, []string{"-C", worktreePath, "config", fmt.Sprintf("branch.%s.merge", branchName), mergeRef})

	return cmds, nil
}

func (c *Creator) storePRMetadata(worktreePath string, pr *github.PullRequest) error {
	// Validate and sanitize inputs
	if err := validate.BranchName(pr.Head.Ref); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}
	
	branchName := pr.Head.Ref
	sanitizedTitle := validate.SanitizeForGitConfig(pr.Title)
	
	// Validate PR number
	if err := validate.PRNumber(pr.Number); err != nil {
		return fmt.Errorf("invalid PR number: %w", err)
	}
	
	// Set PR metadata
	err := git.SetConfig(worktreePath, fmt.Sprintf("branch.%s.gh-worktree-pr-number", branchName), strconv.Itoa(pr.Number))
	if err != nil {
		return fmt.Errorf("failed to set PR number config: %w", err)
	}
	
	err = git.SetConfig(worktreePath, fmt.Sprintf("branch.%s.gh-worktree-pr-title", branchName), sanitizedTitle)
	if err != nil {
		return fmt.Errorf("failed to set PR title config: %w", err)
	}

	return nil
}
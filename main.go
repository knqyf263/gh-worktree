package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/prompter"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/spf13/cobra"
)

type CheckoutOptions struct {
	HttpClient        func() (*http.Client, error)
	RecurseSubmodules bool
	Force             bool
	Detach            bool
	BranchName        string
}

type WorktreeInfo struct {
	Path      string
	Commit    string
	Branch    string
	PRNumber  int
	Title     string
}

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

type Remote struct {
	Name string
	URL  string
}

func main() {
	var opts CheckoutOptions

	rootCmd := &cobra.Command{
		Use:   "gh-worktree",
		Short: "A gh extension for git worktree operations",
	}

	prCmd := &cobra.Command{
		Use:   "pr",
		Short: "Operations on pull requests",
	}

	checkoutCmd := &cobra.Command{
		Use:   "checkout [<number> | <url> | <branch>]",
		Short: "Check out a pull request in a new git worktree",
		Example: `  # Interactively select a PR to check out
  $ gh worktree pr checkout

  # Check out a specific PR in a new worktree
  $ gh worktree pr checkout 32

  # Check out PR from URL in a new worktree  
  $ gh worktree pr checkout https://github.com/OWNER/REPO/pull/32`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.HttpClient = func() (*http.Client, error) {
				return api.DefaultHTTPClient()
			}
			if len(args) > 0 {
				return checkoutRun(&opts, args[0])
			}
			return checkoutRunInteractive(&opts)
		},
	}

	checkoutCmd.Flags().BoolVarP(&opts.RecurseSubmodules, "recurse-submodules", "", false, "Update all submodules after checkout")
	checkoutCmd.Flags().BoolVarP(&opts.Force, "force", "f", false, "Reset the existing local branch to the latest state of the pull request")
	checkoutCmd.Flags().BoolVarP(&opts.Detach, "detach", "", false, "Checkout PR with a detached HEAD")
	checkoutCmd.Flags().StringVarP(&opts.BranchName, "branch", "b", "", "Local branch name to use (default [the name of the head branch])")

	removeCmd := &cobra.Command{
		Use:   "remove [<number> | <url> | <branch>]",
		Short: "Remove a pull request worktree",
		Example: `  # Interactively select a worktree to remove
  $ gh worktree pr remove

  # Remove a specific PR worktree
  $ gh worktree pr remove 32

  # Remove PR worktree from URL  
  $ gh worktree pr remove https://github.com/OWNER/REPO/pull/32`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return removeRun(args[0])
			}
			return removeRunInteractive()
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List pull request worktrees",
		Example: `  # List all PR worktrees
  $ gh worktree pr list`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listRun()
		},
	}

	switchCmd := &cobra.Command{
		Use:   "switch [<number>]",
		Short: "Switch to an existing pull request worktree",
		Example: `  # Interactively select a worktree to switch to
  $ gh worktree pr switch
  
  # Switch to specific PR worktree
  $ gh worktree pr switch 9060
  
  # Use as shell function (add to ~/.bashrc or ~/.zshrc):
  $ ghws() { 
      local target=$(gh worktree pr switch --shell "$@")
      [ -n "$target" ] && cd "$target"
    }
  $ ghws     # interactive selection
  $ ghws 9060  # switch to specific PR`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shellMode, _ := cmd.Flags().GetBool("shell")
			prNumber := ""
			if len(args) > 0 {
				prNumber = args[0]
			}
			return switchRun(shellMode, prNumber)
		},
	}
	
	switchCmd.Flags().BoolP("shell", "s", false, "Output path only for use in shell functions")

	prCmd.AddCommand(checkoutCmd)
	prCmd.AddCommand(removeCmd)
	prCmd.AddCommand(listCmd)
	prCmd.AddCommand(switchCmd)
	rootCmd.AddCommand(prCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func checkoutRunInteractive(opts *CheckoutOptions) error {
	// Get current repository
	repo, err := repository.Current()
	if err != nil {
		return fmt.Errorf("failed to get current repository: %w", err)
	}

	// Get PRs from API
	client, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("failed to create REST client: %w", err)
	}
	
	var prs []PullRequest
	err = client.Get(fmt.Sprintf("repos/%s/%s/pulls?state=open&per_page=100", repo.Owner, repo.Name), &prs)
	if err != nil {
		return fmt.Errorf("failed to get PRs: %w", err)
	}

	if len(prs) == 0 {
		fmt.Println("No open pull requests found.")
		return nil
	}

	// Create candidates list in the same format as gh CLI
	candidates := []string{}
	for _, pr := range prs {
		candidates = append(candidates, fmt.Sprintf("#%d\t%s\t%s", 
			pr.Number, 
			pr.Head.Ref, 
			pr.Head.Repo.Owner.Login+"/"+pr.Head.Repo.Name))
	}

	// Use gh CLI's built-in selection
	selection, err := promptSelect("Select a pull request to check out", candidates)
	if err != nil {
		return err
	}

	if selection == -1 {
		fmt.Println("Cancelled.")
		return nil
	}

	selectedPR := prs[selection]
	
	// Generate worktree path based on git repository root
	gitRoot, err := getGitRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}
	
	repoName := filepath.Base(gitRoot)
	worktreePath := filepath.Join(filepath.Dir(gitRoot), fmt.Sprintf("%s-pr%d", repoName, selectedPR.Number))

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("worktree for PR #%d already exists at %s", selectedPR.Number, worktreePath)
	}

	// Create worktree
	err = createWorktree(worktreePath, &selectedPR, opts)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	fmt.Printf("Created worktree for #%d at %s\n", selectedPR.Number, worktreePath)
	if selectedPR.Title != "" {
		fmt.Printf("Title: %s\n", selectedPR.Title)
	}
	return nil
}

func checkoutRun(opts *CheckoutOptions, selector string) error {
	// Get current repository
	repo, err := repository.Current()
	if err != nil {
		return fmt.Errorf("failed to get current repository: %w", err)
	}

	// Parse PR number from selector
	prNumber, err := parsePRNumber(selector)
	if err != nil {
		return fmt.Errorf("failed to parse PR number: %w", err)
	}

	// Get PR details
	client, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("failed to create REST client: %w", err)
	}
	
	var pr PullRequest
	err = client.Get(fmt.Sprintf("repos/%s/%s/pulls/%d", repo.Owner, repo.Name, prNumber), &pr)
	if err != nil {
		return fmt.Errorf("failed to get PR details: %w", err)
	}


	// Generate worktree path based on git repository root
	gitRoot, err := getGitRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}
	
	repoName := filepath.Base(gitRoot)
	worktreePath := filepath.Join(filepath.Dir(gitRoot), fmt.Sprintf("%s-pr%d", repoName, prNumber))

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("worktree for PR #%d already exists at %s", prNumber, worktreePath)
	}

	// Create worktree
	err = createWorktree(worktreePath, &pr, opts)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	fmt.Printf("Created worktree for #%d at %s\n", prNumber, worktreePath)
	if pr.Title != "" {
		fmt.Printf("Title: %s\n", pr.Title)
	}
	return nil
}

func removeRun(selector string) error {
	// Parse PR number from selector
	prNumber, err := parsePRNumber(selector)
	if err != nil {
		return fmt.Errorf("failed to parse PR number: %w", err)
	}

	// Generate worktree path based on git repository root
	gitRoot, err := getGitRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}
	
	repoName := filepath.Base(gitRoot)
	worktreePath := filepath.Join(filepath.Dir(gitRoot), fmt.Sprintf("%s-pr%d", repoName, prNumber))

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree for PR #%d does not exist at %s", prNumber, worktreePath)
	}

	// Get branch name before removing worktree
	branchName := getBranchName(worktreePath)
	
	// Get PR title from git config before removing
	title := ""
	if branchName != "" {
		title = getPRTitle(worktreePath, branchName)
	}

	// Remove the worktree
	cmd := exec.Command("git", "worktree", "remove", worktreePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// Delete the branch (this also removes branch-specific metadata)
	if branchName != "" {
		cmd = exec.Command("git", "branch", "-D", branchName)
		if err := cmd.Run(); err != nil {
			// Ignore error as branch might not exist or be checked out elsewhere
			fmt.Fprintf(os.Stderr, "Warning: failed to delete branch %s: %v\n", branchName, err)
		}
	}

	fmt.Printf("Removed worktree for #%d at %s\n", prNumber, worktreePath)
	if title != "" {
		fmt.Printf("Title: %s\n", title)
	}
	return nil
}

func parsePRNumber(selector string) (int, error) {
	// Handle URL format: https://github.com/OWNER/REPO/pull/NUMBER
	if strings.Contains(selector, "/pull/") {
		parts := strings.Split(selector, "/pull/")
		if len(parts) == 2 {
			var prNumber int
			_, err := fmt.Sscanf(parts[1], "%d", &prNumber)
			if err != nil {
				return 0, fmt.Errorf("invalid PR URL format")
			}
			return prNumber, nil
		}
	}

	// Handle direct number
	var prNumber int
	_, err := fmt.Sscanf(selector, "%d", &prNumber)
	if err != nil {
		return 0, fmt.Errorf("invalid PR number format")
	}
	return prNumber, nil
}

func createWorktree(worktreePath string, pr *PullRequest, opts *CheckoutOptions) error {
	// Get current repository info
	repo, err := repository.Current()
	if err != nil {
		return fmt.Errorf("failed to get current repository: %w", err)
	}

	// Get remotes
	remotes, err := getRemotes()
	if err != nil {
		return fmt.Errorf("failed to get remotes: %w", err)
	}

	// Find base remote (origin or upstream)
	baseRemote := findBaseRemote(remotes, repo)
	if baseRemote == nil {
		return fmt.Errorf("no suitable remote found")
	}

	// Determine if we have a head remote
	headRemote := baseRemote
	isCrossRepo := pr.Head.Repo.Owner.Login != repo.Owner
	if isCrossRepo {
		headRemote = findHeadRemote(remotes, pr)
	}

	branchName := pr.Head.Ref
	if opts.BranchName != "" {
		branchName = opts.BranchName
	}

	var cmdQueue [][]string

	if headRemote != nil {
		cmdQueue = append(cmdQueue, cmdsForExistingRemote(headRemote, pr, opts, worktreePath, branchName)...)
	} else {
		cmdQueue = append(cmdQueue, cmdsForMissingRemote(pr, baseRemote, repo, opts, worktreePath, branchName)...)
	}

	if opts.RecurseSubmodules {
		cmdQueue = append(cmdQueue, []string{"submodule", "sync", "--recursive"})
		cmdQueue = append(cmdQueue, []string{"submodule", "update", "--init", "--recursive"})
	}

	err = executeCmds(cmdQueue)
	if err != nil {
		return err
	}

	// Store PR metadata in worktree git config
	err = storePRMetadata(worktreePath, pr)
	if err != nil {
		return fmt.Errorf("failed to store PR metadata: %w", err)
	}

	return nil
}

func storePRMetadata(worktreePath string, pr *PullRequest) error {
	// Store PR metadata in branch-specific git config
	branchName := pr.Head.Ref
	configs := [][]string{
		{"-C", worktreePath, "config", fmt.Sprintf("branch.%s.gh-worktree-pr-number", branchName), fmt.Sprintf("%d", pr.Number)},
		{"-C", worktreePath, "config", fmt.Sprintf("branch.%s.gh-worktree-pr-title", branchName), pr.Title},
	}

	for _, configCmd := range configs {
		cmd := exec.Command("git", configCmd...)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to set config %s: %w", strings.Join(configCmd[3:], " "), err)
		}
	}

	return nil
}

func getRemotes() ([]*Remote, error) {
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

func findBaseRemote(remotes []*Remote, repo repository.Repository) *Remote {
	repoName := repo.Name
	
	for _, remote := range remotes {
		if remote.Name == "origin" && strings.Contains(remote.URL, repoName) {
			return remote
		}
	}
	
	for _, remote := range remotes {
		if strings.Contains(remote.URL, repoName) {
			return remote
		}
	}
	
	if len(remotes) > 0 {
		return remotes[0]
	}
	
	return nil
}

func findHeadRemote(remotes []*Remote, pr *PullRequest) *Remote {
	headRepoName := pr.Head.Repo.Name
	headOwner := pr.Head.Repo.Owner.Login
	
	for _, remote := range remotes {
		if strings.Contains(remote.URL, headOwner) && strings.Contains(remote.URL, headRepoName) {
			return remote
		}
	}
	
	return nil
}

func cmdsForExistingRemote(remote *Remote, pr *PullRequest, opts *CheckoutOptions, worktreePath, branchName string) [][]string {
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
		if localBranchExists(branchName) {
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

	return cmds
}

func cmdsForMissingRemote(pr *PullRequest, baseRemote *Remote, repo repository.Repository, opts *CheckoutOptions, worktreePath, branchName string) [][]string {
	var cmds [][]string
	ref := fmt.Sprintf("refs/pull/%d/head", pr.Number)

	if opts.Detach {
		cmds = append(cmds, []string{"fetch", baseRemote.Name, ref, "--no-tags"})
		cmds = append(cmds, []string{"worktree", "add", "--detach", worktreePath, "FETCH_HEAD"})
		return cmds
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
		// If maintainer can modify, set up for push to head repository
		pushRemote := fmt.Sprintf("https://github.com/%s/%s", pr.Head.Repo.Owner.Login, pr.Head.Repo.Name)
		mergeRef = fmt.Sprintf("refs/heads/%s", pr.Head.Ref)
		cmds = append(cmds, []string{"-C", worktreePath, "config", fmt.Sprintf("branch.%s.pushRemote", branchName), pushRemote})
	}

	cmds = append(cmds, []string{"-C", worktreePath, "config", fmt.Sprintf("branch.%s.remote", branchName), remoteName})
	cmds = append(cmds, []string{"-C", worktreePath, "config", fmt.Sprintf("branch.%s.merge", branchName), mergeRef})

	return cmds
}

func localBranchExists(branchName string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", fmt.Sprintf("refs/heads/%s", branchName))
	return cmd.Run() == nil
}

func getGitRoot() (string, error) {
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

func getWorktrees() ([]*WorktreeInfo, error) {
	// Run git worktree list from the main repository to get all worktrees
	gitRoot, err := getGitRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get git root: %w", err)
	}
	
	cmd := exec.Command("git", "-C", gitRoot, "worktree", "list", "--porcelain")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree list: %w", err)
	}

	var worktrees []*WorktreeInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	
	var currentWorktree *WorktreeInfo
	for _, line := range lines {
		if line == "" {
			if currentWorktree != nil {
				worktrees = append(worktrees, currentWorktree)
				currentWorktree = nil
			}
			continue
		}
		
		if strings.HasPrefix(line, "worktree ") {
			currentWorktree = &WorktreeInfo{
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

func getPRWorktrees() ([]*WorktreeInfo, error) {
	allWorktrees, err := getWorktrees()
	if err != nil {
		return nil, err
	}

	gitRoot, err := getGitRoot()
	if err != nil {
		return nil, fmt.Errorf("failed to get git root: %w", err)
	}
	
	repoName := filepath.Base(gitRoot)
	parentDir := filepath.Dir(gitRoot)
	
	var prWorktrees []*WorktreeInfo
	for _, wt := range allWorktrees {
		// Check if this is a PR worktree based on naming pattern
		if strings.HasPrefix(filepath.Base(wt.Path), repoName+"-pr") {
			// Extract PR number from path
			baseName := filepath.Base(wt.Path)
			prPrefix := repoName + "-pr"
			if len(baseName) > len(prPrefix) {
				prNumberStr := baseName[len(prPrefix):]
				var prNumber int
				if n, err := fmt.Sscanf(prNumberStr, "%d", &prNumber); err == nil && n == 1 {
					wt.PRNumber = prNumber
					// Verify it's in the expected parent directory
					if filepath.Dir(wt.Path) == parentDir {
						// Try to get PR title from git config
						wt.Title = getPRTitle(wt.Path, wt.Branch)
						prWorktrees = append(prWorktrees, wt)
					}
				}
			}
		}
	}

	return prWorktrees, nil
}

func getBranchName(worktreePath string) string {
	cmd := exec.Command("git", "-C", worktreePath, "rev-parse", "--abbrev-ref", "HEAD")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func getPRTitle(worktreePath string, branchName string) string {
	if branchName == "" {
		return ""
	}
	cmd := exec.Command("git", "-C", worktreePath, "config", "--local", fmt.Sprintf("branch.%s.gh-worktree-pr-title", branchName))
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func listRun() error {
	prWorktrees, err := getPRWorktrees()
	if err != nil {
		return fmt.Errorf("failed to get PR worktrees: %w", err)
	}

	if len(prWorktrees) == 0 {
		fmt.Println("No PR worktrees found.")
		return nil
	}

	// Get current working directory for relative path calculation
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	fmt.Printf("PR worktrees:\n")
	for _, wt := range prWorktrees {
		title := wt.Title
		if title == "" {
			title = "(no title)"
		}
		
		// Convert absolute path to relative path
		relPath, err := filepath.Rel(cwd, wt.Path)
		if err != nil {
			relPath = wt.Path // Fall back to absolute path if relative path fails
		}
		
		fmt.Printf("  #%d\t%s\t%s\t%s\n", wt.PRNumber, wt.Branch, title, relPath)
	}

	return nil
}

func switchRun(shellMode bool, prNumber string) error {
	prWorktrees, err := getPRWorktrees()
	if err != nil {
		return fmt.Errorf("failed to get PR worktrees: %w", err)
	}

	if len(prWorktrees) == 0 {
		if !shellMode {
			fmt.Println("No PR worktrees found.")
		}
		// In shell mode, output nothing when no worktrees found
		return nil
	}

	var selectedWorktree *WorktreeInfo

	// If PR number is specified, find that specific worktree
	if prNumber != "" {
		prNum, err := parsePRNumber(prNumber)
		if err != nil {
			return fmt.Errorf("invalid PR number: %w", err)
		}
		
		for _, wt := range prWorktrees {
			if wt.PRNumber == prNum {
				selectedWorktree = wt
				break
			}
		}
		
		if selectedWorktree == nil {
			if !shellMode {
				fmt.Printf("Worktree for #%d not found.\n", prNum)
			}
			return nil
		}
	} else {
		// Interactive selection
		candidates := []string{}
		for _, wt := range prWorktrees {
			title := wt.Title
			if title == "" {
				title = "(no title)"
			}
			candidates = append(candidates, fmt.Sprintf("#%d\t%s\t%s", 
				wt.PRNumber, 
				wt.Branch, 
				title))
		}

		// Use gh CLI's built-in selection
		selection, err := promptSelect("Select a worktree to switch to", candidates)
		if err != nil {
			// If prompting fails (e.g., in non-interactive mode), try alternative approach
			if shellMode {
				// In shell mode, if prompting fails, just return empty to avoid cd errors
				return nil
			}
			return err
		}

		if selection == -1 {
			if !shellMode {
				fmt.Println("Cancelled.")
			}
			// In shell mode, output nothing when cancelled so cd doesn't change directory
			return nil
		}

		selectedWorktree = prWorktrees[selection]
	}
	
	// Get current working directory for relative path calculation
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Convert absolute path to relative path
	relPath, err := filepath.Rel(cwd, selectedWorktree.Path)
	if err != nil {
		relPath = selectedWorktree.Path // Fall back to absolute path
	}

	// Output based on mode
	if shellMode {
		// Shell mode: output only the path for use in shell functions
		fmt.Print(relPath)
	} else {
		// Normal mode: output a friendly message with command
		fmt.Printf("To switch to worktree for #%d:\n", selectedWorktree.PRNumber)
		fmt.Printf("cd %s\n", relPath)
	}
	
	return nil
}

func removeRunInteractive() error {
	prWorktrees, err := getPRWorktrees()
	if err != nil {
		return fmt.Errorf("failed to get PR worktrees: %w", err)
	}

	if len(prWorktrees) == 0 {
		fmt.Println("No PR worktrees found.")
		return nil
	}

	// Create candidates list in the same format as gh CLI
	candidates := []string{}
	for _, wt := range prWorktrees {
		candidates = append(candidates, fmt.Sprintf("#%d\t%s\t%s", 
			wt.PRNumber, 
			wt.Branch, 
			filepath.Base(wt.Path)))
	}

	// Use gh CLI's built-in selection
	selection, err := promptSelect("Select a worktree to remove", candidates)
	if err != nil {
		return err
	}

	if selection == -1 {
		fmt.Println("Cancelled.")
		return nil
	}

	selectedWorktree := prWorktrees[selection]
	
	// Remove the worktree
	cmd := exec.Command("git", "worktree", "remove", selectedWorktree.Path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// Delete the branch (this also removes branch-specific metadata)
	if selectedWorktree.Branch != "" {
		cmd = exec.Command("git", "branch", "-D", selectedWorktree.Branch)
		if err := cmd.Run(); err != nil {
			// Ignore error as branch might not exist or be checked out elsewhere
			fmt.Fprintf(os.Stderr, "Warning: failed to delete branch %s: %v\n", selectedWorktree.Branch, err)
		}
	}

	fmt.Printf("Removed worktree for #%d at %s\n", selectedWorktree.PRNumber, selectedWorktree.Path)
	if selectedWorktree.Title != "" {
		fmt.Printf("Title: %s\n", selectedWorktree.Title)
	}
	return nil
}

func promptSelect(message string, candidates []string) (int, error) {
	// Use gh CLI's built-in prompter - output prompts to stderr to avoid capture by $()
	p := prompter.New(os.Stdin, os.Stderr, os.Stderr)
	return p.Select(message, "", candidates)
}



func executeCmds(cmdQueue [][]string) error {
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
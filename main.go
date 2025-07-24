package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/cli/go-gh/v2/pkg/prompter"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/knqyf263/gh-worktree/internal/git"
	"github.com/knqyf263/gh-worktree/internal/github"
	"github.com/knqyf263/gh-worktree/internal/validate"
	"github.com/knqyf263/gh-worktree/internal/worktree"
	"github.com/spf13/cobra"
)

func main() {
	var opts worktree.CheckoutOptions

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
  $ gh worktree pr checkout https://github.com/OWNER/REPO/pull/32
  
  # Use as shell function to checkout and cd (add to ~/.bashrc or ~/.zshrc):
  $ ghwc() { 
      local target=$(gh worktree pr checkout --shell "$@")
      [ -n "$target" ] && cd "$target"
    }
  $ ghwc     # interactive checkout
  $ ghwc 9060  # checkout specific PR`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shellMode, _ := cmd.Flags().GetBool("shell")
			opts.ShellMode = shellMode
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
	checkoutCmd.Flags().BoolP("shell", "s", false, "Output path only for use in shell functions")

	var removeOpts struct {
		Force bool
	}

	removeCmd := &cobra.Command{
		Use:   "remove [<number> | <url> | <branch>]",
		Short: "Remove a pull request worktree",
		Example: `  # Interactively select a worktree to remove
  $ gh worktree pr remove

  # Remove a specific PR worktree
  $ gh worktree pr remove 32

  # Remove PR worktree from URL  
  $ gh worktree pr remove https://github.com/OWNER/REPO/pull/32

  # Force remove without confirmation
  $ gh worktree pr remove 32 --force`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return removeRun(args[0], removeOpts.Force)
			}
			return removeRunInteractive(removeOpts.Force)
		},
	}

	removeCmd.Flags().BoolVarP(&removeOpts.Force, "force", "f", false, "Force removal without confirmation")

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
		Use:   "switch [<number> | main]",
		Short: "Switch to an existing pull request worktree or main worktree",
		Example: `  # Interactively select a worktree to switch to
  $ gh worktree pr switch
  
  # Switch to specific PR worktree
  $ gh worktree pr switch 9060
  
  # Switch to main worktree
  $ gh worktree pr switch main
  
  # Use as shell function (add to ~/.bashrc or ~/.zshrc):
  $ ghws() { 
      local target=$(gh worktree pr switch --shell "$@")
      [ -n "$target" ] && cd "$target"
    }
  $ ghws     # interactive selection
  $ ghws 9060  # switch to specific PR
  $ ghws main  # switch to main worktree`,
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

func checkoutRunInteractive(opts *worktree.CheckoutOptions) error {
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

	var prs []github.PullRequest
	err = client.Get(fmt.Sprintf("repos/%s/%s/pulls?state=open&per_page=100", repo.Owner, repo.Name), &prs)
	if err != nil {
		return fmt.Errorf("failed to get PRs: %w", err)
	}

	if len(prs) == 0 {
		fmt.Println("No open pull requests found.")
		return nil
	}

	// Create candidates list
	candidates := []string{}
	for _, pr := range prs {
		candidates = append(candidates, github.FormatPRCandidate(&pr))
	}

	// Use gh CLI's built-in selection
	selection, err := promptSelect("Select a pull request to check out", candidates)
	if err != nil {
		if opts.ShellMode {
			// In shell mode, if prompting fails, just return empty to avoid cd errors
			return nil
		}
		return err
	}

	if selection == -1 {
		if !opts.ShellMode {
			fmt.Println("Cancelled.")
		}
		// In shell mode, output nothing when cancelled so cd doesn't change directory
		return nil
	}

	selectedPR := prs[selection]

	// Generate worktree path
	gitRoot, err := git.GetRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}

	repoName := filepath.Base(gitRoot)
	if err := validate.RepoName(repoName); err != nil {
		return fmt.Errorf("invalid repository name: %w", err)
	}
	if err := validate.PRNumber(selectedPR.Number); err != nil {
		return fmt.Errorf("invalid PR number: %w", err)
	}

	worktreePath, err := worktree.GeneratePath(repoName, selectedPR.Number)
	if err != nil {
		return fmt.Errorf("failed to generate worktree path: %w", err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("worktree for PR #%d already exists at %s", selectedPR.Number, worktreePath)
	}

	// Create worktree
	creator, err := worktree.NewCreator(repo)
	if err != nil {
		return fmt.Errorf("failed to create worktree creator: %w", err)
	}

	err = creator.Create(worktreePath, &selectedPR, opts)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Output based on mode
	if opts.ShellMode {
		// Shell mode: output only the path for use in shell functions
		// Get current working directory for relative path calculation
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		
		// Convert absolute path to relative path
		relPath, err := filepath.Rel(cwd, worktreePath)
		if err != nil {
			relPath = worktreePath // Fall back to absolute path
		}
		fmt.Print(relPath)
	} else {
		// Normal mode: output a friendly message
		fmt.Printf("Created worktree for #%d at %s\n", selectedPR.Number, worktreePath)
		if selectedPR.Title != "" {
			fmt.Printf("Title: %s\n", selectedPR.Title)
		}
	}
	return nil
}

func checkoutRun(opts *worktree.CheckoutOptions, selector string) error {
	// Get current repository
	repo, err := repository.Current()
	if err != nil {
		return fmt.Errorf("failed to get current repository: %w", err)
	}

	// Parse PR number from selector
	prNumber, err := github.ParsePRNumber(selector)
	if err != nil {
		return fmt.Errorf("failed to parse PR number: %w", err)
	}

	// Get PR details
	client, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("failed to create REST client: %w", err)
	}

	var pr github.PullRequest
	err = client.Get(fmt.Sprintf("repos/%s/%s/pulls/%d", repo.Owner, repo.Name, prNumber), &pr)
	if err != nil {
		return fmt.Errorf("failed to get PR details: %w", err)
	}

	// Generate worktree path
	gitRoot, err := git.GetRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}

	repoName := filepath.Base(gitRoot)
	if err := validate.RepoName(repoName); err != nil {
		return fmt.Errorf("invalid repository name: %w", err)
	}
	if err := validate.PRNumber(prNumber); err != nil {
		return fmt.Errorf("invalid PR number: %w", err)
	}

	worktreePath, err := worktree.GeneratePath(repoName, prNumber)
	if err != nil {
		return fmt.Errorf("failed to generate worktree path: %w", err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("worktree for PR #%d already exists at %s", prNumber, worktreePath)
	}

	// Create worktree
	creator, err := worktree.NewCreator(repo)
	if err != nil {
		return fmt.Errorf("failed to create worktree creator: %w", err)
	}

	err = creator.Create(worktreePath, &pr, opts)
	if err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Output based on mode
	if opts.ShellMode {
		// Shell mode: output only the path for use in shell functions
		// Get current working directory for relative path calculation
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		
		// Convert absolute path to relative path
		relPath, err := filepath.Rel(cwd, worktreePath)
		if err != nil {
			relPath = worktreePath // Fall back to absolute path
		}
		fmt.Print(relPath)
	} else {
		// Normal mode: output a friendly message
		fmt.Printf("Created worktree for #%d at %s\n", prNumber, worktreePath)
		if pr.Title != "" {
			fmt.Printf("Title: %s\n", pr.Title)
		}
	}
	return nil
}

func removeRun(selector string, force bool) error {
	// Parse PR number from selector
	prNumber, err := github.ParsePRNumber(selector)
	if err != nil {
		return fmt.Errorf("failed to parse PR number: %w", err)
	}

	// Generate worktree path
	gitRoot, err := git.GetRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}

	repoName := filepath.Base(gitRoot)
	if err := validate.RepoName(repoName); err != nil {
		return fmt.Errorf("invalid repository name: %w", err)
	}
	if err := validate.PRNumber(prNumber); err != nil {
		return fmt.Errorf("invalid PR number: %w", err)
	}

	worktreePath, err := worktree.GeneratePath(repoName, prNumber)
	if err != nil {
		return fmt.Errorf("failed to generate worktree path: %w", err)
	}

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		return fmt.Errorf("worktree for PR #%d does not exist at %s", prNumber, worktreePath)
	}

	// Get branch name before removing worktree
	branchName := git.GetBranchName(worktreePath)

	// Get PR title from git config before removing
	title := ""
	if branchName != "" {
		title = worktree.GetPRTitle(worktreePath, branchName)
	}

	// Remove the worktree
	err = worktree.Remove(worktreePath, force)
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// Delete the branch (this also removes branch-specific metadata)
	if branchName != "" && branchName != "HEAD" {
		if err := validate.BranchName(branchName); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid branch name %s: %v\n", branchName, err)
		} else {
			err := worktree.DeleteBranch(branchName)
			if err != nil {
				// Ignore error as branch might not exist or be checked out elsewhere
				fmt.Fprintf(os.Stderr, "Warning: failed to delete branch %s: %v\n", branchName, err)
			}
		}
	}

	fmt.Printf("Removed worktree for #%d at %s\n", prNumber, worktreePath)
	if title != "" {
		fmt.Printf("Title: %s\n", title)
	}
	return nil
}

func listRun() error {
	gitRoot, err := git.GetRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}

	repoName := filepath.Base(gitRoot)
	prWorktrees, err := worktree.ListPRWorktrees(repoName)
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
	gitRoot, err := git.GetRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}

	repoName := filepath.Base(gitRoot)
	prWorktrees, err := worktree.ListPRWorktrees(repoName)
	if err != nil {
		return fmt.Errorf("failed to get PR worktrees: %w", err)
	}

	var selectedWorktree *worktree.Info
	var targetPath string

	// Handle direct selection
	if prNumber != "" {
		if prNumber == "main" {
			// Handle main worktree selection
			targetPath = gitRoot
		} else {
			// Handle PR number selection
			prNum, err := github.ParsePRNumber(prNumber)
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
			targetPath = selectedWorktree.Path
		}
	} else {
		// Interactive selection
		candidates := []string{}

		// Add main worktree as first option
		candidates = append(candidates, "main\tmain\t(main worktree)")

		// Add PR worktrees
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

		if selection == 0 {
			// Main worktree selected
			targetPath = gitRoot
		} else {
			// PR worktree selected (adjust index since main is first)
			selectedWorktree = prWorktrees[selection-1]
			targetPath = selectedWorktree.Path
		}
	}

	// Get current working directory for relative path calculation
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Convert absolute path to relative path
	relPath, err := filepath.Rel(cwd, targetPath)
	if err != nil {
		relPath = targetPath // Fall back to absolute path
	}

	// Output based on mode
	if shellMode {
		// Shell mode: output only the path for use in shell functions
		fmt.Print(relPath)
	} else {
		// Normal mode: output a friendly message with command
		if prNumber == "main" || (prNumber == "" && targetPath == gitRoot) {
			fmt.Printf("To switch to main worktree:\n")
		} else {
			fmt.Printf("To switch to worktree for #%d:\n", selectedWorktree.PRNumber)
		}
		fmt.Printf("cd %s\n", relPath)
	}

	return nil
}

func removeRunInteractive(force bool) error {
	gitRoot, err := git.GetRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}

	repoName := filepath.Base(gitRoot)
	prWorktrees, err := worktree.ListPRWorktrees(repoName)
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
	err = worktree.Remove(selectedWorktree.Path, force)
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// Delete the branch (this also removes branch-specific metadata)
	if selectedWorktree.Branch != "" && selectedWorktree.Branch != "HEAD" {
		if err := validate.BranchName(selectedWorktree.Branch); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid branch name %s: %v\n", selectedWorktree.Branch, err)
		} else {
			err := worktree.DeleteBranch(selectedWorktree.Branch)
			if err != nil {
				// Ignore error as branch might not exist or be checked out elsewhere
				fmt.Fprintf(os.Stderr, "Warning: failed to delete branch %s: %v\n", selectedWorktree.Branch, err)
			}
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

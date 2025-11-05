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
	var shellMode bool

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

  # Create a new branch worktree for local development
  $ gh worktree pr checkout --create feature-auth
  $ gh worktree pr checkout -c feature-auth

  # Use as shell function to checkout and cd (add to ~/.bashrc or ~/.zshrc):
  $ ghwc() {
      local target=$(gh worktree pr checkout --shell "$@")
      [ -n "$target" ] && cd "$target"
    }
  $ ghwc     # interactive checkout
  $ ghwc 9060  # checkout specific PR`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shellModeFlag, _ := cmd.Flags().GetBool("shell")
			createBranch, _ := cmd.Flags().GetString("create")
			opts.ShellMode = shellModeFlag
			shellMode = shellModeFlag // Set the outer shellMode variable
			if shellModeFlag {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
			}

			// Handle --create flag for branch worktrees
			if createBranch != "" {
				return checkoutBranchWorktree(createBranch, &opts)
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
	checkoutCmd.Flags().BoolP("shell", "s", false, "Output path only for use in shell functions")
	checkoutCmd.Flags().StringP("create", "c", "", "Create a new branch worktree for local development")

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

	var listOpts struct {
		All bool
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List pull request worktrees",
		Example: `  # List all PR worktrees
  $ gh worktree pr list

  # List all worktrees (PR and branch)
  $ gh worktree pr list --all`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return listRun(listOpts.All)
		},
	}

	listCmd.Flags().BoolVarP(&listOpts.All, "all", "a", false, "List all worktrees (PR and branch)")

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
			shellModeFlag, _ := cmd.Flags().GetBool("shell")
			shellMode = shellModeFlag // Set the outer shellMode variable
			if shellModeFlag {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
			}
			prNumber := ""
			if len(args) > 0 {
				prNumber = args[0]
			}
			return switchRun(shellModeFlag, prNumber)
		},
	}

	switchCmd.Flags().BoolP("shell", "s", false, "Output path only for use in shell functions")

	promoteCmd := &cobra.Command{
		Use:   "promote [<branch>] [<pr-number>]",
		Short: "Promote a branch worktree to a PR worktree",
		Example: `  # Promote current branch worktree after creating a PR
  $ gh worktree pr promote

  # Promote a specific branch worktree
  $ gh worktree pr promote feature-auth

  # Promote with explicit PR number
  $ gh worktree pr promote feature-auth 1234`,
		Args: cobra.RangeArgs(0, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			var branchName string
			prNumber := 0

			if len(args) == 0 {
				// Get current branch name
				currentBranch := git.GetBranchName(".")
				if currentBranch == "" || currentBranch == "HEAD" {
					return fmt.Errorf("could not determine current branch. Please specify branch name")
				}
				branchName = currentBranch
			} else {
				branchName = args[0]
				if len(args) > 1 {
					// Parse PR number if provided
					var err error
					prNumber, err = github.ParsePRNumber(args[1])
					if err != nil {
						return fmt.Errorf("invalid PR number: %w", err)
					}
				}
			}
			return promoteRun(branchName, prNumber)
		},
	}

	prCmd.AddCommand(checkoutCmd)
	prCmd.AddCommand(removeCmd)
	prCmd.AddCommand(listCmd)
	prCmd.AddCommand(switchCmd)
	prCmd.AddCommand(promoteCmd)
	rootCmd.AddCommand(prCmd)

	// Root-level switch command (unified switcher)
	rootSwitchCmd := &cobra.Command{
		Use:   "switch [<identifier> | main]",
		Short: "Switch to any worktree (PR, branch, or main)",
		Example: `  # Interactively select from all worktrees
  $ gh worktree switch

  # Switch to specific PR worktree
  $ gh worktree switch 9060

  # Switch to specific branch worktree
  $ gh worktree switch feature-auth

  # Switch to main worktree
  $ gh worktree switch main

  # Use as shell function (add to ~/.bashrc or ~/.zshrc):
  $ ghws() {
      local target=$(gh worktree switch --shell "$@")
      [ -n "$target" ] && cd "$target"
    }`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shellModeFlag, _ := cmd.Flags().GetBool("shell")
			shellMode = shellModeFlag
			if shellModeFlag {
				cmd.SilenceUsage = true
				cmd.SilenceErrors = true
			}
			identifier := ""
			if len(args) > 0 {
				identifier = args[0]
			}
			return switchAllRun(shellModeFlag, identifier)
		},
	}
	rootSwitchCmd.Flags().BoolP("shell", "s", false, "Output path only for use in shell functions")
	rootCmd.AddCommand(rootSwitchCmd)

	if err := rootCmd.Execute(); err != nil {
		if !shellMode {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
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
	
	// Fetch full PR details to get maintainer_can_modify and other fields
	var fullPR github.PullRequest
	err = client.Get(fmt.Sprintf("repos/%s/%s/pulls/%d", repo.Owner, repo.Name, selectedPR.Number), &fullPR)
	if err != nil {
		return fmt.Errorf("failed to get full PR details: %w", err)
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
	if err := validate.PRNumber(fullPR.Number); err != nil {
		return fmt.Errorf("invalid PR number: %w", err)
	}

	worktreePath, err := worktree.GeneratePath(repoName, fullPR.Number)
	if err != nil {
		return fmt.Errorf("failed to generate worktree path: %w", err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		if opts.ShellMode {
			// In shell mode, output the existing path so cd still works
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			relPath, err := filepath.Rel(cwd, worktreePath)
			if err != nil {
				relPath = worktreePath
			}
			fmt.Print(relPath)
			return nil
		}
		return fmt.Errorf("worktree for PR #%d already exists at %s", fullPR.Number, worktreePath)
	}

	// Create worktree
	creator, err := worktree.NewCreator(repo)
	if err != nil {
		return fmt.Errorf("failed to create worktree creator: %w", err)
	}

	err = creator.Create(worktreePath, &fullPR, opts)
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
		fmt.Printf("Created worktree for #%d at %s\n", fullPR.Number, worktreePath)
		if fullPR.Title != "" {
			fmt.Printf("Title: %s\n", fullPR.Title)
		}
	}
	return nil
}

// checkoutBranchWorktree creates a new worktree for local development.
func checkoutBranchWorktree(branchName string, opts *worktree.CheckoutOptions) error {
	// Validate branch name
	if err := validate.BranchName(branchName); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	// Get git root and repo name
	gitRoot, err := git.GetRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}

	repoName := filepath.Base(gitRoot)
	if err := validate.RepoName(repoName); err != nil {
		return fmt.Errorf("invalid repository name: %w", err)
	}

	// Generate worktree path for branch
	worktreePath, err := worktree.GeneratePathForBranch(repoName, branchName)
	if err != nil {
		return fmt.Errorf("failed to generate worktree path: %w", err)
	}

	// Check if worktree already exists
	if _, err := os.Stat(worktreePath); err == nil {
		if opts.ShellMode {
			// In shell mode, output the existing path so cd still works
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			relPath, err := filepath.Rel(cwd, worktreePath)
			if err != nil {
				relPath = worktreePath
			}
			fmt.Print(relPath)
			return nil
		}
		return fmt.Errorf("worktree for branch %s already exists at %s", branchName, worktreePath)
	}

	// Check if branch already exists
	branchExists := git.BranchExists(branchName)

	// Create worktree with new branch from HEAD
	var cmd [][]string
	if branchExists {
		// Branch exists, checkout existing branch
		cmd = [][]string{{"worktree", "add", worktreePath, branchName}}
	} else {
		// Create new branch from HEAD
		cmd = [][]string{{"worktree", "add", "-b", branchName, worktreePath}}
	}

	if err := git.ExecuteCommands(cmd); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Set worktree type metadata
	if err := worktree.SetWorktreeType(branchName, "branch"); err != nil {
		return fmt.Errorf("failed to set worktree type: %w", err)
	}

	// Output based on mode
	if opts.ShellMode {
		// Shell mode: output only the path for use in shell functions
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		relPath, err := filepath.Rel(cwd, worktreePath)
		if err != nil {
			relPath = worktreePath
		}
		fmt.Print(relPath)
	} else {
		// Normal mode: output a friendly message
		fmt.Printf("Created worktree for branch '%s' at %s\n", branchName, worktreePath)
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
		if opts.ShellMode {
			// In shell mode, output the existing path so cd still works
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}
			relPath, err := filepath.Rel(cwd, worktreePath)
			if err != nil {
				relPath = worktreePath
			}
			fmt.Print(relPath)
			return nil
		}
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
	gitRoot, err := git.GetRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}

	repoName := filepath.Base(gitRoot)
	if err := validate.RepoName(repoName); err != nil {
		return fmt.Errorf("invalid repository name: %w", err)
	}

	var worktreePath string
	var prNumber int
	var isBranchWorktree bool

	// Try to parse as PR number
	prNum, err := github.ParsePRNumber(selector)
	if err == nil {
		// It's a PR number
		if err := validate.PRNumber(prNum); err != nil {
			return fmt.Errorf("invalid PR number: %w", err)
		}
		prNumber = prNum

		worktreePath, err = worktree.GeneratePath(repoName, prNumber)
		if err != nil {
			return fmt.Errorf("failed to generate worktree path: %w", err)
		}
	} else {
		// Try as branch name
		if err := validate.BranchName(selector); err != nil {
			return fmt.Errorf("invalid identifier: not a valid PR number or branch name: %w", err)
		}

		worktreePath, err = worktree.GeneratePathForBranch(repoName, selector)
		if err != nil {
			return fmt.Errorf("failed to generate worktree path: %w", err)
		}
		isBranchWorktree = true
	}

	// Check if worktree exists
	if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
		if isBranchWorktree {
			return fmt.Errorf("worktree for branch %s does not exist at %s", selector, worktreePath)
		}
		return fmt.Errorf("worktree for PR #%d does not exist at %s", prNumber, worktreePath)
	}

	// Get branch name before removing worktree
	branchName := git.GetBranchName(worktreePath)

	// Get title/metadata from git config before removing
	title := ""
	if branchName != "" {
		if isBranchWorktree {
			title = "(local development)"
		} else {
			title = worktree.GetPRTitle(worktreePath, branchName)
		}
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

	// Output based on worktree type
	if isBranchWorktree {
		fmt.Printf("Removed worktree for branch '%s' at %s\n", selector, worktreePath)
	} else {
		fmt.Printf("Removed worktree for #%d at %s\n", prNumber, worktreePath)
		if title != "" {
			fmt.Printf("Title: %s\n", title)
		}
	}

	return nil
}

func listRun(showAll bool) error {
	gitRoot, err := git.GetRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}

	repoName := filepath.Base(gitRoot)

	// Get current working directory for relative path calculation
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if showAll {
		// List both PR and branch worktrees
		prWorktrees, branchWorktrees, err := worktree.ListAllWorktrees(repoName)
		if err != nil {
			return fmt.Errorf("failed to get worktrees: %w", err)
		}

		if len(prWorktrees) == 0 && len(branchWorktrees) == 0 {
			fmt.Println("No worktrees found.")
			return nil
		}

		// List PR worktrees
		if len(prWorktrees) > 0 {
			fmt.Printf("PR worktrees:\n")
			for _, wt := range prWorktrees {
				title := wt.Title
				if title == "" {
					title = "(no title)"
				}

				relPath, err := filepath.Rel(cwd, wt.Path)
				if err != nil {
					relPath = wt.Path
				}

				fmt.Printf("  #%d\t%s\t%s\t%s\n", wt.PRNumber, wt.Branch, title, relPath)
			}
		}

		// List branch worktrees
		if len(branchWorktrees) > 0 {
			if len(prWorktrees) > 0 {
				fmt.Println()
			}
			fmt.Printf("Branch worktrees:\n")
			for _, wt := range branchWorktrees {
				relPath, err := filepath.Rel(cwd, wt.Path)
				if err != nil {
					relPath = wt.Path
				}

				fmt.Printf("  %s\t(local development)\t%s\n", wt.Branch, relPath)
			}
		}
	} else {
		// List only PR worktrees (default behavior)
		prWorktrees, err := worktree.ListPRWorktrees(repoName)
		if err != nil {
			return fmt.Errorf("failed to get PR worktrees: %w", err)
		}

		if len(prWorktrees) == 0 {
			fmt.Println("No PR worktrees found.")
			return nil
		}

		fmt.Printf("PR worktrees:\n")
		for _, wt := range prWorktrees {
			title := wt.Title
			if title == "" {
				title = "(no title)"
			}

			relPath, err := filepath.Rel(cwd, wt.Path)
			if err != nil {
				relPath = wt.Path
			}

			fmt.Printf("  #%d\t%s\t%s\t%s\n", wt.PRNumber, wt.Branch, title, relPath)
		}
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

// promoteRun promotes a branch worktree to a PR worktree.
func promoteRun(branchName string, prNumber int) error {
	// Validate branch name
	if err := validate.BranchName(branchName); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	// Check if it's already a PR worktree
	worktreeType, err := worktree.GetWorktreeType(branchName)
	if err != nil {
		return fmt.Errorf("failed to get worktree type: %w", err)
	}
	if worktreeType == "pr" {
		return fmt.Errorf("branch %s is already a PR worktree", branchName)
	}

	// If PR number not provided, try to find it from the branch
	if prNumber == 0 {
		// Get current repository
		repo, err := repository.Current()
		if err != nil {
			return fmt.Errorf("failed to get current repository: %w", err)
		}

		// Get all PRs for this branch
		client, err := api.DefaultRESTClient()
		if err != nil {
			return fmt.Errorf("failed to create REST client: %w", err)
		}

		var prs []github.PullRequest
		err = client.Get(fmt.Sprintf("repos/%s/%s/pulls?head=%s:%s&state=open", 
			repo.Owner, repo.Name, repo.Owner, branchName), &prs)
		if err != nil {
			return fmt.Errorf("failed to get PRs for branch: %w", err)
		}

		if len(prs) == 0 {
			return fmt.Errorf("no open PR found for branch %s. Please create a PR first or specify the PR number", branchName)
		}

		if len(prs) > 1 {
			return fmt.Errorf("multiple PRs found for branch %s. Please specify the PR number", branchName)
		}

		prNumber = prs[0].Number
	}

	// Get PR details to get the title
	repo, err := repository.Current()
	if err != nil {
		return fmt.Errorf("failed to get current repository: %w", err)
	}

	client, err := api.DefaultRESTClient()
	if err != nil {
		return fmt.Errorf("failed to create REST client: %w", err)
	}

	var pr github.PullRequest
	err = client.Get(fmt.Sprintf("repos/%s/%s/pulls/%d", repo.Owner, repo.Name, prNumber), &pr)
	if err != nil {
		return fmt.Errorf("failed to get PR details: %w", err)
	}

	// Promote to PR worktree
	if err := worktree.PromoteToPR(branchName, prNumber, pr.Title); err != nil {
		return fmt.Errorf("failed to promote worktree: %w", err)
	}

	fmt.Printf("Promoted worktree for branch '%s' to PR #%d\n", branchName, prNumber)
	if pr.Title != "" {
		fmt.Printf("Title: %s\n", pr.Title)
	}

	return nil
}

// switchAllRun switches to any worktree (PR, branch, or main).
func switchAllRun(shellMode bool, identifier string) error {
	gitRoot, err := git.GetRoot()
	if err != nil {
		return fmt.Errorf("failed to get git root: %w", err)
	}

	repoName := filepath.Base(gitRoot)
	prWorktrees, branchWorktrees, err := worktree.ListAllWorktrees(repoName)
	if err != nil {
		return fmt.Errorf("failed to get worktrees: %w", err)
	}

	var targetPath string

	// Handle direct selection
	if identifier != "" {
		if identifier == "main" {
			targetPath = gitRoot
		} else {
			// Try to parse as PR number
			if prNum, err := github.ParsePRNumber(identifier); err == nil {
				for _, wt := range prWorktrees {
					if wt.PRNumber == prNum {
						targetPath = wt.Path
						break
					}
				}
			}

			// If not found as PR, try to find as branch name
			if targetPath == "" {
				for _, wt := range branchWorktrees {
					if wt.Branch == identifier {
						targetPath = wt.Path
						break
					}
				}
			}

			if targetPath == "" {
				if !shellMode {
					fmt.Printf("Worktree '%s' not found.\n", identifier)
				}
				return nil
			}
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

		// Add branch worktrees
		for _, wt := range branchWorktrees {
			candidates = append(candidates, fmt.Sprintf("branch:%s\t%s\t(local development)",
				wt.Branch,
				wt.Branch))
		}

		// Use gh CLI's built-in selection
		selection, err := promptSelect("Select a worktree to switch to", candidates)
		if err != nil {
			if shellMode {
				return nil
			}
			return err
		}

		if selection == -1 {
			if !shellMode {
				fmt.Println("Cancelled.")
			}
			return nil
		}

		if selection == 0 {
			// Main worktree selected
			targetPath = gitRoot
		} else if selection <= len(prWorktrees) {
			// PR worktree selected
			targetPath = prWorktrees[selection-1].Path
		} else {
			// Branch worktree selected
			targetPath = branchWorktrees[selection-1-len(prWorktrees)].Path
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
		relPath = targetPath
	}

	// Output based on mode
	if shellMode {
		// Shell mode: output only the path
		fmt.Print(relPath)
	} else {
		// Normal mode: output a friendly message
		if targetPath == gitRoot {
			fmt.Printf("To switch to main worktree:\n")
		} else {
			fmt.Printf("To switch to worktree:\n")
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
	prWorktrees, branchWorktrees, err := worktree.ListAllWorktrees(repoName)
	if err != nil {
		return fmt.Errorf("failed to get worktrees: %w", err)
	}

	if len(prWorktrees) == 0 && len(branchWorktrees) == 0 {
		fmt.Println("No worktrees found.")
		return nil
	}

	// Create candidates list
	candidates := []string{}
	
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

	// Add branch worktrees
	for _, wt := range branchWorktrees {
		candidates = append(candidates, fmt.Sprintf("branch:%s\t%s\t(local development)",
			wt.Branch,
			wt.Branch))
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

	var selectedWorktree *worktree.Info
	var isBranchWorktree bool

	if selection < len(prWorktrees) {
		// PR worktree selected
		selectedWorktree = prWorktrees[selection]
	} else {
		// Branch worktree selected
		selectedWorktree = branchWorktrees[selection-len(prWorktrees)]
		isBranchWorktree = true
	}

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

	// Output based on worktree type
	if isBranchWorktree {
		fmt.Printf("Removed worktree for branch '%s' at %s\n", selectedWorktree.Branch, selectedWorktree.Path)
	} else {
		fmt.Printf("Removed worktree for #%d at %s\n", selectedWorktree.PRNumber, selectedWorktree.Path)
		if selectedWorktree.Title != "" {
			fmt.Printf("Title: %s\n", selectedWorktree.Title)
		}
	}

	return nil
}

func promptSelect(message string, candidates []string) (int, error) {
	// Use gh CLI's built-in prompter - output prompts to stderr to avoid capture by $()
	p := prompter.New(os.Stdin, os.Stderr, os.Stderr)
	return p.Select(message, "", candidates)
}

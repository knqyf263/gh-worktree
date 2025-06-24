# gh-worktree

A GitHub CLI extension for managing git worktrees with pull requests.

Instead of checking out pull requests in your current directory, this extension creates dedicated worktrees in a separate directory with a clear naming convention (`../repo-name-pr1234`).

## Features

- 🌳 **Create worktrees for PRs** - Each PR gets its own isolated working directory
- 🔄 **Switch between worktrees** - Quick navigation between different PR worktrees
- 🗑️ **Clean removal** - Remove worktrees and associated branches in one command
- 📋 **List all PR worktrees** - See all your PR worktrees at a glance
- 🎯 **Interactive selection** - Use arrow keys and filtering to select PRs

## Installation

```bash
gh extension install knqyf263/gh-worktree
```

## Commands

### `gh worktree pr checkout`

Create a new worktree for a pull request.

```bash
# Interactive selection from open PRs
gh worktree pr checkout

# Checkout specific PR by number
gh worktree pr checkout 1234

# Checkout specific PR by URL
gh worktree pr checkout https://github.com/owner/repo/pull/1234
```

**Example Output:**
```
Created worktree for #1234 at ../repo-name-pr1234
Title: Add new feature for user management
```

### `gh worktree pr list`

List all existing PR worktrees.

```bash
gh worktree pr list
```

**Example Output:**
```
PR worktrees:
  #1234	feature-branch	Add new feature for user management	../repo-name-pr1234
  #5678	bugfix-branch	Fix critical security vulnerability	../repo-name-pr5678
```

### `gh worktree pr remove`

Remove a PR worktree and its associated branch.

```bash
# Interactive selection
gh worktree pr remove

# Remove specific PR worktree
gh worktree pr remove 1234
```

**Example Output:**
```
Removed worktree for #1234 at ../repo-name-pr1234
Title: Add new feature for user management
```

### `gh worktree pr switch`

Switch to an existing PR worktree directory.

```bash
# Interactive selection
gh worktree pr switch

# Switch to specific PR worktree
gh worktree pr switch 1234

# Shell mode (outputs path only)
gh worktree pr switch --shell 1234
```

## Directory Structure

The extension creates worktrees in the parent directory of your current repository:

```
parent-directory/
├── my-repo/                    # Original repository
├── my-repo-pr1234/            # PR #1234 worktree
├── my-repo-pr5678/            # PR #5678 worktree
└── my-repo-pr9999/            # PR #9999 worktree
```

## Shell Integration

For the best experience, add this shell function to your `~/.bashrc` or `~/.zshrc`:

```bash
# Quick worktree switcher
ghws() { 
  local target=$(gh worktree pr switch --shell "$@")
  [ -n "$target" ] && cd "$target"
}
```

**Usage:**
```bash
# Interactive selection and switch
ghws

# Switch to specific PR
ghws 1234
```

## How It Works

1. **Worktree Creation**: Creates git worktrees in `../repo-name-pr{number}` format
2. **Branch Management**: Sets up proper remote tracking and handles cross-repository PRs similar to `gh pr checkout`
3. **Clean Removal**: Removes both worktree and branch when cleaning up

## Comparison with `gh pr checkout`

| Feature          | `gh pr checkout`   | `gh worktree pr checkout`   |
|------------------|--------------------|-----------------------------|
| Location         | Current directory  | Separate worktree directory |
| Multiple PRs     | Requires switching | Parallel development        |
| Branch conflicts | Possible           | Isolated per worktree       |

## Advanced Usage

### Working with Multiple PRs

```bash
# Checkout multiple PRs for parallel development
gh worktree pr checkout 1234
gh worktree pr checkout 5678

# List all active worktrees
gh worktree pr list

# Switch between them quickly
ghws 1234  # Switch to PR 1234
ghws 5678  # Switch to PR 5678
```

### Cross-Repository PRs

The extension handles PRs from forks correctly:

```bash
# This works even if the PR is from a fork
gh worktree pr checkout 1234
```

The extension will:
- Set up appropriate remotes
- Configure push/pull settings
- Handle `maintainer_can_modify` permissions

## Requirements

- [GitHub CLI](https://cli.github.com/) (gh)
- Git with worktree support (Git 2.5+)
- Access to the repository (for API calls)

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

This project is licensed under the MIT License.
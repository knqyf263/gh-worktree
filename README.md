# gh-worktree

A GitHub CLI extension for managing git worktrees with pull requests and local development branches.

Instead of checking out pull requests in your current directory, this extension creates dedicated worktrees in separate directories:
- PR worktrees: `../repo-name-pr1234`
- Branch worktrees: `../repo-name-feature-name`

## Features

- 🌳 **Create worktrees for PRs** - Each PR gets its own isolated working directory
- 🔧 **Create worktrees for local development** - Work on features before creating PRs
- ⬆️ **Promote branch worktrees to PR worktrees** - Link local branches to PRs after creation
- 🔄 **Switch between all worktrees** - Quick navigation between PR, branch, and main worktrees
- 🗑️ **Clean removal** - Remove worktrees and associated branches in one command
- 📋 **List all worktrees** - See all your PR and branch worktrees at a glance
- 🎯 **Interactive selection** - Use arrow keys and filtering to select from all worktrees
- ⚙️ **Post-creation setup** - Automatically copy files and run commands in new worktrees

## Installation

```bash
gh extension install knqyf263/gh-worktree
```

## Commands

### `gh worktree pr checkout`

Create a new worktree for a pull request or local development.

```bash
# Interactive selection from open PRs
gh worktree pr checkout

# Checkout specific PR by number
gh worktree pr checkout 1234

# Checkout specific PR by URL
gh worktree pr checkout https://github.com/owner/repo/pull/1234

# Create a new branch worktree for local development
gh worktree pr checkout --create feature-auth
gh worktree pr checkout -c feature-auth
```

**Example Output:**
```
Created worktree for #1234 at ../repo-name-pr1234
Title: Add new feature for user management

# Or for branch worktrees:
Created worktree for branch 'feature-auth' at ../repo-name-feature-auth
```

### `gh worktree pr list`

List PR worktrees or all worktrees.

```bash
# List only PR worktrees (default)
gh worktree pr list

# List all worktrees (PR and branch)
gh worktree pr list --all
```

**Example Output:**
```
PR worktrees:
  #1234	feature-branch	Add new feature for user management	../repo-name-pr1234
  #5678	bugfix-branch	Fix critical security vulnerability	../repo-name-pr5678

Branch worktrees:
  feature-auth	(local development)	../repo-name-feature-auth
  experiment-api	(local development)	../repo-name-experiment-api
```

### `gh worktree pr remove`

Remove a PR or branch worktree and its associated branch.

```bash
# Interactive selection (shows all worktrees)
gh worktree pr remove

# Remove specific PR worktree
gh worktree pr remove 1234

# Remove specific branch worktree
gh worktree pr remove feature-auth
```

**Example Output:**
```
Removed worktree for #1234 at ../repo-name-pr1234
Title: Add new feature for user management

# Or for branch worktrees:
Removed worktree for branch 'feature-auth' at ../repo-name-feature-auth
```

### `gh worktree pr promote`

Promote a branch worktree to a PR worktree after creating a pull request.

```bash
# Promote current branch (auto-detects branch and PR number)
gh worktree pr promote

# Promote specific branch (auto-detects PR number)
gh worktree pr promote feature-auth

# Promote with explicit PR number
gh worktree pr promote feature-auth 1234
```

**Example Output:**
```
Promoted worktree for branch 'feature-auth' to PR #1234
Title: Add authentication system
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

### `gh worktree switch` (Unified Switcher)

Switch to any worktree (PR, branch, or main).

```bash
# Interactive selection from all worktrees
gh worktree switch

# Switch to specific PR worktree
gh worktree switch 1234

# Switch to specific branch worktree
gh worktree switch feature-auth

# Switch to main worktree
gh worktree switch main

# Shell mode (outputs path only)
gh worktree switch --shell
```

## Directory Structure

The extension creates worktrees in the parent directory of your current repository:

```
parent-directory/
├── my-repo/                    # Original repository
├── my-repo-pr1234/            # PR #1234 worktree
├── my-repo-pr5678/            # PR #5678 worktree
├── my-repo-feature-auth/      # Branch worktree for local development
└── my-repo-experiment-api/    # Branch worktree for local development
```

## Shell Integration

For the best experience, add these shell functions to your `~/.bashrc` or `~/.zshrc`:

```bash
# Unified worktree switcher (recommended - switches between all worktrees)
ghws() {
  local target=$(gh worktree switch --shell "$@")
  [ -n "$target" ] && cd "$target"
}

# Checkout and cd into new worktree (PR or branch)
ghwc() {
  local target=$(gh worktree pr checkout --shell "$@")
  [ -n "$target" ] && cd "$target"
}
```

**Usage:**
```bash
# Interactive selection and switch (all worktrees)
ghws

# Switch to specific PR
ghws 1234

# Switch to specific branch worktree
ghws feature-auth

# Interactive checkout and cd
ghwc

# Checkout specific PR and cd
ghwc 1234

# Create branch worktree and cd
ghwc --create feature-auth
```

## How It Works

1. **Worktree Creation**: Creates git worktrees in separate directories
   - PR worktrees: `../repo-name-pr{number}`
   - Branch worktrees: `../repo-name-{branch-name}`
2. **Branch Management**: Sets up proper remote tracking and handles cross-repository PRs similar to `gh pr checkout`
3. **Metadata Storage**: Uses git config to track worktree types (`pr` or `branch`)
4. **Promotion**: Converts branch worktrees to PR worktrees after PR creation
5. **Clean Removal**: Removes both worktree and branch when cleaning up

## Comparison with `gh pr checkout`

| Feature          | `gh pr checkout`   | `gh worktree pr checkout`   |
|------------------|--------------------|-----------------------------|
| Location         | Current directory  | Separate worktree directory |
| Multiple PRs     | Requires switching | Parallel development        |
| Branch conflicts | Possible           | Isolated per worktree       |

## Advanced Usage

### Local Development Workflow

```bash
# 1. Create a branch worktree for local development
gh worktree pr checkout --create feature-auth

# 2. Work on your feature...
cd ../my-repo-feature-auth
# ... make changes, commit ...

# 3. Create a pull request
gh pr create

# 4. Promote the branch worktree to PR worktree (auto-detects current branch)
gh worktree pr promote

# 5. Now it appears in PR worktree list
gh worktree pr list
```

### Working with Multiple Worktrees

```bash
# Checkout multiple PRs for parallel development
gh worktree pr checkout 1234
gh worktree pr checkout 5678

# Create branch worktrees for new features
gh worktree pr checkout --create feature-auth
gh worktree pr checkout --create experiment-api

# List all active worktrees
gh worktree pr list --all

# Switch between them quickly
ghws 1234           # Switch to PR 1234
ghws feature-auth   # Switch to branch worktree
ghws main           # Switch to main worktree
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

## Post-Creation Setup

Automatically run commands when creating new worktrees, such as copying configuration files or installing dependencies.

### Configuration

Create a `.gh-worktree.yml` file in your repository root:

```yaml
setup:
  run:
    - cp -r "$GH_WORKTREE_MAIN_DIR/.claude" .
    - cp "$GH_WORKTREE_MAIN_DIR/.env.local" . || true
    - pnpm install
```

### How It Works

- Commands run automatically after creating any worktree (PR or branch)
- Commands execute in the new worktree directory
- `GH_WORKTREE_MAIN_DIR` environment variable points to the main worktree
- Setup continues even if commands fail (shows warnings)
- Works correctly when creating worktrees from other worktrees

### Skipping Setup

Skip post-creation setup with the `--no-setup` flag:

```bash
gh worktree pr checkout 1234 --no-setup
gh worktree pr checkout --create feature-auth --no-setup
```

### Common Use Cases

**Copy configuration files:**
```yaml
setup:
  run:
    - cp "$GH_WORKTREE_MAIN_DIR/.env.local" .
    - cp -r "$GH_WORKTREE_MAIN_DIR/.vscode" .
```

**Install dependencies:**
```yaml
setup:
  run:
    - pnpm install
    - npm run build
```

**Initialize development environment:**
```yaml
setup:
  run:
    - cp "$GH_WORKTREE_MAIN_DIR/.env.example" .env
    - pnpm install
    - pnpm run db:migrate
```

**Claude Code setup:**
```yaml
setup:
  run:
    - cp -r "$GH_WORKTREE_MAIN_DIR/.claude" .
    - echo "Worktree ready for Claude Code!"
```

## Requirements

- [GitHub CLI](https://cli.github.com/) (gh)
- Git with worktree support (Git 2.5+)
- Access to the repository (for API calls)

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

This project is licensed under the MIT License.
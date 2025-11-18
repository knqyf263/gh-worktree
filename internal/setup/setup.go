package setup

import (
	"fmt"
	"os"
	"os/exec"
)

// RunSetup executes post-creation setup commands in the new worktree
func RunSetup(newWorktreePath, mainWorktreePath string) error {
	config, err := LoadConfig(mainWorktreePath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// If no setup commands are configured, skip
	if len(config.Setup.Run) == 0 {
		return nil
	}

	fmt.Println("→ Running post-creation setup...")

	var warnings []string

	for _, cmdStr := range config.Setup.Run {
		fmt.Printf("  ✓ %s\n", cmdStr)

		// Execute command in the new worktree directory with GH_WORKTREE_MAIN_DIR env var
		cmd := exec.Command("sh", "-c", cmdStr)
		cmd.Dir = newWorktreePath
		cmd.Env = append(os.Environ(), fmt.Sprintf("GH_WORKTREE_MAIN_DIR=%s", mainWorktreePath))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			warning := fmt.Sprintf("Command failed (exit %d): %s", cmd.ProcessState.ExitCode(), cmdStr)
			warnings = append(warnings, warning)
			fmt.Printf("  ⚠ %s\n", warning)
		}
	}

	if len(warnings) > 0 {
		fmt.Println("  ⚠ Setup completed with warnings")
	} else {
		fmt.Println("  ✓ Setup completed")
	}

	return nil
}

// ShouldRunSetup checks if setup should be executed
func ShouldRunSetup(mainWorktreePath string) bool {
	config, err := LoadConfig(mainWorktreePath)
	if err != nil {
		return false
	}
	return len(config.Setup.Run) > 0
}

// PrintSkippedMessage prints a message when setup is skipped
func PrintSkippedMessage() {
	fmt.Println("  (setup skipped)")
}

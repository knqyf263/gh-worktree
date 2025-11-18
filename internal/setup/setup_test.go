package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldRunSetup(t *testing.T) {
	tests := []struct {
		name       string
		configYAML string
		want       bool
	}{
		{
			name: "config with commands",
			configYAML: `setup:
  run:
    - echo "test"`,
			want: true,
		},
		{
			name:       "empty config",
			configYAML: `setup:`,
			want:       false,
		},
		{
			name:       "no config file",
			configYAML: "",
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			if tt.configYAML != "" {
				configPath := filepath.Join(tmpDir, ".gh-worktree.yml")
				if err := os.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
					t.Fatalf("failed to write test config: %v", err)
				}
			}

			got := ShouldRunSetup(tmpDir)
			if got != tt.want {
				t.Errorf("ShouldRunSetup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRunSetup_NoConfig(t *testing.T) {
	// Create temporary directories
	mainDir := t.TempDir()
	newDir := t.TempDir()

	// Run setup with no config file (should succeed without doing anything)
	err := RunSetup(newDir, mainDir)
	if err != nil {
		t.Errorf("RunSetup() with no config should not error, got: %v", err)
	}
}

func TestRunSetup_WithSimpleCommand(t *testing.T) {
	// Create temporary directories
	mainDir := t.TempDir()
	newDir := t.TempDir()

	// Create a simple config that creates a test file
	configYAML := `setup:
  run:
    - touch test-file.txt`
	configPath := filepath.Join(mainDir, ".gh-worktree.yml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Run setup
	err := RunSetup(newDir, mainDir)
	if err != nil {
		t.Errorf("RunSetup() error = %v", err)
	}

	// Verify the file was created in the new directory
	testFile := filepath.Join(newDir, "test-file.txt")
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Errorf("Expected file %s to be created", testFile)
	}
}

func TestRunSetup_WithEnvironmentVariable(t *testing.T) {
	// Create temporary directories
	mainDir := t.TempDir()
	newDir := t.TempDir()

	// Create a marker file in main directory
	markerFile := filepath.Join(mainDir, "main-marker.txt")
	if err := os.WriteFile(markerFile, []byte("marker"), 0644); err != nil {
		t.Fatalf("failed to write marker file: %v", err)
	}

	// Create config that uses GH_WORKTREE_MAIN_DIR
	configYAML := `setup:
  run:
    - cp "$GH_WORKTREE_MAIN_DIR/main-marker.txt" ./copied-marker.txt`
	configPath := filepath.Join(mainDir, ".gh-worktree.yml")
	if err := os.WriteFile(configPath, []byte(configYAML), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Run setup
	err := RunSetup(newDir, mainDir)
	if err != nil {
		t.Errorf("RunSetup() error = %v", err)
	}

	// Verify the file was copied to the new directory
	copiedFile := filepath.Join(newDir, "copied-marker.txt")
	if _, err := os.Stat(copiedFile); os.IsNotExist(err) {
		t.Errorf("Expected file %s to be created via environment variable", copiedFile)
	}
}

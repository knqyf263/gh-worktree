package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		configYAML  string
		wantErr     bool
		wantRunLen  int
		wantFirstCmd string
	}{
		{
			name: "valid config with multiple commands",
			configYAML: `setup:
  run:
    - echo "test1"
    - echo "test2"
    - pnpm install`,
			wantErr:      false,
			wantRunLen:   3,
			wantFirstCmd: `echo "test1"`,
		},
		{
			name: "valid config with single command",
			configYAML: `setup:
  run:
    - pnpm install`,
			wantErr:      false,
			wantRunLen:   1,
			wantFirstCmd: "pnpm install",
		},
		{
			name:       "empty config",
			configYAML: `setup:`,
			wantErr:    false,
			wantRunLen: 0,
		},
		{
			name:       "no config file (handled by LoadConfig returning empty)",
			configYAML: "",
			wantErr:    false,
			wantRunLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary directory
			tmpDir := t.TempDir()

			if tt.configYAML != "" {
				// Write the config file
				configPath := filepath.Join(tmpDir, ".gh-worktree.yml")
				if err := os.WriteFile(configPath, []byte(tt.configYAML), 0644); err != nil {
					t.Fatalf("failed to write test config: %v", err)
				}
			}

			// Load config
			config, err := LoadConfig(tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err != nil {
				return
			}

			// Check run commands length
			if len(config.Setup.Run) != tt.wantRunLen {
				t.Errorf("LoadConfig() got %d run commands, want %d", len(config.Setup.Run), tt.wantRunLen)
			}

			// Check first command if expected
			if tt.wantRunLen > 0 && config.Setup.Run[0] != tt.wantFirstCmd {
				t.Errorf("LoadConfig() first command = %q, want %q", config.Setup.Run[0], tt.wantFirstCmd)
			}
		})
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".gh-worktree.yml")

	// Write invalid YAML
	invalidYAML := `setup:
  run:
    - echo "test
    invalid yaml`
	if err := os.WriteFile(configPath, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	// Load config should fail
	_, err := LoadConfig(tmpDir)
	if err == nil {
		t.Error("LoadConfig() expected error for invalid YAML, got nil")
	}
}

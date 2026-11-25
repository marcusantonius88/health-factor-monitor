package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigPath(t *testing.T) {
	t.Run("explicit config has highest priority", func(t *testing.T) {
		got := resolveConfigPathFrom("/tmp/from-flag.json", "/tmp/from-env.json", t.TempDir(), "/tmp/config.json")
		if got != "/tmp/from-flag.json" {
			t.Errorf("resolveConfigPath() = %q, want explicit path", got)
		}
	})

	t.Run("environment config overrides discovery", func(t *testing.T) {
		got := resolveConfigPathFrom("", "/tmp/from-env.json", t.TempDir(), "/tmp/config.json")
		if got != "/tmp/from-env.json" {
			t.Errorf("resolveConfigPath() = %q, want environment path", got)
		}
	})

	t.Run("global config is discovered outside project", func(t *testing.T) {
		userConfigDir := t.TempDir()
		globalPath := filepath.Join(userConfigDir, "hf", "config.json")
		if err := os.MkdirAll(filepath.Dir(globalPath), 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if err := os.WriteFile(globalPath, []byte("{}"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		localPath := filepath.Join(t.TempDir(), "config.json")
		got := resolveConfigPathFrom("", "", userConfigDir, localPath)
		if got != globalPath {
			t.Errorf("resolveConfigPath() = %q, want %q", got, globalPath)
		}
	})

	t.Run("local config remains supported", func(t *testing.T) {
		localPath := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(localPath, []byte("{}"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		got := resolveConfigPathFrom("", "", filepath.Join(t.TempDir(), "missing-user-config"), localPath)
		if got != localPath {
			t.Errorf("resolveConfigPath() = %q, want local config.json", got)
		}
	})
}

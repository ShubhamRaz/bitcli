// Package config tests BitCLI configuration validation (expanded).
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfigIsValid(t *testing.T) {
	if err := Validate(DefaultConfig()); err != nil {
		t.Fatalf("default config should validate: %v", err)
	}
}

func TestValidateRejectsInvalidPort(t *testing.T) {
	cfg := DefaultConfig()
	cfg.API.Port = 70000
	if err := Validate(cfg); err == nil {
		t.Fatal("expected invalid port to fail")
	}
}

func TestValidateRejectsNegativePort(t *testing.T) {
	cfg := DefaultConfig()
	cfg.API.Port = -1
	if err := Validate(cfg); err == nil {
		t.Fatal("expected negative port to fail")
	}
}

func TestValidateRejectsZeroVersion(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Version = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected version 0 to fail")
	}
}

func TestValidateRejectsEmptyBackend(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultBackend = ""
	if err := Validate(cfg); err == nil {
		t.Fatal("expected empty default_backend to fail")
	}
}

func TestValidateRejectsNegativeTemperature(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.Temperature = -0.5
	if err := Validate(cfg); err == nil {
		t.Fatal("expected negative temperature to fail")
	}
}

func TestValidateRejectsTemperatureAbove2(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.Temperature = 2.1
	if err := Validate(cfg); err == nil {
		t.Fatal("expected temperature > 2 to fail")
	}
}

func TestValidateRejectsTopPAbove1(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.TopP = 1.5
	if err := Validate(cfg); err == nil {
		t.Fatal("expected top_p > 1 to fail")
	}
}

func TestValidateRejectsZeroContextLength(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Runtime.ContextLength = 0
	if err := Validate(cfg); err == nil {
		t.Fatal("expected context_length = 0 to fail")
	}
}

func TestValidateRejectsEmptyMirror(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Download.Mirror = ""
	if err := Validate(cfg); err == nil {
		t.Fatal("expected empty download mirror to fail")
	}
}

// Regression test: runtime.min_p must round-trip through Write/Load and be
// reachable via Get/Set. It was previously dropped from serialization, so a
// configured value silently vanished and `config get runtime.min_p` errored.
func TestMinPRoundTripsThroughFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	svc, err := NewService()
	if err != nil {
		t.Fatal(err)
	}
	cfg := DefaultConfig()
	cfg.Runtime.MinP = 0.11
	if err := svc.Write(cfgFile, cfg); err != nil {
		t.Fatalf("write config: %v", err)
	}
	loaded, _, err := svc.Load(cfgFile)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.Runtime.MinP != 0.11 {
		t.Fatalf("min_p should round-trip, got %v", loaded.Runtime.MinP)
	}

	if _, err := svc.Get(cfgFile, "runtime.min_p"); err != nil {
		t.Fatalf("config get runtime.min_p should work: %v", err)
	}
	if err := svc.Set(cfgFile, "runtime.min_p", 0.2); err != nil {
		t.Fatalf("config set runtime.min_p should work: %v", err)
	}
	after, _, err := svc.Load(cfgFile)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if after.Runtime.MinP != 0.2 {
		t.Fatalf("min_p should persist after Set, got %v", after.Runtime.MinP)
	}
}

// A custom config path must not relocate the data root (models, database,
// backends, logs stay under the default BitCLI home).
func TestCustomConfigPathKeepsDefaultDataPaths(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".bitcli"), 0o755); err != nil {
		t.Fatal(err)
	}
	svc, err := NewService()
	if err != nil {
		t.Fatal(err)
	}
	defaultPaths, err := DefaultPaths()
	if err != nil {
		t.Fatal(err)
	}

	custom := filepath.Join(t.TempDir(), "prod", "config.yaml")
	_, paths, err := svc.Load(custom)
	if err != nil {
		t.Fatalf("load with custom config path: %v", err)
	}
	if paths.ConfigFile != custom {
		t.Fatalf("config file should be the custom path, got %s", paths.ConfigFile)
	}
	if paths.ModelDir != defaultPaths.ModelDir {
		t.Fatalf("model dir should stay at the default root, got %s", paths.ModelDir)
	}
	if paths.DatabaseFile != defaultPaths.DatabaseFile {
		t.Fatalf("database should stay at the default root, got %s", paths.DatabaseFile)
	}
}

// Set must reject keys outside the schema instead of writing stray YAML.
func TestSetRejectsUnknownKey(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "config.yaml")
	svc, err := NewService()
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Write(cfgFile, DefaultConfig()); err != nil {
		t.Fatal(err)
	}
	if err := svc.Set(cfgFile, "bogus_key", "value"); err == nil {
		t.Fatal("expected Set to reject an unknown key")
	}
}

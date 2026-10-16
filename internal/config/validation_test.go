// Package config tests BitCLI configuration validation (expanded).
package config

import "testing"

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

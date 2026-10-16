// Package hardware detects local CPU, memory, and accelerator capabilities for BitCLI.
package hardware

import (
	"context"
	"os/exec"
	"runtime"
	"strings"
)

// Report contains detected hardware facts and recommendations.
type Report struct {
	OS                string   `json:"os"`
	Arch              string   `json:"arch"`
	CPUName           string   `json:"cpu_name"`
	CPUThreads        int      `json:"cpu_threads"`
	AVX2              bool     `json:"avx2"`
	AVX512            bool     `json:"avx512"`
	CUDA              bool     `json:"cuda"`
	Metal             bool     `json:"metal"`
	ROCm              bool     `json:"rocm"`
	RAMBytes          uint64   `json:"ram_bytes"`
	GPUs              []GPU    `json:"gpus"`
	RecommendedModel  string   `json:"recommended_model"`
	EstimatedRAMBytes uint64   `json:"estimated_ram_bytes"`
	EstimatedTokSec   string   `json:"estimated_tokens_sec"`
	Warnings          []string `json:"warnings"`
}

// GPU describes a detected accelerator.
type GPU struct {
	Name      string `json:"name"`
	Backend   string `json:"backend"`
	VRAMBytes uint64 `json:"vram_bytes"`
}

// Detector performs best-effort platform detection.
type Detector struct{}

// NewDetector creates a hardware detector.
func NewDetector() *Detector {
	return &Detector{}
}

// Detect returns best-effort hardware facts without failing for missing platform tools.
func (d *Detector) Detect(ctx context.Context) Report {
	report := Report{
		OS:         runtime.GOOS,
		Arch:       runtime.GOARCH,
		CPUThreads: runtime.NumCPU(),
		Metal:      runtime.GOOS == "darwin",
	}
	fillCPU(ctx, &report)
	fillMemory(ctx, &report)
	fillGPU(ctx, &report)
	ApplyRecommendation(&report)
	return report
}

func commandOutput(ctx context.Context, name string, args ...string) string {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}


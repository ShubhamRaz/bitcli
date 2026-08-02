// Package hardware detects local CPU, memory, and accelerator capabilities for BitCLI.
//go:build linux

package hardware

import (
	"context"
	"strconv"
	"strings"
)

func fillGPU(ctx context.Context, report *Report) {
	if out := commandOutput(ctx, "nvidia-smi", "--query-gpu=name,memory.total", "--format=csv,noheader,nounits"); out != "" {
		report.CUDA = true
		for _, line := range strings.Split(out, "\n") {
			parts := strings.Split(line, ",")
			if len(parts) >= 2 {
				mib, _ := strconv.ParseUint(strings.TrimSpace(parts[1]), 10, 64)
				report.GPUs = append(report.GPUs, GPU{Name: strings.TrimSpace(parts[0]), Backend: "cuda", VRAMBytes: mib * 1024 * 1024})
			}
		}
	}
	if out := commandOutput(ctx, "rocm-smi", "--showproductname"); out != "" {
		report.ROCm = true
		report.GPUs = append(report.GPUs, GPU{Name: firstNonEmptyLine(out), Backend: "rocm"})
	}
}

func firstNonEmptyLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line)
		}
	}
	return "unknown"
}


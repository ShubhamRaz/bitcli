// Package hardware detects local CPU, memory, and accelerator capabilities for BitCLI.
//go:build windows

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
	out := commandOutput(ctx, "wmic", "path", "win32_VideoController", "get", "name", "/value")
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name=") {
			name := strings.TrimSpace(strings.TrimPrefix(line, "Name="))
			if name != "" {
				report.GPUs = append(report.GPUs, GPU{Name: name, Backend: "unknown"})
			}
		}
	}
}


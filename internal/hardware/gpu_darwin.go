// Package hardware detects local CPU, memory, and accelerator capabilities for BitCLI.
//go:build darwin

package hardware

import (
	"context"
	"strings"
)

func fillGPU(ctx context.Context, report *Report) {
	report.Metal = true
	out := commandOutput(ctx, "system_profiler", "SPDisplaysDataType")
	name := "Apple GPU"
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Chipset Model:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "Chipset Model:"))
			break
		}
	}
	report.GPUs = append(report.GPUs, GPU{Name: name, Backend: "metal"})
}


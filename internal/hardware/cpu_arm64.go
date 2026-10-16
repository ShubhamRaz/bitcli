// Package hardware detects local CPU, memory, and accelerator capabilities for BitCLI.
package hardware

import (
	"context"
	"os"
	"runtime"
	"strings"
)

func fillCPU(ctx context.Context, report *Report) {
	// ARM64 always has NEON — noted in CPUName, no separate field yet.
	switch runtime.GOOS {
	case "linux":
		data, _ := os.ReadFile("/proc/cpuinfo")
		for _, line := range strings.Split(string(data), "\n") {
			lower := strings.ToLower(line)
			if strings.HasPrefix(lower, "model name") || strings.HasPrefix(lower, "hardware") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					report.CPUName = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	case "darwin":
		report.CPUName = commandOutput(ctx, "sysctl", "-n", "machdep.cpu.brand_string")
		if report.CPUName == "" {
			report.CPUName = commandOutput(ctx, "sysctl", "-n", "hw.model")
		}
	}
	if report.CPUName == "" {
		report.CPUName = "arm64"
	}
}


// Package hardware detects local CPU, memory, and accelerator capabilities for BitCLI.
package hardware

import (
	"context"
	"os"
	"runtime"
	"strings"
)

func fillCPU(ctx context.Context, report *Report) {
	switch runtime.GOOS {
	case "linux":
		data, _ := os.ReadFile("/proc/cpuinfo")
		text := strings.ToLower(string(data))
		report.AVX2 = strings.Contains(text, "avx2")
		report.AVX512 = strings.Contains(text, "avx512")
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.ToLower(line), "model name") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					report.CPUName = strings.TrimSpace(parts[1])
					break
				}
			}
		}
	case "darwin":
		report.CPUName = commandOutput(ctx, "sysctl", "-n", "machdep.cpu.brand_string")
		features := strings.ToLower(commandOutput(ctx, "sysctl", "-a"))
		report.AVX2 = strings.Contains(features, "avx2")
		report.AVX512 = strings.Contains(features, "avx512")
	case "windows":
		report.CPUName = commandOutput(ctx, "wmic", "cpu", "get", "Name", "/value")
		id := strings.ToLower(os.Getenv("PROCESSOR_IDENTIFIER"))
		report.AVX2 = strings.Contains(id, "avx2")
		report.AVX512 = strings.Contains(id, "avx512")
	}
	if report.CPUName == "" {
		report.CPUName = runtime.GOARCH
	}
}


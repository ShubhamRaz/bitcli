// Package hardware detects local CPU, memory, and accelerator capabilities for BitCLI.
package hardware

import (
	"context"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func fillMemory(ctx context.Context, report *Report) {
	switch runtime.GOOS {
	case "linux":
		data, _ := os.ReadFile("/proc/meminfo")
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "MemTotal:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					kib, _ := strconv.ParseUint(fields[1], 10, 64)
					report.RAMBytes = kib * 1024
				}
				return
			}
		}
	case "darwin":
		out := commandOutput(ctx, "sysctl", "-n", "hw.memsize")
		report.RAMBytes, _ = strconv.ParseUint(strings.TrimSpace(out), 10, 64)
	case "windows":
		out := commandOutput(ctx, "wmic", "ComputerSystem", "get", "TotalPhysicalMemory", "/value")
		for _, line := range strings.Split(out, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "TotalPhysicalMemory=") {
				report.RAMBytes, _ = strconv.ParseUint(strings.TrimPrefix(strings.TrimSpace(line), "TotalPhysicalMemory="), 10, 64)
			}
		}
	}
}


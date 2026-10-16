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
		// Use PowerShell Get-CimInstance instead of deprecated wmic.
		report.CPUName = windowsCPUName(ctx)
		// PROCESSOR_IDENTIFIER does NOT contain AVX feature flags,
		// so we probe the registry for CPU feature bits.
		report.AVX2 = windowsHasAVX2()
	}
	if report.CPUName == "" {
		report.CPUName = runtime.GOARCH
	}
}

// windowsCPUName uses PowerShell CIM to get the CPU brand string.
func windowsCPUName(ctx context.Context) string {
	out := commandOutput(ctx, "powershell", "-NoProfile", "-Command",
		"(Get-CimInstance Win32_Processor).Name")
	out = strings.TrimSpace(out)
	if out != "" {
		return out
	}
	// Fallback to wmic for older Windows versions.
	out = commandOutput(ctx, "wmic", "cpu", "get", "Name", "/value")
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name=") {
			return strings.TrimPrefix(line, "Name=")
		}
	}
	return ""
}

// windowsHasAVX2 reads the ProcessorFeatures registry to detect AVX2 support.
// Feature index 40 = PF_AVX2_INSTRUCTIONS_AVAILABLE on Windows 10+.
func windowsHasAVX2() bool {
	out := strings.TrimSpace(commandOutput(context.Background(), "powershell", "-NoProfile", "-Command",
		"(Get-CimInstance Win32_Processor).Description"))
	lower := strings.ToLower(out)
	// If Description mentions AVX2 (some builds do), use that.
	if strings.Contains(lower, "avx2") {
		return true
	}
	// Check via Win32 API using PowerShell Add-Type.
	check := commandOutput(context.Background(), "powershell", "-NoProfile", "-Command", `
Add-Type -MemberDefinition '[DllImport("kernel32.dll")] public static extern bool IsProcessorFeaturePresent(int feature);' -Name PF -Namespace Win32
[Win32.PF]::IsProcessorFeaturePresent(40)
`)
	return strings.TrimSpace(check) == "True"
}

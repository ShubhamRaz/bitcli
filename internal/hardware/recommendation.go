// Package hardware detects local CPU, memory, and accelerator capabilities for BitCLI.
package hardware

// ApplyRecommendation adds model and runtime recommendations to a report.
func ApplyRecommendation(report *Report) {
	report.RecommendedModel = "microsoft/BitNet-b1.58-2B-4T"
	report.EstimatedRAMBytes = 3 * 1024 * 1024 * 1024
	switch {
	case report.AVX512:
		report.EstimatedTokSec = "fast for local CPU inference"
	case report.AVX2:
		report.EstimatedTokSec = "good for local CPU inference"
	case report.Arch == "arm64":
		report.EstimatedTokSec = "good on supported ARM kernels"
	default:
		report.EstimatedTokSec = "limited; install optimized CPU support where available"
		report.Warnings = append(report.Warnings, "AVX2 or ARM64 support was not detected")
	}
	if report.RAMBytes > 0 && report.RAMBytes < report.EstimatedRAMBytes {
		report.Warnings = append(report.Warnings, "available RAM may be too low for the recommended model")
	}
	if (report.CUDA || report.Metal || report.ROCm) && len(report.GPUs) > 0 {
		report.Warnings = append(report.Warnings, "GPU detected; BitCLI enables GPU options only when the official BitNet backend exposes support")
	}
}


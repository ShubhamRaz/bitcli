// Package hardware tests hardware recommendation logic.
package hardware

import "testing"

func TestApplyRecommendationSetsModel(t *testing.T) {
	report := &Report{}
	ApplyRecommendation(report)
	if report.RecommendedModel == "" {
		t.Fatal("recommended model should not be empty")
	}
}

func TestApplyRecommendationSetsEstimatedRAM(t *testing.T) {
	report := &Report{}
	ApplyRecommendation(report)
	if report.EstimatedRAMBytes == 0 {
		t.Fatal("estimated RAM bytes should not be 0")
	}
}

func TestApplyRecommendationAVX512IsFast(t *testing.T) {
	report := &Report{AVX512: true}
	ApplyRecommendation(report)
	if report.EstimatedTokSec == "" {
		t.Fatal("EstimatedTokSec should be set")
	}
	// Should mention fast
	if report.EstimatedTokSec == "limited; install optimized CPU support where available" {
		t.Fatal("AVX512 should not result in limited estimate")
	}
}

func TestApplyRecommendationAVX2IsGood(t *testing.T) {
	report := &Report{AVX2: true}
	ApplyRecommendation(report)
	if report.EstimatedTokSec == "" {
		t.Fatal("EstimatedTokSec should be set")
	}
}

func TestApplyRecommendationWarnsOnLowRAM(t *testing.T) {
	report := &Report{
		RAMBytes: 1 * 1024 * 1024 * 1024, // 1 GiB
	}
	ApplyRecommendation(report)
	warned := false
	for _, w := range report.Warnings {
		if w != "" {
			warned = true
		}
	}
	if !warned {
		t.Fatal("expected a warning for low RAM")
	}
}

func TestApplyRecommendationNoWarningForSufficientRAM(t *testing.T) {
	report := &Report{
		RAMBytes: 16 * 1024 * 1024 * 1024, // 16 GiB
		AVX2:     true,
	}
	ApplyRecommendation(report)
	for _, w := range report.Warnings {
		if w == "available RAM may be too low for the recommended model" {
			t.Fatal("should not warn about RAM when there is plenty")
		}
	}
}

func TestApplyRecommendationGPUWarning(t *testing.T) {
	report := &Report{
		CUDA: true,
		GPUs: []GPU{{Name: "RTX 3080", Backend: "cuda"}},
	}
	ApplyRecommendation(report)
	warned := false
	for _, w := range report.Warnings {
		if w != "" {
			warned = true
		}
	}
	if !warned {
		t.Fatal("expected a GPU-related warning")
	}
}

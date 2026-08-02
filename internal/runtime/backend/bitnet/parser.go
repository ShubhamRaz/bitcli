// Package bitnet adapts BitCLI runtime requests to official Microsoft bitnet.cpp scripts.
package bitnet

import (
	"regexp"
	"strings"
)

var (
	promptStatsRegex = regexp.MustCompile(`\[\s*Prompt:\s*[\d\.]+\s*t/s\s*\|\s*Generation:\s*[\d\.]+\s*t/s\s*\]`)
	specialTokens    = []string{"<|begin_of_text|>", "<|end_of_text|>", "<|eot_id|>", "<s>", "</s>"}
)

// StreamFilter filters out backend runtime banners, metadata, prompt echoes, and trailing stats.
type StreamFilter struct {
	prompt  string
	started bool
	ended   bool
}

// NewStreamFilter creates a stream filter for the given prompt.
func NewStreamFilter(prompt string) *StreamFilter {
	return &StreamFilter{
		prompt: strings.TrimSpace(prompt),
	}
}

// Filter processes a line/chunk from backend stdout and returns the cleaned text to emit (or empty string if skipped).
func (f *StreamFilter) Filter(text string) string {
	if f.ended {
		return ""
	}

	cleaned := strings.ReplaceAll(text, "\r\n", "\n")

	// If it contains prompt stats footer, mark as ended
	if promptStatsRegex.MatchString(cleaned) {
		f.ended = true
		return ""
	}

	trimmed := strings.TrimSpace(cleaned)

	// Check if this line is header/banner noise
	if !f.started {
		if f.isHeaderNoise(trimmed) {
			return ""
		}
		// Check for prompt echo
		if f.isPromptEcho(trimmed) {
			return ""
		}
		if trimmed == "" {
			return ""
		}
		f.started = true
	}

	// Check for trailer/footer noise
	if f.isFooterNoise(trimmed) {
		f.ended = true
		return ""
	}

	// Clean out any embedded special tokens or stats
	result := cleaned
	for _, tok := range specialTokens {
		result = strings.ReplaceAll(result, tok, "")
	}
	result = promptStatsRegex.ReplaceAllString(result, "")

	return result
}

func (f *StreamFilter) isHeaderNoise(trimmed string) bool {
	if trimmed == "" {
		return true
	}
	// "Loading model..."
	if strings.HasPrefix(trimmed, "Loading model") || strings.HasPrefix(trimmed, "loading model") {
		return true
	}
	// ASCII / Unicode Art banner characters
	if strings.ContainsAny(trimmed, "█▄▀▌▐") {
		return true
	}
	// Metadata lines
	prefixes := []string{
		"build      :",
		"build:",
		"model      :",
		"model:",
		"ftype      :",
		"ftype:",
		"modalities :",
		"modalities:",
		"available commands:",
		"/exit",
		"/regen",
		"/clear",
		"/read",
		"/glob",
		"--no-conversation is not supported",
		"please use llama-completion",
		"create_tensor:",
		"done_getting_tensors:",
		"load_tensors:",
		"llama_context:",
		"llama_kv_cache:",
		"sched_reserve:",
		"graph_reserve:",
		"set_abort_callback:",
		"llama_perf_",
		"~llama_context:",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(trimmed, p) {
			return true
		}
	}
	return false
}

func (f *StreamFilter) isPromptEcho(trimmed string) bool {
	if trimmed == ">" {
		return true
	}
	if strings.HasPrefix(trimmed, "> ") {
		return true
	}
	if trimmed == "Assistant:" || trimmed == "User:" {
		return true
	}
	return false
}

func (f *StreamFilter) isFooterNoise(trimmed string) bool {
	if trimmed == "Exiting..." || trimmed == "Exiting" {
		return true
	}
	if trimmed == ">" {
		return true
	}
	if strings.HasPrefix(trimmed, "main: decoded") || strings.HasPrefix(trimmed, "llama_perf_") {
		return true
	}
	if promptStatsRegex.MatchString(trimmed) {
		return true
	}
	return false
}

// CleanToken normalizes backend output before it becomes a streamed token event.
func CleanToken(text string) string {
	cleaned := strings.ReplaceAll(text, "\r\n", "\n")
	for _, tok := range specialTokens {
		cleaned = strings.ReplaceAll(cleaned, tok, "")
	}
	return promptStatsRegex.ReplaceAllString(cleaned, "")
}

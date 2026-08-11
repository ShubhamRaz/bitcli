// Package bitnet adapts BitCLI runtime requests to official Microsoft bitnet.cpp scripts.
package bitnet

import (
	"regexp"
	"strings"
)

var (
	promptStatsRegex = regexp.MustCompile(`\[\s*Prompt:\s*[\d\.]+\s*t/s\s*\|\s*Generation:\s*[\d\.]+\s*t/s\s*\]`)
	specialTokens    = []string{
		"<|begin_of_text|>",
		"<|end_of_text|>",
		"<|eot_id|>",
		"<|start_header_id|>",
		"<|end_header_id|>",
		"<s>",
		"</s>",
	}
	// eotToken signals the model has finished its assistant turn.
	eotToken = "<|eot_id|>"
)

// StreamFilter filters out backend runtime banners, metadata, prompt echoes, and trailing stats.
type StreamFilter struct {
	promptLines     map[string]bool
	seenLines       map[string]int
	started         bool
	firstToken      bool
	ended           bool
	totalEmitted    int
	consecutiveQs   int      // consecutive lines ending with '?'
	recentLines     []string // rolling window for similarity detection
}

func normalizeLine(s string) string {
	s = strings.TrimSpace(s)
	for _, tok := range specialTokens {
		s = strings.ReplaceAll(s, tok, "")
	}
	s = strings.TrimPrefix(s, "> ")
	s = strings.TrimPrefix(s, ">")
	return strings.TrimSpace(s)
}

// NewStreamFilter creates a stream filter for the given prompt.
func NewStreamFilter(prompt string) *StreamFilter {
	lines := make(map[string]bool)
	for _, line := range strings.Split(prompt, "\n") {
		norm := normalizeLine(line)
		if norm != "" {
			lines[norm] = true
		}
	}
	return &StreamFilter{
		promptLines: lines,
		seenLines:   make(map[string]int),
		firstToken:  true,
		recentLines: make([]string, 0, 16),
	}
}

// Filter processes a line/chunk from backend stdout and returns the cleaned text to emit (or empty string if skipped).
func (f *StreamFilter) Filter(text string) string {
	if f.ended {
		return ""
	}

	cleaned := strings.ReplaceAll(text, "\r\n", "\n")
	trimmed := strings.TrimSpace(cleaned)

	// Check if this line is header/banner/prompt-echo noise
	if !f.started {
		if f.isHeaderNoise(trimmed) {
			return ""
		}
		if f.isPromptEcho(trimmed) {
			return ""
		}
		if trimmed == "" || trimmed == ":" {
			return ""
		}
		f.started = true
	}

	// Once streaming has started, check for end-of-turn token
	if strings.Contains(cleaned, eotToken) {
		before, _, _ := strings.Cut(cleaned, eotToken)
		f.ended = true
		for _, tok := range specialTokens {
			before = strings.ReplaceAll(before, tok, "")
		}
		return before
	}

	// Check for trailer/footer noise or timing stats
	if f.isFooterNoise(trimmed) || promptStatsRegex.MatchString(cleaned) {
		f.ended = true
		return ""
	}

	// Anti-loop protection: if the model begins repeating identical sentences/lines, stop cleanly.
	if trimmed != "" && len(trimmed) > 8 {
		f.seenLines[trimmed]++
		if f.seenLines[trimmed] >= 2 {
			f.ended = true
			return ""
		}
	}

	// Question-flood detection: if the model keeps asking questions instead of answering, stop.
	if trimmed != "" {
		if strings.HasSuffix(trimmed, "?") {
			f.consecutiveQs++
		} else {
			f.consecutiveQs = 0
		}
		if f.consecutiveQs >= 3 {
			f.ended = true
			return ""
		}
	}

	// Similarity-based repetition: detect when recent lines share long common substrings.
	if trimmed != "" && len(trimmed) > 20 {
		f.recentLines = append(f.recentLines, trimmed)
		if len(f.recentLines) > 10 {
			f.recentLines = f.recentLines[len(f.recentLines)-10:]
		}
		if f.hasSimilarRepetition(trimmed) {
			f.ended = true
			return ""
		}
	}

	// Clean out any embedded special tokens or stats
	result := cleaned
	for _, tok := range specialTokens {
		result = strings.ReplaceAll(result, tok, "")
	}
	result = promptStatsRegex.ReplaceAllString(result, "")

	if f.firstToken {
		result = strings.TrimPrefix(result, ": ")
		result = strings.TrimPrefix(result, ":")
		result = strings.TrimLeft(result, " ")
		if result != "" {
			f.firstToken = false
		}
	}

	f.totalEmitted += len(result)
	return result
}

// hasSimilarRepetition checks if the current line is very similar to multiple recent lines,
// indicating the model is rehashing the same idea with minor rewording.
func (f *StreamFilter) hasSimilarRepetition(current string) bool {
	if len(f.recentLines) < 4 {
		return false
	}
	// Count how many recent lines share a long common substring with the current line.
	similarCount := 0
	currentLower := strings.ToLower(current)
	for i := 0; i < len(f.recentLines)-1; i++ {
		pastLower := strings.ToLower(f.recentLines[i])
		if longestCommonSubstringLen(currentLower, pastLower) >= 30 {
			similarCount++
		}
	}
	return similarCount >= 3
}

// longestCommonSubstringLen returns the length of the longest common substring between a and b.
// Uses a sliding window approach bounded to avoid excessive computation.
func longestCommonSubstringLen(a, b string) int {
	if len(a) > 200 {
		a = a[:200]
	}
	if len(b) > 200 {
		b = b[:200]
	}
	best := 0
	for i := 0; i < len(a); i++ {
		for j := 0; j < len(b); j++ {
			k := 0
			for i+k < len(a) && j+k < len(b) && a[i+k] == b[j+k] {
				k++
			}
			if k > best {
				best = k
				if best >= 30 {
					return best // Early exit, threshold met
				}
			}
		}
	}
	return best
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
	if trimmed == ">" || trimmed == "> " {
		return true
	}
	norm := normalizeLine(trimmed)
	if norm == "" {
		return true
	}
	if f.promptLines != nil && f.promptLines[norm] {
		return true
	}
	// Common prompt turn labels echoed by llama-cli
	if norm == "Assistant:" || norm == "Assistant" ||
		norm == "User:" || norm == "User" ||
		norm == "Human:" || norm == "Human" ||
		norm == "System:" || norm == "System" ||
		norm == "BITNETAssistant:" || norm == "BITNETAssistant" {
		return true
	}
	if strings.HasPrefix(trimmed, "> ") {
		return true
	}
	return false
}

func (f *StreamFilter) isFooterNoise(trimmed string) bool {
	if trimmed == "Exiting..." || trimmed == "Exiting" || trimmed == ">" || trimmed == "> " {
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

// Package bitnet adapts BitCLI runtime requests to official Microsoft bitnet.cpp scripts.
package bitnet

import "strings"

// CleanToken normalizes backend output before it becomes a streamed token event.
func CleanToken(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}


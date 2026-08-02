// Package utils contains shared helpers that are intentionally small and dependency-light.
package utils

import "runtime"

// Platform describes the OS and architecture running BitCLI.
type Platform struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// CurrentPlatform returns the Go runtime platform identifiers.
func CurrentPlatform() Platform {
	return Platform{OS: runtime.GOOS, Arch: runtime.GOARCH}
}


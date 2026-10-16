// Package setup provides dependency detection, portable tool installation,
// and shell environment activation for the BitCLI self-contained setup flow.
package setup

import (
	"os/exec"
	"path/filepath"
	"runtime"
)

// ToolStatus describes the presence and resolved path of a required tool.
type ToolStatus struct {
	Name    string
	Path    string // empty when missing
	Bundled bool   // true when found inside ~/.bitcli/tools/
	Missing bool
}

// Report holds the detection result for every required tool.
type Report struct {
	Git    ToolStatus
	CMake  ToolStatus
	Ninja  ToolStatus
	Clang  ToolStatus
	UV     ToolStatus
	Python ToolStatus
}

// Detector checks for required build tools, preferring bundled portables
// inside bitcliHome/tools/ over system-wide installations.
type Detector struct {
	bitcliHome string // e.g. ~/.bitcli
}

// NewDetector returns a Detector rooted at the given BitCLI home directory.
func NewDetector(bitcliHome string) *Detector {
	return &Detector{bitcliHome: bitcliHome}
}

// Detect probes all required tools and returns a Report.
func (d *Detector) Detect() Report {
	return Report{
		Git:    d.probe("git", ""),
		CMake:  d.probe("cmake", filepath.Join(d.bitcliHome, "tools", "cmake"), filepath.Join(d.bitcliHome, "tools", "cmake", "bin")),
		Ninja:  d.probe("ninja", filepath.Join(d.bitcliHome, "tools", "ninja")),
		Clang:  d.probe("clang", filepath.Join(d.bitcliHome, "tools", "clang", "bin"), filepath.Join(d.bitcliHome, "tools", "clang"), filepath.Join(d.bitcliHome, "tools", "llvm", "bin")),
		UV:     d.probe(uvBinary(), filepath.Join(d.bitcliHome, "tools", "uv")),
		Python: d.probe("python3", ""),
	}
}

// probe tries to resolve a binary first in candidateDirs (if non-empty), then on PATH.
func (d *Detector) probe(name string, candidateDirs ...string) ToolStatus {
	s := ToolStatus{Name: name}

	// Try bundled locations first.
	for _, dir := range candidateDirs {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, execName(name))
		if path, err := exec.LookPath(candidate); err == nil {
			s.Path = path
			s.Bundled = true
			return s
		}
	}

	// Fall back to system PATH.
	if path, err := exec.LookPath(name); err == nil {
		s.Path = path
		return s
	}

	// Windows fallback: python may be exposed as "python" not "python3".
	if name == "python3" && runtime.GOOS == "windows" {
		if path, err := exec.LookPath("python"); err == nil {
			s.Name = "python"
			s.Path = path
			return s
		}
	}

	s.Missing = true
	return s
}

// uvBinary returns the uv executable name for the current platform.
func uvBinary() string {
	return execName("uv")
}

// execName appends .exe on Windows.
func execName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// AllPresent returns true when every required tool is found.
func (r Report) AllPresent() bool {
	return !r.Git.Missing && !r.CMake.Missing && !r.Ninja.Missing && !r.Clang.Missing && !r.UV.Missing
}

// Missing returns a slice of ToolStatus for tools that were not found.
func (r Report) Missing() []ToolStatus {
	var out []ToolStatus
	for _, s := range []ToolStatus{r.Git, r.CMake, r.Ninja, r.Clang, r.UV} {
		if s.Missing {
			out = append(out, s)
		}
	}
	return out
}

// Package setup provides dependency detection, portable tool installation,
// and shell environment activation for the BitCLI self-contained setup flow.
package setup

import "runtime"

// toolSpec describes where to download a portable binary for each platform.
type toolSpec struct {
	Name    string
	URL     string // direct download URL
	Archive string // "zip" | "tar.gz" | "exe"
	BinDir  string // sub-path inside extracted archive that contains the binary
}

// ToolURLs returns the appropriate portable binary URL for cmake, llvm/clang, and uv
// based on the current OS and CPU architecture.
func ToolURLs() map[string]toolSpec {
	switch runtime.GOOS {
	case "windows":
		return windowsSpecs()
	case "darwin":
		return darwinSpecs()
	default:
		return linuxSpecs()
	}
}

func windowsSpecs() map[string]toolSpec {
	return map[string]toolSpec{
		"cmake": {
			Name:    "cmake",
			URL:     "https://github.com/Kitware/CMake/releases/download/v3.31.7/cmake-3.31.7-windows-x86_64.zip",
			Archive: "zip",
			BinDir:  "cmake-3.31.7-windows-x86_64/bin",
		},
		"clang": {
			Name:    "clang",
			URL:     "https://github.com/mstorsjo/llvm-mingw/releases/download/20241119/llvm-mingw-20241119-ucrt-x86_64.zip",
			Archive: "zip",
			BinDir:  "llvm-mingw-20241119-ucrt-x86_64",
		},
		"uv": {
			Name:    "uv",
			URL:     "https://github.com/astral-sh/uv/releases/download/0.7.13/uv-x86_64-pc-windows-msvc.zip",
			Archive: "zip",
			BinDir:  "uv-x86_64-pc-windows-msvc",
		},
	}
}

func darwinSpecs() map[string]toolSpec {
	arch := "x86_64"
	if runtime.GOARCH == "arm64" {
		arch = "arm64"
	}
	uvArch := "x86_64-apple-darwin"
	if runtime.GOARCH == "arm64" {
		uvArch = "aarch64-apple-darwin"
	}
	cmakeArch := "macos-universal"
	return map[string]toolSpec{
		"cmake": {
			Name:    "cmake",
			URL:     "https://github.com/Kitware/CMake/releases/download/v3.31.7/cmake-3.31.7-" + cmakeArch + ".tar.gz",
			Archive: "tar.gz",
			BinDir:  "cmake-3.31.7-" + cmakeArch + "/CMake.app/Contents/bin",
		},
		"clang": {
			Name:    "clang",
			URL:     "https://github.com/llvm/llvm-project/releases/download/llvmorg-19.1.7/clang+llvm-19.1.7-" + arch + "-apple-macosx11.0.tar.xz",
			Archive: "tar.gz",
			BinDir:  "clang+llvm-19.1.7-" + arch + "-apple-macosx11.0/bin",
		},
		"uv": {
			Name:    "uv",
			URL:     "https://github.com/astral-sh/uv/releases/download/0.7.13/uv-" + uvArch + ".tar.gz",
			Archive: "tar.gz",
			BinDir:  "uv-" + uvArch,
		},
	}
}

func linuxSpecs() map[string]toolSpec {
	arch := "x86_64"
	uvArch := "x86_64-unknown-linux-gnu"
	if runtime.GOARCH == "arm64" {
		arch = "aarch64"
		uvArch = "aarch64-unknown-linux-gnu"
	}
	return map[string]toolSpec{
		"cmake": {
			Name:    "cmake",
			URL:     "https://github.com/Kitware/CMake/releases/download/v3.31.7/cmake-3.31.7-linux-" + arch + ".tar.gz",
			Archive: "tar.gz",
			BinDir:  "cmake-3.31.7-linux-" + arch + "/bin",
		},
		"clang": {
			Name:    "clang",
			URL:     "https://github.com/llvm/llvm-project/releases/download/llvmorg-19.1.7/clang+llvm-19.1.7-" + arch + "-linux-gnu.tar.xz",
			Archive: "tar.gz",
			BinDir:  "clang+llvm-19.1.7-" + arch + "-linux-gnu/bin",
		},
		"uv": {
			Name:    "uv",
			URL:     "https://github.com/astral-sh/uv/releases/download/0.7.13/uv-" + uvArch + ".tar.gz",
			Archive: "tar.gz",
			BinDir:  "uv-" + uvArch,
		},
	}
}

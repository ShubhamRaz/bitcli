// Package setup provides dependency detection, portable tool installation,
// and shell environment activation for the BitCLI self-contained setup flow.
package setup

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

// ToolInstaller downloads and extracts portable tool binaries into
// bitcliHome/tools/<tool>/ with a progress bar displayed to out.
type ToolInstaller struct {
	BitcliHome string
	Out        io.Writer
}

// NewToolInstaller returns an installer targeting the given BitCLI home.
func NewToolInstaller(bitcliHome string, out io.Writer) *ToolInstaller {
	if out == nil {
		out = io.Discard
	}
	return &ToolInstaller{BitcliHome: bitcliHome, Out: out}
}

// Install downloads and installs the portable binary for the named tool
// (one of "cmake", "clang", "uv") if it is not already present.
// Returns the directory that should be prepended to PATH.
func (t *ToolInstaller) Install(ctx context.Context, toolName string) (string, error) {
	specs := ToolURLs()
	spec, ok := specs[toolName]
	if !ok {
		return "", fmt.Errorf("no portable binary available for %q on this platform", toolName)
	}

	destBin := filepath.Join(t.BitcliHome, "tools", toolName)

	// Check if already installed.
	binary := filepath.Join(destBin, execName(toolName))
	if _, err := os.Stat(binary); err == nil {
		fmt.Fprintf(t.Out, "  ✓ %s already installed at %s\n", toolName, destBin)
		return destBin, nil
	}

	fmt.Fprintf(t.Out, "  ↓ Downloading %s...\n", toolName)

	// Download into a temp file.
	tmpFile, err := os.CreateTemp("", "bitcli-tool-*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if err := t.download(ctx, spec.URL, tmpFile); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("download %s: %w", toolName, err)
	}
	tmpFile.Close()

	// Extract.
	if err := os.MkdirAll(filepath.Join(t.BitcliHome, "tools", toolName+"_extract"), 0o755); err != nil {
		return "", err
	}
	extractDir := filepath.Join(t.BitcliHome, "tools", toolName+"_extract")
	defer os.RemoveAll(extractDir)

	fmt.Fprintf(t.Out, "  ⚙ Extracting %s...\n", toolName)

	switch spec.Archive {
	case "zip":
		if err := extractZip(tmpPath, extractDir); err != nil {
			return "", fmt.Errorf("extract %s: %w", toolName, err)
		}
	case "tar.gz", "tar.xz":
		if err := extractTar(ctx, tmpPath, extractDir); err != nil {
			return "", fmt.Errorf("extract %s: %w", toolName, err)
		}
	case "exe":
		// LLVM Windows self-extracting installer: run with /S /D=<dest>
		if err := t.runSilentInstaller(ctx, tmpPath, destBin); err != nil {
			return "", fmt.Errorf("install %s: %w", toolName, err)
		}
		fmt.Fprintf(t.Out, "  ✓ %s installed\n", toolName)
		return destBin, nil
	default:
		return "", fmt.Errorf("unsupported archive format %q", spec.Archive)
	}

	// Move the bin dir into place.
	srcBin := filepath.Join(extractDir, filepath.FromSlash(spec.BinDir))
	if err := os.MkdirAll(filepath.Dir(destBin), 0o755); err != nil {
		return "", err
	}
	if err := os.Rename(srcBin, destBin); err != nil {
		// Rename across volumes may fail — fall back to copy.
		if err2 := copyDir(srcBin, destBin); err2 != nil {
			return "", fmt.Errorf("install %s: %w", toolName, err2)
		}
	}

	fmt.Fprintf(t.Out, "  ✓ %s installed at %s\n", toolName, destBin)
	return destBin, nil
}

// download streams url into dst with an mpb progress bar.
func (t *ToolInstaller) download(ctx context.Context, url string, dst *os.File) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "bitcli/0.1")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s", resp.Status)
	}

	total := resp.ContentLength
	progress := mpb.New(mpb.WithOutput(t.Out))
	name := url[strings.LastIndex(url, "/")+1:]
	bar := progress.AddBar(total,
		mpb.PrependDecorators(
			decor.Name("    "+name+" ", decor.WCSyncWidthR),
			decor.CountersKibiByte("% .2f / % .2f"),
		),
		mpb.AppendDecorators(decor.Percentage()),
	)

	_, err = io.Copy(dst, bar.ProxyReader(resp.Body))
	bar.SetTotal(total, true)
	progress.Wait()
	return err
}

// runSilentInstaller runs a Windows NSIS/LLVM self-extracting .exe silently.
func (t *ToolInstaller) runSilentInstaller(ctx context.Context, exe, dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	// LLVM uses NSIS: /S = silent, /D = absolute destination (must be last flag).
	// #nosec G204 — controlled internal installer path.
	cmd := newCommand(ctx, exe, "/S", "/D="+filepath.ToSlash(dest))
	cmd.Stdout = t.Out
	cmd.Stderr = t.Out
	return cmd.Run()
}

// extractZip extracts a .zip archive into destDir.
func extractZip(src, destDir string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if err := extractZipEntry(f, destDir); err != nil {
			return err
		}
	}
	return nil
}

func extractZipEntry(f *zip.File, destDir string) error {
	// Sanitize path traversal.
	dest := filepath.Join(destDir, filepath.FromSlash(f.Name))
	if !strings.HasPrefix(dest, filepath.Clean(destDir)+string(os.PathSeparator)) && dest != filepath.Clean(destDir) {
		return fmt.Errorf("zip entry %q would escape destination directory", f.Name)
	}
	if f.FileInfo().IsDir() {
		return os.MkdirAll(dest, f.Mode())
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, rc)
	return err
}

// extractTar extracts .tar.gz or .tar.xz using the system tar command.
func extractTar(ctx context.Context, src, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	cmd := newCommand(ctx, "tar", "-xf", src, "-C", destDir)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

// copyDir recursively copies src directory to dst.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		return copyFile(path, target, info.Mode())
	})
}

func copyFile(src, dst string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

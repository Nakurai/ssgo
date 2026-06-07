package template

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	// maxArchiveSize caps the total size of a downloaded archive.
	maxArchiveSize = 50 << 20 // 50 MiB
	// maxFileSize caps the uncompressed size of any single extracted file.
	maxFileSize = 10 << 20 // 10 MiB
)

// Fetch retrieves a template style from source and writes it into destDir.
// source may be an http(s) URL or local path to a .zip archive, or a local
// directory containing the template's .html files. After writing, Fetch
// verifies that base.html is present so callers fail fast on bad sources.
func Fetch(source, destDir string) error {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	var err error
	switch {
	case strings.HasPrefix(source, "http://"), strings.HasPrefix(source, "https://"):
		err = fetchURL(source, destDir)
	case strings.HasSuffix(strings.ToLower(source), ".zip"):
		err = fetchZipFile(source, destDir)
	default:
		err = fetchDir(source, destDir)
	}
	if err != nil {
		return err
	}

	if _, err := os.Stat(filepath.Join(destDir, "base.html")); err != nil {
		return fmt.Errorf("invalid template from %s: base.html not found", source)
	}
	return nil
}

// StyleName derives a default style name from a source path or URL by taking
// the final path element without any ".zip" suffix or query/fragment.
func StyleName(source string) string {
	s := source
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	s = strings.ReplaceAll(s, "\\", "/")
	s = strings.TrimRight(s, "/")
	base := strings.TrimSuffix(path.Base(s), ".zip")
	if base == "" || base == "." || base == "/" {
		return "default"
	}
	return base
}

func fetchURL(url, destDir string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxArchiveSize+1))
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	if int64(len(data)) > maxArchiveSize {
		return fmt.Errorf("download %s: archive exceeds %d bytes", url, maxArchiveSize)
	}
	return extractZip(bytes.NewReader(data), int64(len(data)), destDir)
}

func fetchZipFile(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip %s: %w", zipPath, err)
	}
	defer r.Close()
	return extractFiles(r.File, destDir)
}

func extractZip(r io.ReaderAt, size int64, destDir string) error {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return fmt.Errorf("read zip: %w", err)
	}
	return extractFiles(zr.File, destDir)
}

func extractFiles(files []*zip.File, destDir string) error {
	prefix := commonTopDir(files)
	for _, f := range files {
		name := strings.TrimPrefix(strings.TrimPrefix(f.Name, prefix), "/")
		if name == "" {
			continue
		}

		dest := filepath.Join(destDir, filepath.FromSlash(name))
		if !withinDir(destDir, dest) {
			return fmt.Errorf("unsafe path in archive: %q", f.Name)
		}

		info := f.FileInfo()
		if info.IsDir() {
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return err
			}
			continue
		}
		if !info.Mode().IsRegular() {
			continue // skip symlinks, devices, etc.
		}
		if f.UncompressedSize64 > maxFileSize {
			return fmt.Errorf("file %q exceeds %d bytes", f.Name, maxFileSize)
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := writeZipEntry(f, dest); err != nil {
			return err
		}
	}
	return nil
}

func writeZipEntry(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, io.LimitReader(rc, maxFileSize)); err != nil {
		return err
	}
	return nil
}

func fetchDir(srcDir, destDir string) error {
	info, err := os.Stat(srcDir)
	if err != nil {
		return fmt.Errorf("template source %q: %w", srcDir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("template source %q is not a directory or .zip archive", srcDir)
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(srcDir, e.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(destDir, e.Name()), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// commonTopDir returns the single top-level directory shared by every entry
// (e.g. "minimal/"), or "" if entries live at the archive root or under
// differing top-level directories. This lets a zip wrap its files in a folder
// or not, transparently.
func commonTopDir(files []*zip.File) string {
	var top string
	for _, f := range files {
		name := strings.TrimPrefix(f.Name, "/")
		if name == "" {
			continue
		}
		i := strings.Index(name, "/")
		if i < 0 {
			return "" // a file sits at the root → no common wrapper
		}
		dir := name[:i]
		switch {
		case top == "":
			top = dir
		case top != dir:
			return ""
		}
	}
	if top == "" {
		return ""
	}
	return top + "/"
}

// withinDir reports whether target stays inside base, guarding against
// Zip-Slip path traversal from crafted archive entries.
func withinDir(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

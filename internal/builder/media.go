package builder

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var localRefRe = regexp.MustCompile(`(?i)(src|href)="([^"]*)"`)

// processMedia rewrites local references in rawHTML. Links to other content
// pages (.md files) are rewritten to that page's clean route; all other local
// references are treated as media and copied into outDir/assets/ (deduped by
// content hash). srcDir is the directory containing the source .md file, and
// routes maps each page's absolute source path to its public route.
func processMedia(rawHTML, srcDir, assetsOutDir string, routes map[string]string) (string, error) {
	var err error
	result := localRefRe.ReplaceAllStringFunc(rawHTML, func(match string) string {
		if err != nil {
			return match
		}
		parts := localRefRe.FindStringSubmatch(match)
		if len(parts) < 3 {
			return match
		}
		attr, ref := parts[1], parts[2]
		if isExternal(ref) {
			return match
		}

		// A reference to another content page: rewrite to that page's route
		// instead of copying the raw .md file into assets. Any #fragment or
		// ?query is preserved and re-appended to the route.
		refPath, suffix := splitRefSuffix(ref)
		if strings.EqualFold(filepath.Ext(refPath), ".md") {
			target, resolveErr := resolvePath(refPath, srcDir)
			if resolveErr != nil {
				return match
			}
			if route, ok := routes[filepath.Clean(target)]; ok {
				return attr + `="` + route + suffix + `"`
			}
			log.Printf("broken link: %q in %s does not resolve to a page", ref, srcDir)
			return match
		}

		srcPath, resolveErr := resolvePath(ref, srcDir)
		if resolveErr != nil {
			return match
		}
		if _, statErr := os.Stat(srcPath); statErr != nil {
			return match
		}

		newRef, copyErr := copyToAssets(srcPath, assetsOutDir)
		if copyErr != nil {
			err = copyErr
			return match
		}
		return attr + `="` + newRef + `"`
	})
	return result, err
}

// resolveAsset turns a configured asset path (logo, favicon, …) into a public
// URL reference. An empty path yields "". References that are already usable as
// URLs — external (http(s), //, data:) or site-absolute (leading "/") — pass
// through unchanged. Otherwise the value is treated as a file path relative to
// projectDir (where ssg.json lives), or used as given when absolute, and copied
// into the output assets directory.
func resolveAsset(ref, projectDir, assetsDir string) (string, error) {
	if ref == "" {
		return "", nil
	}
	if isExternal(ref) || strings.HasPrefix(ref, "/") {
		return ref, nil
	}
	srcPath := ref
	if !filepath.IsAbs(srcPath) {
		srcPath = filepath.Join(projectDir, ref)
	}
	if _, err := os.Stat(srcPath); err != nil {
		return "", fmt.Errorf("asset file %q: %w", ref, err)
	}
	return copyToAssets(srcPath, assetsDir)
}

func isExternal(ref string) bool {
	lower := strings.ToLower(ref)
	return strings.HasPrefix(lower, "http://") ||
		strings.HasPrefix(lower, "https://") ||
		strings.HasPrefix(lower, "//") ||
		strings.HasPrefix(ref, "#") ||
		strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "data:")
}

// splitRefSuffix separates a reference into its path and any trailing
// #fragment or ?query portion (whichever comes first).
func splitRefSuffix(ref string) (path, suffix string) {
	if i := strings.IndexAny(ref, "#?"); i != -1 {
		return ref[:i], ref[i:]
	}
	return ref, ""
}

func resolvePath(ref, srcDir string) (string, error) {
	if filepath.IsAbs(ref) {
		return ref, nil
	}
	return filepath.Join(srcDir, ref), nil
}

// copyToAssets copies src into assetsDir named by <sha256[:16]>.<ext>,
// returning the public reference /assets/<name>. Skips copy if already present.
func copyToAssets(srcPath, assetsDir string) (string, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	data, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	h.Write(data)
	hash := fmt.Sprintf("%x", h.Sum(nil))[:16]

	ext := filepath.Ext(srcPath)
	name := hash + ext
	destPath := filepath.Join(assetsDir, name)

	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		if err := os.MkdirAll(assetsDir, 0755); err != nil {
			return "", err
		}
		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return "", err
		}
	}
	return "/assets/" + name, nil
}

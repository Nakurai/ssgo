package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Page represents one parsed .md file.
type Page struct {
	SrcPath string
	SrcDir  string
	Meta    Meta

	BodyHTML  string
	PlainText string
	Route     string

	// Typed front-matter fields
	InMenu bool
	Status string // "draft" | "published"
	Date   time.Time
	Tags   []string

	// Set by the builder (not from content)
	IsDraft  bool
	IsFuture bool
}

func (p *Page) Type() string {
	t := p.Meta.str("type")
	if t == "" {
		return "default"
	}
	return t
}

func (p *Page) Title() string {
	if t := p.Meta.str("title"); t != "" {
		return t
	}
	base := filepath.Base(p.SrcPath)
	return strings.TrimSuffix(base, ".md")
}

func (p *Page) Description() string { return p.Meta.str("description") }

// Excerpt returns a short plain-text summary of the page: the "description"
// front matter if present, otherwise a truncated form of the rendered body.
func (p *Page) Excerpt() string {
	if d := p.Description(); d != "" {
		return d
	}
	return excerpt(p.PlainText, 200)
}

// Load parses a single .md file. contentDir is the root content/ folder.
func Load(srcPath, contentDir string) (*Page, error) {
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", srcPath, err)
	}

	meta, body, err := splitFrontMatter(raw)
	if err != nil {
		return nil, fmt.Errorf("parse front matter %s: %w", srcPath, err)
	}

	bodyHTML, err := renderMarkdown(body)
	if err != nil {
		return nil, fmt.Errorf("render markdown %s: %w", srcPath, err)
	}

	rel, err := filepath.Rel(contentDir, srcPath)
	if err != nil {
		return nil, err
	}

	// Typed fields
	inMenu := meta.str("in-menu") == "yes"

	status := meta.str("status")
	if status == "" {
		status = "draft"
	}

	var date time.Time
	date, err = meta.date()
	if err != nil && meta.str("type") == "blog-post" {
		return nil, fmt.Errorf("%s: invalid date: %w", srcPath, err)
	}
	err = nil

	return &Page{
		SrcPath:   srcPath,
		SrcDir:    filepath.Dir(srcPath),
		Meta:      meta,
		BodyHTML:  bodyHTML,
		PlainText: plainText(bodyHTML),
		Route:     computeRoute(rel),
		InMenu:    inMenu,
		Status:    status,
		Date:      date,
		Tags:      meta.tags(),
	}, nil
}

// excerpt returns text truncated to at most maxLen runes, cut at a word
// boundary where possible and suffixed with an ellipsis when shortened.
func excerpt(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxLen {
		return text
	}
	for i := maxLen; i > maxLen-20 && i > 0; i-- {
		if text[i] == ' ' {
			return text[:i] + "…"
		}
	}
	return text[:maxLen] + "…"
}

// Discover returns all .md paths under dir (recursive).
func Discover(dir string) ([]string, error) {
	var paths []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			paths = append(paths, path)
		}
		return nil
	})
	return paths, err
}

// computeRoute maps a content-relative path to a clean URL.
// content/blog/hello.md → /blog/hello/
// content/index.md      → /
func computeRoute(rel string) string {
	rel = filepath.ToSlash(rel)
	rel = strings.TrimSuffix(rel, ".md")
	parts := strings.Split(rel, "/")

	if len(parts) == 1 && parts[0] == "index" {
		return "/"
	}
	if parts[len(parts)-1] == "index" {
		parts = parts[:len(parts)-1]
	}
	return "/" + strings.Join(parts, "/") + "/"
}

func parseDate(s string) (time.Time, error) {
	for _, f := range []string{"2006-01-02", "2006-01-02T15:04:05Z", "2006-01-02T15:04:05-07:00"} {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse date %q", s)
}

package template

import (
	"bytes"
	"fmt"
	ht "html/template"
	"io/fs"
	"os"
	"strings"
)

// Set holds one compiled *ht.Template per page type.
type Set struct {
	templates map[string]*ht.Template
}

// Load reads the template set from a directory on disk.
func Load(dir string) (*Set, error) {
	return loadFromFS(os.DirFS(dir))
}

// LoadFS reads the template set from an fs.FS (e.g. embed.FS sub-tree).
func LoadFS(fsys fs.FS) (*Set, error) {
	return loadFromFS(fsys)
}

func loadFromFS(fsys fs.FS) (*Set, error) {
	baseData, err := fs.ReadFile(fsys, "base.html")
	if err != nil {
		return nil, fmt.Errorf("missing base.html: %w", err)
	}

	s := &Set{templates: make(map[string]*ht.Template)}

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") || e.Name() == "base.html" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".html")
		typeData, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return nil, err
		}
		// Each type gets its own template set: base + type definitions combined.
		t, err := ht.New("root").Funcs(funcs()).Parse(string(baseData))
		if err != nil {
			return nil, fmt.Errorf("parse base.html: %w", err)
		}
		if _, err = t.Parse(string(typeData)); err != nil {
			return nil, fmt.Errorf("parse %s: %w", e.Name(), err)
		}
		s.templates[name] = t
	}
	return s, nil
}

// Render executes the named type's template with data.
// Falls back to "default" if the type is not found.
func (s *Set) Render(typeName string, data any) (string, error) {
	t, ok := s.templates[typeName]
	if !ok {
		t, ok = s.templates["default"]
		if !ok {
			return "", fmt.Errorf("no template for type %q and no default template", typeName)
		}
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "base", data); err != nil {
		return "", fmt.Errorf("execute template %q: %w", typeName, err)
	}
	return buf.String(), nil
}

func funcs() ht.FuncMap {
	return ht.FuncMap{
		"safeHTML": func(s string) ht.HTML { return ht.HTML(s) },
		"safeCSS":  func(s string) ht.CSS  { return ht.CSS(s) },
		"safeJS":   func(s string) ht.JS   { return ht.JS(s) },
	}
}

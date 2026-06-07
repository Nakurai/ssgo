package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nakurai/ssgo/assets"
	"github.com/nakurai/ssgo/internal/config"
	"github.com/nakurai/ssgo/internal/template"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new site in the current directory",
	RunE:  runInit,
}

var (
	initURL   string
	initForce bool
	initFrom  string
	initStyle string
)

func init() {
	initCmd.Flags().StringVar(&initURL, "url", "", "Production base URL (e.g. https://example.com)")
	initCmd.Flags().BoolVar(&initForce, "force", false, "Overwrite an existing ssgo.json")
	initCmd.Flags().StringVar(&initFrom, "from", "", "Template source: http(s) URL, local .zip, or directory (defaults to the embedded template)")
	initCmd.Flags().StringVar(&initStyle, "style", "", "Name to install the style as (defaults to the source name, or \"default\")")
}

func runInit(cmd *cobra.Command, args []string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	cfgPath := filepath.Join(wd, config.Filename)
	if !initForce {
		if _, err := os.Stat(cfgPath); err == nil {
			return fmt.Errorf("%s already exists; use --force to overwrite", config.Filename)
		}
	}

	// Resolve the style name to install under template/style/.
	style := initStyle
	if style == "" {
		if initFrom != "" {
			style = template.StyleName(initFrom)
		} else {
			style = "default"
		}
	}
	styleDir := filepath.Join(wd, "template", "style", style)

	// Create directory structure
	dirs := []string{
		styleDir,
		filepath.Join(wd, "content"),
		filepath.Join(wd, "build", "prod"),
		filepath.Join(wd, "build", "dev"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	// Write ssgo.json
	prodURL := initURL
	if prodURL == "" {
		prodURL = "https://example.com"
	}
	cfg := &config.Config{
		Title:  "My Site",
		Style:  style,
		Colors: config.DefaultColors,
		Dev:    config.Profile{BaseURL: "http://localhost:8088"},
		Prod:   config.Profile{BaseURL: prodURL},
	}
	if err := config.Save(wd, cfg); err != nil {
		return err
	}

	// Install templates into template/style/<style>/ — from the given source,
	// or the embedded default when no --from is supplied.
	if initFrom != "" {
		if err := template.Fetch(initFrom, styleDir); err != nil {
			os.RemoveAll(styleDir)
			return fmt.Errorf("fetch template: %w", err)
		}
	} else if err := copyEmbeddedTemplates(styleDir); err != nil {
		return fmt.Errorf("write templates: %w", err)
	}

	// Seed sample content
	if err := writeSampleContent(filepath.Join(wd, "content")); err != nil {
		return fmt.Errorf("write sample content: %w", err)
	}

	fmt.Printf("Initialized site in %s (style %q)\n", wd, style)
	if initURL == "" {
		fmt.Printf("  Note: edit %s and set prod.baseURL to your real URL (currently %q)\n", config.Filename, prodURL)
	}
	fmt.Println("  Run 'ssgo generate' to build or 'ssgo watch' to start the dev server.")
	return nil
}

func copyEmbeddedTemplates(outDir string) error {
	return fs.WalkDir(assets.FS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(path, "templates")
		rel = strings.TrimPrefix(rel, "/")
		if rel == "" {
			return nil
		}
		dest := filepath.Join(outDir, rel)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		data, err := assets.FS.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(dest, data, 0644)
	})
}

func writeSampleContent(contentDir string) error {
	index := `---
type: index
title: Welcome
in-menu: no
---
This is your new site. Edit the content in the **content/** folder and templates in **template/style/default/**.
`
	about := `---
type: page
title: About
in-menu: yes
---
This is the about page. Tell visitors who you are.
`
	post := `---
type: blog-post
title: Hello World
status: published
date: 2025-01-01
tags: intro, welcome
in-menu: no
---
My first post. Write something here.
`
	secondPost := `---
type: blog-post
title: A Second Post
status: published
date: 2025-02-14
tags: updates, welcome
in-menu: no
---
Another post, so the archive and tag lists have something to show.
`
	postIndex := `---
type: post-index
title: Blog
in-menu: yes
---
Browse every post below. The sidebar groups them by date and by tag.
`
	files := map[string]string{
		"index.md":            index,
		"about.md":            about,
		"blog/index.md":       postIndex,
		"blog/hello.md":       post,
		"blog/second-post.md": secondPost,
	}
	for name, body := range files {
		path := filepath.Join(contentDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		// Don't overwrite existing files.
		if _, err := os.Stat(path); err == nil {
			continue
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			return err
		}
	}
	return nil
}

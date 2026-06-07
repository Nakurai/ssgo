package builder

import (
	"fmt"
	ht "html/template"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tdewolff/minify/v2"
	mhtml "github.com/tdewolff/minify/v2/html"
	mjs "github.com/tdewolff/minify/v2/js"

	"github.com/nakurai/ssgo/assets"
	"github.com/nakurai/ssgo/internal/config"
	"github.com/nakurai/ssgo/internal/content"
	"github.com/nakurai/ssgo/internal/search"
	tmpl "github.com/nakurai/ssgo/internal/template"
)

// Options controls a single build run.
type Options struct {
	ProjectDir string
	OutputDir  string
	Profile    config.Profile
	IsDev      bool // skips blog-post filtering, sets IsDraft/IsFuture flags
	Minify     bool
	LiveReload bool
}

// NavItem is a single navigation link.
type NavItem struct {
	Title string
	URL   string
}

// SiteContext holds site-wide template data.
type SiteContext struct {
	Title string
	Style string
	// Logo is the public URL of the site logo, or "" when none is configured.
	Logo string
}

// PostSummary is one blog post as shown in the post-index listing.
type PostSummary struct {
	Title   string
	URL     string
	Excerpt string
	Date    time.Time
	Tags    []string
}

// ArchiveDay/ArchiveMonth/ArchiveYear form the year→month→day date tree, each
// node carrying the number of posts beneath it.
type ArchiveDay struct {
	Day   int
	Count int
}

type ArchiveMonth struct {
	Month int
	Name  string
	Count int
	Days  []ArchiveDay
}

type ArchiveYear struct {
	Year   int
	Count  int
	Months []ArchiveMonth
}

// TagCount is a tag and the number of posts carrying it.
type TagCount struct {
	Tag   string
	Count int
}

// RenderContext is the data passed to every template.
type RenderContext struct {
	Site      SiteContext
	Page      *content.Page
	Content   ht.HTML
	BaseURL   string
	Favicon   string
	ColorVars ht.CSS
	Nav       []NavItem
	IsDraft   bool
	IsFuture  bool

	// Populated for the post-index template (available to every template).
	Posts   []PostSummary
	Archive []ArchiveYear
	Tags    []TagCount
}

// Build performs a full build according to opts.
func Build(cfg *config.Config, opts Options) error {
	// 1. Load templates
	ts, err := loadTemplates(opts.ProjectDir, cfg.Style)
	if err != nil {
		return fmt.Errorf("load templates: %w", err)
	}

	// 2. Discover and parse pages
	contentDir := filepath.Join(opts.ProjectDir, "content")
	paths, err := content.Discover(contentDir)
	if err != nil {
		return fmt.Errorf("discover content: %w", err)
	}

	pages := make([]*content.Page, 0, len(paths))
	for _, p := range paths {
		page, err := content.Load(p, contentDir)
		if err != nil {
			return fmt.Errorf("load %s: %w", p, err)
		}
		pages = append(pages, page)
	}

	// 3. Blog-post lifecycle
	now := time.Now().UTC()
	if !opts.IsDev {
		pages = filterProd(pages, now)
	} else {
		setDevFlags(pages, now)
	}

	// 4. Nav + post index (listing, date archive, tag counts)
	nav := collectNav(pages)
	posts, archive, tagCounts := collectPostIndex(pages)

	// 5. Shared context values
	colorVars := ht.CSS(cfg.Colors.CSSVars())
	assetsDir := filepath.Join(opts.OutputDir, "assets")
	logoURL, err := resolveAsset(cfg.Logo, opts.ProjectDir, assetsDir)
	if err != nil {
		return fmt.Errorf("logo: %w", err)
	}
	faviconURL, err := resolveAsset(cfg.Favicon, opts.ProjectDir, assetsDir)
	if err != nil {
		return fmt.Errorf("favicon: %w", err)
	}
	site := SiteContext{Title: cfg.Title, Style: cfg.Style, Logo: logoURL}

	var mini *minify.M
	if opts.Minify {
		mini = minify.New()
		mini.AddFunc("text/html", mhtml.Minify)
		mini.AddFunc("application/javascript", mjs.Minify)
	}

	// 6. Prepare output root
	if err := os.MkdirAll(opts.OutputDir, 0755); err != nil {
		return err
	}

	// 7. Render pages
	routes := make(map[string]string, len(pages))
	for _, p := range pages {
		routes[filepath.Clean(p.SrcPath)] = p.Route
	}
	for _, p := range pages {
		body := p.BodyHTML
		if body, err = processMedia(body, p.SrcDir, assetsDir, routes); err != nil {
			return fmt.Errorf("media %s: %w", p.SrcPath, err)
		}

		ctx := RenderContext{
			Site:      site,
			Page:      p,
			Content:   ht.HTML(body),
			BaseURL:   opts.Profile.BaseURL,
			Favicon:   faviconURL,
			ColorVars: colorVars,
			Nav:       nav,
			IsDraft:   p.IsDraft,
			IsFuture:  p.IsFuture,
			Posts:     posts,
			Archive:   archive,
			Tags:      tagCounts,
		}

		html, err := ts.Render(p.Type(), ctx)
		if err != nil {
			return fmt.Errorf("render %s: %w", p.SrcPath, err)
		}

		if opts.LiveReload {
			html = injectLiveReload(html)
		}
		if opts.Minify && mini != nil {
			html, err = mini.String("text/html", html)
			if err != nil {
				log.Printf("minify %s: %v (skipping)", p.SrcPath, err)
			}
		}

		if err := writePage(opts.OutputDir, p.Route, html); err != nil {
			return fmt.Errorf("write %s: %w", p.Route, err)
		}
	}

	// 8. Search index
	if err := search.Build(pages, opts.OutputDir); err != nil {
		return fmt.Errorf("search index: %w", err)
	}

	// 9. Copy search.js
	searchJS, err := assets.FS.ReadFile("search/search.js")
	if err != nil {
		return fmt.Errorf("read search.js: %w", err)
	}
	if opts.Minify && mini != nil {
		if min, err := mini.String("application/javascript", string(searchJS)); err == nil {
			searchJS = []byte(min)
		}
	}
	if err := os.WriteFile(filepath.Join(opts.OutputDir, "search.js"), searchJS, 0644); err != nil {
		return err
	}

	// 10. 404 page
	ctx404 := RenderContext{
		Site:      site,
		Page:      &content.Page{Meta: content.Meta{"title": "Page Not Found"}},
		BaseURL:   opts.Profile.BaseURL,
		Favicon:   faviconURL,
		ColorVars: colorVars,
		Nav:       nav,
	}
	html404, err := ts.Render("404", ctx404)
	if err != nil {
		html404 = "<h1>404 Not Found</h1>"
	}
	if opts.Minify && mini != nil {
		if min, err := mini.String("text/html", html404); err == nil {
			html404 = min
		}
	}
	if err := os.WriteFile(filepath.Join(opts.OutputDir, "404.html"), []byte(html404), 0644); err != nil {
		return err
	}

	return nil
}

// loadTemplates resolves the active style folder and returns its template Set.
// Falls back to the embedded default templates if no template/style/ dir exists.
func loadTemplates(projectDir, style string) (*tmpl.Set, error) {
	if style == "" {
		style = "default"
	}
	styleDir := filepath.Join(projectDir, "template", "style", style)
	if _, err := os.Stat(styleDir); err == nil {
		return tmpl.Load(styleDir)
	}

	// If the style root doesn't exist at all, silently use embedded defaults.
	styleRoot := filepath.Join(projectDir, "template", "style")
	if _, err := os.Stat(styleRoot); err != nil {
		log.Println("hint: run 'ssgo init' to write default templates")
		sub, err := fs.Sub(assets.FS, "templates")
		if err != nil {
			return nil, err
		}
		return tmpl.LoadFS(sub)
	}

	// style/ exists but the requested style is missing — show what's available.
	entries, _ := os.ReadDir(styleRoot)
	var available []string
	for _, e := range entries {
		if e.IsDir() {
			available = append(available, e.Name())
		}
	}
	return nil, fmt.Errorf("style %q not found in template/style/; available: %v", style, available)
}

func writePage(outDir, route, html string) error {
	var outPath string
	if route == "/" {
		outPath = filepath.Join(outDir, "index.html")
	} else {
		outPath = filepath.Join(outDir, filepath.FromSlash(strings.Trim(route, "/")), "index.html")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(outPath, []byte(html), 0644)
}

func filterProd(pages []*content.Page, now time.Time) []*content.Page {
	out := pages[:0]
	for _, p := range pages {
		if p.Type() != "blog-post" {
			out = append(out, p)
			continue
		}
		if p.Status != "published" {
			log.Printf("skip (draft): %s", p.SrcPath)
			continue
		}
		if !p.Date.IsZero() && p.Date.After(now) {
			log.Printf("skip (future %s): %s", p.Date.Format("2006-01-02"), p.SrcPath)
			continue
		}
		out = append(out, p)
	}
	return out
}

func setDevFlags(pages []*content.Page, now time.Time) {
	for _, p := range pages {
		if p.Type() != "blog-post" {
			continue
		}
		p.IsDraft = p.Status == "draft"
		p.IsFuture = !p.Date.IsZero() && p.Date.After(now)
	}
}

func collectNav(pages []*content.Page) []NavItem {
	var items []NavItem
	for _, p := range pages {
		if !p.InMenu {
			continue
		}
		title := p.Title()
		if title == "" {
			title = p.Route
		}
		items = append(items, NavItem{Title: title, URL: p.Route})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Title < items[j].Title
	})
	return items
}

// collectPostIndex gathers every blog-post page into a newest-first listing
// plus the derived date archive (year/month/day counts) and tag counts.
func collectPostIndex(pages []*content.Page) ([]PostSummary, []ArchiveYear, []TagCount) {
	var posts []PostSummary
	for _, p := range pages {
		if p.Type() != "blog-post" {
			continue
		}
		posts = append(posts, PostSummary{
			Title:   p.Title(),
			URL:     p.Route,
			Excerpt: p.Excerpt(),
			Date:    p.Date,
			Tags:    p.Tags,
		})
	}
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})
	return posts, buildArchive(posts), buildTagCounts(posts)
}

// buildArchive groups posts into a year→month→day tree, each node counting the
// posts beneath it. Years, months, and days are ordered newest first.
func buildArchive(posts []PostSummary) []ArchiveYear {
	years := map[int]map[int]map[int]int{}
	for _, p := range posts {
		if p.Date.IsZero() {
			continue
		}
		y, mo, d := p.Date.Year(), int(p.Date.Month()), p.Date.Day()
		if years[y] == nil {
			years[y] = map[int]map[int]int{}
		}
		if years[y][mo] == nil {
			years[y][mo] = map[int]int{}
		}
		years[y][mo][d]++
	}

	var out []ArchiveYear
	for y, months := range years {
		ay := ArchiveYear{Year: y}
		for mo, days := range months {
			am := ArchiveMonth{Month: mo, Name: time.Month(mo).String()}
			for d, c := range days {
				am.Days = append(am.Days, ArchiveDay{Day: d, Count: c})
				am.Count += c
			}
			sort.Slice(am.Days, func(i, j int) bool { return am.Days[i].Day > am.Days[j].Day })
			ay.Months = append(ay.Months, am)
			ay.Count += am.Count
		}
		sort.Slice(ay.Months, func(i, j int) bool { return ay.Months[i].Month > ay.Months[j].Month })
		out = append(out, ay)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Year > out[j].Year })
	return out
}

// buildTagCounts tallies how many posts carry each tag, sorted alphabetically.
func buildTagCounts(posts []PostSummary) []TagCount {
	counts := map[string]int{}
	for _, p := range posts {
		for _, t := range p.Tags {
			counts[t]++
		}
	}
	out := make([]TagCount, 0, len(counts))
	for t, c := range counts {
		out = append(out, TagCount{Tag: t, Count: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Tag < out[j].Tag })
	return out
}

func injectLiveReload(html string) string {
	const tag = `<script src="/__livereload.js"></script>`
	if idx := strings.LastIndex(html, "</body>"); idx != -1 {
		return html[:idx] + tag + html[idx:]
	}
	return html + tag
}

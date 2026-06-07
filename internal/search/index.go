package search

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"ssg.nakurai/internal/content"
)

type Doc struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

type Index struct {
	Docs  []Doc              `json:"docs"`
	Index map[string][]int   `json:"index"`
}

// Build creates search-index.json in outDir from the given pages.
func Build(pages []*content.Page, outDir string) error {
	idx := &Index{
		Docs:  make([]Doc, 0, len(pages)),
		Index: make(map[string][]int),
	}

	for i, p := range pages {
		title := p.Title()
		snippet := snippet(p.PlainText, 160)

		idx.Docs = append(idx.Docs, Doc{
			ID:      i,
			Title:   title,
			URL:     p.Route,
			Snippet: snippet,
		})

		text := normalize(title + " " + p.PlainText)
		for _, tg := range trigrams(text) {
			idx.Index[tg] = appendUniq(idx.Index[tg], i)
		}
	}

	data, err := json.Marshal(idx)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "search-index.json"), data, 0644)
}

func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func trigrams(s string) []string {
	if len(s) < 3 {
		return nil
	}
	seen := make(map[string]bool)
	var out []string
	for i := 0; i <= len(s)-3; i++ {
		tg := s[i : i+3]
		if !seen[tg] {
			seen[tg] = true
			out = append(out, tg)
		}
	}
	return out
}

func appendUniq(ids []int, id int) []int {
	for _, v := range ids {
		if v == id {
			return ids
		}
	}
	return append(ids, id)
}

func snippet(text string, maxLen int) string {
	text = strings.TrimSpace(text)
	if len(text) <= maxLen {
		return text
	}
	// cut at a word boundary
	for i := maxLen; i > maxLen-20 && i > 0; i-- {
		if text[i] == ' ' {
			return text[:i] + "…"
		}
	}
	return text[:maxLen] + "…"
}

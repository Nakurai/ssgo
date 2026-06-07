package content

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Meta is the raw key-value front matter map.
type Meta map[string]any

func splitFrontMatter(src []byte) (Meta, []byte, error) {
	s := string(src)
	if !strings.HasPrefix(s, "---") {
		return Meta{}, src, nil
	}
	rest := s[3:]
	// accept "---\n" or "---\r\n"
	if len(rest) > 0 && (rest[0] == '\n' || rest[0] == '\r') {
		rest = strings.TrimLeft(rest, "\r\n")
	}
	idx := strings.Index(rest, "\n---")
	if idx == -1 {
		return Meta{}, src, fmt.Errorf("unclosed front matter")
	}
	yamlSrc := strings.TrimSpace(rest[:idx])
	bodyStr := rest[idx+4:]
	if strings.HasPrefix(bodyStr, "\n") {
		bodyStr = bodyStr[1:]
	} else if strings.HasPrefix(bodyStr, "\r\n") {
		bodyStr = bodyStr[2:]
	}

	m := Meta{}
	if yamlSrc != "" {
		if err := yaml.NewDecoder(bytes.NewBufferString(yamlSrc)).Decode(&m); err != nil {
			return Meta{}, src, fmt.Errorf("front matter YAML: %w", err)
		}
	}
	return m, []byte(bodyStr), nil
}

// date returns the "date" field as a time.Time.
// gopkg.in/yaml.v3 parses bare date strings (2006-01-02) into time.Time directly,
// so we must handle both time.Time and string values.
func (m Meta) date() (time.Time, error) {
	v, ok := m["date"]
	if !ok {
		return time.Time{}, nil
	}
	switch t := v.(type) {
	case time.Time:
		return t.UTC(), nil
	case string:
		return parseDate(t)
	default:
		return parseDate(fmt.Sprintf("%v", v))
	}
}

// tags returns the "tags" field as a slice. The documented form is a
// comma-separated string ("go, web, ssg"), but a YAML list is also accepted.
// Whitespace is trimmed and empty entries are dropped.
func (m Meta) tags() []string {
	v, ok := m["tags"]
	if !ok {
		return nil
	}
	var raw []string
	switch t := v.(type) {
	case string:
		raw = strings.Split(t, ",")
	case []any:
		for _, e := range t {
			raw = append(raw, fmt.Sprintf("%v", e))
		}
	default:
		raw = strings.Split(fmt.Sprintf("%v", v), ",")
	}
	out := make([]string, 0, len(raw))
	for _, p := range raw {
		if s := strings.TrimSpace(p); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func (m Meta) str(key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	case bool:
		if s {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", v)
	}
}

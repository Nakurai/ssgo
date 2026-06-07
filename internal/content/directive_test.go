package content

import (
	"strings"
	"testing"
)

func TestDirectiveRendering(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		contains []string // substrings that must appear in the output
		absent   []string // substrings that must NOT appear
	}{
		{
			name:     "simple directive wraps in prefixed div",
			src:      ":::hero\n# Welcome\n:::\n",
			contains: []string{`<div class="ssg-hero">`, "<h1", "Welcome", "</h1>", "</div>"},
		},
		{
			name:     "inner content is parsed as markdown",
			src:      ":::centered\nSome **bold** text.\n:::\n",
			contains: []string{`<div class="ssg-centered">`, "<strong>bold</strong>"},
		},
		{
			name:     "hyphenated name is allowed",
			src:      ":::call-to-action\nClick me\n:::\n",
			contains: []string{`<div class="ssg-call-to-action">`},
		},
		{
			name:     "nested directives use more colons on the outer fence",
			src:      "::::hero\n:::centered\n![pic](pic.png)\n:::\n::::\n",
			contains: []string{`<div class="ssg-hero">`, `<div class="ssg-centered">`},
		},
		{
			name:     "invalid name is not treated as a directive",
			src:      ":::Bad Name\nhello\n:::\n",
			absent:   []string{`class="ssg-`},
			contains: []string{"hello"},
		},
		{
			name:     "fewer than three colons is not a directive",
			src:      "::hero\nhello\n::\n",
			absent:   []string{`class="ssg-`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := renderMarkdown([]byte(tt.src))
			if err != nil {
				t.Fatalf("renderMarkdown: %v", err)
			}
			for _, want := range tt.contains {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q\ngot:\n%s", want, out)
				}
			}
			for _, bad := range tt.absent {
				if strings.Contains(out, bad) {
					t.Errorf("output should not contain %q\ngot:\n%s", bad, out)
				}
			}
		})
	}
}

func TestDirectiveRegistryOverride(t *testing.T) {
	directiveRegistry["note"] = Directive{
		Open:  func(name, args string) string { return `<aside class="callout" data-args="` + args + `">` },
		Close: "</aside>",
	}
	t.Cleanup(func() { delete(directiveRegistry, "note") })

	out, err := renderMarkdown([]byte(":::note warning\nHeads up.\n:::\n"))
	if err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	for _, want := range []string{`<aside class="callout" data-args="warning">`, "Heads up.", "</aside>"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\ngot:\n%s", want, out)
		}
	}
	if strings.Contains(out, `class="ssg-note"`) {
		t.Errorf("registry override should suppress the generic wrapper\ngot:\n%s", out)
	}
}

func TestDirectiveInnerTextIsPlainTextIndexable(t *testing.T) {
	out, err := renderMarkdown([]byte(":::hero\nFindable words here.\n:::\n"))
	if err != nil {
		t.Fatalf("renderMarkdown: %v", err)
	}
	if got := plainText(out); !strings.Contains(got, "Findable words here.") {
		t.Errorf("plainText dropped directive inner text, got %q", got)
	}
}

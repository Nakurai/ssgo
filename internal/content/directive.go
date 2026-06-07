package content

import (
	"bytes"
	"regexp"

	"github.com/yuin/goldmark"
	gast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// Directives let content authors express layout intent from Markdown without
// writing any HTML or CSS. A fenced block
//
//	:::hero
//	# Welcome
//	:::
//
// renders to `<div class="ssg-hero"> … </div>` with its inner content parsed as
// normal Markdown. The matching `.ssg-hero` styling lives in each style's
// base.html, so authors only ever need to know the directive *name*.
//
// Adding a new directive is, in the common case, zero Go: any valid name renders
// as `<div class="ssg-<name>">`, so a new directive is just a new CSS rule. The
// directiveRegistry below is the escape hatch for the rare directive that needs
// richer markup than a class wrapper.

// Directive customises how a named directive renders. Both fields are optional;
// when Open is nil the generic `<div class="ssg-<name>">` wrapper is used, and
// when Close is empty the wrapper closes with `</div>`.
type Directive struct {
	Open  func(name, args string) string // custom opening HTML
	Close string                         // custom closing HTML
}

// directiveRegistry holds per-name overrides. It is intentionally empty by
// default: the built-in `hero` and `centered` directives need only CSS. Register
// an entry here when a directive must emit something other than a class wrapper.
var directiveRegistry = map[string]Directive{}

// directiveNameRE constrains names to a safe CSS class suffix.
var directiveNameRE = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// directiveNode is a container block holding the directive's name, the raw
// arguments after the name on the opening line (reserved for future directives),
// and the number of colons in its opening fence so the matching close can be
// found even when directives are nested.
type directiveNode struct {
	gast.BaseBlock
	Name     string
	Args     string
	fenceLen int
}

var kindDirective = gast.NewNodeKind("Directive")

func (n *directiveNode) Kind() gast.NodeKind { return kindDirective }

func (n *directiveNode) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{
		"Name": n.Name,
		"Args": n.Args,
	}, nil)
}

// closingFenceLen reports the colon count if line is a closing fence (only
// colons, surrounded by optional whitespace), or 0 otherwise.
func closingFenceLen(line []byte) int {
	s := bytes.TrimSpace(line)
	if len(s) == 0 {
		return 0
	}
	for _, c := range s {
		if c != ':' {
			return 0
		}
	}
	return len(s)
}

type directiveParser struct{}

func (p *directiveParser) Trigger() []byte { return []byte{':'} }

func (p *directiveParser) Open(parent gast.Node, reader text.Reader, pc parser.Context) (gast.Node, parser.State) {
	line, _ := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 { // blank line
		return nil, parser.NoChildren
	}
	i := pos
	for i < len(line) && line[i] == ':' {
		i++
	}
	fenceLen := i - pos
	if fenceLen < 3 {
		return nil, parser.NoChildren
	}
	// The remainder is `name [args]`. A line of only colons is a closing fence,
	// not an opener, so it is left for Continue to handle.
	rest := bytes.TrimSpace(line[i:])
	if len(rest) == 0 {
		return nil, parser.NoChildren
	}
	name := rest
	var args []byte
	if sp := bytes.IndexAny(rest, " \t"); sp >= 0 {
		name = rest[:sp]
		args = bytes.TrimSpace(rest[sp+1:])
	}
	if !directiveNameRE.Match(name) {
		return nil, parser.NoChildren
	}
	reader.AdvanceLine()
	return &directiveNode{
		Name:     string(name),
		Args:     string(args),
		fenceLen: fenceLen,
	}, parser.HasChildren
}

func (p *directiveParser) Continue(node gast.Node, reader text.Reader, pc parser.Context) parser.State {
	d := node.(*directiveNode)
	line, _ := reader.PeekLine()
	if n := closingFenceLen(line); n >= d.fenceLen {
		reader.AdvanceLine()
		return parser.Close
	}
	return parser.Continue | parser.HasChildren
}

func (p *directiveParser) Close(node gast.Node, reader text.Reader, pc parser.Context) {}

func (p *directiveParser) CanInterruptParagraph() bool { return true }

func (p *directiveParser) CanAcceptIndentedLine() bool { return false }

type directiveRenderer struct{}

func (r *directiveRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindDirective, r.render)
}

func (r *directiveRenderer) render(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	d := n.(*directiveNode)
	if custom, ok := directiveRegistry[d.Name]; ok && custom.Open != nil {
		if entering {
			_, _ = w.WriteString(custom.Open(d.Name, d.Args))
		} else if custom.Close != "" {
			_, _ = w.WriteString(custom.Close)
		} else {
			_, _ = w.WriteString("</div>")
		}
		return gast.WalkContinue, nil
	}
	if entering {
		_, _ = w.WriteString(`<div class="ssg-`)
		_, _ = w.WriteString(d.Name)
		_, _ = w.WriteString("\">\n")
	} else {
		_, _ = w.WriteString("</div>\n")
	}
	return gast.WalkContinue, nil
}

// directiveExtension wires the directive parser and renderer into Goldmark.
type directiveExtension struct{}

func (e *directiveExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithBlockParsers(
		util.Prioritized(&directiveParser{}, 99),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&directiveRenderer{}, 100),
	))
}

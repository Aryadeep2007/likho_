package main

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

// Editor me text select karke "preset" laga sakte ho - jaisa Clash Royale
// card ka shine ya legendary item ka glow. Syntax jaanbujhke naya banaya
// hai: {fx:legendary}text{/fx}. Raw HTML ko humne jaanbujhke off rakha hai
// (render.go dekho - blog LAN/internet pe khulta hai), toh ye seedha
// <span class="fx-..."> nahi likh sakte the - koi bhi <script> daal deta.
//
// Isliye ye ek asli goldmark extension hai (inline parser + AST node +
// renderer), bilkul waise hi jaise goldmark ka apna "strikethrough"
// (~~text~~) kaam karta hai. Fayda ye hai ki sirf yahan whitelist kiye
// hue class names hi kabhi HTML me jaate hain - koi arbitrary tag/attribute
// nahi ghus sakta, aur agar koi selection do paragraphs me phailta hai
// (matlab galत jagah band hui tag) toh bas wo silently ignore ho jata hai,
// tooti hui HTML kabhi nahi banti.
//
// Naya preset add karna ho: yahan whitelist me naam daalo, style.css me
// `.fx-<naam>` likho, aur editor.js ke FX_PRESETS me bhi wahi naam daalo
// (teeno jagah same list honi chahiye).
var fxPresets = map[string]bool{
	"legendary": true, // dheere dheere pulse karta warm gold glow
	"champion":  true, // metallic gradient + Clash Royale card jaisi diagonal chamak
	"holo":      true, // shift hota hua iridescent rainbow foil
}

var (
	fxOpenRe  = regexp.MustCompile(`^\{fx:([a-z]+)\}`)
	fxCloseTag = []byte("{/fx}")
)

// ---------- AST node ----------

type fxSpan struct {
	gast.BaseInline
	Preset string
}

var kindFxSpan = gast.NewNodeKind("FxSpan")

func (n *fxSpan) Kind() gast.NodeKind { return kindFxSpan }

func (n *fxSpan) Dump(source []byte, level int) {
	gast.DumpHelper(n, source, level, map[string]string{"Preset": n.Preset}, nil)
}

func newFxSpan(preset string) *fxSpan { return &fxSpan{Preset: preset} }

// ---------- delimiters ----------
// parser.Delimiter goldmark ka wahi mechanism hai jo emphasis/strikethrough
// ke liye "kaunsa opener kaunse closer se match karta hai" sambhalta hai.
// Farak sirf itna hai ki wo ek hi character ke repeated runs (jaise **, ~~)
// ke liye bana hai, aur hamara syntax asymmetric hai ({fx:x} ... {/fx}) -
// isliye parser.ScanDelimiter use nahi karte, khud Delimiter bana ke
// pc.PushDelimiter() karte hain. Baaki matching goldmark khud sambhal leta hai.

type fxOpenProcessor struct{ preset string }

func (p *fxOpenProcessor) IsDelimiter(b byte) bool { return false } // ScanDelimiter use nahi hota, isliye irrelevant

func (p *fxOpenProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool {
	_, ok := closer.Processor.(*fxCloseProcessor)
	return ok
}

func (p *fxOpenProcessor) OnMatch(consumes int) gast.Node { return newFxSpan(p.preset) }

type fxCloseProcessor struct{}

func (p *fxCloseProcessor) IsDelimiter(b byte) bool                             { return false }
func (p *fxCloseProcessor) CanOpenCloser(opener, closer *parser.Delimiter) bool { return false }
func (p *fxCloseProcessor) OnMatch(consumes int) gast.Node                     { return nil }

var defaultFxCloseProcessor = &fxCloseProcessor{}

// ---------- inline parser ----------

type fxParser struct{}

func (s *fxParser) Trigger() []byte { return []byte{'{'} }

func (s *fxParser) Parse(parent gast.Node, block text.Reader, pc parser.Context) gast.Node {
	line, segment := block.PeekLine()

	if bytes.HasPrefix(line, fxCloseTag) {
		node := parser.NewDelimiter(false, true, 1, 0, defaultFxCloseProcessor)
		node.Segment = segment.WithStop(segment.Start + len(fxCloseTag))
		block.Advance(len(fxCloseTag))
		pc.PushDelimiter(node)
		return node
	}

	m := fxOpenRe.FindSubmatch(line)
	if m == nil {
		return nil // '{' bas literal text hai, aage badho
	}
	preset := string(m[1])
	if !fxPresets[preset] {
		return nil // syntax sahi hai par preset whitelist me nahi - literal text jaisa treat karo
	}

	node := parser.NewDelimiter(true, false, 1, 0, &fxOpenProcessor{preset: preset})
	node.Segment = segment.WithStop(segment.Start + len(m[0]))
	block.Advance(len(m[0]))
	pc.PushDelimiter(node)
	return node
}

// ---------- HTML renderer ----------

type fxHTMLRenderer struct{}

func (r *fxHTMLRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindFxSpan, r.renderFxSpan)
}

func (r *fxHTMLRenderer) renderFxSpan(w util.BufWriter, source []byte, n gast.Node, entering bool) (gast.WalkStatus, error) {
	span := n.(*fxSpan)
	// span.Preset yahan tak pahunchne se pehle hi whitelist se guzar chuka hai
	// (Parse() upar dekho), isliye seedha class me daalna safe hai.
	if entering {
		_, _ = w.WriteString(`<span class="fx fx-`)
		_, _ = w.WriteString(span.Preset)
		_, _ = w.WriteString(`">`)
	} else {
		_, _ = w.WriteString(`</span>`)
	}
	return gast.WalkContinue, nil
}

// ---------- extension ----------

type fxExtension struct{}

// FxExtension render.go me goldmark.New() ke saath register hota hai.
var FxExtension = &fxExtension{}

func (e *fxExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(parser.WithInlineParsers(
		util.Prioritized(&fxParser{}, 500),
	))
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(&fxHTMLRenderer{}, 500),
	))
}

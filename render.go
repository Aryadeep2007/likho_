package main

import (
	"bytes"
	"html/template"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// [[kuch bhi]] - Obsidian wala style. Yahi feature poore graph view ko chalata hai.
var wikiRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM, // tables, strikethrough, autolink
		FxExtension,   // {fx:legendary}...{/fx} jaisi text-animation presets (fx.go)
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(), // blog likhte waqt enter dabao toh line break dikhna chahiye
	),
	// Note: raw HTML deliberately allow nahi kiya. Blog ab public/internet pe
	// khulta hai, aur kisi ne <script> daal diya toh sabke browser me chalega.
	// Nahi chahiye - isliye fx presets bhi raw HTML se nahi, apne extension se aate hain.
)

// Markdown ko HTML banao. Pehle wikilinks ko normal links me badalte hain,
// phir goldmark ko de dete hain - isse do alag parser likhne se bach gaye.
func renderMarkdown(body string) template.HTML {
	src := wikiRe.ReplaceAllStringFunc(body, func(m string) string {
		name := strings.TrimSpace(wikiRe.FindStringSubmatch(m)[1])

		if slug, ok := resolveWikiTarget(name); ok {
			return "[" + name + "](/p/" + slug + ")"
		}
		// Post exist nahi karta? Toh link ko "banane ka bahana" bana do.
		// Click karo aur editor us title ke saath khul jayega. Obsidian jaisa feel.
		return "[" + name + "](/new?title=" + urlish(name) + ")"
	})

	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		// convert fail ho hi nahi sakta normally, par agar ho jaye toh
		// kam se kam plain text dikha do - blank page se behtar hai
		return template.HTML("<pre>" + template.HTMLEscapeString(body) + "</pre>")
	}
	return template.HTML(buf.String())
}

// [[naam]] ko asli post se match karo. Pehle slug try karo, phir title.
// Case ka farak nahi padta - log "[[Go Tips]]" bhi likhte hain aur "[[go-tips]]" bhi.
func resolveWikiTarget(name string) (string, bool) {
	if p, ok := store.Get(name); ok {
		return p.Slug, true
	}
	if p, ok := store.Get(slugify(name)); ok {
		return p.Slug, true
	}
	low := strings.ToLower(name)
	for _, p := range store.All() {
		if strings.ToLower(p.Title) == low {
			return p.Slug, true
		}
	}
	return "", false
}

// Ek post kis kis post ko point kar raha hai. Graph ke edges yahi se bante hain.
func linksFrom(p *Post) []string {
	var out []string
	seen := map[string]bool{}

	for _, m := range wikiRe.FindAllStringSubmatch(p.Body, -1) {
		slug, ok := resolveWikiTarget(strings.TrimSpace(m[1]))
		if !ok || slug == p.Slug || seen[slug] {
			continue // apne aap se link, ya duplicate - skip
		}
		seen[slug] = true
		out = append(out, slug)
	}
	return out
}

// Kaun kaun is post ko point kar raha hai. Post page ke neeche
// "yahan se link aaya hai" section me dikhta hai.
func backlinksTo(slug string) []*Post {
	var out []*Post
	for _, p := range store.All() {
		if p.Slug == slug {
			continue
		}
		for _, l := range linksFrom(p) {
			if l == slug {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

// Card pe dikhane ke liye chhota sa preview. Markdown ke symbols hata dete hain
// warna preview me "## ** [[" jaisa kachra dikhta hai.
func excerpt(body string, max int) string {
	s := wikiRe.ReplaceAllString(body, "$1")

	// code blocks poore uda do, preview me unka koi matlab nahi
	if i := strings.Index(s, "```"); i >= 0 {
		s = s[:i]
	}

	var clean []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ">") {
			continue
		}
		clean = append(clean, line)
	}

	s = strings.Join(clean, " ")
	for _, ch := range []string{"**", "__", "*", "`", "![", "]("} {
		s = strings.ReplaceAll(s, ch, "")
	}
	s = strings.Join(strings.Fields(s), " ") // extra spaces saaf

	if len(s) > max {
		// beech me se word na kate isliye aakhri space tak trim karte hain
		cut := s[:max]
		if i := strings.LastIndex(cut, " "); i > max/2 {
			cut = cut[:i]
		}
		return cut + "..."
	}
	return s
}

// URL me daalne layak bana do. net/url import kar sakte the par
// sirf space aur & ke liye poora package laana zaroori nahi laga.
func urlish(s string) string {
	r := strings.NewReplacer(" ", "%20", "&", "%26", "?", "%3F", "#", "%23")
	return r.Replace(s)
}

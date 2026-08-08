package main

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

// Ye sab words har post me aate hain, inse koi matlab nahi nikalta.
// Hinglish wale bhi daal diye kyunki hum aise hi likhte hain.
var stopWords = map[string]bool{
	"the": true, "and": true, "for": true, "that": true, "this": true, "with": true,
	"you": true, "your": true, "are": true, "was": true, "but": true, "not": true,
	"from": true, "have": true, "has": true, "can": true, "will": true, "just": true,
	"what": true, "when": true, "how": true, "all": true, "its": true, "it's": true,
	"hai": true, "hain": true, "tha": true, "the_": true, "kya": true, "aur": true,
	"toh": true, "bhi": true, "koi": true, "kuch": true, "yeh": true, "woh": true,
	"mein": true, "me": true, "ka": true, "ki": true, "ke": true, "se": true,
}

type SearchHit struct {
	Post    *Post
	Score   float64
	Snippet string
}

// Words nikalo. Punctuation, emoji, sab hata do - sirf saaf tokens chahiye.
func tokenize(s string) []string {
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	var out []string
	for _, f := range fields {
		if len(f) < 2 || stopWords[f] {
			continue
		}
		out = append(out, f)
	}
	return out
}

// Search. Poore corpus pe har baar loop chalta hai.
// Haan, index bana sakte the - par 50-100 posts pe ye 1ms se kam leta hai,
// aur index ko sync me rakhna ek aur bug ka source hota. Rehne diya.
func searchPosts(q string) []SearchHit {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		return nil
	}
	terms := tokenize(q)
	if len(terms) == 0 {
		// user ne "a" ya "??" type kiya - raw string se hi try karte hain
		terms = []string{q}
	}

	var hits []SearchHit
	for _, p := range store.All() {
		title := strings.ToLower(p.Title)
		body := strings.ToLower(p.Body)
		tags := strings.ToLower(strings.Join(p.Tags, " "))

		score := 0.0
		for _, t := range terms {
			// title match sabse important - agar title me hai toh
			// wahi post chahiye hoga 90% cases me
			if strings.Contains(title, t) {
				score += 10
			}
			if strings.Contains(tags, t) {
				score += 5
			}
			score += float64(strings.Count(body, t)) * 0.5
		}
		if score == 0 {
			continue
		}
		// naye posts thoda upar aane chahiye, tie break ke liye
		if p.Updated.After(p.Created) {
			score += 0.1
		}
		hits = append(hits, SearchHit{
			Post:    p,
			Score:   score,
			Snippet: snippetAround(p.Body, terms[0]),
		})
	}

	sort.Slice(hits, func(i, j int) bool { return hits[i].Score > hits[j].Score })
	if len(hits) > 20 {
		hits = hits[:20]
	}
	return hits
}

// Search result me word ke aas paas ka thoda text dikhao,
// taki pata chale match kahan hua.
func snippetAround(body, term string) string {
	low := strings.ToLower(body)
	i := strings.Index(low, strings.ToLower(term))
	if i < 0 {
		return excerpt(body, 120)
	}

	start := i - 50
	if start < 0 {
		start = 0
	}
	end := i + 110
	if end > len(body) {
		end = len(body)
	}

	s := strings.Join(strings.Fields(body[start:end]), " ")
	if start > 0 {
		s = "..." + s
	}
	if end < len(body) {
		s = s + "..."
	}
	return s
}

// "Related posts" - bina kisi AI ke. Ye purana TF-IDF hai:
// jo word ek hi post me baar baar aata hai wo important, aur jo har
// post me aata hai wo bekaar. Phir cosine similarity se compare.
// Maths thoda dikhta hai par ye 1970s ki technique hai, koi magic nahi.
func relatedTo(target *Post, n int) []*Post {
	all := store.All()
	if len(all) < 2 {
		return nil
	}

	// Step 1: har post ke word counts
	docs := map[string]map[string]float64{}
	// Step 2: kitne posts me ye word aata hai (document frequency)
	df := map[string]int{}

	for _, p := range all {
		tf := map[string]float64{}
		// title aur tags ko 3 baar count karte hain - wo zyada matter karte hain
		words := tokenize(p.Body)
		words = append(words, tokenize(p.Title)...)
		words = append(words, tokenize(p.Title)...)
		words = append(words, tokenize(strings.Join(p.Tags, " "))...)

		for _, w := range words {
			tf[w]++
		}
		for w := range tf {
			df[w]++
		}
		docs[p.Slug] = tf
	}

	total := float64(len(all))
	// tf-idf weight nikaal lo
	for _, tf := range docs {
		for w, c := range tf {
			idf := math.Log(total / float64(df[w]))
			tf[w] = c * idf
		}
	}

	me := docs[target.Slug]

	type scored struct {
		p *Post
		s float64
	}
	var out []scored

	for _, p := range all {
		if p.Slug == target.Slug {
			continue
		}
		sim := cosine(me, docs[p.Slug])

		// jo posts direct [[link]] se jude hain unhe bonus milta hai
		for _, l := range linksFrom(target) {
			if l == p.Slug {
				sim += 0.35
			}
		}
		if sim <= 0.01 {
			continue // itna kam similarity dikhane ka koi fayda nahi
		}
		out = append(out, scored{p, sim})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].s > out[j].s })

	var posts []*Post
	for i, s := range out {
		if i >= n {
			break
		}
		posts = append(posts, s.p)
	}
	return posts
}

func cosine(a, b map[string]float64) float64 {
	var dot, na, nb float64

	// chhote map pe loop karo, thoda fast rehta hai
	small, big := a, b
	if len(b) < len(a) {
		small, big = b, a
	}
	for w, v := range small {
		if bv, ok := big[w]; ok {
			dot += v * bv
		}
	}
	for _, v := range a {
		na += v * v
	}
	for _, v := range b {
		nb += v * v
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

// Editor me "ye tags daal lo" wala suggestion.
// Sabse zyada aane wale non-boring words utha lete hain.
func suggestTags(body, title string) []string {
	counts := map[string]int{}
	for _, w := range tokenize(body + " " + title + " " + title) {
		if len(w) < 4 {
			continue // "abc" type ke tags kaam ke nahi hote
		}
		counts[w]++
	}

	type kv struct {
		w string
		n int
	}
	var list []kv
	for w, n := range counts {
		if n < 2 {
			continue // ek baar aaya word tag banne layak nahi hai
		}
		list = append(list, kv{w, n})
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].n == list[j].n {
			return list[i].w < list[j].w // stable rahe, warna har refresh pe order badalta hai
		}
		return list[i].n > list[j].n
	})

	var out []string
	for i, k := range list {
		if i >= 5 {
			break
		}
		out = append(out, k.w)
	}
	return out
}

// Saare tags, kitni baar aaye uske hisaab se. Sidebar me dikhta hai.
func allTags() []TagCount {
	counts := map[string]int{}
	for _, p := range store.All() {
		for _, t := range p.Tags {
			counts[t]++
		}
	}

	var out []TagCount
	for t, n := range counts {
		out = append(out, TagCount{Tag: t, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count == out[j].Count {
			return out[i].Tag < out[j].Tag
		}
		return out[i].Count > out[j].Count
	})
	return out
}

type TagCount struct {
	Tag   string
	Count int
}

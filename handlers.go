package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"
)

// Har page ko yehi base data milta hai. Struct banane ke bajaye map use kiya
// kyunki har page ko alag alag cheezein chahiye thi aur ek bada struct
// jisme aadhe fields khali ho, wo dekhne me ganda lagta hai.
func base(r *http.Request, title, nav string) map[string]any {
	return map[string]any{
		"Title":     title,
		"Nav":       nav,
		"LanURL":    lanURL(currentPort),
		"Tags":      allTags(),
		"Count":     store.Count(),
		"User":      currentUser(r),
		"CSRFToken": currentCSRF(r),
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	// Go ka "/" pattern sab kuch match karta hai, toh 404 khud handle karna padta hai.
	// Warna /kuch-bhi pe bhi homepage khul jata hai aur weird lagta hai.
	if r.URL.Path != "/" {
		notFound(w, r)
		return
	}

	owner := currentUser(r) != nil
	posts := store.All()

	// draft aur scheduled sirf owner ko dikhte hain - anonymous reader
	// ke liye sirf "live" posts exist karte hain.
	var live, drafts, scheduled []*Post
	for _, p := range posts {
		switch {
		case p.IsLive():
			live = append(live, p)
		case owner && p.IsScheduled():
			scheduled = append(scheduled, p)
		case owner:
			drafts = append(drafts, p)
		}
	}

	d := base(r, "Likho", "home")
	d["Posts"] = live
	d["Drafts"] = drafts
	d["Scheduled"] = scheduled
	d["Excerpt"] = excerpt // template se call karne ke liye
	d["Imported"] = r.URL.Query().Get("imported")
	render(w, "index", d)
}

func handlePost(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	p, ok := store.Get(slug)
	if !ok {
		notFound(w, r)
		return
	}

	// Draft ya abhi-schedule-me-hai post sirf logged-in owner ko dikhta hai -
	// seedha link se bhi anonymous banda nahi dekh sakta.
	if !p.IsLive() && currentUser(r) == nil {
		notFound(w, r)
		return
	}

	// view count badha do. Alag goroutine me isliye kyunki isme DB write hoti hai
	// aur reader ko uske liye wait nahi karna chahiye.
	go store.AddView(slug)

	d := base(r, p.Title, "")
	d["Post"] = p
	d["HTML"] = renderMarkdown(p.Body)
	d["Related"] = relatedTo(p, 3)
	d["Backlinks"] = backlinksTo(slug)
	d["Revisions"] = len(store.Revisions(slug))
	render(w, "post", d)
}

func handleEditor(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	// naya post: /new?title=Kuch  (broken [[wikilink]] click karne se yahi aata hai)
	p := &Post{Published: true, Title: r.URL.Query().Get("title")}
	isNew := true

	if slug != "" {
		found, ok := store.Get(slug)
		if !ok {
			notFound(w, r)
			return
		}
		p = found
		isNew = false
	}

	d := base(r, map[bool]string{true: "Naya post", false: "Edit: " + p.Title}[isNew], "new")
	d["Post"] = p
	d["IsNew"] = isNew
	d["TagString"] = strings.Join(p.Tags, ", ")
	d["Suggested"] = suggestTags(p.Body, p.Title)
	render(w, "editor", d)
}

func handleTag(w http.ResponseWriter, r *http.Request) {
	tag := r.PathValue("tag")
	owner := currentUser(r) != nil

	var out []*Post
	for _, p := range store.All() {
		if !owner && !p.IsLive() {
			continue // anonymous ko draft/scheduled posts tag page pe bhi nahi dikhne chahiye
		}
		for _, t := range p.Tags {
			if strings.EqualFold(t, tag) {
				out = append(out, p)
				break
			}
		}
	}

	d := base(r, "#"+tag, "")
	d["Posts"] = out
	d["Drafts"] = nil
	d["Scheduled"] = nil
	d["Excerpt"] = excerpt
	d["FilterTag"] = tag
	render(w, "index", d)
}

func handleGraphPage(w http.ResponseWriter, r *http.Request) {
	render(w, "graph", base(r, "Blog ka naksha", "graph"))
}

func handleHistory(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")

	p, ok := store.Get(slug)
	if !ok {
		notFound(w, r)
		return
	}

	d := base(r, "History: "+p.Title, "")
	d["Post"] = p
	d["Revs"] = store.Revisions(slug)
	render(w, "history", d)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	posts := store.All()

	totalViews, totalWords := 0, 0
	for _, p := range posts {
		totalViews += p.Views
		totalWords += len(strings.Fields(p.Body))
	}

	// sabse zyada padhe gaye 5 posts
	top := make([]*Post, len(posts))
	copy(top, posts)
	sort.Slice(top, func(i, j int) bool { return top[i].Views > top[j].Views })
	if len(top) > 5 {
		top = top[:5]
	}

	d := base(r, "Dashboard", "dashboard")
	d["Posts"] = posts
	d["TotalViews"] = totalViews
	d["TotalWords"] = totalWords
	d["Streak"] = writingStreak(posts)
	d["Top"] = top
	render(w, "dashboard", d)
}

// Kitne din se lagatar likh rahe ho. Duolingo wala streak, par blog ke liye.
// Motivation ke liye rakha hai - aur demo me acha dikhta hai.
func writingStreak(posts []*Post) int {
	if len(posts) == 0 {
		return 0
	}

	// har din ko "2006-01-02" string bana ke set me daal do
	days := map[string]bool{}
	for _, p := range posts {
		days[p.Created.Format("2006-01-02")] = true
		days[p.Updated.Format("2006-01-02")] = true
	}

	streak := 0
	day := time.Now()

	// agar aaj nahi likha toh kal se ginna shuru karo -
	// warna raat 12 baje sabka streak toot jata, jo bakwas hoga
	if !days[day.Format("2006-01-02")] {
		day = day.AddDate(0, 0, -1)
	}
	for days[day.Format("2006-01-02")] {
		streak++
		day = day.AddDate(0, 0, -1)
	}
	return streak
}

// ---------- JSON APIs ----------

func apiSave(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Slug      string `json:"slug"`
		Title     string `json:"title"`
		Body      string `json:"body"`
		Tags      string `json:"tags"`
		Published bool   `json:"published"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		jsonErr(w, "form samajh nahi aaya", http.StatusBadRequest)
		return
	}

	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		jsonErr(w, "title toh daal do yaar", http.StatusBadRequest)
		return
	}

	p := &Post{
		Title:     in.Title,
		Body:      in.Body,
		Tags:      splitTags(in.Tags),
		Published: in.Published,
		Slug:      in.Slug,
	}
	// requireAuthAPI ne pehle hi check kar liya hai ki koi logged in hai
	if u := currentUser(r); u != nil {
		p.AuthorID = u.ID
	}
	// purana post edit ho raha hai toh series/schedule jo pehle se set tha wahi rakho -
	// editor abhi in dono ko badalne ka UI nahi deta (wo Task 4 me aayega)
	if old, existed := store.Get(p.Slug); existed {
		p.SeriesID = old.SeriesID
		p.PublishAt = old.PublishAt
	}

	// naya post hai toh title se slug banao
	if p.Slug == "" {
		p.Slug = slugify(in.Title)

		// same naam ka post pehle se hai? Peeche number laga do.
		// "-2", "-3" karte jao jab tak khali slot na mile.
		if _, taken := store.Get(p.Slug); taken {
			orig := p.Slug
			for i := 2; ; i++ {
				try := orig + "-" + itoa(i)
				if _, taken := store.Get(try); !taken {
					p.Slug = try
					break
				}
			}
		}
	}

	if err := store.Save(p); err != nil {
		jsonErr(w, "save nahi hua: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{"ok": true, "slug": p.Slug})
}

func apiDelete(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Slug string `json:"slug"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		jsonErr(w, "slug chahiye", http.StatusBadRequest)
		return
	}

	if err := store.Delete(in.Slug); err != nil {
		jsonErr(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

// Reader kitna scroll kiya. Browser ye sendBeacon se bhejta hai jab tab band hota hai,
// isliye response ka koi matlab nahi - bas 204 de do.
func apiDepth(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Slug   string `json:"slug"`
		Decile int    `json:"decile"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err == nil {
		store.RecordDepth(in.Slug, in.Decile)
	}
	w.WriteHeader(http.StatusNoContent)
}

func apiRestore(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Slug       string `json:"slug"`
		RevisionID int64  `json:"revisionId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		jsonErr(w, "kaunsa version restore karna hai?", http.StatusBadRequest)
		return
	}

	if err := store.RestoreRevision(in.Slug, in.RevisionID); err != nil {
		jsonErr(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func apiSearch(w http.ResponseWriter, r *http.Request) {
	hits := searchPosts(r.URL.Query().Get("q"))
	owner := currentUser(r) != nil

	// template ko struct bhejne ke bajaye JSON me flat kar dete hain,
	// front-end ka kaam aasan ho jata hai
	out := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		if !owner && !h.Post.IsLive() {
			continue // anonymous ko search se bhi draft/scheduled posts nahi milne chahiye
		}
		out = append(out, map[string]any{
			"slug":    h.Post.Slug,
			"title":   h.Post.Title,
			"snippet": h.Snippet,
			"tags":    h.Post.Tags,
			"draft":   !h.Post.Published,
		})
	}
	writeJSON(w, map[string]any{"results": out})
}

// Graph view ka data. Nodes = posts, links = [[wikilinks]].
func apiGraph(w http.ResponseWriter, r *http.Request) {
	owner := currentUser(r) != nil
	posts := store.All()

	nodes := make([]map[string]any, 0, len(posts))
	links := make([]map[string]any, 0)
	visible := map[string]bool{}

	for _, p := range posts {
		if !owner && !p.IsLive() {
			continue // anonymous ko naksha me bhi draft/scheduled posts nahi dikhne chahiye
		}
		visible[p.Slug] = true
		nodes = append(nodes, map[string]any{
			"id":    p.Slug,
			"title": p.Title,
			"tags":  p.Tags,
			"views": p.Views,
			// node ka size words pe depend karta hai - bada post, bada circle
			"words": len(strings.Fields(p.Body)),
			"draft": !p.Published,
		})
	}
	for _, p := range posts {
		if !visible[p.Slug] {
			continue
		}
		for _, target := range linksFrom(p) {
			if !visible[target] {
				continue // hidden post ka link bhi leak nahi hona chahiye
			}
			links = append(links, map[string]any{"source": p.Slug, "target": target})
		}
	}

	writeJSON(w, map[string]any{"nodes": nodes, "links": links})
}

// ---------- chhote helpers ----------

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		// yahan tak aaye toh client already chala gaya hoga
		return
	}
}

func jsonErr(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": msg})
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	d := base(r, "Ye page nahi mila", "")
	d["Posts"] = nil
	d["Drafts"] = nil
	d["Scheduled"] = nil
	d["Excerpt"] = excerpt
	d["NotFound"] = true
	render(w, "index", d)
}

// strconv sirf ek jagah chahiye tha, chhota sa likh diya
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}

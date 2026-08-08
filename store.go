package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Ek post. Pehle ye seedha .md file thi, ab SQLite ki ek row hai -
// par struct ka shape zyada nahi badla, isliye templates aur baaki
// code ko zyada chhedna nahi pada.
type Post struct {
	ID        int64
	Slug      string
	Title     string
	Body      string // raw markdown, jaisa user ne likha
	Tags      []string
	Created   time.Time
	Updated   time.Time
	Published bool
	Views     int

	// Set hai aur future me hai toh post "scheduled" hai - Published=true
	// hone ke bawajood anonymous reader ko tab tak nahi dikhega.
	PublishAt *time.Time

	AuthorID   int64
	AuthorName string // users.display_name, join karke bhar diya jata hai

	SeriesID    *int64
	SeriesSlug  string
	SeriesTitle string

	// Reading analytics ke liye. Post ko 10 hisso me baanta hai,
	// Depth[3] matlab kitne log 40% tak pahunche.
	Depth [10]int
}

// Anonymous/logged-out reader ko dikhna chahiye ya nahi.
func (p *Post) IsLive() bool {
	return p.Published && (p.PublishAt == nil || !p.PublishAt.After(time.Now()))
}

// Published hai par future me schedule hai.
func (p *Post) IsScheduled() bool {
	return p.Published && p.PublishAt != nil && p.PublishAt.After(time.Now())
}

func (p *Post) IsDraft() bool { return !p.Published }

type Revision struct {
	ID    int64
	Stamp time.Time
	Title string
	Body  string
}

// Store ab SQLite ke upar ek chhota sa layer hai. Posts ka poora set
// memory me bhi cache rehta hai (map + mutex, bilkul pehle jaisa) taaki
// har page view par baar baar DB query na maarni pade - graph, related
// posts, tags, search sab isi cache pe chalte hain jaisa pehle files pe
// chalte the. Sirf writes aur naye per-post features (comments, reactions)
// seedha DB se baat karte hain.
type Store struct {
	db  *sql.DB
	dir string

	mu    sync.RWMutex
	posts map[string]*Post // slug -> post
	byID  map[int64]*Post
}

func OpenStore(dir string) (*Store, error) {
	for _, sub := range []string{"posts", "revisions", "media"} {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return nil, err
		}
	}

	db, err := openDB(dir)
	if err != nil {
		return nil, err
	}

	s := &Store{db: db, dir: dir}

	// Purane .md files hain aur DB abhi khali hai? Ek hi baar import kar do.
	// (Ye check khud migrate.go ke andar hai, taaki dobara chalane pe
	// dobara import na ho jaye.)
	if err := migrateMarkdownIfNeeded(db, dir); err != nil {
		fmt.Printf("  ⚠️  purane posts migrate karte waqt dikkat aayi: %v\n", err)
	}

	if err := s.reloadLocked(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *Store) Close() error   { return s.db.Close() }
func (s *Store) DB() *sql.DB    { return s.db }
func (s *Store) DataDir() string { return s.dir }
func (s *Store) MediaDir() string { return filepath.Join(s.dir, "media") }

func (s *Store) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.posts)
}

// Saare posts, naye pehle. Har baar sort karte hain - 50-100 posts ke liye
// ye bilkul theek hai, isko optimize karna waste of time hoga.
func (s *Store) All() []*Post {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*Post, 0, len(s.posts))
	for _, p := range s.posts {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Created.After(out[j].Created)
	})
	return out
}

// Jo kuch bhi ek anonymous/logged-out reader ko dikhna chahiye.
func (s *Store) Live() []*Post {
	var out []*Post
	for _, p := range s.All() {
		if p.IsLive() {
			out = append(out, p)
		}
	}
	return out
}

func (s *Store) Drafts() []*Post {
	var out []*Post
	for _, p := range s.All() {
		if p.IsDraft() {
			out = append(out, p)
		}
	}
	return out
}

func (s *Store) Scheduled() []*Post {
	var out []*Post
	for _, p := range s.All() {
		if p.IsScheduled() {
			out = append(out, p)
		}
	}
	return out
}

func (s *Store) Get(slug string) (*Post, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.posts[slug]
	return p, ok
}

func (s *Store) GetByID(id int64) (*Post, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.byID[id]
	return p, ok
}

// Series ke andar ke posts, series ke andar ke order me (purana pehle).
func (s *Store) BySeries(seriesID int64) []*Post {
	var out []*Post
	for _, p := range s.All() {
		if p.SeriesID != nil && *p.SeriesID == seriesID {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Created.Before(out[j].Created) })
	return out
}

// Editor se save. Naya post ho ya purana edit - dono yahin se guzarte hain.
// Purana version revisions table me chala jata hai taaki "time travel" kaam kare.
func (s *Store) Save(p *Post) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	old, existed := s.posts[p.Slug]
	now := time.Now()

	if existed {
		// purana content revision bana do. Fail ho bhi jaye toh save nahi rokna -
		// history nice-to-have hai, post zaroori hai.
		if _, err := s.db.Exec(
			`INSERT INTO revisions (post_id, title, body, created_at) VALUES (?,?,?,?)`,
			old.ID, old.Title, old.Body, timeToStr(now)); err != nil {
			fmt.Printf("  ⚠️  history save nahi hui: %v\n", err)
		}
		// views aur depth editor se nahi aate, purane wale hi rakho
		p.ID = old.ID
		p.Views = old.Views
		p.Depth = old.Depth
		p.Created = old.Created
	}
	if p.Created.IsZero() {
		p.Created = now
	}
	p.Updated = now

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	depthCSV := depthToCSV(p.Depth)

	if existed {
		_, err = tx.Exec(`UPDATE posts SET
				title=?, body=?, author_id=?, series_id=?, published=?,
				publish_at=?, updated_at=?, views=?, depth=?
			WHERE id=?`,
			p.Title, p.Body, nullInt64(p.AuthorID), nullSeriesID(p.SeriesID),
			boolToInt(p.Published), nullTimeToStr(p.PublishAt), timeToStr(p.Updated),
			p.Views, depthCSV, p.ID)
	} else {
		var res sql.Result
		res, err = tx.Exec(`INSERT INTO posts
				(slug, title, body, author_id, series_id, published, publish_at, created_at, updated_at, views, depth)
			VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
			p.Slug, p.Title, p.Body, nullInt64(p.AuthorID), nullSeriesID(p.SeriesID),
			boolToInt(p.Published), nullTimeToStr(p.PublishAt), timeToStr(p.Created),
			timeToStr(p.Updated), p.Views, depthCSV)
		if err == nil {
			p.ID, _ = res.LastInsertId()
		}
	}
	if err != nil {
		return fmt.Errorf("post save nahi hua: %w", err)
	}

	// tags dobara likh do - saaf tarika hai, delete karke jo abhi chahiye wo daal do
	if _, err := tx.Exec(`DELETE FROM post_tags WHERE post_id=?`, p.ID); err != nil {
		return err
	}
	for _, t := range p.Tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		tagID, err := upsertTag(tx, t)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO post_tags (post_id, tag_id) VALUES (?,?)`, p.ID, tagID); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return s.reloadLocked()
}

func (s *Store) Delete(slug string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.posts[slug]
	if !ok {
		return fmt.Errorf("post nahi mila: %s", slug)
	}
	// comments/reactions/revisions/post_tags sab ON DELETE CASCADE se apne aap saaf ho jate hain
	if _, err := s.db.Exec(`DELETE FROM posts WHERE id=?`, p.ID); err != nil {
		return err
	}
	return s.reloadLocked()
}

// View count badhao. Sirf DB row + in-memory field update - poora cache
// reload karna is hot path ke liye zaroorat se zyada mehnga hoga.
func (s *Store) AddView(slug string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.posts[slug]
	if !ok {
		return
	}
	p.Views++
	if _, err := s.db.Exec(`UPDATE posts SET views=? WHERE id=?`, p.Views, p.ID); err != nil {
		fmt.Printf("  ⚠️  view count save nahi hua: %v\n", err)
	}
}

// Reader kitna neeche tak scroll kiya, wo record karo (0 se 9 tak).
func (s *Store) RecordDepth(slug string, decile int) {
	if decile < 0 || decile > 9 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	p, ok := s.posts[slug]
	if !ok {
		return
	}
	for i := 0; i <= decile; i++ {
		p.Depth[i]++
	}
	if _, err := s.db.Exec(`UPDATE posts SET depth=? WHERE id=?`, depthToCSV(p.Depth), p.ID); err != nil {
		fmt.Printf("  ⚠️  depth save nahi hua: %v\n", err)
	}
}

func (s *Store) Revisions(slug string) []Revision {
	p, ok := s.Get(slug)
	if !ok {
		return nil
	}
	rows, err := s.db.Query(`SELECT id, title, body, created_at FROM revisions WHERE post_id=? ORDER BY created_at DESC`, p.ID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []Revision
	for rows.Next() {
		var r Revision
		var stamp string
		if err := rows.Scan(&r.ID, &r.Title, &r.Body, &stamp); err != nil {
			continue
		}
		r.Stamp = strToTime(stamp)
		out = append(out, r)
	}
	return out
}

// Purane version pe wapas jao. Note: ye khud bhi ek naya revision banata hai
// (Save() ke through), toh restore ko undo bhi kiya ja sakta hai.
func (s *Store) RestoreRevision(slug string, revisionID int64) error {
	p, ok := s.Get(slug)
	if !ok {
		return fmt.Errorf("post ab exist hi nahi karta: %s", slug)
	}

	var title, body string
	err := s.db.QueryRow(`SELECT title, body FROM revisions WHERE id=? AND post_id=?`, revisionID, p.ID).
		Scan(&title, &body)
	if err != nil {
		return fmt.Errorf("wo version padha nahi gaya: %w", err)
	}

	cp := *p // copy - cache wale pointer ko seedha nahi chhedna
	cp.Title = title
	cp.Body = body
	return s.Save(&cp)
}

// Poora in-memory cache DB se dobara bhar do. Save/Delete jaisi kam-baar
// wali operations ke baad chalta hai - 50-100 posts ke liye do query
// (posts + tags) ka kharcha kuch bhi nahi hai.
func (s *Store) reloadLocked() error {
	rows, err := s.db.Query(`
		SELECT p.id, p.slug, p.title, p.body, p.author_id, COALESCE(u.display_name, ''),
		       p.series_id, COALESCE(sr.slug, ''), COALESCE(sr.title, ''),
		       p.published, p.publish_at, p.created_at, p.updated_at, p.views, p.depth
		FROM posts p
		LEFT JOIN users  u  ON u.id = p.author_id
		LEFT JOIN series sr ON sr.id = p.series_id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	posts := map[string]*Post{}
	byID := map[int64]*Post{}

	for rows.Next() {
		var p Post
		var authorID sql.NullInt64
		var seriesID sql.NullInt64
		var publishAt sql.NullString
		var createdAt, updatedAt, depthCSV string
		var publishedInt int

		if err := rows.Scan(&p.ID, &p.Slug, &p.Title, &p.Body, &authorID, &p.AuthorName,
			&seriesID, &p.SeriesSlug, &p.SeriesTitle,
			&publishedInt, &publishAt, &createdAt, &updatedAt, &p.Views, &depthCSV); err != nil {
			return err
		}
		p.AuthorID = authorID.Int64
		if seriesID.Valid {
			id := seriesID.Int64
			p.SeriesID = &id
		}
		p.Published = publishedInt != 0
		p.PublishAt = strToNullTime(publishAt)
		p.Created = strToTime(createdAt)
		p.Updated = strToTime(updatedAt)
		p.Depth = csvToDepth(depthCSV)

		posts[p.Slug] = &p
		byID[p.ID] = &p
	}
	if err := rows.Err(); err != nil {
		return err
	}

	tagRows, err := s.db.Query(`SELECT pt.post_id, t.name FROM post_tags pt JOIN tags t ON t.id = pt.tag_id`)
	if err != nil {
		return err
	}
	defer tagRows.Close()
	for tagRows.Next() {
		var postID int64
		var name string
		if err := tagRows.Scan(&postID, &name); err != nil {
			return err
		}
		if p, ok := byID[postID]; ok {
			p.Tags = append(p.Tags, name)
		}
	}
	if err := tagRows.Err(); err != nil {
		return err
	}
	for _, p := range posts {
		sort.Strings(p.Tags) // stable order, warna har reload pe tags idhar-udhar hote
	}

	s.posts = posts
	s.byID = byID
	return nil
}

func upsertTag(tx *sql.Tx, name string) (int64, error) {
	if _, err := tx.Exec(`INSERT OR IGNORE INTO tags (name) VALUES (?)`, name); err != nil {
		return 0, err
	}
	var id int64
	if err := tx.QueryRow(`SELECT id FROM tags WHERE name=?`, name).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

// ---------- chhote SQL helpers ----------

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullInt64(v int64) sql.NullInt64 {
	if v == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: v, Valid: true}
}

func nullSeriesID(id *int64) sql.NullInt64 {
	if id == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *id, Valid: true}
}

func depthToCSV(d [10]int) string {
	parts := make([]string, 10)
	for i, v := range d {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ",")
}

func csvToDepth(s string) [10]int {
	var d [10]int
	for i, part := range strings.Split(s, ",") {
		if i >= 10 {
			break
		}
		d[i], _ = strconv.Atoi(strings.TrimSpace(part))
	}
	return d
}

// ---------- chhote helpers (pure functions, DB se koi lena dena nahi) ----------

var slugKill = regexp.MustCompile(`[^a-z0-9]+`)

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugKill.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		// title agar poora hindi/emoji me hai toh slug khali reh jayega,
		// aise me timestamp se kaam chala lete hain
		s = "post-" + time.Now().Format("20060102-150405")
	}
	if len(s) > 60 {
		s = strings.Trim(s[:60], "-")
	}
	return s
}

func splitTags(s string) []string {
	var out []string
	for _, t := range strings.Split(s, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func readingTime(body string) int {
	// average banda ~200 word/min padhta hai. Kam se kam 1 min dikhao,
	// "0 min read" ajeeb lagta hai.
	n := len(strings.Fields(body))
	m := n / 200
	if m < 1 {
		return 1
	}
	return m
}

func humanTime(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "abhi abhi"
	case d < time.Hour:
		return fmt.Sprintf("%d min pehle", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d ghante pehle", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%d din pehle", int(d.Hours()/24))
	default:
		return t.Format("2 Jan 2006")
	}
}

package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// legacyPost - purane blog-data/posts/*.md format se nikala hua data.
// Sirf migration ke liye zinda hai, ab koi post isse seedha nahi bachta.
type legacyPost struct {
	Slug      string
	Title     string
	Body      string
	Tags      []string
	Created   time.Time
	Updated   time.Time
	Published bool
	Views     int
	Depth     [10]int
}

// Purane version me Likho posts ko flat markdown files me rakhta tha.
// Ab wo SQLite me jaate hain - agar kisi ke paas purani .md files hain
// aur DB abhi khali hai, toh unhe ek hi transaction me import kar dete
// hain taaki upgrade karne pe kisi ka likha hua na chhute.
//
// Naye install pe ye function turant return ho jata hai (na .md files
// hoti hain, na DB me kuch hota) - seedFirstPost baad me demo post daal deta hai.
func migrateMarkdownIfNeeded(db *sql.DB, dir string) error {
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM posts`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil // pehle se migrate ho chuka hai (ya koi naya post ban chuka)
	}

	files, err := filepath.Glob(filepath.Join(dir, "posts", "*.md"))
	if err != nil || len(files) == 0 {
		return nil // purana kuch hai hi nahi
	}

	var legacy []legacyPost
	for _, f := range files {
		lp, err := parseLegacyPostFile(f)
		if err != nil {
			fmt.Printf("  ⚠️  %s migrate nahi hua (%v) - skip kar rahe hain\n", filepath.Base(f), err)
			continue
		}
		legacy = append(legacy, lp)
	}
	if len(legacy) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, lp := range legacy {
		res, err := tx.Exec(`INSERT INTO posts
				(slug, title, body, published, created_at, updated_at, views, depth)
			VALUES (?,?,?,?,?,?,?,?)`,
			lp.Slug, lp.Title, lp.Body, boolToInt(lp.Published),
			timeToStr(lp.Created), timeToStr(lp.Updated), lp.Views, depthToCSV(lp.Depth))
		if err != nil {
			return fmt.Errorf("%s insert nahi hua: %w", lp.Slug, err)
		}
		postID, _ := res.LastInsertId()

		for _, tag := range lp.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			if _, err := tx.Exec(`INSERT OR IGNORE INTO tags (name) VALUES (?)`, tag); err != nil {
				return err
			}
			var tagID int64
			if err := tx.QueryRow(`SELECT id FROM tags WHERE name=?`, tag).Scan(&tagID); err != nil {
				return err
			}
			if _, err := tx.Exec(`INSERT OR IGNORE INTO post_tags (post_id, tag_id) VALUES (?,?)`, postID, tagID); err != nil {
				return err
			}
		}

		// purani revisions bhi le aate hain, taaki History page khali na dikhe
		revFiles, _ := filepath.Glob(filepath.Join(dir, "revisions", lp.Slug, "*.md"))
		for _, rf := range revFiles {
			rp, err := parseLegacyPostFile(rf)
			if err != nil {
				continue
			}
			stamp := rp.Updated
			base := strings.TrimSuffix(filepath.Base(rf), ".md")
			if t, err := time.ParseInLocation("20060102-150405.000", base, time.Local); err == nil {
				stamp = t
			}
			if _, err := tx.Exec(`INSERT INTO revisions (post_id, title, body, created_at) VALUES (?,?,?,?)`,
				postID, rp.Title, rp.Body, timeToStr(stamp)); err != nil {
				return err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	fmt.Printf("  📦  %d purane post migrate ho gaye purani markdown files se\n", len(legacy))
	return nil
}

// Purana frontmatter format:
//
//	---
//	title: Mera pehla post
//	tags: go, hackathon
//	---
//	yahan se markdown shuru
//
// Hu-ba-hu wahi parser jo pehle store.go me tha - bas ab sirf migration
// ke liye use hota hai, isliye naya *Post nahi, legacyPost banata hai.
func parseLegacyPostFile(path string) (legacyPost, error) {
	f, err := os.Open(path)
	if err != nil {
		return legacyPost{}, err
	}
	defer f.Close()

	p := legacyPost{
		Slug:      strings.TrimSuffix(filepath.Base(path), ".md"),
		Published: true,
	}

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // lambi lines ke liye

	inHeader := false
	first := true
	var body []string

	for sc.Scan() {
		line := sc.Text()

		if first {
			first = false
			if strings.TrimSpace(line) == "---" {
				inHeader = true
				continue
			}
		}

		if inHeader {
			if strings.TrimSpace(line) == "---" {
				inHeader = false
				continue
			}
			key, val, found := strings.Cut(line, ":")
			if !found {
				continue
			}
			applyLegacyHeader(&p, strings.TrimSpace(key), strings.TrimSpace(val))
			continue
		}

		body = append(body, line)
	}
	if err := sc.Err(); err != nil {
		return legacyPost{}, err
	}

	p.Body = strings.TrimLeft(strings.Join(body, "\n"), "\n")
	if p.Title == "" {
		p.Title = p.Slug
	}
	if p.Created.IsZero() {
		if st, err := os.Stat(path); err == nil {
			p.Created = st.ModTime()
		} else {
			p.Created = time.Now()
		}
	}
	if p.Updated.IsZero() {
		p.Updated = p.Created
	}
	return p, nil
}

func applyLegacyHeader(p *legacyPost, key, val string) {
	switch strings.ToLower(key) {
	case "title":
		p.Title = val
	case "tags":
		p.Tags = splitTags(val)
	case "created":
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			p.Created = t
		}
	case "updated":
		if t, err := time.Parse(time.RFC3339, val); err == nil {
			p.Updated = t
		}
	case "published":
		p.Published = val != "false" && val != "no" && val != "0"
	case "views":
		p.Views, _ = strconv.Atoi(val)
	case "depth":
		for i, part := range strings.Split(val, ",") {
			if i >= 10 {
				break
			}
			p.Depth[i], _ = strconv.Atoi(strings.TrimSpace(part))
		}
	}
}

package main

import (
	"database/sql"
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	// modernc.org/sqlite pure Go me likha hua hai - koi C compiler, koi cgo
	// nahi chahiye. Isliye CGO_ENABLED=0 wala cross-compile (build.sh dekho)
	// abhi bhi kaam karta hai aur binary abhi bhi ek hi .exe rehta hai.
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// Poori app me ek hi *sql.DB, ek hi connection. Blog chhota hai (LAN pe
// chalne wala, ek writer), isliye concurrency ke liye fancy pooling ki
// zarurat nahi - ek connection pe sab serialize ho jaye, yahi sabse
// simple aur sabse safe hai. WAL mode isliye taki reads block na ho
// jab koi write chal raha ho.
func openDB(dir string) (*sql.DB, error) {
	path := filepath.Join(dir, "likho.db")

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("database khul nahi payi: %w", err)
	}

	// Ek hi connection - isse "database is locked" wali dikkat kabhi nahi aati,
	// aur code me alag se mutex rakhne ki zarurat khatam ho jati hai.
	db.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			db.Close()
			return nil, fmt.Errorf("pragma set nahi hua (%s): %w", p, err)
		}
	}

	if err := runSchema(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// schema.sql me kayi CREATE TABLE hain, ek ke baad ek semicolon se alag.
// Kuch SQLite drivers ek Exec call me multiple statements nahi chalate,
// isliye khud split karke ek ek statement bhejte hain - itni si DDL ke
// liye poora SQL parser laana overkill hota.
func runSchema(db *sql.DB) error {
	for _, stmt := range strings.Split(schemaSQL, ";") {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("schema apply nahi hua: %w\n  statement: %s", err, stmt)
		}
	}
	return nil
}

// ---------- chhote time helpers, saari dates DB me RFC3339 text hoti hain ----------
// (mattn/modernc sqlite drivers ke time.Time scanning me farak hota hai,
// isliye khud text me store/parse karna zyada bharosemand hai - jaisa
// purana store.go bhi karta tha.)

func timeToStr(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func strToTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// nullable version - scheduling (PublishAt) aur series (SeriesID) jaisi
// optional cheezon ke liye.
func nullTimeToStr(t *time.Time) sql.NullString {
	if t == nil || t.IsZero() {
		return sql.NullString{}
	}
	return sql.NullString{String: timeToStr(*t), Valid: true}
}

func strToNullTime(ns sql.NullString) *time.Time {
	if !ns.Valid || ns.String == "" {
		return nil
	}
	t := strToTime(ns.String)
	if t.IsZero() {
		return nil
	}
	return &t
}

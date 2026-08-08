package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Likho ab "single admin" model pe chalta hai - ek hi owner account,
// pehle run pe /setup se banta hai. Us account ke bina koi post likh/edit/
// delete nahi kar sakta, dashboard nahi dekh sakta - par published posts
// koi bhi (login ke bina) padh sakta hai. Ye ab ek asli "blog" hai,
// koi bhi-kisi-ka-bhi-editable LAN tool nahi.

type User struct {
	ID          int64
	Username    string
	DisplayName string
	Bio         string
	Avatar      string
	CreatedAt   time.Time
}

const (
	sessionCookieName = "likho_session"
	sessionTTL        = 30 * 24 * time.Hour
)

// ---------- users ----------

func usersExist() bool {
	var n int
	_ = store.DB().QueryRow(`SELECT COUNT(*) FROM users`).Scan(&n)
	return n > 0
}

func getUserByID(id int64) (*User, bool) {
	var u User
	var createdAt string
	err := store.DB().QueryRow(
		`SELECT id, username, display_name, bio, avatar, created_at FROM users WHERE id=?`, id,
	).Scan(&u.ID, &u.Username, &u.DisplayName, &u.Bio, &u.Avatar, &createdAt)
	if err != nil {
		return nil, false
	}
	u.CreatedAt = strToTime(createdAt)
	return &u, true
}

// username + password_hash dono chahiye login verify karne ke liye,
// isliye hash alag return karte hain (User struct me rakhna theek nahi lagta).
func getUserByUsername(username string) (*User, string, bool) {
	var u User
	var hash, createdAt string
	err := store.DB().QueryRow(
		`SELECT id, username, password_hash, display_name, bio, avatar, created_at FROM users WHERE username=?`, username,
	).Scan(&u.ID, &u.Username, &hash, &u.DisplayName, &u.Bio, &u.Avatar, &createdAt)
	if err != nil {
		return nil, "", false
	}
	u.CreatedAt = strToTime(createdAt)
	return &u, hash, true
}

func createUser(username, password, displayName string) (*User, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, fmt.Errorf("username khali nahi ho sakta")
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("password kam se kam 8 characters ka hona chahiye")
	}
	if displayName = strings.TrimSpace(displayName); displayName == "" {
		displayName = username
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	res, err := store.DB().Exec(
		`INSERT INTO users (username, password_hash, display_name, avatar, created_at) VALUES (?,?,?,?,?)`,
		username, string(hash), displayName, "✍️", timeToStr(now))
	if err != nil {
		return nil, fmt.Errorf("account ban nahi paya (shayad ye username pehle se hai)")
	}
	id, _ := res.LastInsertId()
	return &User{ID: id, Username: username, DisplayName: displayName, Avatar: "✍️", CreatedAt: now}, nil
}

// ---------- sessions + CSRF ----------

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Session banao aur cookie set kar do. Token DB me hai (file me nahi),
// isliye logout/revoke bilkul seedha hai - bas row delete karo.
func createSession(w http.ResponseWriter, r *http.Request, userID int64) error {
	token, err := randomToken()
	if err != nil {
		return err
	}
	csrf, err := randomToken()
	if err != nil {
		return err
	}

	now := time.Now()
	if _, err := store.DB().Exec(
		`INSERT INTO sessions (token, user_id, csrf_secret, created_at, expires_at) VALUES (?,?,?,?,?)`,
		token, userID, csrf, timeToStr(now), timeToStr(now.Add(sessionTTL))); err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		Expires:  now.Add(sessionTTL),
	})
	return nil
}

func destroySession(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		_, _ = store.DB().Exec(`DELETE FROM sessions WHERE token=?`, c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1})
}

type authInfo struct {
	User *User
	CSRF string
}

type ctxKey int

const authCtxKey ctxKey = 0

// Cookie se session dhoondo, user aur uska CSRF secret request context me
// chipka do. Ye har request pe chalta hai (chahe login ho ya na ho) taaki
// templates hamesha jaan sakein "kaun dekh raha hai" - bina isse har
// public page bhi login-gated dikhta.
func withUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(sessionCookieName); err == nil && c.Value != "" {
			var userID int64
			var csrf, expiresAt string
			err := store.DB().QueryRow(
				`SELECT user_id, csrf_secret, expires_at FROM sessions WHERE token=?`, c.Value,
			).Scan(&userID, &csrf, &expiresAt)

			if err == nil && strToTime(expiresAt).After(time.Now()) {
				if u, ok := getUserByID(userID); ok {
					r = r.WithContext(context.WithValue(r.Context(), authCtxKey, &authInfo{User: u, CSRF: csrf}))
				}
			} else if err == nil {
				// expire ho chuka session, saaf kar do
				_, _ = store.DB().Exec(`DELETE FROM sessions WHERE token=?`, c.Value)
			}
		}
		next.ServeHTTP(w, r)
	})
}

func currentUser(r *http.Request) *User {
	info, _ := r.Context().Value(authCtxKey).(*authInfo)
	if info == nil {
		return nil
	}
	return info.User
}

func currentCSRF(r *http.Request) string {
	info, _ := r.Context().Value(authCtxKey).(*authInfo)
	if info == nil {
		return ""
	}
	return info.CSRF
}

// Header se CSRF token check karo. Sirf header dekhte hain (form/body nahi) -
// warna JSON POST ka body yahin consume ho jayega aur handler ka json.Decode
// khali haath rahega.
func checkCSRFHeader(r *http.Request) bool {
	want := currentCSRF(r)
	if want == "" {
		return false
	}
	got := r.Header.Get("X-CSRF-Token")
	return got != "" && got == want
}

// ---------- route guards ----------

// Page routes (browser navigation) ke liye - login nahi hai toh /login
// bhej do, wapas yahin aane ke liye ?next bhi bhej dete hain.
func requireAuthPage(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if currentUser(r) == nil {
			http.Redirect(w, r, "/login?next="+urlish(r.URL.Path), http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// JSON API routes ke liye - auth + CSRF dono check karte hain, aur
// dono fail hone pe browser redirect nahi, JSON error dete hain
// (warna fetch() ko HTML mil jata aur .json() crash ho jata).
func requireAuthAPI(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if currentUser(r) == nil {
			jsonErr(w, "pehle login karo", http.StatusUnauthorized)
			return
		}
		if !checkCSRFHeader(r) {
			jsonErr(w, "session purani ho gayi lagti hai, page reload karke phir try karo", http.StatusForbidden)
			return
		}
		next(w, r)
	}
}

// ---------- login rate limiting ----------
// Blog ab internet-facing ho sakta hai, isliye brute-force ko rokna zaroori
// hai. Kuch fancy nahi - ek IP se 5 galat try / 5 minute se zyada nahi.

var loginLimiter = struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
}{attempts: map[string][]time.Time{}}

const (
	loginMaxAttempts = 5
	loginWindow      = 5 * time.Minute
)

func loginAllowed(ip string) bool {
	loginLimiter.mu.Lock()
	defer loginLimiter.mu.Unlock()

	cutoff := time.Now().Add(-loginWindow)
	var recent []time.Time
	for _, t := range loginLimiter.attempts[ip] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	loginLimiter.attempts[ip] = recent
	return len(recent) < loginMaxAttempts
}

func loginRecordFailure(ip string) {
	loginLimiter.mu.Lock()
	defer loginLimiter.mu.Unlock()
	loginLimiter.attempts[ip] = append(loginLimiter.attempts[ip], time.Now())
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ---------- handlers: setup, login, logout ----------

// Pehli baar chalane pe koi user nahi hota - ye wizard admin account banata hai.
// Ek baar ban gaya toh ye route khud ko band kar leta hai (upar redirect dekho).
func handleSetup(w http.ResponseWriter, r *http.Request) {
	if usersExist() {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			renderSetup(w, r, "form samajh nahi aaya")
			return
		}
		password := r.FormValue("password")
		if password != r.FormValue("confirm") {
			renderSetup(w, r, "dono password same nahi hain")
			return
		}
		u, err := createUser(r.FormValue("username"), password, r.FormValue("display_name"))
		if err != nil {
			renderSetup(w, r, err.Error())
			return
		}
		if err := createSession(w, r, u.ID); err != nil {
			renderSetup(w, r, "account ban gaya par login nahi ho paya, /login se try karo")
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	renderSetup(w, r, "")
}

func renderSetup(w http.ResponseWriter, r *http.Request, errMsg string) {
	d := base(r, "Setup", "")
	d["Error"] = errMsg
	render(w, "setup", d)
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	if !usersExist() {
		http.Redirect(w, r, "/setup", http.StatusSeeOther)
		return
	}
	if currentUser(r) != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	next := r.URL.Query().Get("next")

	if r.Method == http.MethodPost {
		ip := clientIP(r)
		if !loginAllowed(ip) {
			renderLogin(w, r, "bahut saari koshishein ho gayi - thodi der ruk ke try karo", next)
			return
		}
		if err := r.ParseForm(); err != nil {
			renderLogin(w, r, "form samajh nahi aaya", next)
			return
		}
		next = r.FormValue("next")

		u, hash, ok := getUserByUsername(strings.TrimSpace(r.FormValue("username")))
		if !ok || bcrypt.CompareHashAndPassword([]byte(hash), []byte(r.FormValue("password"))) != nil {
			loginRecordFailure(ip)
			renderLogin(w, r, "username ya password galat hai", next)
			return
		}
		if err := createSession(w, r, u.ID); err != nil {
			renderLogin(w, r, "login nahi ho paya, dobara try karo", next)
			return
		}

		dest := "/dashboard"
		if strings.HasPrefix(next, "/") && !strings.HasPrefix(next, "//") {
			dest = next
		}
		http.Redirect(w, r, dest, http.StatusSeeOther)
		return
	}

	renderLogin(w, r, "", next)
}

func renderLogin(w http.ResponseWriter, r *http.Request, errMsg, next string) {
	d := base(r, "Log in", "")
	d["Error"] = errMsg
	d["Next"] = next
	render(w, "login", d)
}

func handleLogout(w http.ResponseWriter, r *http.Request) {
	// GET se koi seedha link bana ke logout na karva de isliye POST + CSRF hi chalta hai.
	if currentUser(r) != nil && r.FormValue("csrf_token") == currentCSRF(r) {
		destroySession(w, r)
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

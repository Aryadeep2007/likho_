// Likho - ek chhota sa blog jo kahin bhi chal jata hai.
// Idea simple hai: koi database nahi, koi install nahi. Bas ek .exe.
// Posts seedha .md files me save hote hain, toh tum Notepad me bhi kholke edit kar sakte ho.
//
// Chalane ka tarika: run.bat pe double click. Bas. Aur kuch nahi karna.
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Sab kuch binary ke andar hi ghusa diya hai, taki ek hi file bheji ja sake.
// Warna templates alag folder me rakhne padte aur log unhe delete kar dete :')
//
//go:embed web/templates/*.html
var tmplFS embed.FS

//go:embed web/static/*
var staticFS embed.FS

// global variables - haan pata hai, purist log naaraz honge, par ye ek hackathon app hai
// aur poore codebase me dependency injection karne ka time nahi tha.
var (
	store       *Store
	pages       = map[string]*template.Template{}
	currentPort = 4000 // QR banate waqt chahiye hota hai
)

// jo functions templates ke andar se call karne hote hain
var funcMap = template.FuncMap{
	"humanTime":   humanTime,
	"readingTime": readingTime,
	"join":        strings.Join,
	"safeHTML":    func(s string) template.HTML { return template.HTML(s) },
	"add":         func(a, b int) int { return a + b },
	"now":         func() int { return time.Now().Year() },

	// Go ki value ko JS me bhejne ke liye. Bina iske {{.Depth}} print hota hai
	// "[3 1 0]" jaisa, jo JSON nahi hai aur JS parse nahi kar pata.
	"json": func(v any) template.JS {
		b, err := json.Marshal(v)
		if err != nil {
			return template.JS("null")
		}
		return template.JS(b)
	},
}

func main() {
	port := flag.Int("port", 4000, "kaunse port pe server chalega")
	dataDir := flag.String("data", "blog-data", "posts is folder me rakhe jayenge")
	noBrowser := flag.Bool("no-browser", false, "browser apne aap mat kholo")
	flag.Parse()
	currentPort = *port

	banner()

	// data folder ka absolute path nikal lete hain. Windows pe agar koi
	// shortcut se chalaye toh working directory kahin aur point kar sakti hai,
	// isliye exe ke bagal wala folder use karte hain.
	base := *dataDir
	if !filepath.IsAbs(base) {
		if exe, err := os.Executable(); err == nil {
			base = filepath.Join(filepath.Dir(exe), *dataDir)
		}
	}

	var err error
	store, err = OpenStore(base)
	if err != nil {
		fatal("Blog folder khol nahi paye: %v", err)
	}
	log.Printf("[likho] %d post mile -> %s", store.Count(), base)

	// pehli baar chala rahe ho toh ek welcome post daal dete hain,
	// warna khali screen dekhke lagta hai app tuta hua hai.
	if store.Count() == 0 {
		seedFirstPost(store)
		log.Printf("[likho] pehli baar chala rahe ho, ek sample post daal diya hai")
	}

	loadTemplates()

	mux := http.NewServeMux()
	routes(mux)

	addr := fmt.Sprintf(":%d", *port)
	lan := lanURL(*port)

	fmt.Println()
	fmt.Println("  ✅  Blog chalu ho gaya!")
	fmt.Printf("      Is computer pe    :  http://localhost:%d\n", *port)
	if lan != "" {
		fmt.Printf("      Phone / dost ke PC:  %s\n", lan)
		fmt.Println("      (same wifi pe hona chahiye - ya QR scan kar lo header me)")
	}
	fmt.Println()
	fmt.Println("  Band karna ho toh is window me Ctrl+C dabao, ya bas window band kar do.")
	fmt.Println()

	if !*noBrowser {
		go openBrowser(fmt.Sprintf("http://localhost:%d", *port))
	}

	srv := &http.Server{
		Addr:         addr,
		Handler:      logRequests(withUser(mux)),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		// sabse common dikkat: port already in use. Uska seedha matlab bata dete hain
		// warna banda "bind: address already in use" dekhke ghabra jayega.
		if strings.Contains(err.Error(), "in use") || strings.Contains(err.Error(), "permission") {
			fatal("Port %d already busy hai.\n   Shayad Likho pehle se chal raha hai?\n   Doosre port pe try karo:  likho.exe -port 4001", *port)
		}
		fatal("Server band ho gaya: %v", err)
	}
}

func routes(mux *http.ServeMux) {
	// Go 1.22 se mux me method + path params aa gaye hain, toh koi router
	// library import karne ki zarurat hi nahi padi. Ek dependency bachi.

	// ---- public: koi bhi bina login ke padh sakta hai ----
	mux.HandleFunc("GET /", handleIndex)
	mux.HandleFunc("GET /p/{slug}", handlePost)
	mux.HandleFunc("GET /graph", handleGraphPage)
	mux.HandleFunc("GET /tag/{tag}", handleTag)
	mux.HandleFunc("GET /api/search", apiSearch)
	mux.HandleFunc("GET /api/graph", apiGraph)
	mux.HandleFunc("POST /api/depth", apiDepth) // reading analytics, anonymous hi bhejta hai

	// ---- auth ----
	mux.HandleFunc("GET /setup", handleSetup)
	mux.HandleFunc("POST /setup", handleSetup)
	mux.HandleFunc("GET /login", handleLogin)
	mux.HandleFunc("POST /login", handleLogin)
	mux.HandleFunc("POST /logout", handleLogout)

	// ---- sirf logged-in owner: likhna, edit karna, dekhna dashboard ----
	mux.HandleFunc("GET /new", requireAuthPage(handleEditor))
	mux.HandleFunc("GET /edit/{slug}", requireAuthPage(handleEditor))
	mux.HandleFunc("GET /dashboard", requireAuthPage(handleDashboard))
	mux.HandleFunc("GET /history/{slug}", requireAuthPage(handleHistory))
	mux.HandleFunc("GET /qr.png", requireAuthPage(handleQR))
	mux.HandleFunc("GET /export", requireAuthPage(handleExport))
	mux.HandleFunc("POST /import", requireAuthPage(handleImport))

	mux.HandleFunc("POST /api/save", requireAuthAPI(apiSave))
	mux.HandleFunc("POST /api/delete", requireAuthAPI(apiDelete))
	mux.HandleFunc("POST /api/restore", requireAuthAPI(apiRestore))

	// embed me files "web/static/..." pe hain, par browser "/static/..." maangta hai.
	// fs.Sub se "web/" wala prefix hata dete hain, warna sab 404 ho jata hai.
	assets, err := fs.Sub(staticFS, "web")
	if err != nil {
		fatal("static files nahi mile: %v", err)
	}
	mux.Handle("GET /static/", http.FileServer(http.FS(assets)))
}

// har page ke liye alag template set banana padta hai kyunki sab apna
// "content" block define karte hain. Ek hi set me daalo toh aakhri wala jeet jata hai.
func loadTemplates() {
	names := []string{"index", "post", "editor", "graph", "dashboard", "history", "setup", "login"}
	for _, n := range names {
		t := template.New("base.html").Funcs(funcMap)
		t = template.Must(t.ParseFS(tmplFS,
			"web/templates/base.html",
			"web/templates/"+n+".html",
		))
		pages[n] = t
	}
}

func render(w http.ResponseWriter, name string, data any) {
	t, ok := pages[name]
	if !ok {
		http.Error(w, "aisa koi page nahi hai: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "base.html", data); err != nil {
		// yahan tak aaye toh header already ja chuka hoga, isliye sirf log karte hain
		log.Printf("[likho] template %s render nahi hua: %v", name, err)
	}
}

// chhota sa request logger. Terminal me kuch chalta dikhta rahe toh
// bharosa rehta hai ki app zinda hai.
func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		// static files ka log nahi chahiye, terminal bhar jata hai
		if !strings.HasPrefix(r.URL.Path, "/static/") {
			log.Printf("%s %s  (%v)", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
		}
	})
}

func banner() {
	fmt.Println()
	fmt.Println("   __    _ __   __         ")
	fmt.Println("  / /   (_) /__/ /_  ____  ")
	fmt.Println(" / /   / / //_/ __ \\/ __ \\ ")
	fmt.Println("/ /___/ / ,< / / / / /_/ / ")
	fmt.Println("\\____/_/_/|_/_/ /_/\\____/  ")
	fmt.Println("  likho. apna blog, apne computer pe.")
	fmt.Println()
}

// error aane pe chillake exit karo, par window turant band mat hone do -
// warna Windows pe user ko error dikhega hi nahi, screen bas flash hogi.
func fatal(format string, args ...any) {
	fmt.Println()
	fmt.Printf("  ❌  "+format+"\n", args...)
	fmt.Println()
	if runtime.GOOS == "windows" {
		fmt.Println("  Band karne ke liye Enter dabao...")
		fmt.Scanln()
	}
	os.Exit(1)
}

func openBrowser(url string) {
	// server ko thoda time do listen karne ka, warna browser
	// "connection refused" dikha dega aur banda soch lega app kharab hai.
	time.Sleep(700 * time.Millisecond)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("[likho] browser khud nahi khula, manually kholo: %s", url)
	}
}

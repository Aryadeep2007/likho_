package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Poora blog ek file me. Ye ".likho" file bas ek zip hai (naam cool lagta hai).
// Pendrive me daalo, dost ko bhejo, doosre laptop pe import karo - blog waisa ka waisa.
func handleExport(w http.ResponseWriter, r *http.Request) {
	name := fmt.Sprintf("mera-blog-%s.likho", time.Now().Format("2006-01-02"))

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)

	zw := zip.NewWriter(w)
	defer zw.Close()

	root := store.DataDir()

	// blog-data ka poora folder ghum ke zip me daal do
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// .tmp files bach jati hain kabhi kabhi, unhe mat le jao
		if strings.HasSuffix(path, ".tmp") {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// zip me hamesha forward slash chalta hai, Windows pe bhi
		rel = filepath.ToSlash(rel)

		f, err := zw.Create(rel)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()

		_, err = io.Copy(f, src)
		return err
	})

	if err != nil {
		// header ja chuka hai toh ab kuch keh nahi sakte, bas log kar do
		fmt.Printf("  ⚠️  export beech me atak gaya: %v\n", err)
	}
}

// .likho file wapas import karo. Purane posts overwrite ho jayenge
// agar same naam ka post hai - warning UI me de rakhi hai.
func handleImport(w http.ResponseWriter, r *http.Request) {
	// 64 MB kaafi hai; text blog itna bada hoga hi nahi
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		http.Error(w, "file bahut badi hai ya form kharab hai", http.StatusBadRequest)
		return
	}

	file, hdr, err := r.FormFile("bundle")
	if err != nil {
		http.Error(w, "koi file hi nahi mili", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// zip.NewReader ko size chahiye, isliye pehle disk pe rakh lete hain.
	// Memory me poora padhne se bhi kaam chal jata par bade file pe RAM kha jata.
	tmp, err := os.CreateTemp("", "likho-import-*.zip")
	if err != nil {
		http.Error(w, "temp file nahi bani", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	size, err := io.Copy(tmp, file)
	if err != nil {
		http.Error(w, "file save nahi hui", http.StatusInternalServerError)
		return
	}

	zr, err := zip.NewReader(tmp, size)
	if err != nil {
		http.Error(w, "ye valid .likho file nahi lag rahi (zip nahi khuli)", http.StatusBadRequest)
		return
	}

	root := store.DataDir()
	count := 0

	for _, zf := range zr.File {
		if zf.FileInfo().IsDir() {
			continue
		}

		// ZIP SLIP se bachao. Koi "../../Windows/System32/..." naam ki entry
		// daal de toh wo file blog folder ke bahar likhi ja sakti hai.
		// Isliye har path ko check karte hain ki wo root ke andar hi hai.
		dest := filepath.Join(root, filepath.FromSlash(zf.Name))
		if !strings.HasPrefix(filepath.Clean(dest)+string(os.PathSeparator),
			filepath.Clean(root)+string(os.PathSeparator)) {
			fmt.Printf("  ⚠️  shaqi entry skip ki: %s\n", zf.Name)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			continue
		}

		rc, err := zf.Open()
		if err != nil {
			continue
		}
		out, err := os.Create(dest)
		if err != nil {
			rc.Close()
			continue
		}
		// limit laga ke copy karo - "zip bomb" 10 KB ki file se 10 GB nikal sakta hai
		if _, err := io.Copy(out, io.LimitReader(rc, 32<<20)); err == nil {
			count++
		}
		out.Close()
		rc.Close()
	}

	// store ko dobara load karo taki naye posts turant dikhein
	if ns, err := OpenStore(root); err == nil {
		store = ns
	}

	fmt.Printf("  📥  import ho gaya: %s se %d file\n", hdr.Filename, count)
	http.Redirect(w, r, "/?imported="+fmt.Sprint(count), http.StatusSeeOther)
}

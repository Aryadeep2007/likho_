#!/usr/bin/env bash
# ---------------------------------------------------------------
# Likho ka build script.
#
# Linux pe chalao, Windows ke liye .exe nikal jayega. Go install
# karne ki zarurat nahi - Docker sab kaam kar deta hai.
#
#   ./build.sh          -> sab kuch banao + Windows zip package karo
#   ./build.sh windows  -> sirf .exe
# ---------------------------------------------------------------
set -euo pipefail

cd "$(dirname "$0")"
ROOT="$(pwd)"
OUT="$ROOT/dist"
TARGET="${1:-all}"

mkdir -p "$OUT"

# Go local me hai toh wahi use karo (fast). Nahi hai toh Docker.
if command -v go >/dev/null 2>&1; then
  echo "==> local Go mil gaya: $(go version)"
  gorun() { env "$@"; }
else
  echo "==> local Go nahi hai, Docker se build kar rahe hain"
  mkdir -p "$HOME/.cache/likho-go"
  gorun() {
    docker run --rm \
      -u "$(id -u):$(id -g)" \
      -e HOME=/tmp -e GOFLAGS=-mod=mod \
      -e GOMODCACHE=/gocache/mod -e GOCACHE=/gocache/build \
      -v "$ROOT":/src -v "$HOME/.cache/likho-go":/gocache \
      -w /src golang:1.25-bookworm \
      env "$@"
  }
fi

# ldflags: -s -w debug symbols hata deta hai, binary ~30% chhota ho jata hai
LDFLAGS='-s -w'

build_one() {
  local goos="$1" goarch="$2" out="$3"
  echo "  -> $goos/$goarch  ($out)"
  gorun GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
    go build -trimpath -ldflags="$LDFLAGS" -o "dist/$out" .
}

echo "==> compile kar rahe hain..."
case "$TARGET" in
  windows) build_one windows amd64 likho.exe ;;
  linux)   build_one linux   amd64 likho-linux ;;
  mac)     build_one darwin  arm64 likho-mac ;;
  all)
    build_one windows amd64 likho.exe
    build_one linux   amd64 likho-linux
    build_one darwin  arm64 likho-mac
    ;;
  *) echo "pata nahi '$TARGET' kya hai. windows / linux / mac / all likho."; exit 1 ;;
esac

# ---------------------------------------------------------------
# Windows wala zip package
# ---------------------------------------------------------------
if [[ "$TARGET" == "all" || "$TARGET" == "windows" ]]; then
  echo "==> Windows package bana rahe hain..."

  PKG="$OUT/Likho"
  rm -rf "$PKG"
  mkdir -p "$PKG"

  cp "$OUT/likho.exe" "$PKG/"
  cp "$ROOT/run.bat"  "$PKG/"
  cp "$ROOT/PADHO-PEHLE.txt" "$PKG/" 2>/dev/null || true

  # zip banane ke liye python use kar rahe hain kyunki har machine pe
  # `zip` command install nahi hoti, par python aksar hota hai.
  # Saath me .txt files ko CRLF me convert karte hain - warna Windows
  # Notepad me sab ek hi lambi line me dikhta hai.
  python3 - "$PKG" "$OUT/Likho-Windows.zip" <<'PY'
import os, sys, zipfile

src, dest = sys.argv[1], sys.argv[2]

with zipfile.ZipFile(dest, "w", zipfile.ZIP_DEFLATED) as z:
    for root, _, files in os.walk(src):
        for f in sorted(files):
            full = os.path.join(root, f)
            arc = os.path.join("Likho", os.path.relpath(full, src))

            if f.endswith(".txt") or f.endswith(".bat"):
                # Windows ko CRLF chahiye
                data = open(full, "rb").read().replace(b"\r\n", b"\n").replace(b"\n", b"\r\n")
                z.writestr(arc, data)
            else:
                z.write(full, arc)

print("   zip ban gaya:", dest, "-", round(os.path.getsize(dest) / 1e6, 1), "MB")
PY
fi

echo
echo "==> ho gaya. dist/ me ye hai:"
ls -lh "$OUT" | tail -n +2 | awk '{printf "   %-24s %s\n", $9, $5}'

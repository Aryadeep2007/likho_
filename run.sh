#!/usr/bin/env bash
# Linux / Mac wala run script. Windows wale run.bat use karein.
set -euo pipefail

cd "$(dirname "$0")"

PORT="${1:-4000}"

echo
echo "  ==========================================="
echo "     Likho - apna blog, apne computer pe"
echo "  ==========================================="
echo

# kaunsa binary chalayein - OS ke hisaab se
BIN=""
if [[ -x "./likho-linux" ]]; then
  BIN="./likho-linux"
elif [[ -x "./likho-mac" ]]; then
  BIN="./likho-mac"
elif [[ -x "./likho" ]]; then
  BIN="./likho"
fi

if [[ -z "$BIN" ]]; then
  echo "  [X] Koi binary nahi mila (likho-linux / likho-mac)."
  echo
  echo "  Source se banana ho toh (Go chahiye hoga):"
  echo "      go build -o likho-linux ."
  echo
  echo "  Ya Docker se:"
  echo "      docker compose up"
  echo
  exit 1
fi

# exec isliye taki Ctrl+C seedha server tak pahunche,
# warna bash beech me aa jata hai aur do baar dabana padta hai
exec "$BIN" -port "$PORT"

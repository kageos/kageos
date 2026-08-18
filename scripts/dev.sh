#!/usr/bin/env bash
set -Eeuo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND_PID=""
FRONTEND_PID=""

cleanup() {
  trap - EXIT INT TERM
  [[ -z "$FRONTEND_PID" ]] || kill "$FRONTEND_PID" 2>/dev/null || true
  [[ -z "$BACKEND_PID" ]] || kill "$BACKEND_PID" 2>/dev/null || true
  [[ -z "$FRONTEND_PID" ]] || wait "$FRONTEND_PID" 2>/dev/null || true
  [[ -z "$BACKEND_PID" ]] || wait "$BACKEND_PID" 2>/dev/null || true
}

trap cleanup EXIT INT TERM

command -v go >/dev/null 2>&1 || { printf 'ERROR: Go 1.25 or newer is required.\n' >&2; exit 1; }
command -v node >/dev/null 2>&1 || { printf 'ERROR: Node.js 20.19+ or 22.12+ is required.\n' >&2; exit 1; }
command -v npm >/dev/null 2>&1 || { printf 'ERROR: npm is required.\n' >&2; exit 1; }

cd "$REPO_DIR"

printf '==> Preparing the web frontend\n'
(
  cd "$REPO_DIR/web"
  npm install
)

printf '==> Starting kageos backend and local infrastructure\n'
go run ./cmd/kagectl bootstrap --dev "$@" &
BACKEND_PID=$!

printf '==> Starting the web frontend\n'
(
  cd "$REPO_DIR/web"
  exec npm run dev
) &
FRONTEND_PID=$!

cat <<'EOF'

kageos development startup is running.
Open: http://localhost:5173
User: system
The generated password is printed in the backend initialization summary.

Press Ctrl-C to stop the frontend and backend processes.
Local infrastructure and data are preserved. To stop infrastructure later:
  go run ./cmd/kagectl down
EOF

while kill -0 "$BACKEND_PID" 2>/dev/null && kill -0 "$FRONTEND_PID" 2>/dev/null; do
  sleep 1
done

if ! kill -0 "$BACKEND_PID" 2>/dev/null; then
  wait "$BACKEND_PID"
else
  wait "$FRONTEND_PID"
fi

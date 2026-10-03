#!/bin/bash
cd "$(dirname "$0")"
chmod +x "$0" 2>/dev/null
xattr -d com.apple.quarantine "$0" 2>/dev/null

DIR="$(pwd)"

command -v go >/dev/null 2>&1 || { echo "[ERROR] go not found. Install Go first: https://go.dev/dl/"; read -r -p "Press Enter to close..."; exit 1; }
command -v bun >/dev/null 2>&1 || { echo "[ERROR] bun not found. Install Bun first: https://bun.sh/"; read -r -p "Press Enter to close..."; exit 1; }

if [ ! -d "web/node_modules" ]; then
  echo "[INFO] web/node_modules not found. Running bun install (may take a few minutes)..."
  (cd web && bun install) || { echo "[ERROR] bun install failed."; read -r -p "Press Enter to close..."; exit 1; }
fi

mkdir -p ".local/project-workbench-debug"

pick_port() {
  local p=$1
  local end=$(( $1 + 200 ))
  while [ "$p" -lt "$end" ]; do
    if ! nc -z 127.0.0.1 "$p" >/dev/null 2>&1; then echo "$p"; return 0; fi
    p=$(( p + 1 ))
  done
  return 1
}

BACKEND_PORT="$(pick_port 8080)" || { echo "[ERROR] no free backend port found near 8080."; read -r -p "Press Enter to close..."; exit 1; }
FRONTEND_PORT="$(pick_port 3000)" || { echo "[ERROR] no free frontend port found near 3000."; read -r -p "Press Enter to close..."; exit 1; }

echo "[INFO] backend  port: $BACKEND_PORT"
echo "[INFO] frontend port: $FRONTEND_PORT"

osascript -e "tell application \"Terminal\" to do script \"cd '$DIR/backend' && CANVAS_BACKEND_DATA_DIR=../.local/project-workbench-debug CANVAS_BACKEND_ADDR=127.0.0.1:$BACKEND_PORT go run ./cmd/server\"" >/dev/null
osascript -e "tell application \"Terminal\" to do script \"cd '$DIR/web' && VITE_API_PROXY_TARGET=http://127.0.0.1:$BACKEND_PORT bun x vite --host 127.0.0.1 --port $FRONTEND_PORT --strictPort\"" >/dev/null

echo "Waiting for services to become ready..."
for _ in $(seq 1 180); do
  if nc -z 127.0.0.1 "$BACKEND_PORT" >/dev/null 2>&1 && nc -z 127.0.0.1 "$FRONTEND_PORT" >/dev/null 2>&1; then
    open "http://127.0.0.1:$FRONTEND_PORT"
    echo "Ready: http://127.0.0.1:$FRONTEND_PORT"
    read -r -p "Press Enter to close..."
    exit 0
  fi
  sleep 1
done

echo "[WARN] Services did not become ready in 3 minutes. Check the Terminal windows."
read -r -p "Press Enter to close..."
exit 1

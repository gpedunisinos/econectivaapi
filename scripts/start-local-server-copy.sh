#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
docker start rieonlife-server-copy-db >/dev/null
ssh -N -L 127.0.0.1:18000:127.0.0.1:8000 \
  -o BatchMode=yes -o ExitOnForwardFailure=yes -o ServerAliveInterval=30 \
  ubuntu@54.94.142.29 &
tunnel_pid=$!
trap 'kill "$tunnel_pid" 2>/dev/null || true' EXIT
# Wait for SSH's local listener before starting the API.
for attempt in {1..30}; do
  kill -0 "$tunnel_pid" 2>/dev/null || { wait "$tunnel_pid"; exit 1; }
  if (echo >/dev/tcp/127.0.0.1/18000) 2>/dev/null; then break; fi
  sleep 1
done
for attempt in {1..60}; do
  if docker exec rieonlife-server-copy-db mysqladmin ping --silent >/dev/null 2>&1; then break; fi
  sleep 1
done
export PRESENCE_ALLOWED_ORIGINS='http://127.0.0.1:8772,http://localhost:8772'
go run ./tools/local-preview

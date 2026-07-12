#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
ROOT_DIR="$(cd "$BACKEND_DIR/../.." && pwd)"
BUILD_DIR="$(mktemp -d)"
trap 'rm -rf "$BUILD_DIR"' EXIT

# E2E_DB_* has precedence over repository .env files loaded by Make. This
# prevents a local black-box run from accidentally targeting a development DB.
export DB_HOST="${E2E_DB_HOST:-${DB_HOST:-localhost}}"
export DB_PORT="${E2E_DB_PORT:-${DB_PORT:-5432}}"
export DB_NAME="${E2E_DB_NAME:-${DB_NAME:-testdb}}"
export DB_USER="${E2E_DB_USER:-${DB_USER:-testuser}}"
export DB_PASSWORD="${E2E_DB_PASSWORD:-${DB_PASSWORD:-testpass}}"

cd "$ROOT_DIR"
pnpm -F bot-engine run build

cd "$BACKEND_DIR"
go build -o "$BUILD_DIR/purrchat-backend" ./cmd/server
go build -o "$BUILD_DIR/purrchat-migrate" ./cmd/migrate

# 本地 testdb 可能被 `go test` 的 SetupTestDB 直接建表（有 schema 但无 migrate history）。
# migrate up 遇到此情况会要求 baseline；自动 baseline 后重试，使脚本在本地与 CI 均自洽。
if ! out="$("$BUILD_DIR/purrchat-migrate" up 2>&1)"; then
	printf '%s\n' "$out" >&2
	if printf '%s\n' "$out" | grep -q "migrate baseline"; then
		echo "→ 检测到已有 schema 但无 migration history，自动 baseline 后重试 up..." >&2
		"$BUILD_DIR/purrchat-migrate" baseline
		"$BUILD_DIR/purrchat-migrate" up
	else
		exit 1
	fi
else
	printf '%s\n' "$out"
fi

E2E_BACKEND_BIN="$BUILD_DIR/purrchat-backend" \
E2E_BOT_ENGINE_ENTRY="$ROOT_DIR/apps/bot-engine/dist/index.js" \
go test -count=1 -tags=e2e -v ./e2e

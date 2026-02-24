#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN_DIR="${ROOT_DIR}/bin"
CACHE_DIR="${ROOT_DIR}/.cache"
ENTRY="${ENTRY:-./cmd}"
APP_NAME="${APP_NAME:-groupmasterbot}"
TARGET="${1:-local}"
GOCACHE="${GOCACHE:-${CACHE_DIR}/go-build}"

mkdir -p "${BIN_DIR}"
mkdir -p "${GOCACHE}"

export GOCACHE

require_go() {
  if ! command -v go >/dev/null 2>&1; then
    echo "go command not found, please install Go first." >&2
    exit 1
  fi

  local required current
  required="$(awk '/^go / {print $2; exit}' "${ROOT_DIR}/go.mod")"
  current="$(go env GOVERSION 2>/dev/null | sed 's/^go//')"

  if [[ -z "${required}" || -z "${current}" ]]; then
    return
  fi

  if ! version_ge "${current}" "${required}"; then
    echo "Go ${required}+ is required, current is ${current}." >&2
    exit 1
  fi
}

version_ge() {
  local current="$1"
  local required="$2"

  awk -v c="${current}" -v r="${required}" '
    BEGIN {
      cn = split(c, ca, ".")
      rn = split(r, ra, ".")
      max = (cn > rn ? cn : rn)
      for (i = 1; i <= max; i++) {
        cv = (i <= cn ? ca[i] : 0) + 0
        rv = (i <= rn ? ra[i] : 0) + 0
        if (cv > rv) { exit 0 }
        if (cv < rv) { exit 1 }
      }
      exit 0
    }
  '
}

usage() {
  cat <<'EOF'
Usage:
  ./scripts/build.sh [target]

Targets:
  local          Build for current machine (default)
  linux-amd64    Build for Linux amd64
  linux-arm64    Build for Linux arm64
  darwin-amd64   Build for macOS amd64
  darwin-arm64   Build for macOS arm64
  windows-amd64  Build for Windows amd64
  all            Build all targets above

Optional env:
  APP_NAME       Output binary base name (default: go-tg-supervisor)
  ENTRY          Go entry package (default: ./cmd)
  GOCACHE        Go build cache (default: ./.cache/go-build)
  GOMODCACHE     Go module cache (default: Go toolchain default)
  CGO_ENABLED    Go cgo flag (default follows Go toolchain behavior)
EOF
}

build_one() {
  local goos="$1"
  local goarch="$2"
  local ext="$3"
  local output="${BIN_DIR}/${APP_NAME}-${goos}-${goarch}${ext}"

  echo "==> Building ${goos}/${goarch} -> ${output}"
  GOOS="${goos}" GOARCH="${goarch}" go build -trimpath -o "${output}" "${ENTRY}"
}

build_local() {
  local output="${BIN_DIR}/${APP_NAME}"
  echo "==> Building local -> ${output}"
  go build -trimpath -o "${output}" "${ENTRY}"
}

case "${TARGET}" in
  local)
    require_go
    build_local
    ;;
  linux-amd64)
    require_go
    build_one linux amd64 ""
    ;;
  linux-arm64)
    require_go
    build_one linux arm64 ""
    ;;
  darwin-amd64)
    require_go
    build_one darwin amd64 ""
    ;;
  darwin-arm64)
    require_go
    build_one darwin arm64 ""
    ;;
  windows-amd64)
    require_go
    build_one windows amd64 ".exe"
    ;;
  all)
    require_go
    build_one linux amd64 ""
    build_one linux arm64 ""
    build_one darwin amd64 ""
    build_one darwin arm64 ""
    build_one windows amd64 ".exe"
    ;;
  -h|--help|help)
    usage
    ;;
  *)
    echo "Unknown target: ${TARGET}" >&2
    usage
    exit 1
    ;;
esac

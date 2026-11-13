#!/usr/bin/env bash
# Install the pinned Quint CLI and the pinned Apalache distribution into the
# ignored .engmod tree. Nothing is downloaded at run time afterwards:
#   quint:    npm ci --ignore-scripts from scripts/quint-tool/package-lock.json
#   apalache: scripts/quint-toolchain.env URL, sha256-checked, unpacked as
#             $QUINT_HOME/apalache-dist-<version>/apalache (the layout Quint
#             looks for before it would download anything).
# After installing, point engquint at them:
#   ENGMOD_QUINT=.engmod/tooling/quint/node_modules/.bin/quint
#   QUINT_HOME=.engmod/tooling/quint-home
# Java 21+ must be on PATH for `verify` mode (not needed for `run` mode).
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
# shellcheck source=quint-toolchain.env
source scripts/quint-toolchain.env

tool_root="$repo_root/.engmod/tooling"
download_dir="$tool_root/downloads"
quint_dir="$tool_root/quint"
quint_home="$tool_root/quint-home"
mkdir -p "$download_dir" "$quint_dir" "$quint_home"

verify_sha256() {
  local checksum="$1" path="$2"
  if command -v sha256sum >/dev/null 2>&1; then
    printf '%s  %s\n' "$checksum" "$path" | sha256sum -c -
  elif command -v shasum >/dev/null 2>&1; then
    test "$(shasum -a 256 "$path" | awk '{print $1}')" = "$checksum"
  else
    echo "sha256sum or shasum is required" >&2
    return 1
  fi
}

install_quint() {
  command -v npm >/dev/null || { echo "npm is required" >&2; return 1; }
  cp scripts/quint-tool/package.json scripts/quint-tool/package-lock.json "$quint_dir/"
  (cd "$quint_dir" && npm ci --ignore-scripts --no-audit --no-fund >/dev/null)
  test "$("$quint_dir/node_modules/.bin/quint" --version)" = "$QUINT_VERSION"
}

install_apalache() {
  local archive="$download_dir/apalache-${APALACHE_VERSION}.tgz"
  local url="https://github.com/apalache-mc/apalache/releases/download/v${APALACHE_VERSION}/apalache.tgz"
  local dist="$quint_home/apalache-dist-${APALACHE_VERSION}"
  # An existing dist counts only when the marker written at unpacking names
  # the pinned archive hash; a dist of another build is replaced.
  if [ -x "$dist/apalache/bin/apalache-mc" ] && [ "$(cat "$dist/.sha256" 2>/dev/null || true)" = "$APALACHE_SHA256" ]; then
    return
  fi
  if ! { [ -f "$archive" ] && verify_sha256 "$APALACHE_SHA256" "$archive" >/dev/null 2>&1; }; then
    rm -f "$archive"
    curl --fail --location --proto '=https' --proto-redir '=https' --tlsv1.2 --retry 3 --output "$archive" "$url"
    verify_sha256 "$APALACHE_SHA256" "$archive"
  fi
  rm -rf "$dist"
  mkdir -p "$dist"
  tar -xzf "$archive" -C "$dist"
  test -x "$dist/apalache/bin/apalache-mc"
  printf '%s\n' "$APALACHE_SHA256" > "$dist/.sha256"
}

case "${1:-all}" in
  all)
    install_quint
    install_apalache
    ;;
  quint)
    install_quint
    ;;
  apalache)
    install_apalache
    ;;
  *)
    echo "usage: $0 [all|quint|apalache]" >&2
    exit 2
    ;;
esac

printf 'installed pinned Quint toolchain in %s\n' "$tool_root"
printf '  ENGMOD_QUINT=%s\n  QUINT_HOME=%s\n' "$quint_dir/node_modules/.bin/quint" "$quint_home"

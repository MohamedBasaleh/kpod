#!/usr/bin/env sh
# Install kpod from GitHub Releases into ~/.local/bin.
# Does not modify kubeconfig, kpod config, or shell rc files.
set -eu

REPO="${KPOD_REPO:-MohamedBasaleh/kpod}"
DEST_DIR="${KPOD_INSTALL_DIR:-${HOME}/.local/bin}"
TMPDIR="${TMPDIR:-/tmp}"

usage() {
  cat <<EOF
kpod installer

Installs the kpod binary only. Configuration happens when you run: kpod

  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/install.sh | sh

Installs the latest *stable* GitHub Release. Prereleases are not selected
automatically.

  KPOD_VERSION=v2.0.0-rc.2 sh

Environment:
  KPOD_REPO          GitHub owner/name (default: ${REPO})
  KPOD_INSTALL_DIR   install directory (default: ~/.local/bin)
  KPOD_VERSION       exact tag (example: v2.0.0 or v2.0.0-rc.2)

This script never uses sudo. It never edits ~/.zshrc or kubeconfig.
EOF
}

kpod_tag_from_latest_json() {
  sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1
}

kpod_validate_release_tag() {
  tag=$1
  case "$tag" in
    v[0-9]*.[0-9]*.[0-9]*) ;;
    *) return 1 ;;
  esac
  case "$tag" in
    *[!A-Za-z0-9._-]*) return 1 ;;
  esac
  case "$tag" in
    releases|latest|download|tag|tags) return 1 ;;
  esac
  return 0
}

kpod_resolved_kpod() {
  command -v kpod 2>/dev/null || true
}

kpod_same_file() {
  a=$1
  b=$2
  [ -e "$a" ] && [ -e "$b" ] || return 1
  if command -v python3 >/dev/null 2>&1; then
    python3 -c 'import os,sys; sys.exit(0 if os.path.samefile(sys.argv[1], sys.argv[2]) else 1)' "$a" "$b"
    return $?
  fi
  [ "$a" = "$b" ]
}

kpod_warn_path() {
  installed=$1
  resolved=$(kpod_resolved_kpod)
  path_hint="  export PATH=\"${DEST_DIR}:\$PATH\""

  if [ -z "$resolved" ]; then
    echo
    echo "kpod is not on PATH yet. Add this to your shell rc, then open a new terminal:"
    echo "$path_hint"
    echo "Or run: ${installed} --version"
    return 0
  fi

  if kpod_same_file "$resolved" "$installed"; then
    return 0
  fi

  echo
  echo "warning: another kpod is earlier on PATH. The installer did not change it."
  echo "  installed: ${installed}"
  echo "  currently runs: ${resolved}"
  echo "Put the installer directory first:"
  echo "$path_hint"
  echo "Then open a new terminal, or run: ${installed} --version"
}

if [ "${KPOD_INSTALL_TEST:-}" = 1 ]; then
  return 0 2>/dev/null || exit 0
fi

case "${1:-}" in
  -h|--help) usage; exit 0 ;;
esac

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: $1 is required" >&2
    exit 1
  fi
}

need curl
need tar
need uname

os=$(uname -s | tr '[:upper:]' '[:lower:]')
arch=$(uname -m)
case "$os" in
  darwin|linux) ;;
  *) echo "error: unsupported OS: $os" >&2; exit 1 ;;
esac
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo "error: unsupported architecture: $arch" >&2; exit 1 ;;
esac

work=$(mktemp -d "${TMPDIR}/kpod-install.XXXXXX")
trap 'rm -rf "$work"' EXIT

if [ -n "${KPOD_VERSION:-}" ]; then
  tag=$KPOD_VERSION
  case "$tag" in
    v*) ;;
    *) tag="v${tag}" ;;
  esac
else
  code=$(curl -sS -o "${work}/latest.json" -w '%{http_code}' \
    -H "Accept: application/vnd.github+json" \
    -H "User-Agent: kpod-installer" \
    "https://api.github.com/repos/${REPO}/releases/latest" || true)
  if [ "$code" = 404 ]; then
    echo "error: no stable GitHub Release found for ${REPO}." >&2
    echo "This installer installs the latest stable release only (not RCs)." >&2
    echo "Install a prerelease with:" >&2
    echo "  curl -fsSL https://raw.githubusercontent.com/${REPO}/main/scripts/install.sh | KPOD_VERSION=v2.0.0-rc.3 sh" >&2
    exit 1
  fi
  if [ "$code" != 200 ]; then
    echo "error: GitHub latest-release lookup failed (HTTP ${code:-0})." >&2
    exit 1
  fi
  tag=$(kpod_tag_from_latest_json <"${work}/latest.json")
fi

if ! kpod_validate_release_tag "$tag"; then
  echo "error: refusing to treat '${tag:-}' as a release tag." >&2
  echo "Set KPOD_VERSION to an exact tag such as v2.0.0." >&2
  exit 1
fi

ver=${tag#v}
asset="kpod_${ver}_${os}_${arch}.tar.gz"
base="https://github.com/${REPO}/releases/download/${tag}"

echo "Downloading ${asset}"
if ! curl -fsSL "${base}/${asset}" -o "${work}/${asset}"; then
  echo "error: download failed. Is ${tag} published for ${os}/${arch}?" >&2
  exit 1
fi
if ! curl -fsSL "${base}/checksums.txt" -o "${work}/checksums.txt"; then
  echo "error: checksums.txt download failed" >&2
  exit 1
fi

want=$(awk -v f="$asset" '$NF==f || $NF=="*"f {print $1; exit}' "${work}/checksums.txt")
if [ -z "$want" ]; then
  echo "error: ${asset} is not listed in checksums.txt" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  got=$(sha256sum "${work}/${asset}" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  got=$(shasum -a 256 "${work}/${asset}" | awk '{print $1}')
else
  echo "error: sha256sum or shasum is required" >&2
  exit 1
fi
if [ "$got" != "$want" ]; then
  echo "error: checksum mismatch" >&2
  exit 1
fi

tar -xzf "${work}/${asset}" -C "$work"
if [ ! -f "${work}/kpod" ]; then
  echo "error: archive did not contain kpod" >&2
  exit 1
fi

dest="${DEST_DIR}/kpod"
if [ -e "$dest" ] && [ ! -f "$dest" ]; then
  echo "error: ${dest} exists and is not a regular file; refusing to overwrite" >&2
  exit 1
fi

mkdir -p "$DEST_DIR"
tmp="${DEST_DIR}/.kpod.new.$$"
cp "${work}/kpod" "$tmp"
chmod 0755 "$tmp"
mv "$tmp" "$dest"

echo "Installed ${dest}  ($("${dest}" --version 2>/dev/null || echo kpod))"
echo "Config is not modified."
kpod_warn_path "$dest"

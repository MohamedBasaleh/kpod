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

Environment:
  KPOD_REPO          GitHub owner/name (default: ${REPO})
  KPOD_INSTALL_DIR   install directory (default: ~/.local/bin)
  KPOD_VERSION       tag without downloading latest (example: v2.0.0)

This script never uses sudo. It never edits ~/.zshrc.
EOF
}

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

if [ -n "${KPOD_VERSION:-}" ]; then
  tag="$KPOD_VERSION"
else
  latest=$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/${REPO}/releases/latest")
  tag=${latest##*/}
fi
ver=${tag#v}
asset="kpod_${ver}_${os}_${arch}.tar.gz"
base="https://github.com/${REPO}/releases/download/${tag}"

work=$(mktemp -d "${TMPDIR}/kpod-install.XXXXXX")
trap 'rm -rf "$work"' EXIT

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

mkdir -p "$DEST_DIR"
tmp="${DEST_DIR}/.kpod.new.$$"
cp "${work}/kpod" "$tmp"
chmod 0755 "$tmp"
mv "$tmp" "${DEST_DIR}/kpod"

echo "Installed ${DEST_DIR}/kpod  ($("${DEST_DIR}/kpod" --version 2>/dev/null || echo kpod))"
echo "Config is not modified."
case ":$PATH:" in
  *":${DEST_DIR}:"*) ;;
  *)
    echo
    echo "Add this to your shell rc if kpod is not found:"
    echo "  export PATH=\"${DEST_DIR}:\$PATH\""
    echo "Then open a new terminal and run: kpod"
    ;;
esac

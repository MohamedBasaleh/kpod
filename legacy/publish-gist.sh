#!/usr/bin/env bash
# Create or update the public GitHub Gist for kpod.
# Requires GitHub CLI:  gh auth login -h github.com -p https -w -s gist

set -euo pipefail

SRC_DIR="$(cd "$(dirname "$0")" && pwd)"
GIST_USER="${KPOD_GIST_USER:-MohamedBasaleh}"
DESC="kpod — Kubernetes Pod Manager (install + kubectl CLI for Rails)"
ID_FILE="${SRC_DIR}/.gist_id"

need_gh() {
  if ! command -v gh >/dev/null 2>&1; then
    echo "error: install GitHub CLI (gh)" >&2
    exit 1
  fi
  if ! gh auth status -h github.com >/dev/null 2>&1; then
    echo "Not logged in to GitHub CLI. Run this once:"
    echo "  gh auth login -h github.com -p https -w -s gist"
    exit 1
  fi
}

write_id() {
  local id="$1"
  printf '%s\n' "$id" > "$ID_FILE"
  perl -i -pe "s/__KPOD_GIST_ID__/${id}/g" \
    "${SRC_DIR}/kpod" \
    "${SRC_DIR}/install.sh" \
    "${SRC_DIR}/README.md"
}

push_files() {
  local id="$1"
  gh gist edit "$id" --filename kpod "${SRC_DIR}/kpod"
  gh gist edit "$id" --filename install.sh "${SRC_DIR}/install.sh"
  gh gist edit "$id" --filename README.md "${SRC_DIR}/README.md"
}

need_gh
bash -n "${SRC_DIR}/kpod"
bash -n "${SRC_DIR}/install.sh"
chmod +x "${SRC_DIR}/kpod" "${SRC_DIR}/install.sh"

id=""
if [[ -n "${KPOD_GIST_ID:-}" && "${KPOD_GIST_ID}" != "__KPOD_GIST_ID__" ]]; then
  id="$KPOD_GIST_ID"
elif [[ -f "$ID_FILE" ]]; then
  id="$(tr -d '[:space:]' < "$ID_FILE")"
fi

if [[ -z "$id" ]]; then
  echo "Creating public gist..."
  url="$(
    gh gist create --public --desc "$DESC" \
      "${SRC_DIR}/README.md" \
      "${SRC_DIR}/kpod" \
      "${SRC_DIR}/install.sh"
  )"
  echo "$url"
  id="${url##*/}"
  write_id "$id"
  echo "Embedding gist id and pushing again..."
  push_files "$id"
else
  write_id "$id"
  echo "Updating gist ${id}..."
  push_files "$id"
fi

echo
echo "Gist:    https://gist.github.com/${GIST_USER}/${id}"
echo "Install: curl -fsSL https://gist.githubusercontent.com/${GIST_USER}/${id}/raw/install.sh | bash"
echo "Update:  kpod --update"

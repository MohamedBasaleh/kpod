# Changelog

## Unreleased

- MIT License (Copyright (c) 2026 Mohamed Basaleh)
- Curl installer is the primary supported install method for v2.0.0; Homebrew is postponed
- Resolve the latest **stable** GitHub Release via the GitHub API (do not parse `/releases/latest` redirects as a tag)
- Fail clearly when no stable release exists; keep `KPOD_VERSION` for explicit tags including RCs
- Warn when another `kpod` on `PATH` shadows `~/.local/bin/kpod`
- GitHub Releases: `MohamedBasaleh/kpod`
- Homebrew formula upload remains skipped (`skip_upload: true`)
- Do not reuse tags `v2.0.0-rc.1`, `v2.0.0-rc.2`, or `v2.0.0-rc.3`

## 2.0.0-rc.2

Go toolchain 1.26 to match `golang.org/x/term`.

## 2.0.0-rc.1

Go rewrite of kpod.

- Interactive TUI dashboard and pod browser
- Environment model: alias → Kubernetes context + namespace
- Automatic context/namespace discovery (informational until you add)
- Pod actions: console, dbconsole, runner, shell, logs, describe, YAML, events, metrics, rollout, restart, delete, port-forward, upload/download
- Optional shell shortcuts (`k-dc`, …) without unsolicited rc edits
- Migration from Bash v1 `~/.config/kpod/contexts`
- **Behavior change:** kpod never changes kubectl’s global `current-context`

Release-blocking until GitHub Releases exist:

- public GitHub repository
- license selection
- Homebrew tap `MohamedBasaleh/homebrew-kpod`

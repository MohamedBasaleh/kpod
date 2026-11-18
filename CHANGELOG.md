# Changelog

## Unreleased

Installer:

- Resolve the latest **stable** GitHub Release via the GitHub API (do not parse `/releases/latest` redirects as a tag)
- Fail clearly when no stable release exists; keep `KPOD_VERSION` for explicit tags including RCs
- Warn when another `kpod` on `PATH` shadows `~/.local/bin/kpod`

Release configuration (not tagged):

- GitHub Releases and installer: `MohamedBasaleh/kpod` (`https://github.com/MohamedBasaleh/kpod/releases`)
- Homebrew formula publish is postponed (`skip_upload: true`); curl install does not depend on the tap
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

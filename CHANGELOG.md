# Changelog

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

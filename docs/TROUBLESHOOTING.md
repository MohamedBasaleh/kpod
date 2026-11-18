# Troubleshooting

## kubectl not found

Install kubectl and ensure it is on `PATH`. kpod never talks to the Kubernetes API itself.

## Expired EKS / cloud credentials

kpod uses the selected `--context`. Refresh credentials with your cloud tool, then retry. kpod does not store tokens.

## VPN / cluster unreachable

The dashboard still opens from local config. Pod lists and actions fail until the cluster is reachable. Cached kube results may appear briefly.

## Forbidden namespace listing

Your account cannot `get namespaces`. Configured environments still work. Discovery will not treat this as “zero namespaces.”

## Metrics Server unavailable

The metrics action needs Metrics Server in the cluster. Other actions are unaffected.

## Multiple pod matches

Pass a tighter pod filter (`kpod dev logs api`). Interactive mode lets you pick.

## Shell executable unavailable

kpod tries `bash`, then `sh`. If the container has neither, use **Shell** after checking the image.

## Wrong `kpod` after the curl installer

The installer writes `~/.local/bin/kpod` and does not change other binaries. If `kpod --version` is still old, another file is earlier on `PATH` (often `~/bin/kpod`).

```bash
command -v kpod
export PATH="$HOME/.local/bin:$PATH"
```

The installer prints this warning when it detects the mismatch. It will not delete the older file.

## Homebrew

Homebrew is not a supported install method for v2.0.0. Use the curl installer. If `kpod` happens to live under a Homebrew Cellar path, `kpod --update` will refuse to replace that file.

## TUI will not start

`kpod` with no arguments needs a real terminal. Direct actions (`kpod dev logs api`) work without a TUI.

## `NO_COLOR`

Disables the welcome animation and uses a static header.

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

## Homebrew-installed kpod

`kpod --update` will not replace Homebrew files. Use:

```bash
brew upgrade kpod
```

## TUI will not start

`kpod` with no arguments needs a real terminal. Direct actions (`kpod dev logs api`) work without a TUI.

## `NO_COLOR`

Disables the welcome animation and uses a static header.

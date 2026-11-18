# kpod

Fast, interactive Kubernetes access for developers.

kpod maps short **environment** names to a Kubernetes context and namespace, then gets you to pods and common actions without changing kubectl’s global current-context.

## Installation

Needs `kubectl` on your `PATH`. The installed binary does not require Go.

### Homebrew

```bash
brew install MohamedBasaleh/kpod/kpod
```

The formula lives in the tap `MohamedBasaleh/homebrew-kpod` and installs binaries from [GitHub Releases](https://github.com/MohamedBasaleh/kpod/releases).

### Direct installer

Installs into `~/.local/bin` without sudo. It does not edit kubeconfig, kpod config, or shell rc files.

The default command installs the latest **stable** GitHub Release (not RCs):

```bash
curl -fsSL https://raw.githubusercontent.com/MohamedBasaleh/kpod/main/scripts/install.sh | sh
```

Install a specific tag, including a prerelease:

```bash
curl -fsSL https://raw.githubusercontent.com/MohamedBasaleh/kpod/main/scripts/install.sh \
  | KPOD_VERSION=v2.0.0-rc.2 sh
```

If `kpod` is not found, or an older `kpod` (for example `~/bin/kpod`) still runs, put the installer directory first:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

The installer does not delete or replace a shadowed binary. Use `command -v kpod` to see which file the shell runs.

### Manual download

Binaries are published at [github.com/MohamedBasaleh/kpod/releases](https://github.com/MohamedBasaleh/kpod/releases). Each release includes:

- `kpod_<version>_darwin_arm64.tar.gz`
- `kpod_<version>_darwin_amd64.tar.gz`
- `kpod_<version>_linux_amd64.tar.gz`
- `kpod_<version>_linux_arm64.tar.gz`
- `checksums.txt`

Verify the checksum, extract `kpod`, and place it on your `PATH`.

## First run

```bash
kpod
```

kpod discovers kubeconfig contexts and lets you pick namespaces, then create environment names. You do not need to edit YAML by hand.

## Environments

An **environment** is a Kubernetes **context + namespace** with a short alias.

```text
my-company-dev
├── dev       → dev-core
└── staging   → staging-core
```

After that:

```bash
kpod dev
kpod staging console
```

kpod always passes `--context` and `-n` to kubectl. It never runs `kubectl config use-context`.

## Everyday commands

```bash
kpod
kpod dev
kpod dev console
kpod staging bash
kpod dev logs api
kpod discover
kpod shortcut generate
```

## Interactive UI

From the dashboard:

| Key | Action |
| --- | --- |
| Enter | Open environment / pod |
| `/` | Search |
| `n` | Namespaces |
| `k` | Contexts |
| `u` | Kubernetes updates |
| `s` | Shortcut settings |
| `r` | Refresh |
| `?` | Help |
| Esc | Back |
| `q` | Quit |

The welcome header types once per process. Any key finishes it and still performs that key.

## Actions

Connect: **console**, **dbconsole**, **runner**, **shell**

Observe: **logs**, **describe**, **YAML**, **events**, **metrics**

Manage (mutating; restart/delete ask for confirmation): **rollout**, **restart**, **delete**

Transfer / network: **upload**, **download**, **port-forward**

## Shortcuts

Optional shell functions such as `k-dc`, `k-sc`, `k-sb`:

```bash
kpod shortcut generate
kpod shortcut enable
kpod shortcut install
```

Shell rc files are **not** changed unless you confirm install.

## Discovery

kpod can notice new kubeconfig contexts and new namespaces in contexts you already manage. It does **not** add them until you choose Add.

| Word | Meaning |
| --- | --- |
| Seen | Already observed; no “new” notice |
| Ignored | You asked not to be notified |
| Configured | Has an environment mapping |

```bash
kpod discover
```

Disable the dashboard badge with `settings.notifications: false`. `kpod discover` still works.

## Configuration

Default directory: `~/.config/kpod/`

- `config.yaml` — environments, shortcuts, settings
- `state.yaml` — last environment, discovery memory
- `shell/` — generated shortcut files

Manual editing is normally unnecessary. See [docs/SCHEMA.md](docs/SCHEMA.md).

kpod stores mappings locally. It does **not** store kube credentials, tokens, or cluster secrets. There is no telemetry. Network use is kubectl plus optional GitHub Releases for install/update.

## Update

```bash
kpod --update
```

Downloads the latest **stable** GitHub Release from [github.com/MohamedBasaleh/kpod/releases](https://github.com/MohamedBasaleh/kpod/releases), verifies checksums, and replaces the binary. Config and shortcuts are not touched.

If Homebrew installed kpod:

```bash
brew upgrade kpod
```

## Completions

```bash
kpod completion zsh
kpod completion bash
kpod completion fish
```

Homebrew installs these automatically. Completions read local config for environment names; they do not call kubectl.

## Migration from v1

If `~/.config/kpod/contexts` exists and there is no v2 `config.yaml`, `kpod` offers migration.

- Environment mappings are preserved
- A backup `contexts.v1.bak` is written
- The original file is kept
- kubeconfig is never rewritten

```bash
kpod migrate
```

v1 is legacy (`legacy/kpod-v1` in this repository). v2 does not invoke it. Existing `~/bin/kpod-v1` is not deleted by the installer.

## Troubleshooting

See [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) for kubectl, credentials, VPN, RBAC, metrics, Homebrew updates, and terminals.

## Development

Go **1.26** is the minimum supported toolchain. CI and release builds use Go 1.26. Users of the binary do not need Go.

```bash
go test ./...
go test -race ./...
go vet ./...
```

## License

No license has been selected by the owner. This repository is not currently offered under an OSI-approved license; do not assume you may redistribute it until a LICENSE file is added.

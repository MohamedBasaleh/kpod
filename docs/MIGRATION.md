# Migrating from kpod v1

v1 stored mappings in `~/.config/kpod/contexts`:

```text
alias|kube-context|namespace
```

When v2 finds that file and no `config.yaml`, running `kpod` offers migration. You can also run:

```bash
kpod migrate
```

What happens:

1. Rows become v2 environments (same alias, context, namespace).
2. `contexts.v1.bak` is written once.
3. The original `contexts` file is left in place.
4. kubeconfig is not modified.
5. Shortcuts and notification settings start at defaults (v1 had no v2 shortcuts file).

v2 does not call Bash v1. Keep `~/bin/kpod-v1` if you want a rollback binary; the installer will not delete it.

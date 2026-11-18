# Configuration and state schema

kpod stores local YAML only. It does not store kube credentials, tokens, or cluster secrets.

| File | Version | Purpose |
| --- | --- | --- |
| `config.yaml` | **2** | User configuration |
| `state.yaml` | **1** | Runtime / discovery memory |

Unknown optional fields are ignored by the YAML decoder. Missing optional fields are normalized to empty/default values on load.

## When to bump a version

Bump `config.yaml` / `state.yaml` **only** when a change cannot be expressed as an additive, optional field.

| Change | Version bump? |
| --- | --- |
| New optional field (`ignored_namespaces`, `discovery_ready`) | No |
| New map/list that defaults to empty | No |
| Rename or remove a required field | Yes |
| Change the meaning of an existing field | Yes |
| Reject previously valid documents | Yes |

Future incompatible changes must ship a migrator that writes a new version and leaves a backup.

## config.yaml (version 2)

- `contexts.<name>.environments.<alias>.namespace`
- `contexts.<name>.ignored_namespaces` (optional)
- `ignored_contexts`
- `shortcuts`
- `settings.notifications`
- `settings.shell_shortcuts`

## state.yaml (version 1)

- `last_environment`
- `seen_contexts`
- `seen_namespaces`
- `discovery_ready` (optional; absent means “inventory never completed”)

`discovery_ready` is additive. Older files without it remain valid and are treated as uninitialized discovery, not as “everything is new.”

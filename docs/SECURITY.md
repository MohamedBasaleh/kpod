# Security notes

- kubectl is invoked with argument arrays (`exec.Command`), never `bash -c`.
- kpod never runs `kubectl config use-context`.
- Config and state files store names only, not credentials.
- Shortcut IDs allow `[A-Za-z0-9_-]` only.
- Environment aliases reject spaces and shell metacharacters.
- Generated shortcut functions quote environment and action names.
- Installer and `kpod --update` download from GitHub Releases on `MohamedBasaleh/kpod` and verify SHA-256 checksums before replacing a binary.
- Checksum mismatch leaves the existing binary in place.
- Homebrew-managed binaries are not self-replaced.
- Destructive pod actions require confirmation.
- `PAGER` is split into argv; it is not executed through a shell.
- Debug logs print kubectl args, exit code, duration, and stderr — not kubeconfig contents.

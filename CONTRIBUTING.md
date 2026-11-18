# Contributing

kpod is a Go CLI. Kubernetes access goes through `kubectl` argv arrays, never a shell string.

```bash
gofmt -w .
go test ./...
go test -race ./...
go vet ./...
go build -o /tmp/kpod ./cmd/kpod
```

Unit tests must not need a cluster. Use fake kubectl runners.

Do not commit kubeconfig files, tokens, or production data.

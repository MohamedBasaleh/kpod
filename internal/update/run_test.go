package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func tarGz(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{Name: name, Mode: 0755, Size: int64(len(body))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(body); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestParseChecksums(t *testing.T) {
	text := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa  kpod_2.0.0_darwin_arm64.tar.gz\n"
	got, err := ParseChecksums(text, "kpod_2.0.0_darwin_arm64.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if got != strings.Repeat("a", 64) {
		t.Fatalf("%s", got)
	}
	if _, err := ParseChecksums(text, "missing.tar.gz"); err == nil {
		t.Fatal("expected missing")
	}
}

func TestVerifyMismatch(t *testing.T) {
	if err := Verify([]byte("x"), strings.Repeat("a", 64)); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestInstalledByHomebrew(t *testing.T) {
	if !InstalledByHomebrew("/opt/homebrew/Cellar/kpod/2.0.0/bin/kpod") {
		t.Fatal("cellar")
	}
	if InstalledByHomebrew("/Users/me/.local/bin/kpod") {
		t.Fatal("user bin")
	}
}

func TestRunUpdate(t *testing.T) {
	archive := tarGz(t, "kpod", []byte("new-binary"))
	sum := sha256.Sum256(archive)
	hexSum := hex.EncodeToString(sum[:])
	asset := ArchiveName("2.0.0", runtime.GOOS, runtime.GOARCH)
	checksums := hexSum + "  " + asset + "\n"

	mux := http.NewServeMux()
	mux.HandleFunc("/repos/MohamedBasaleh/kpod/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v2.0.0","prerelease":false,"assets":[{"name":%q,"browser_download_url":"http://%s/a.tar.gz"},{"name":"checksums.txt","browser_download_url":"http://%s/checksums.txt"}]}`, asset, r.Host, r.Host)
	})
	mux.HandleFunc("/a.tar.gz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) })
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(checksums)) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	bin := filepath.Join(dir, "kpod")
	if err := os.WriteFile(bin, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfg, []byte("keep-me"), 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	res, err := Run(Options{
		Stdout:   &out,
		ExecPath: bin,
		Current:  "2.0.0-dev",
		Client:   Client{HTTP: srv.Client(), BaseURL: srv.URL, Repo: "MohamedBasaleh/kpod"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated {
		t.Fatalf("%+v %s", res, out.String())
	}
	got, _ := os.ReadFile(bin)
	if string(got) != "new-binary" {
		t.Fatalf("binary %q", got)
	}
	cfgGot, _ := os.ReadFile(cfg)
	if string(cfgGot) != "keep-me" {
		t.Fatal("config mutated")
	}
}

func TestRunAlreadyLatest(t *testing.T) {
	mux := http.NewServeMux()
	asset := ArchiveName("2.0.0", runtime.GOOS, runtime.GOARCH)
	mux.HandleFunc("/repos/x/y/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v2.0.0","prerelease":false,"assets":[{"name":%q,"browser_download_url":"http://example/a"},{"name":"checksums.txt","browser_download_url":"http://example/c"}]}`, asset)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	var out bytes.Buffer
	res, err := Run(Options{
		Stdout:   &out,
		ExecPath: filepath.Join(t.TempDir(), "kpod"),
		Current:  "2.0.0",
		Client:   Client{HTTP: srv.Client(), BaseURL: srv.URL, Repo: "x/y"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated || !strings.Contains(out.String(), "already the latest") {
		t.Fatalf("%+v %s", res, out.String())
	}
}

func TestRunPrereleaseLatestRejected(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/x/y/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"tag_name":"v2.0.0-rc.1","prerelease":true,"assets":[]}`)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	_, err := Run(Options{
		Stdout:   ioDiscard{},
		ExecPath: filepath.Join(t.TempDir(), "kpod"),
		Current:  "2.0.0-dev",
		Client:   Client{HTTP: srv.Client(), BaseURL: srv.URL, Repo: "x/y"},
	})
	if err == nil || !strings.Contains(err.Error(), "prerelease") {
		t.Fatalf("got %v", err)
	}
}

func TestRunChecksumMismatch(t *testing.T) {
	archive := tarGz(t, "kpod", []byte("new-binary"))
	asset := ArchiveName("2.0.0", runtime.GOOS, runtime.GOARCH)
	checksums := strings.Repeat("b", 64) + "  " + asset + "\n"
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/x/y/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v2.0.0","prerelease":false,"assets":[{"name":%q,"browser_download_url":"http://%s/a.tar.gz"},{"name":"checksums.txt","browser_download_url":"http://%s/checksums.txt"}]}`, asset, r.Host, r.Host)
	})
	mux.HandleFunc("/a.tar.gz", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(archive) })
	mux.HandleFunc("/checksums.txt", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(checksums)) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	bin := filepath.Join(t.TempDir(), "kpod")
	_ = os.WriteFile(bin, []byte("old"), 0o755)
	_, err := Run(Options{
		Stdout:   ioDiscard{},
		ExecPath: bin,
		Current:  "2.0.0-dev",
		Client:   Client{HTTP: srv.Client(), BaseURL: srv.URL, Repo: "x/y"},
	})
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("got %v", err)
	}
	got, _ := os.ReadFile(bin)
	if string(got) != "old" {
		t.Fatal("binary replaced after checksum failure")
	}
}

func TestRunHomebrew(t *testing.T) {
	var out bytes.Buffer
	res, err := Run(Options{
		Stdout:   &out,
		ExecPath: "/opt/homebrew/Cellar/kpod/2.0.0/bin/kpod",
		Current:  "2.0.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "homebrew" || !strings.Contains(out.String(), "brew upgrade kpod") {
		t.Fatalf("%+v %s", res, out.String())
	}
}

func TestRunDownloadFailure(t *testing.T) {
	asset := ArchiveName("2.0.0", runtime.GOOS, runtime.GOARCH)
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/x/y/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":"v2.0.0","prerelease":false,"assets":[{"name":%q,"browser_download_url":"http://%s/missing"},{"name":"checksums.txt","browser_download_url":"http://%s/c"}]}`, asset, r.Host, r.Host)
	})
	mux.HandleFunc("/c", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	_, err := Run(Options{
		Stdout:   ioDiscard{},
		ExecPath: filepath.Join(t.TempDir(), "kpod"),
		Current:  "2.0.0-dev",
		Client:   Client{HTTP: srv.Client(), BaseURL: srv.URL, Repo: "x/y"},
	})
	if err == nil {
		t.Fatal("expected download error")
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }

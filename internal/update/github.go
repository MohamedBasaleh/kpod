package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/MohamedBasaleh/kpod/internal/version"
)

const defaultTimeout = 20 * time.Second

type Release struct {
	Tag      string
	Assets   []Asset
	Pre      bool
	Checksum string
	Archive  string
	URL      string
}

type Asset struct {
	Name string
	URL  string
}

type githubRelease struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

type Client struct {
	HTTP    *http.Client
	BaseURL string
	Repo    string
}

func NewClient() Client {
	return Client{
		HTTP:    &http.Client{Timeout: defaultTimeout},
		BaseURL: "https://api.github.com",
		Repo:    version.Repo,
	}
}

func ArchiveName(ver, goos, goarch string) string {
	ver = strings.TrimPrefix(ver, "v")
	return fmt.Sprintf("kpod_%s_%s_%s.tar.gz", ver, goos, goarch)
}

func CurrentArchiveName() string {
	return ArchiveName(version.Display(), runtime.GOOS, runtime.GOARCH)
}

func (c Client) LatestStable() (Release, error) {
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: defaultTimeout}
	}
	if c.BaseURL == "" {
		c.BaseURL = "https://api.github.com"
	}
	if c.Repo == "" {
		c.Repo = version.Repo
	}
	url := strings.TrimRight(c.BaseURL, "/") + "/repos/" + c.Repo + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", version.UserAgent())
	res, err := c.HTTP.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("check GitHub releases: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return Release{}, err
	}
	if res.StatusCode == http.StatusNotFound {
		return Release{}, fmt.Errorf("no GitHub releases found for %s", c.Repo)
	}
	if res.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("GitHub releases: HTTP %d", res.StatusCode)
	}
	var raw githubRelease
	if err := json.Unmarshal(body, &raw); err != nil {
		return Release{}, fmt.Errorf("parse GitHub release: %w", err)
	}
	if raw.Prerelease {
		return Release{}, fmt.Errorf("latest GitHub release is marked prerelease; refusing to install")
	}
	rel := Release{Tag: raw.TagName, Pre: raw.Prerelease}
	want := ArchiveName(raw.TagName, runtime.GOOS, runtime.GOARCH)
	for _, a := range raw.Assets {
		rel.Assets = append(rel.Assets, Asset{Name: a.Name, URL: a.URL})
		switch a.Name {
		case want:
			rel.Archive = a.Name
			rel.URL = a.URL
		case "checksums.txt":
			rel.Checksum = a.URL
		}
	}
	if rel.URL == "" {
		return rel, fmt.Errorf("release %s has no asset %s", raw.TagName, want)
	}
	if rel.Checksum == "" {
		return rel, fmt.Errorf("release %s is missing checksums.txt", raw.TagName)
	}
	return rel, nil
}

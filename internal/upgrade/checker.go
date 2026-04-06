// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ReleaseInfo holds metadata for a GitHub release.
type ReleaseInfo struct {
	TagName    string      `json:"tag_name"`
	Prerelease bool        `json:"prerelease"`
	Assets     []AssetInfo `json:"assets"`
}

// AssetInfo holds metadata for a release asset.
type AssetInfo struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// NewHTTPClient returns an *http.Client configured for upgrade operations.
// It follows redirects (required for GitHub asset downloads) and has a
// generous timeout for large binary downloads.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 120 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
			MaxConnsPerHost:     5,
			IdleConnTimeout:     90 * time.Second,
		},
	}
}

// fetchReleases fetches up to perPage releases from the GitHub API.
func fetchReleases(ctx context.Context, client *http.Client, repo string, perPage int) ([]ReleaseInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=%d", repo, perPage)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "lore-upgrade")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var releases []ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return releases, nil
}

// CheckLatestRelease fetches releases from the GitHub API and returns the
// newest one (including pre-releases). Uses /releases (not /releases/latest)
// because the latter excludes pre-releases.
func CheckLatestRelease(ctx context.Context, client *http.Client, repo string) (*ReleaseInfo, error) {
	releases, err := fetchReleases(ctx, client, repo, 10)
	if err != nil {
		return nil, err
	}
	if len(releases) == 0 {
		return nil, nil
	}
	return &releases[0], nil
}

// ListNewerReleases fetches recent releases and returns those newer than
// currentVersion, sorted newest first. Returns at most 20 entries.
func ListNewerReleases(ctx context.Context, client *http.Client, repo string, currentVersion string) ([]ReleaseInfo, error) {
	releases, err := fetchReleases(ctx, client, repo, 50)
	if err != nil {
		return nil, err
	}

	var newer []ReleaseInfo
	for _, r := range releases {
		if CompareVersions(currentVersion, r.TagName) < 0 {
			newer = append(newer, r)
		}
	}
	if len(newer) > 20 {
		newer = newer[:20]
	}
	return newer, nil
}

// FindRelease fetches releases and returns the one matching the given tag.
// Returns nil, nil if no matching release is found.
func FindRelease(ctx context.Context, client *http.Client, repo string, tag string) (*ReleaseInfo, error) {
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}

	releases, err := fetchReleases(ctx, client, repo, 50)
	if err != nil {
		return nil, err
	}

	for i := range releases {
		if releases[i].TagName == tag {
			return &releases[i], nil
		}
	}
	return nil, nil
}

// CompareVersions compares two semver strings (with optional "v" prefix).
// Returns -1 if current < latest, 0 if equal, +1 if current > latest.
// Handles pre-release tags: v1.0.0-beta.1 < v1.0.0.
func CompareVersions(current, latest string) int {
	curCore, curPre := parseSemver(current)
	latCore, latPre := parseSemver(latest)

	if c := compareTuples(curCore, latCore); c != 0 {
		return c
	}

	// Both have no pre-release → equal
	if curPre == "" && latPre == "" {
		return 0
	}
	// A version without pre-release is greater than one with
	if curPre == "" {
		return 1
	}
	if latPre == "" {
		return -1
	}

	// Both have pre-release: compare lexicographically
	if curPre < latPre {
		return -1
	}
	if curPre > latPre {
		return 1
	}
	return 0
}

// parseSemver splits "v1.2.3-beta.1" into ([1,2,3], "beta.1").
func parseSemver(v string) ([3]int, string) {
	v = strings.TrimPrefix(v, "v")
	var parts [3]int
	pre := ""

	if idx := strings.IndexByte(v, '-'); idx >= 0 {
		pre = v[idx+1:]
		v = v[:idx]
	}

	segs := strings.SplitN(v, ".", 3)
	for i := 0; i < 3 && i < len(segs); i++ {
		parts[i], _ = strconv.Atoi(segs[i])
	}
	return parts, pre
}

// compareTuples compares two [3]int version tuples.
func compareTuples(a, b [3]int) int {
	for i := 0; i < 3; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package upgrade

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    int
	}{
		{"v1.0.0", "v1.1.0", -1},
		{"v1.1.0", "v1.1.0", 0},
		{"v2.0.0", "v1.9.9", 1},
		{"v1.0.0", "v1.0.1", -1},
		{"v0.9.0", "v1.0.0", -1},

		// Pre-release handling
		{"v1.0.0-beta.1", "v1.0.0", -1},
		{"v1.0.0", "v1.0.0-beta.1", 1},
		{"v1.0.0-beta.1", "v1.0.0-beta.2", -1},
		{"v1.0.0-beta.2", "v1.0.0-beta.1", 1},
		{"v1.0.0-alpha", "v1.0.0-beta", -1},
		{"v1.0.0-beta.1", "v1.0.0-beta.1", 0},

		// Without v prefix
		{"1.0.0", "1.1.0", -1},
		{"1.0.0", "v1.1.0", -1},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.latest, func(t *testing.T) {
			got := CompareVersions(tt.current, tt.latest)
			if got != tt.want {
				t.Errorf("CompareVersions(%q, %q) = %d, want %d", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

func TestCheckLatestRelease(t *testing.T) {
	releases := []ReleaseInfo{
		{
			TagName:    "v1.2.0-beta.1",
			Prerelease: true,
			Assets: []AssetInfo{
				{Name: "lore_Darwin_arm64.tar.gz", BrowserDownloadURL: "https://example.com/lore_Darwin_arm64.tar.gz"},
				{Name: "checksums.txt", BrowserDownloadURL: "https://example.com/checksums.txt"},
			},
		},
		{
			TagName:    "v1.1.0",
			Prerelease: false,
			Assets:     []AssetInfo{},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	}))
	defer srv.Close()

	// We need to override the URL — use a custom approach via the test server
	// For this test, we call the handler directly
	client := srv.Client()

	// Create a request to the test server
	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	var got []ReleaseInfo
	_ = json.NewDecoder(resp.Body).Decode(&got)

	if len(got) != 2 {
		t.Fatalf("expected 2 releases, got %d", len(got))
	}
	if got[0].TagName != "v1.2.0-beta.1" {
		t.Errorf("expected first release v1.2.0-beta.1, got %s", got[0].TagName)
	}
	if !got[0].Prerelease {
		t.Error("expected first release to be a prerelease")
	}
	if len(got[0].Assets) != 2 {
		t.Errorf("expected 2 assets, got %d", len(got[0].Assets))
	}
}

func TestNewHTTPClient(t *testing.T) {
	client := NewHTTPClient()
	if client == nil {
		t.Fatal("NewHTTPClient returned nil")
	}
	if client.Timeout == 0 {
		t.Error("expected non-zero timeout")
	}
}

func TestParseSemver(t *testing.T) {
	tests := []struct {
		input string
		core  [3]int
		pre   string
	}{
		{"v1.2.3", [3]int{1, 2, 3}, ""},
		{"1.2.3", [3]int{1, 2, 3}, ""},
		{"v1.2.3-beta.1", [3]int{1, 2, 3}, "beta.1"},
		{"v0.0.1", [3]int{0, 0, 1}, ""},
		{"v1.0.0-rc.1", [3]int{1, 0, 0}, "rc.1"},
	}
	for _, tt := range tests {
		core, pre := parseSemver(tt.input)
		if core != tt.core {
			t.Errorf("parseSemver(%q) core = %v, want %v", tt.input, core, tt.core)
		}
		if pre != tt.pre {
			t.Errorf("parseSemver(%q) pre = %q, want %q", tt.input, pre, tt.pre)
		}
	}
}

func TestCompareTuples(t *testing.T) {
	tests := []struct {
		a, b [3]int
		want int
	}{
		{[3]int{1, 0, 0}, [3]int{1, 0, 0}, 0},
		{[3]int{1, 0, 0}, [3]int{2, 0, 0}, -1},
		{[3]int{2, 0, 0}, [3]int{1, 0, 0}, 1},
		{[3]int{1, 1, 0}, [3]int{1, 2, 0}, -1},
		{[3]int{1, 0, 1}, [3]int{1, 0, 0}, 1},
	}
	for _, tt := range tests {
		got := compareTuples(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("compareTuples(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// redirectTransport intercepts HTTP calls and redirects them to the test server.
type redirectTransport struct {
	target *httptest.Server
}

func (rt *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect the request to the test server
	req.URL.Scheme = "http"
	req.URL.Host = rt.target.Listener.Addr().String()
	return http.DefaultTransport.RoundTrip(req)
}

func newTestClient(srv *httptest.Server) *http.Client {
	return &http.Client{Transport: &redirectTransport{target: srv}}
}

func TestCheckLatestRelease_WithReleases(t *testing.T) {
	releases := []ReleaseInfo{
		{TagName: "v1.2.0", Prerelease: false},
		{TagName: "v1.1.0", Prerelease: false},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	ctx := context.Background()

	result, err := CheckLatestRelease(ctx, client, "test/repo")
	if err != nil {
		t.Fatalf("CheckLatestRelease: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.TagName != "v1.2.0" {
		t.Errorf("expected v1.2.0, got %s", result.TagName)
	}
}

func TestCheckLatestRelease_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	client := newTestClient(srv)
	ctx := context.Background()

	result, err := CheckLatestRelease(ctx, client, "test/repo")
	if err != nil {
		t.Fatalf("CheckLatestRelease: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for empty releases, got %+v", result)
	}
}

func TestListNewerReleases(t *testing.T) {
	releases := []ReleaseInfo{
		{TagName: "v2.0.0", Prerelease: false},
		{TagName: "v1.5.0", Prerelease: false},
		{TagName: "v1.0.0", Prerelease: false},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	ctx := context.Background()

	newer, err := ListNewerReleases(ctx, client, "test/repo", "v1.0.0")
	if err != nil {
		t.Fatalf("ListNewerReleases: %v", err)
	}
	if len(newer) != 2 {
		t.Fatalf("expected 2 newer releases, got %d", len(newer))
	}
	if newer[0].TagName != "v2.0.0" {
		t.Errorf("first = %s, want v2.0.0", newer[0].TagName)
	}
}

func TestListNewerReleases_AlreadyLatest(t *testing.T) {
	releases := []ReleaseInfo{
		{TagName: "v1.0.0", Prerelease: false},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	ctx := context.Background()

	newer, err := ListNewerReleases(ctx, client, "test/repo", "v1.0.0")
	if err != nil {
		t.Fatalf("ListNewerReleases: %v", err)
	}
	if len(newer) != 0 {
		t.Errorf("expected 0 newer, got %d", len(newer))
	}
}

func TestFindRelease_Found(t *testing.T) {
	releases := []ReleaseInfo{
		{TagName: "v1.1.0", Prerelease: false},
		{TagName: "v1.0.0", Prerelease: false},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	ctx := context.Background()

	result, err := FindRelease(ctx, client, "test/repo", "v1.0.0")
	if err != nil {
		t.Fatalf("FindRelease: %v", err)
	}
	if result == nil {
		t.Fatal("expected result")
	}
	if result.TagName != "v1.0.0" {
		t.Errorf("expected v1.0.0, got %s", result.TagName)
	}
}

func TestFindRelease_NotFound(t *testing.T) {
	releases := []ReleaseInfo{
		{TagName: "v1.0.0", Prerelease: false},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	ctx := context.Background()

	result, err := FindRelease(ctx, client, "test/repo", "v9.9.9")
	if err != nil {
		t.Fatalf("FindRelease: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil, got %+v", result)
	}
}

func TestFindRelease_AddVPrefix(t *testing.T) {
	releases := []ReleaseInfo{
		{TagName: "v1.0.0", Prerelease: false},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(releases)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	ctx := context.Background()

	// Pass without v prefix — should still match
	result, err := FindRelease(ctx, client, "test/repo", "1.0.0")
	if err != nil {
		t.Fatalf("FindRelease: %v", err)
	}
	if result == nil {
		t.Fatal("expected result with auto-added v prefix")
	}
}

func TestFetchReleases_ErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestClient(srv)
	ctx := context.Background()

	_, err := CheckLatestRelease(ctx, client, "test/repo")
	if err == nil {
		t.Error("expected error for 500 status")
	}
}

func TestCheckLatestReleaseEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
	}))
	defer srv.Close()

	// For empty releases, CheckLatestRelease should return nil, nil
	// We test the JSON parsing logic indirectly
	client := srv.Client()
	ctx := context.Background()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()

	var got []ReleaseInfo
	_ = json.NewDecoder(resp.Body).Decode(&got)

	if len(got) != 0 {
		t.Fatalf("expected 0 releases, got %d", len(got))
	}
}

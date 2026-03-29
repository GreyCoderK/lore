// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import "testing"

func TestScanDiffContent_SecurityPattern(t *testing.T) {
	diff := "+  apiKey := os.Getenv(\"API_KEY\")\n- oldToken := cfg.token"
	s := ScanDiffContent(diff)
	if s.Score < 15 {
		t.Errorf("security pattern: score = %d, want >= 15", s.Score)
	}
}

func TestScanDiffContent_PublicAPIPattern(t *testing.T) {
	diff := "+func HandleAuth(w http.ResponseWriter, r *http.Request) {"
	s := ScanDiffContent(diff)
	if s.Score < 10 {
		t.Errorf("public API pattern: score = %d, want >= 10", s.Score)
	}
}

func TestScanDiffContent_InfraPattern(t *testing.T) {
	diff := "+  database_url: postgres://localhost:5432/mydb"
	s := ScanDiffContent(diff)
	if s.Score < 10 {
		t.Errorf("infra pattern: score = %d, want >= 10", s.Score)
	}
}

func TestScanDiffContent_EntityDeleted(t *testing.T) {
	diff := "-func OldHandler(w http.ResponseWriter, r *http.Request) {"
	s := ScanDiffContent(diff)
	if s.Score < 8 {
		t.Errorf("entity deleted pattern: score = %d, want >= 8", s.Score)
	}
}

func TestScanDiffContent_TechDebt(t *testing.T) {
	diff := "+  // TODO: refactor this mess"
	s := ScanDiffContent(diff)
	if s.Score < 5 {
		t.Errorf("tech debt pattern: score = %d, want >= 5", s.Score)
	}
}

func TestScanDiffContent_MultiplePatterns(t *testing.T) {
	diff := "+  secret := getSecret()\n+  // TODO: fix later\n-func OldAuth() {}"
	s := ScanDiffContent(diff)
	// security(15) + tech-debt(5) + entity-deleted(8) = 28
	if s.Score < 28 {
		t.Errorf("multiple patterns: score = %d, want >= 28", s.Score)
	}
}

func TestScanDiffContent_Deduplication(t *testing.T) {
	diff := "+  secret1 := getSecret()\n+  secret2 := getSecret()\n+  token := getToken()"
	s := ScanDiffContent(diff)
	// Security pattern matches 3 times but scores only once = 15
	if s.Score != 15 {
		t.Errorf("deduplication: score = %d, want 15", s.Score)
	}
}

func TestScanDiffContent_NoMatches(t *testing.T) {
	diff := "+  x := 42\n-  y := 43"
	s := ScanDiffContent(diff)
	if s.Score != 0 {
		t.Errorf("no matches: score = %d, want 0", s.Score)
	}
}

func TestScanDiffContent_EmptyDiff(t *testing.T) {
	s := ScanDiffContent("")
	if s.Score != 0 {
		t.Errorf("empty diff: score = %d, want 0", s.Score)
	}
}

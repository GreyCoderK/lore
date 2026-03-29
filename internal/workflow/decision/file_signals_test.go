// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import "testing"

func TestFileValueSignal_SingleHighValue(t *testing.T) {
	s := FileValueSignal([]string{"src/api/handler.go"})
	if s.Score != 5 {
		t.Errorf("single high-value: score = %d, want 5", s.Score)
	}
}

func TestFileValueSignal_ThreeHighValue(t *testing.T) {
	s := FileValueSignal([]string{"api/v1/handler.go", "schema/user.sql", "migration/001.sql"})
	if s.Score != 15 {
		t.Errorf("three high-value: score = %d, want 15", s.Score)
	}
}

func TestFileValueSignal_CappedAt15(t *testing.T) {
	s := FileValueSignal([]string{"api/a.go", "schema/b.sql", "migration/c.sql", "security/d.go"})
	if s.Score != 15 {
		t.Errorf("four high-value: score = %d, want 15 (capped)", s.Score)
	}
}

func TestFileValueSignal_AllTests(t *testing.T) {
	s := FileValueSignal([]string{"handler_test.go", "test/fixture.go", "mock/provider.go"})
	if s.Score != -10 {
		t.Errorf("all tests: score = %d, want -10", s.Score)
	}
}

func TestFileValueSignal_MixTestsAndNon(t *testing.T) {
	s := FileValueSignal([]string{"handler_test.go", "handler.go"})
	// Not 100% tests → no penalty, handler.go is not high-value → 0
	if s.Score != 0 {
		t.Errorf("mix: score = %d, want 0", s.Score)
	}
}

func TestFileValueSignal_Empty(t *testing.T) {
	s := FileValueSignal(nil)
	if s.Score != 0 {
		t.Errorf("empty: score = %d, want 0", s.Score)
	}
}

func TestFileValueSignal_ProtoFile(t *testing.T) {
	s := FileValueSignal([]string{"service.proto", "main.go"})
	if s.Score != 5 {
		t.Errorf("proto: score = %d, want 5", s.Score)
	}
}

// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package decision

import "testing"

func TestExtractImplicitWhy_EN_Separator(t *testing.T) {
	what, why, conf := ExtractImplicitWhy("feat(auth): add OAuth2 flow for third-party providers")
	if what != "add OAuth2 flow" {
		t.Errorf("what = %q, want 'add OAuth2 flow'", what)
	}
	if why != "third-party providers" {
		t.Errorf("why = %q, want 'third-party providers'", why)
	}
	if conf != 0.7 {
		t.Errorf("confidence = %f, want 0.7", conf)
	}
}

func TestExtractImplicitWhy_FR_Separator(t *testing.T) {
	what, why, conf := ExtractImplicitWhy("feat(api): ajouter le flux pour éviter les doublons")
	if what != "ajouter le flux" {
		t.Errorf("what = %q, want 'ajouter le flux'", what)
	}
	if why != "éviter les doublons" {
		t.Errorf("why = %q, want 'éviter les doublons'", why)
	}
	if conf != 0.7 {
		t.Errorf("confidence = %f, want 0.7", conf)
	}
}

func TestExtractImplicitWhy_NoSeparator(t *testing.T) {
	what, why, conf := ExtractImplicitWhy("fix: typo in readme")
	if what != "typo in readme" {
		t.Errorf("what = %q, want 'typo in readme'", what)
	}
	if why != "" {
		t.Errorf("why = %q, want empty", why)
	}
	if conf != 0 {
		t.Errorf("confidence = %f, want 0", conf)
	}
}

func TestExtractImplicitWhy_ShortWhy(t *testing.T) {
	what, why, conf := ExtractImplicitWhy("feat: add X for Y")
	if conf != 0.4 {
		t.Errorf("confidence = %f, want 0.4 (short why %q)", conf, why)
	}
	_ = what
}

func TestExtractImplicitWhy_BecauseSeparator(t *testing.T) {
	what, why, _ := ExtractImplicitWhy("refactor(db): simplify query because the old one was too complex")
	if what != "simplify query" {
		t.Errorf("what = %q", what)
	}
	if why != "the old one was too complex" {
		t.Errorf("why = %q", why)
	}
}

func TestExtractImplicitWhy_EmptySubject(t *testing.T) {
	what, why, conf := ExtractImplicitWhy("")
	if what != "" || why != "" || conf != 0 {
		t.Errorf("empty subject: what=%q, why=%q, conf=%f", what, why, conf)
	}
}

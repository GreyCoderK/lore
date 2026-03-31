// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"os"
	"testing"

	"github.com/greycoderk/lore/internal/i18n"
)

// TestMain ensures i18n is initialized to EN before any test in the cmd package.
// Without this, tests that assert on hardcoded English strings from i18n catalogs
// would break if another test calls i18n.Init("fr") without restoring.
func TestMain(m *testing.M) {
	i18n.Init("en")
	os.Exit(m.Run())
}

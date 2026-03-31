// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package workflow

import (
	"os"
	"testing"

	"github.com/greycoderk/lore/internal/i18n"
)

// TestMain ensures i18n is initialized to EN before any test in the workflow package.
func TestMain(m *testing.M) {
	i18n.Init("en")
	os.Exit(m.Run())
}

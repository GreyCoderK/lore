// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package status

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBadgeColor(t *testing.T) {
	assert.Equal(t, "555", BadgeColor(0))
	assert.Equal(t, "555", BadgeColor(49))
	assert.Equal(t, "5c2", BadgeColor(50))
	assert.Equal(t, "5c2", BadgeColor(79))
	assert.Equal(t, "d4a017", BadgeColor(80))
	assert.Equal(t, "d4a017", BadgeColor(100))
}

func TestFormatBadgeMarkdown(t *testing.T) {
	badge := FormatBadgeMarkdown(78, "documented")
	assert.Contains(t, badge, "78%")
	assert.Contains(t, badge, "documented")
	assert.Contains(t, badge, "shields.io")
	assert.Contains(t, badge, "5c2") // green for 78%
}

func TestFormatBadgeMarkdown_100(t *testing.T) {
	badge := FormatBadgeMarkdown(100, "documented")
	assert.Contains(t, badge, "100%")
	assert.Contains(t, badge, "d4a017") // gold
}

func TestFormatBadgeMarkdown_0(t *testing.T) {
	badge := FormatBadgeMarkdown(0, "documented")
	assert.Contains(t, badge, "0%")
	assert.Contains(t, badge, "555") // grey
}

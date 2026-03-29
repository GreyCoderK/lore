// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/greycoderk/lore/internal/domain"
)

// resolveDocTypeFlags resolves document type shorthand flags into a single type string.
// Returns the type and an error if multiple shorthands are used.
func resolveDocTypeFlags(flagType string, flagFeature, flagDecision, flagBugfix, flagRefactor, flagNote bool) (string, error) {
	docType := flagType
	shorthands := 0
	if flagFeature {
		docType = domain.DocTypeFeature
		shorthands++
	}
	if flagDecision {
		docType = domain.DocTypeDecision
		shorthands++
	}
	if flagBugfix {
		docType = domain.DocTypeBugfix
		shorthands++
	}
	if flagRefactor {
		docType = domain.DocTypeRefactor
		shorthands++
	}
	if flagNote {
		docType = domain.DocTypeNote
		shorthands++
	}
	if shorthands > 1 || (flagType != "" && shorthands > 0) {
		return "", fmt.Errorf("specify at most one document type filter")
	}
	return docType, nil
}

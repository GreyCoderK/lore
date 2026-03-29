// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

// Content signals for persona resolution. Each persona has a set of keywords
// (EN + FR cumulative) that trigger activation when found in a document body.
//
// To add a new language: append translated keywords to each slice below.
// No changes to persona.go or ResolvePersonas are needed — signals are cumulative.
//
// Structure: signalsXxx where Xxx matches the persona Name in the registry.

// signalsStoryteller activates on narrative/decision content.
var signalsStoryteller = []string{
	// EN
	"decision", "chose", "trade-off", "context",
	// FR
	"décision", "choisi", "compromis", "contexte", "pourquoi", "raison",
}

// signalsTechWriter activates on technical implementation content.
var signalsTechWriter = []string{
	// EN
	"api", "endpoint", "schema", "module", "configuration",
	// FR
	"implémentation",
}

// signalsQAReviewer activates on testing/validation content.
var signalsQAReviewer = []string{
	// EN
	"bugfix", "test", "validation",
	// FR
	"vérification", "bogue", "correctif", "régression",
}

// signalsArchitect activates on system design content.
var signalsArchitect = []string{
	// EN
	"architecture", "design", "component", "scale",
	// FR
	"conception", "composant", "scalabilité", "système", "dimensionnement",
}

// signalsUXDesigner activates on user-facing content.
var signalsUXDesigner = []string{
	// EN
	"interface",
	// FR
	"utilisateur", "accessibilité", "ergonomie", "expérience",
}

// signalsBusinessAnalyst activates on business/requirements content.
var signalsBusinessAnalyst = []string{
	// EN
	"requirement", "stakeholder", "business", "customer",
	// FR
	"exigence", "partie-prenante", "métier", "client", "besoin",
}

// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

// Package apipostman implements the first concrete ExampleSynthesizer
// an api-postman synthesizer that composes ready-to-import
// HTTP+JSON request examples from information already present in a feature
// doc's Endpoints / Filters / Security sections.
//
// Output strategy (β, per 2026-04-15 decision Q5):
//
//   - Required fields render as Postman variables: "month": "{{month}}".
//   - Optional fields render as null.
//   - Pagination fields (page, size) are treated as normal optional fields
//     unless the source doc explicitly declares defaults.
//
// Security (I5/I5-bis, per decisions Q2 and Q6):
//
//   - When the source doc has a Security section, every field listed there
//     is excluded from the output.
//   - When Security is absent, the framework's well-known list filters
//     likely server-injected names AND a "missing-security-section"
//     warning is emitted (degraded mode).
//
// This synthesizer registers itself with the framework's DefaultRegistry
// via init(), so importing this package (directly or through a side-effect
// blank import) is enough to make "api-postman" available.
package apipostman

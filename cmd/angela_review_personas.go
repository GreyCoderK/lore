// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/angela"
	"github.com/greycoderk/lore/internal/config"
	"github.com/greycoderk/lore/internal/domain"
	"github.com/greycoderk/lore/internal/i18n"
	"github.com/greycoderk/lore/internal/ui"
)

// personaResolution tags which opt-in path was selected after flag parsing.
// Invariant: personas activate only when the user explicitly consents (flag
// or interactive prompt). Config alone never activates.
type personaResolution int

const (
	// personaBaseline: no personas activated. Review runs in pre-8-19 behavior.
	// Produced by: no flag + no configured personas, --no-personas, TTY prompt "N",
	// non-TTY + configured (with info log), or unknown persona names.
	personaBaseline personaResolution = iota

	// personaFromFlag: --persona names resolved to actual personas.
	personaFromFlag

	// personaFromConfig: config personas activated via --use-configured-personas
	// or positive answer to the interactive prompt.
	personaFromConfig

	// personaPromptRequired: TTY + configured personas + no flag. The cmd layer
	// must show the confirmation prompt and re-evaluate.
	personaPromptRequired

	// personaNonTTYInfo: non-TTY + configured personas + no flag. The cmd layer
	// must emit the info log (AC-10) and fall back to baseline.
	personaNonTTYInfo
)

// reviewPersonaDecision is the pure output of the opt-in decision logic.
// The cmd layer consumes this to decide whether to call Review with personas,
// to prompt the user, or to emit the info log.
type reviewPersonaDecision struct {
	Resolution personaResolution
	// Personas is populated only when Resolution is personaFromFlag or
	// personaFromConfig — i.e. when we already know the final list.
	// For personaPromptRequired, the cmd layer resolves Candidates after prompt.
	Personas []angela.PersonaProfile
	// Candidates lists the configured persona names that would be used if the
	// user says "yes" at the prompt (personaPromptRequired) or that were
	// discovered in non-TTY mode (personaNonTTYInfo).
	Candidates []string
}

// errPersonaFlagConflict is returned when two mutually exclusive persona
// flags are set at the same time.
type errPersonaFlagConflict struct {
	flags []string
}

func (e errPersonaFlagConflict) Error() string {
	return fmt.Sprintf(i18n.T().Cmd.AngelaReviewErrMutuallyExclusive, strings.Join(e.flags, ", "))
}

// decideReviewPersonas is a pure function that applies the 8-19 opt-in
// decision matrix given flag state, config, and TTY context. It does NOT
// perform I/O — the caller handles the prompt (for personaPromptRequired)
// and the info log (for personaNonTTYInfo).
//
// Decision matrix (story 8-19 AC-7..AC-12):
//
//	| config personas | --persona | --no-personas | --use-configured | TTY | Result                |
//	| any             | set       | —             | —                | any | personaFromFlag       |
//	| any             | —         | set           | —                | any | personaBaseline       |
//	| configured      | —         | —             | set              | any | personaFromConfig     |
//	| none            | —         | —             | —                | any | personaBaseline       |
//	| configured      | —         | —             | —                | yes | personaPromptRequired |
//	| configured      | —         | —             | —                | no  | personaNonTTYInfo     |
//	| any             | set       | set           | —                | any | ERROR                 |
//	| any             | set       | —             | set              | any | ERROR                 |
//	| any             | —         | set           | set              | any | ERROR                 |
func decideReviewPersonas(
	cfg *config.Config,
	flagPersonaNames []string,
	flagNoPersonas bool,
	flagUseConfigured bool,
	isTTY bool,
) (reviewPersonaDecision, error) {
	// 1. Mutual exclusion (AC-11, AC-12).
	var set []string
	if len(flagPersonaNames) > 0 {
		set = append(set, "--persona")
	}
	if flagNoPersonas {
		set = append(set, "--no-personas")
	}
	if flagUseConfigured {
		set = append(set, "--use-configured-personas")
	}
	if len(set) > 1 {
		return reviewPersonaDecision{}, errPersonaFlagConflict{flags: set}
	}

	// 2. --no-personas hard-forces baseline.
	if flagNoPersonas {
		return reviewPersonaDecision{Resolution: personaBaseline}, nil
	}

	// 3. --persona names: resolve via registry.
	if len(flagPersonaNames) > 0 {
		personas, unknown := resolvePersonaNames(flagPersonaNames)
		if len(unknown) > 0 {
			return reviewPersonaDecision{}, fmt.Errorf(
				i18n.T().Cmd.AngelaReviewErrUnknownPersonas, strings.Join(unknown, ", "))
		}
		return reviewPersonaDecision{
			Resolution: personaFromFlag,
			Personas:   personas,
		}, nil
	}

	// 4. Config source of truth: only ManualList is consulted for persona-aware
	// review. "auto" / "all" / "none" selections do NOT opt the user in silently.
	//
	// When the user explicitly passed --use-configured-personas but the config
	// does not opt in via the manual selection + non-empty list path, fail
	// loudly so the user learns what to fix rather than silently getting a
	// baseline review.
	configured := configuredReviewPersonaNames(cfg)
	if flagUseConfigured && len(configured) == 0 {
		return reviewPersonaDecision{}, fmt.Errorf(
			"%s", i18n.T().Cmd.AngelaReviewErrUseConfiguredNoManual)
	}
	if len(configured) == 0 {
		return reviewPersonaDecision{Resolution: personaBaseline}, nil
	}

	// 5. --use-configured-personas: activate without prompt.
	if flagUseConfigured {
		personas, unknown := resolvePersonaNames(configured)
		if len(unknown) > 0 {
			// Config has unknown persona names — fail loud so user can fix .lorerc.
			return reviewPersonaDecision{}, fmt.Errorf(
				i18n.T().Cmd.AngelaReviewErrUnknownConfiguredPersona, strings.Join(unknown, ", "))
		}
		return reviewPersonaDecision{
			Resolution: personaFromConfig,
			Personas:   personas,
			Candidates: configured,
		}, nil
	}

	// 6. TTY + configured: prompt required.
	if isTTY {
		return reviewPersonaDecision{
			Resolution: personaPromptRequired,
			Candidates: configured,
		}, nil
	}

	// 7. Non-TTY + configured: info log then baseline.
	return reviewPersonaDecision{
		Resolution: personaNonTTYInfo,
		Candidates: configured,
	}, nil
}

// configuredReviewPersonaNames returns the explicit list of persona names
// configured for review, honoring the 8-19 opt-in rule: only the "manual"
// selection mode contributes a list. Auto / all / none are treated as "no
// opt-in signal" — the user must either add --persona or switch to manual.
func configuredReviewPersonaNames(cfg *config.Config) []string {
	if cfg == nil {
		return nil
	}
	pc := cfg.Angela.Review.Personas
	if strings.ToLower(pc.Selection) != "manual" {
		return nil
	}
	out := make([]string, 0, len(pc.ManualList))
	for _, name := range pc.ManualList {
		if name = strings.TrimSpace(name); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// personaPromptInputs carries the data needed to render the 8-19 AC-9/AC-13
// confirmation prompt with a cost delta. Grouped into a struct so the
// callable helper stays testable.
type personaPromptInputs struct {
	CorpusBytes int    // sum of (summary + filename + 50) across docs
	Model       string // cfg.AI.Model
	MaxTokens   int    // ResolveMaxTokens output
	Timeout     time.Duration
	Candidates  []string // configured persona names
}

// promptPersonaConfirmation renders the cost-delta prompt (AC-9 + AC-13) and
// reads a y/N answer from the user. Returns true if the user opted in.
// Reads input byte-by-byte (matching cmd/delete.go pattern) so the helper
// works reliably in tests that inject a bytes.Buffer as streams.In.
//
// The helper handles:
//   - Rendering the configured persona list
//   - Computing baseline vs with-personas cost deltas via Preflight
//   - Collapsing deltas <5% to "~same cost" (AC-13)
//   - Falling back to "cost unknown" when EstimateCost returns -1 (AC-13)
//   - Empty or non-"y"/"yes" answer → false (baseline), per AC-9 default=No
func promptPersonaConfirmation(streams domain.IOStreams, in personaPromptInputs) (bool, error) {
	// Resolve personas up-front so the cost calculation reflects real names.
	personas, unknown := resolvePersonaNames(in.Candidates)
	if len(unknown) > 0 {
		return false, fmt.Errorf(
			i18n.T().Cmd.AngelaReviewErrUnknownConfiguredPersona, strings.Join(unknown, ", "))
	}

	// Two Preflight calls — one baseline, one with the persona prompt inflated
	// into the input size. Preflight is pure (no I/O), so this is cheap.
	baseline := angela.Preflight(
		strings.Repeat("x", in.CorpusBytes), "", in.Model, in.MaxTokens, in.Timeout,
	)
	personaPromptBytes := len(angela.BuildPersonaPrompt(personas))
	augmented := angela.Preflight(
		strings.Repeat("x", in.CorpusBytes+personaPromptBytes), "", in.Model, in.MaxTokens, in.Timeout,
	)

	ta := i18n.T().Angela

	// Header + configured list
	fmt.Fprintf(streams.Err,
		ta.UIPersonaConfiguredHeader,
		len(in.Candidates),
		pluralS(len(in.Candidates)),
		strings.Join(in.Candidates, ", "),
	)

	// Cost delta (AC-13)
	renderPersonaCostDelta(streams, baseline, augmented, len(in.Candidates))

	fmt.Fprint(streams.Err, ta.UIPersonaAddContext)
	fmt.Fprint(streams.Err, ta.UIPersonaPromptQuestion)

	// Read one line from stdin, byte-by-byte.
	answer, err := readLineFromStreams(streams.In)
	if err != nil {
		return false, err
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

// renderPersonaCostDelta prints the baseline vs with-personas cost/token
// comparison. Collapses negligible deltas and falls back gracefully when the
// model's pricing is unknown OR the baseline cost is zero (which would make
// a percentage delta meaningless).
func renderPersonaCostDelta(streams domain.IOStreams, baseline, augmented *angela.PreflightResult, personaCount int) {
	ta := i18n.T().Angela
	baseLine := fmt.Sprintf(ta.UIPersonaInputTokens, formatTokens(baseline.EstimatedInputTokens))
	augLine := fmt.Sprintf(ta.UIPersonaInputTokens, formatTokens(augmented.EstimatedInputTokens))

	// Cost rendering has three branches:
	//   1. Both costs known and baseline > 0 → render percent delta with collapse rule.
	//   2. Both costs known but baseline == 0 → percent is undefined; show absolute
	//      augmented cost but avoid asserting "~same cost" (a 0→X change is ∞%).
	//   3. Either cost unknown → omit $ amounts entirely.
	switch {
	case baseline.EstimatedCost > 0 && augmented.EstimatedCost >= 0:
		baseLine += fmt.Sprintf(ta.UIPersonaCostInline, formatUSD(baseline.EstimatedCost))
		augLine += fmt.Sprintf(ta.UIPersonaCostInline, formatUSD(augmented.EstimatedCost))
		pct := percentDelta(baseline.EstimatedCost, augmented.EstimatedCost)
		if pct < 5 {
			augLine += ta.UIPersonaCostSameRoughly
		} else {
			augLine += fmt.Sprintf(ta.UIPersonaCostDeltaPct, pct)
		}
	case baseline.EstimatedCost == 0 && augmented.EstimatedCost >= 0:
		baseLine += ta.UIPersonaCostBaselineZero
		augLine += fmt.Sprintf(ta.UIPersonaCostDeltaUndefined, formatUSD(augmented.EstimatedCost))
	default:
		augLine += ta.UIPersonaCostUnknown
	}

	fmt.Fprintf(streams.Err, ta.UIPersonaCostDeltaBaselineLabel, baseLine)
	fmt.Fprintf(streams.Err, ta.UIPersonaCostDeltaAugmentedLabel, personaCount, pluralS(personaCount), augLine)
}

// pluralS returns "s" when n != 1, "" otherwise.
func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// formatTokens renders an int token count with thousands separator.
func formatTokens(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	// Simple comma-grouping. Keeps the CLI output non-locale-dependent (spec
	// uses "1,240" in AC-13 example).
	s := fmt.Sprintf("%d", n)
	var out strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out.WriteRune(',')
		}
		out.WriteRune(r)
	}
	return out.String()
}

// formatUSD renders a dollar amount with 4 decimals (matches AC-13 sample).
func formatUSD(v float64) string {
	return fmt.Sprintf("%.4f", v)
}

// percentDelta returns the integer percentage increase from a to b (clamped 0+).
// Returns 0 when a == 0 to avoid div-by-zero (treat as negligible).
func percentDelta(a, b float64) int {
	if a == 0 {
		return 0
	}
	pct := int(((b - a) / a) * 100)
	if pct < 0 {
		return 0
	}
	return pct
}

// readLineFromStreams reads a single line (up to newline) from r byte-by-byte.
// Matches the cmd/delete.go pattern to avoid bufio prefetch in piped tests.
//
// Consent invariant: a partial buffer starting with "y" followed by a read
// error must NOT return (true, nil) to the caller — that would silently
// activate personas the user never confirmed. Any read error before a
// terminating newline discards the partial buffer (returns ""), which the
// caller interprets as "No" per the default=No contract. io.EOF is still
// honored for the "no trailing newline" case when a complete word was read,
// but unexpected errors propagate so callers can log them.
func readLineFromStreams(r io.Reader) (string, error) {
	if r == nil {
		return "", nil
	}
	var buf []byte
	b := make([]byte, 1)
	for {
		n, err := r.Read(b)
		if n > 0 {
			if b[0] == '\n' {
				return string(buf), nil
			}
			buf = append(buf, b[0])
		}
		if err != nil {
			// Normal EOF with a complete word (e.g. "y" then EOF in a test
			// buffer) is legitimate — return what we have.
			if errors.Is(err, io.EOF) {
				return string(buf), nil
			}
			// Any other error (broken pipe, closed fd, network glitch) is
			// treated as "read failed before newline" — discard the partial
			// buffer so the caller cannot opt the user in by accident. The
			// error is still returned so the caller can decide to surface it.
			return "", err
		}
	}
}

// activePersonasInReport collects the UNION of persona names referenced
// across all findings in the report and resolves each to a PersonaProfile.
// Unknown names (no match in the registry) are skipped — the formatter falls
// back to the raw identifier in the per-finding attribution line instead.
//
// Order is stable: first-seen order across the findings slice. This gives a
// deterministic "Review angle" block regardless of how the AI shuffles the
// findings list across runs with the same input.
func activePersonasInReport(report *angela.ReviewReport) []angela.PersonaProfile {
	if report == nil || len(report.Findings) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	out := make([]angela.PersonaProfile, 0, 4)
	for _, f := range report.Findings {
		for _, name := range f.Personas {
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
			if p, ok := angela.PersonaByName(name); ok {
				out = append(out, p)
			}
		}
	}
	return out
}

// formatFlaggedByLine renders the per-finding persona attribution string used
// in the text report. Each name is replaced by its Icon + DisplayName when
// resolvable; unknown names pass through verbatim so debugging an AI
// hallucination on the Personas field stays possible.
func formatFlaggedByLine(names []string) string {
	parts := make([]string, 0, len(names))
	for _, n := range names {
		if p, ok := angela.PersonaByName(n); ok {
			parts = append(parts, fmt.Sprintf("%s %s", p.Icon, p.DisplayName))
		} else {
			parts = append(parts, n)
		}
	}
	return strings.Join(parts, ", ")
}

// renderNonTTYPersonaInfo emits the AC-10 info log when personas are
// configured but not activated because the context is non-interactive.
// Separate helper so tests can exercise it without spinning up the full cmd.
func renderNonTTYPersonaInfo(streams domain.IOStreams, candidates []string) {
	fmt.Fprintf(streams.Err,
		i18n.T().Angela.UIPersonaNonTTYInfo,
		ui.Dim("ℹ"),
		strings.Join(candidates, ", "),
	)
}

// resolvePersonaNames maps persona names to PersonaProfile values using the
// registry. Returns the resolved personas and a slice of names that were not
// found. A nil input yields nil/nil.
//
// Lookup and deduplication are case-insensitive on the persona name: a user
// passing "--persona SEC --persona sec" activates the persona once; a user
// passing the uppercase form of a real registry name resolves successfully
// instead of erroring as "unknown persona".
//
// Names containing control characters (newline, carriage return, NUL, etc.
// — enforced via isSafePersonaName) are rejected as unknown. This prevents
// log-injection via error messages printed back to stderr and protects the
// prompt construction downstream.
func resolvePersonaNames(names []string) (profiles []angela.PersonaProfile, unknown []string) {
	if len(names) == 0 {
		return nil, nil
	}
	registry := angela.GetRegistry()
	byName := make(map[string]angela.PersonaProfile, len(registry))
	for _, p := range registry {
		byName[strings.ToLower(p.Name)] = p
	}
	seen := make(map[string]struct{}, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n == "" {
			continue
		}
		if !isSafePersonaName(n) {
			// Replace control chars with a printable sentinel in the returned
			// unknown value so error messages don't leak raw newlines into logs.
			unknown = append(unknown, sanitizePrintable(n))
			continue
		}
		key := strings.ToLower(n)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		if p, ok := byName[key]; ok {
			profiles = append(profiles, p)
		} else {
			unknown = append(unknown, n)
		}
	}
	return profiles, unknown
}

// isSafePersonaName returns false if s contains any control character or
// byte that would be dangerous to echo back in an error log (newline, CR,
// NUL, tab — only printable ASCII + non-control Unicode allowed).
func isSafePersonaName(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

// sanitizePrintable replaces control characters in s with '?' so the value
// can be safely printed in error messages without breaking log parsers.
func sanitizePrintable(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			b.WriteByte('?')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

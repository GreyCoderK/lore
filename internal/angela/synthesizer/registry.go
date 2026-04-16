// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package synthesizer

import (
	"sort"
	"sync"
)

// Registry is the thread-safe catalog of available synthesizers. Concrete
// implementations register themselves in their package init() via
// DefaultRegistry.Register. At command entry the Angela pipeline calls
// Enabled(cfg) to narrow the catalog to the synthesizers activated by
// AngelaConfig.Synthesizers.Enabled for the current run.
type Registry struct {
	mu      sync.RWMutex
	entries map[string]Synthesizer
}

// NewRegistry builds an empty registry. Tests use this to avoid leaking
// state between cases; production code uses DefaultRegistry.
func NewRegistry() *Registry {
	return &Registry{entries: make(map[string]Synthesizer)}
}

// DefaultRegistry is the process-wide registry populated via init() from
// the impls/<name>/ subpackages. Pipelines read from it.
var DefaultRegistry = NewRegistry()

// Register adds s under s.Name(). Duplicate names panic - a name collision
// across implementation packages is a programming error caught at startup.
func (r *Registry) Register(s Synthesizer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	name := s.Name()
	if name == "" {
		panic("synthesizer: Register called with empty Name")
	}
	if _, exists := r.entries[name]; exists {
		panic("synthesizer: duplicate Register for " + name)
	}
	r.entries[name] = s
}

// Get returns the synthesizer registered under name, or nil if absent.
func (r *Registry) Get(name string) Synthesizer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.entries[name]
}

// Names returns the list of registered synthesizer names, sorted for
// determinism.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.entries))
	for name := range r.entries {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Enabled returns the synthesizers activated by cfg. The returned slice is
// in the order declared by cfg.Enabled (not alphabetical) so that callers
// can present findings in a stable, user-controlled sequence.
//
// Unknown names in cfg.Enabled are silently skipped - this lets config
// files reference a synthesizer that ships in a later binary release
// without breaking older binaries.
func (r *Registry) Enabled(cfg EnabledConfig) []Synthesizer {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if cfg.Disabled {
		return nil
	}

	seen := make(map[string]struct{}, len(cfg.Enabled))
	out := make([]Synthesizer, 0, len(cfg.Enabled))
	for _, name := range cfg.Enabled {
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		if s := r.entries[name]; s != nil {
			out = append(out, s)
		}
	}
	return out
}

// ForDoc returns the synthesizers enabled AND applicable to doc. The order
// matches Enabled's order, so review/polish findings are presented in a
// stable sequence.
func (r *Registry) ForDoc(doc *Doc, cfg EnabledConfig) []Synthesizer {
	enabled := r.Enabled(cfg)
	out := make([]Synthesizer, 0, len(enabled))
	for _, s := range enabled {
		if s.Applies(doc) {
			out = append(out, s)
		}
	}
	return out
}

// EnabledConfig is the minimal selection surface the Registry needs. It is
// populated from AngelaConfig.Synthesizers and (optionally) CLI overrides
// in the command layer, so the Registry itself does not depend on the
// config package.
type EnabledConfig struct {
	// Enabled is the ordered list of synthesizer names to activate. Empty
	// means "no synthesizers this run" (the default before 8-18 merges;
	// after 8-18, defaults.go sets this to ["api-postman"]).
	Enabled []string

	// Disabled forces all synthesizers off, overriding Enabled. Mapped from
	// the --no-synthesizers CLI flag.
	Disabled bool
}

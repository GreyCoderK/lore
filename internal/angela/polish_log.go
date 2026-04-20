// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package angela

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/greycoderk/lore/internal/fileutil"
)

// LogEntry is the schema v1 of a single polish.log record. One entry
// per terminal polish state; every terminal state writes exactly one
// line — this is invariant I30.
//
// The schema is intentionally flat and append-oriented: new optional
// fields may be added in v2 without breaking v1 readers (standard
// JSON compatibility). No field rename, no field removal.
type LogEntry struct {
	Timestamp time.Time    `json:"ts"`
	File      string       `json:"file"`
	Op        string       `json:"op"`   // always "polish" for v1
	Mode      string       `json:"mode"` // see LogMode* constants
	AI        *LogAIInfo   `json:"ai,omitempty"`
	Findings  LogFindings  `json:"findings"`
	Result    string       `json:"result"` // see LogResult* constants
	Exit      int          `json:"exit"`
}

// LogAIInfo captures the provider detail for the AI call that
// produced (or was expected to produce) this polish. May be nil when
// no AI call was issued — e.g. aborted_corrupt_src where the pipeline
// refused before contacting the provider (I28).
type LogAIInfo struct {
	Provider         string `json:"provider"`
	Model            string `json:"model,omitempty"`
	PromptTokens     int    `json:"prompt_tokens,omitempty"`
	CompletionTokens int    `json:"completion_tokens,omitempty"`
}

// LogFindings aggregates the structural events detected during this
// polish invocation: leaked frontmatter stripped, duplicate sections
// arbitrated.
type LogFindings struct {
	LeakedFM   *LogLeakedFM   `json:"leaked_fm,omitempty"`
	Duplicates []LogDuplicate `json:"duplicates,omitempty"`
}

// LogLeakedFM reports whether the AI re-emitted a `---\n...\n---\n`
// block on top of its body and how many bytes were removed by the
// sanitizer (I26).
type LogLeakedFM struct {
	Stripped bool `json:"stripped"`
	Bytes    int  `json:"bytes"`
}

// LogDuplicate reports one duplicate-section group and how it was
// resolved. The Resolution field is formatted as "<source>:<decision>"
// where source is "user" (interactive prompt) or "rule"
// (--arbitrate-rule flag), and decision is first / second / both.
// "user:abort" and "rule:abort" may also appear — in that case the
// owning entry's Result is aborted_arbitrate.
type LogDuplicate struct {
	Heading    string `json:"heading"`
	Count      int    `json:"count"`
	Resolution string `json:"resolution"`
}

// Result sentinel values for LogEntry.Result.
const (
	LogResultWritten           = "written"
	LogResultDryRun            = "dryrun"
	LogResultAbortedArbitrate  = "aborted_arbitrate"
	LogResultAbortedCorruptSrc = "aborted_corrupt_src"
	LogResultAIError           = "ai_error"
)

// Mode sentinel values for LogEntry.Mode.
const (
	LogModeFull        = "full"
	LogModeIncremental = "incremental"
	LogModeDryRun      = "dry-run"
	LogModeInteractive = "interactive"
)

// FormatResolution renders the per-group resolution string for the
// LogDuplicate.Resolution field. Source is "user" or "rule"; choice
// is the ArbitrateChoice from the resolution layer.
func FormatResolution(source string, choice ArbitrateChoice) string {
	var tag string
	switch choice {
	case ChoiceFirst:
		tag = "first"
	case ChoiceSecond:
		tag = "second"
	case ChoiceBoth:
		tag = "both"
	case ChoiceAbort:
		tag = "abort"
	default:
		tag = "unknown"
	}
	return source + ":" + tag
}

// PolishLogPath returns the canonical log file path for the supplied
// state directory. The caller owns the state-dir resolution — this
// function only composes the filename.
func PolishLogPath(stateDir string) string {
	return filepath.Join(stateDir, "polish.log")
}

// AppendLogEntry serializes the entry to a JSON line and appends it
// atomically to <stateDir>/polish.log. An advisory flock is held for
// the duration of the append to serialize concurrent polish
// invocations running in different processes.
//
// I/O errors are wrapped and returned; callers should log but not
// block polish success on a failed log append. The pipeline treats
// log write failure as non-fatal.
//
// If the entry.Timestamp is the zero value, it is set to time.Now().UTC()
// before marshaling — callers that want a specific timestamp (e.g. a
// test injecting a fixed clock) should populate the field themselves.
func AppendLogEntry(stateDir string, entry LogEntry) error {
	logPath := PolishLogPath(stateDir)

	// Best-effort mkdir. fileutil.NewFileLock will also ensure the
	// directory, but do it here too so a fresh repo with no state
	// dir yet gets the polish.log written in the expected place.
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return fmt.Errorf("polish log: mkdir state dir: %w", err)
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	} else {
		entry.Timestamp = entry.Timestamp.UTC()
	}
	if entry.Op == "" {
		entry.Op = "polish"
	}

	payload, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("polish log: marshal: %w", err)
	}
	// Every line MUST end with a single \n. json.Marshal does not
	// emit a trailing newline, so append one manually.
	payload = append(payload, '\n')

	lock, err := fileutil.NewFileLock(logPath)
	if err != nil {
		return fmt.Errorf("polish log: lock: %w", err)
	}
	defer lock.Unlock()

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("polish log: open: %w", err)
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Write(payload); err != nil {
		return fmt.Errorf("polish log: write: %w", err)
	}
	return nil
}

// ReadLogEntries parses every line of polish.log into a LogEntry
// slice, ordered oldest-first (file order). Malformed lines are
// skipped silently — this mirrors the tolerant read path used
// elsewhere in the codebase for audit artifacts (e.g. review state
// corruption quarantine at draft_state.go). Intended for tests and
// for a future `polish --show-log` subcommand.
func ReadLogEntries(stateDir string) ([]LogEntry, error) {
	logPath := PolishLogPath(stateDir)
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("polish log: open: %w", err)
	}
	defer func() { _ = f.Close() }()

	return readLogEntriesFrom(f)
}

// readLogEntriesFrom is the io.Reader-based core of ReadLogEntries,
// split out so tests can drive it without touching the filesystem.
func readLogEntriesFrom(r io.Reader) ([]LogEntry, error) {
	var out []LogEntry
	scanner := bufio.NewScanner(r)
	// Raise the line buffer cap to accommodate verbose entries with
	// many duplicate findings; 1 MB is generous and still safe.
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var e LogEntry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			// Tolerant: skip malformed lines. We do not surface per-line
			// errors because the log is audit-oriented, not transactional.
			continue
		}
		out = append(out, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("polish log: scan: %w", err)
	}
	return out, nil
}

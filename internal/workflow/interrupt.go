package workflow

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/museigen/lore/internal/storage"
	"gopkg.in/yaml.v3"
)

// PendingRecord is the YAML structure written to .lore/pending/{hash}.yaml
// on Ctrl+C or non-TTY deferral. The file is retained for manual inspection
// until `lore pending` (Epic 10) processes it.
type PendingRecord struct {
	Commit  string         `yaml:"commit"`
	Date    string         `yaml:"date"`
	Message string         `yaml:"message"`
	Answers PendingAnswers `yaml:"answers"`
	Status  string         `yaml:"status"` // "partial" | "deferred"
	Reason  string         `yaml:"reason"` // "interrupted" | "non-tty" | "rebase"
}

// PendingAnswers holds the question responses collected before interruption.
type PendingAnswers struct {
	Type         string `yaml:"type"`
	What         string `yaml:"what"`
	Why          string `yaml:"why,omitempty"`
	Alternatives string `yaml:"alternatives,omitempty"`
	Impact       string `yaml:"impact,omitempty"`
}

// SavePending writes partial answers to .lore/pending/{hash}.yaml.
// The directory is created with os.MkdirAll if absent (per NOTE m19).
// Relative paths work when CWD is the git work tree (item L19).
func SavePending(workDir string, record PendingRecord) error {
	pendingDir := filepath.Join(workDir, ".lore", "pending")
	if err := os.MkdirAll(pendingDir, 0o755); err != nil {
		return fmt.Errorf("workflow: save pending: mkdir: %w", err)
	}

	name := record.Commit
	if name == "" {
		name = "unknown-" + time.Now().Format("20060102T150405")
	}

	// Append timestamp suffix if file already exists (e.g. rebase --exec replays).
	path := filepath.Join(pendingDir, name+".yaml")
	if _, err := os.Stat(path); err == nil {
		suffix := time.Now().Format("150405")
		path = filepath.Join(pendingDir, name+"-"+suffix+".yaml")
	}

	data, err := yaml.Marshal(record)
	if err != nil {
		return fmt.Errorf("workflow: save pending: marshal: %w", err)
	}

	if err := storage.AtomicWrite(path, data); err != nil {
		return fmt.Errorf("workflow: save pending: write: %w", err)
	}
	return nil
}

// BuildPendingRecord converts partial answers into a PendingRecord.
// commitHash / commitMsg may be empty if the commit could not be read.
// status must be "partial" (interrupted mid-flow) or "deferred" (non-TTY / rebase batch).
func BuildPendingRecord(answers Answers, commitHash, commitMsg, reason, status string) PendingRecord {
	return PendingRecord{
		Commit:  commitHash,
		Date:    time.Now().UTC().Format(time.RFC3339),
		Message: commitMsg,
		Answers: PendingAnswers{
			Type:         answers.Type,
			What:         answers.What,
			Why:          answers.Why,
			Alternatives: answers.Alternatives,
			Impact:       answers.Impact,
		},
		Status: status,
		Reason: reason,
	}
}

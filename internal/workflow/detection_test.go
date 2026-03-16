package workflow

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/greycoderk/lore/internal/domain"
)

// detectAdapter is a configurable mock for detection tests.
// Embeds mockGitAdapter so only overridden fields need to be set.
type detectAdapter struct {
	mockGitAdapter
	isMerge          bool
	mergeErr         error
	isRebase         bool
	rebaseErr        error
	commitMsgResult  bool
	commitMsgErr     error
	gitDirPath       string
	gitDirErr        error
}

func (d *detectAdapter) IsMergeCommit(_ string) (bool, error)             { return d.isMerge, d.mergeErr }
func (d *detectAdapter) IsRebaseInProgress() (bool, error)                { return d.isRebase, d.rebaseErr }
func (d *detectAdapter) CommitMessageContains(_, _ string) (bool, error)  { return d.commitMsgResult, d.commitMsgErr }
func (d *detectAdapter) GitDir() (string, error)                          { return d.gitDirPath, d.gitDirErr }

// nonTTYStreams returns streams backed by bytes.Buffer (not os.File) → non-TTY.
func nonTTYStreams() domain.IOStreams {
	return domain.IOStreams{
		In:  strings.NewReader(""),
		Out: new(strings.Builder),
		Err: new(strings.Builder),
	}
}

// ttyStreams returns streams backed by os.Stdin/os.Stderr for TTY detection.
// In unit tests the actual TTY check will fail (CI), but we override LORE_LINE_MODE
// to force the non-TTY path; for TTY path we rely on the streams being non-file
// and instead focus on testing the detection logic via adapter flags.
func ttyLikeStreams() domain.IOStreams {
	return nonTTYStreams() // tests use non-tty streams; TTY path tested via integration
}

// TestDetect_DocSkip verifies AC-1: [doc-skip] → skip silencieux, exit 0, 0 output.
func TestDetect_DocSkip(t *testing.T) {
	adapter := &detectAdapter{commitMsgResult: true}
	streams := nonTTYStreams()

	result, err := Detect(context.Background(), "abc", adapter, streams, DetectOpts{})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Action != "skip" {
		t.Errorf("Action = %q, want %q", result.Action, "skip")
	}
	if result.Reason != "doc-skip" {
		t.Errorf("Reason = %q, want %q", result.Reason, "doc-skip")
	}
	if result.Message != "" {
		t.Errorf("Message = %q, want empty (silence total)", result.Message)
	}
}

// TestDetect_NonTTY verifies AC-6: non-TTY → defer pending.
func TestDetect_NonTTY(t *testing.T) {
	adapter := &detectAdapter{} // commitMsg=false (no doc-skip)
	streams := nonTTYStreams()

	result, err := Detect(context.Background(), "abc", adapter, streams, DetectOpts{})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Action != "defer" {
		t.Errorf("Action = %q, want %q", result.Action, "defer")
	}
	if result.Reason != "non-tty" {
		t.Errorf("Reason = %q, want %q", result.Reason, "non-tty")
	}
}

// TestDetect_TermDumb verifies AC-6: TERM=dumb traité comme non-TTY.
func TestDetect_TermDumb(t *testing.T) {
	t.Setenv("TERM", "dumb")

	adapter := &detectAdapter{}
	streams := nonTTYStreams()

	result, err := Detect(context.Background(), "abc", adapter, streams, DetectOpts{})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Action != "defer" {
		t.Errorf("Action = %q, want defer (TERM=dumb)", result.Action)
	}
}

// TestDetect_Rebase verifies AC-3: rebase en cours → defer pending.
// Streams are TTY-like but we force the TTY check to pass via LORE_LINE_MODE=0
// and the adapter returns isRebase=true (checked after non-TTY).
// Since test streams are non-os.File, IsInteractiveTTY() returns false.
// To reach the rebase branch we need TTY — use the injectable opts to bypass non-TTY.
func TestDetect_Rebase(t *testing.T) {
	adapter := &detectAdapter{isRebase: true}
	// Use forceTTY opt to simulate TTY for reaching rebase check
	result, err := Detect(context.Background(), "abc", adapter, nonTTYStreams(), DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Action != "defer" {
		t.Errorf("Action = %q, want %q", result.Action, "defer")
	}
	if result.Reason != "rebase" {
		t.Errorf("Reason = %q, want %q", result.Reason, "rebase")
	}
}

// TestDetect_MergeCommit verifies AC-2: merge commit → skip avec message 1 ligne.
func TestDetect_MergeCommit(t *testing.T) {
	adapter := &detectAdapter{isMerge: true}
	result, err := Detect(context.Background(), "abc", adapter, nonTTYStreams(), DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Action != "skip" {
		t.Errorf("Action = %q, want %q", result.Action, "skip")
	}
	if result.Reason != "merge" {
		t.Errorf("Reason = %q, want %q", result.Reason, "merge")
	}
	if result.Message == "" {
		t.Error("Message should not be empty for merge commit")
	}
	lines := strings.Split(strings.TrimSpace(result.Message), "\n")
	if len(lines) != 1 {
		t.Errorf("Message should be exactly 1 line, got %d: %q", len(lines), result.Message)
	}
}

// TestDetect_CherryPick verifies AC-5: cherry-pick avec CHERRY_PICK_HEAD → skip silencieux.
func TestDetect_CherryPick(t *testing.T) {
	gitDir := t.TempDir()
	// Create .git/CHERRY_PICK_HEAD to simulate cherry-pick in progress.
	if err := os.WriteFile(filepath.Join(gitDir, "CHERRY_PICK_HEAD"), []byte("deadbeef\n"), 0o644); err != nil {
		t.Fatalf("create CHERRY_PICK_HEAD: %v", err)
	}

	adapter := &detectAdapter{gitDirPath: gitDir}
	result, err := Detect(context.Background(), "abc", adapter, nonTTYStreams(), DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Action != "skip" {
		t.Errorf("Action = %q, want %q", result.Action, "skip")
	}
	if result.Reason != "cherry-pick" {
		t.Errorf("Reason = %q, want %q", result.Reason, "cherry-pick")
	}
	if result.Message != "" {
		t.Errorf("Message = %q, want empty (silence)", result.Message)
	}
}

// TestDetect_Amend verifies AC-4: amend → action "amend".
func TestDetect_Amend(t *testing.T) {
	adapter := &detectAdapter{}
	result, err := Detect(context.Background(), "abc", adapter, nonTTYStreams(), DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
		GetEnv: func(key string) string {
			if key == "GIT_REFLOG_ACTION" {
				return "amend"
			}
			return ""
		},
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Action != "amend" {
		t.Errorf("Action = %q, want %q", result.Action, "amend")
	}
	if result.Reason != "amend" {
		t.Errorf("Reason = %q, want %q", result.Reason, "amend")
	}
}

// TestDetect_NormalCommit verifies a plain commit → proceed.
func TestDetect_NormalCommit(t *testing.T) {
	adapter := &detectAdapter{}
	result, err := Detect(context.Background(), "abc", adapter, nonTTYStreams(), DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
		GetEnv:           func(_ string) string { return "" },
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Action != "proceed" {
		t.Errorf("Action = %q, want %q", result.Action, "proceed")
	}
}

// mockCorpus implements domain.CorpusReader for detection tests.
type mockCorpus struct {
	docs []domain.DocMeta
}

func (m *mockCorpus) ReadDoc(_ string) (string, error)                          { return "", nil }
func (m *mockCorpus) ListDocs(_ domain.DocFilter) ([]domain.DocMeta, error)     { return m.docs, nil }

// TestDetect_CherryPick_WithDoc verifies AC-5: cherry-pick + doc exists → skip.
func TestDetect_CherryPick_WithDoc(t *testing.T) {
	gitDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(gitDir, "CHERRY_PICK_HEAD"), []byte("deadbeef\n"), 0o644); err != nil {
		t.Fatalf("create CHERRY_PICK_HEAD: %v", err)
	}

	adapter := &detectAdapter{gitDirPath: gitDir}
	corpus := &mockCorpus{docs: []domain.DocMeta{{Commit: "deadbeef"}}}

	result, err := Detect(context.Background(), "abc", adapter, nonTTYStreams(), DetectOpts{
		IsTTY:  func(_ domain.IOStreams) bool { return true },
		Corpus: corpus,
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Action != "skip" {
		t.Errorf("Action = %q, want skip (doc exists for cherry-picked commit)", result.Action)
	}
	if result.Reason != "cherry-pick" {
		t.Errorf("Reason = %q, want cherry-pick", result.Reason)
	}
}

// TestDetect_CherryPick_WithoutDoc verifies AC-5: cherry-pick but no doc → proceed.
func TestDetect_CherryPick_WithoutDoc(t *testing.T) {
	gitDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(gitDir, "CHERRY_PICK_HEAD"), []byte("deadbeef\n"), 0o644); err != nil {
		t.Fatalf("create CHERRY_PICK_HEAD: %v", err)
	}

	adapter := &detectAdapter{gitDirPath: gitDir}
	corpus := &mockCorpus{docs: []domain.DocMeta{}} // no matching doc

	result, err := Detect(context.Background(), "abc", adapter, nonTTYStreams(), DetectOpts{
		IsTTY:  func(_ domain.IOStreams) bool { return true },
		Corpus: corpus,
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Action != "proceed" {
		t.Errorf("Action = %q, want proceed (no doc for cherry-picked commit)", result.Action)
	}
}

// TestDetect_Amend_WithDoc verifies AC-4: amend + doc exists for pre-amend → amend action.
func TestDetect_Amend_WithDoc(t *testing.T) {
	gitDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(gitDir, "ORIG_HEAD"), []byte("origabc\n"), 0o644); err != nil {
		t.Fatalf("create ORIG_HEAD: %v", err)
	}

	adapter := &detectAdapter{gitDirPath: gitDir}
	corpus := &mockCorpus{docs: []domain.DocMeta{{Commit: "origabc"}}}

	result, err := Detect(context.Background(), "abc", adapter, nonTTYStreams(), DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
		GetEnv: func(key string) string {
			if key == "GIT_REFLOG_ACTION" {
				return "amend"
			}
			return ""
		},
		Corpus: corpus,
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Action != "amend" {
		t.Errorf("Action = %q, want amend (doc exists for pre-amend commit)", result.Action)
	}
}

// TestDetect_Amend_WithoutDoc verifies AC-4: amend but no doc → proceed (create new).
func TestDetect_Amend_WithoutDoc(t *testing.T) {
	gitDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(gitDir, "ORIG_HEAD"), []byte("origabc\n"), 0o644); err != nil {
		t.Fatalf("create ORIG_HEAD: %v", err)
	}

	adapter := &detectAdapter{gitDirPath: gitDir}
	corpus := &mockCorpus{docs: []domain.DocMeta{}} // no matching doc

	result, err := Detect(context.Background(), "abc", adapter, nonTTYStreams(), DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
		GetEnv: func(key string) string {
			if key == "GIT_REFLOG_ACTION" {
				return "amend"
			}
			return ""
		},
		Corpus: corpus,
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Action != "proceed" {
		t.Errorf("Action = %q, want proceed (no doc for pre-amend commit)", result.Action)
	}
}

// TestDetect_DocSkipPriority verifies [doc-skip] wins over all other conditions.
func TestDetect_DocSkipPriority(t *testing.T) {
	// Even with merge=true and rebase=true, doc-skip takes priority.
	adapter := &detectAdapter{
		commitMsgResult: true,
		isMerge:         true,
		isRebase:        true,
	}
	result, err := Detect(context.Background(), "abc", adapter, nonTTYStreams(), DetectOpts{
		IsTTY: func(_ domain.IOStreams) bool { return true },
	})
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if result.Action != "skip" || result.Reason != "doc-skip" {
		t.Errorf("Action=%q Reason=%q, want skip/doc-skip", result.Action, result.Reason)
	}
}


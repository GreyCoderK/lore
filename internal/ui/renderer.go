package ui

// Renderer defines the contract for question flow rendering.
// Two implementations exist:
//   - ProgressRenderer (TTY): condensation progressive, bar, checkmarks, ANSI colors
//   - LineRenderer (non-TTY): one line per event, no ANSI rewriting, CI/pipe compatible
//
// Current implementations live in workflow/ (ProgressRenderer and LineRenderer).
// TODO: migrate workflow.ProgressRenderer and workflow.LineRenderer here post-MVP.
type Renderer interface {
	Progress(current, total int, label string)
	QuestionConfirm(label, value string)
	Result(verb, filename string)
}

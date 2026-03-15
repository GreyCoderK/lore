package workflow

// Renderer abstracts TTY vs non-TTY output during the question flow.
// Implementations: ProgressRenderer (TTY) and LineRenderer (non-TTY/CI).
type Renderer interface {
	// QuestionStart displays a question before the user types.
	QuestionStart(question string, defaultVal string)
	// QuestionConfirm condenses a confirmed answer into the summary bar.
	QuestionConfirm(question string, answer string)
	// Progress shows the current question position and label.
	Progress(current, total int, label string)
	// ExpressSkip announces that optional questions were skipped.
	ExpressSkip(skipped int)
}

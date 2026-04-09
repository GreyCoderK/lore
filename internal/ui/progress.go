// Copyright (C) 2026 Museigen
// SPDX-License-Identifier: AGPL-3.0-or-later

package ui

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/greycoderk/lore/internal/domain"
)

// Progress displays "[##·] 3/5 label" on stderr.
func Progress(streams domain.IOStreams, current, total int, label string) {
	if current < 0 {
		current = 0
	}
	if total < 0 {
		total = 0
	}
	if current > total {
		current = total
	}
	bar := strings.Repeat("#", current) + strings.Repeat("·", total-current)
	fmt.Fprintf(streams.Err, "[%s] %d/%d %s\n", bar, current, total, label)
}

// spinnerFrames are the animation frames for the terminal spinner.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Spinner displays an animated spinner with a label, elapsed time, and
// optional timeout countdown on stderr.
type Spinner struct {
	w       io.Writer
	stopCh  chan struct{}
	done    sync.WaitGroup
	start   time.Time
	timeout time.Duration // 0 = no timeout display
	warned  atomic.Bool
}

// StartSpinner begins a spinner animation with the given label.
// Shows elapsed time: ⠹ Label… (12s)
func StartSpinner(streams domain.IOStreams, label string) *Spinner {
	return startSpinnerInternal(streams.Err, label, 0)
}

// StartSpinnerWithTimeout begins a spinner that also shows a countdown
// to the configured timeout and warns when 80% has elapsed.
func StartSpinnerWithTimeout(streams domain.IOStreams, label string, timeout time.Duration) *Spinner {
	return startSpinnerInternal(streams.Err, label, timeout)
}

func startSpinnerInternal(w io.Writer, label string, timeout time.Duration) *Spinner {
	s := &Spinner{
		w:       w,
		stopCh:  make(chan struct{}),
		start:   time.Now(),
		timeout: timeout,
	}
	// Print the label immediately so the user sees something right away
	fmt.Fprintf(s.w, "%s %s\n", spinnerFrames[0], label)
	s.done.Add(1)
	go func() {
		defer s.done.Done()
		i := 0
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				fmt.Fprintf(s.w, "\r\033[K")
				return
			case <-ticker.C:
				elapsed := time.Since(s.start).Truncate(time.Second)
				frame := spinnerFrames[i%len(spinnerFrames)]

				if s.timeout > 0 {
					remaining := s.timeout - elapsed
					pct := float64(elapsed) / float64(s.timeout)

					if pct >= 0.80 && !s.warned.Load() {
						s.warned.Store(true)
						// Print warning on a new line, then continue spinner
						fmt.Fprintf(s.w, "\r\033[K")
						fmt.Fprintf(s.w, "  ⚠ %s remaining before timeout — waiting for AI response…\n",
							formatDuration(remaining.Truncate(time.Second)))
					}

					if s.warned.Load() {
						fmt.Fprintf(s.w, "\r\033[K%s %s (%s / %s ⚠)",
							frame, label,
							formatDuration(elapsed),
							formatDuration(s.timeout))
					} else {
						fmt.Fprintf(s.w, "\r\033[K%s %s (%s / %s)",
							frame, label,
							formatDuration(elapsed),
							formatDuration(s.timeout))
					}
				} else {
					fmt.Fprintf(s.w, "\r\033[K%s %s (%s)", frame, label, formatDuration(elapsed))
				}
				i++
			}
		}
	}()
	return s
}

// Stop halts the spinner and clears the line.
func (s *Spinner) Stop() {
	select {
	case <-s.stopCh:
		return
	default:
		close(s.stopCh)
	}
	s.done.Wait()
}

// Elapsed returns the time since the spinner started.
func (s *Spinner) Elapsed() time.Duration {
	return time.Since(s.start)
}

// StopWith halts the spinner and prints a final message on the same line.
func (s *Spinner) StopWith(msg string) {
	select {
	case <-s.stopCh:
		return
	default:
		close(s.stopCh)
	}
	s.done.Wait()
	fmt.Fprintf(s.w, "\r\033[K%s\n", msg)
}

// StopWithDuration halts the spinner and prints a message with elapsed time.
func (s *Spinner) StopWithDuration(msg string) {
	elapsed := time.Since(s.start)
	select {
	case <-s.stopCh:
		return
	default:
		close(s.stopCh)
	}
	s.done.Wait()
	fmt.Fprintf(s.w, "\r\033[K%s (%s)\n", msg, formatDuration(elapsed.Truncate(time.Second)))
}

// formatDuration produces a compact human-readable duration: "0s", "12s", "1m30s".
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) - m*60
	return fmt.Sprintf("%dm%ds", m, s)
}

package ui_test

import "github.com/greycoderk/lore/internal/ui"

// Compile-time check: verify the Renderer interface is satisfiable.
// A concrete type implementing all three methods must compile.
var _ ui.Renderer = (*mockRenderer)(nil)

type mockRenderer struct{}

func (m *mockRenderer) Progress(current, total int, label string) {}
func (m *mockRenderer) QuestionConfirm(label, value string)      {}
func (m *mockRenderer) Result(verb, filename string)              {}

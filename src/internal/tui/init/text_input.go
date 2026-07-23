package tui

import (
	bubbleTea "charm.land/bubbletea/v2"
)

// TextInputModel is a hand-rolled single-line text field. No bubbles/textinput
// dependency — not worth adding for one field.
type TextInputModel struct {
	Value string
}

func (t *TextInputModel) Handle(key bubbleTea.KeyPressMsg) {
	if key.String() == "backspace" {
		if len(t.Value) > 0 {
			runes := []rune(t.Value)
			t.Value = string(runes[:len(runes)-1])
		}
		return
	}

	t.Value += key.Key().Text
}

func (t TextInputModel) View(label string) string {
	return label + ": " + t.Value + "█\n"
}

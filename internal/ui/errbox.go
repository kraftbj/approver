package ui

// ErrBox displays a single-line error message.
type ErrBox struct {
	Message string
	Width   int
}

func NewErrBox() ErrBox {
	return ErrBox{}
}

// SetWidth updates the error box width.
func (e *ErrBox) SetWidth(w int) {
	e.Width = w
}

// SetError sets the error message.
func (e *ErrBox) SetError(msg string) {
	e.Message = msg
}

// Clear removes the error message.
func (e *ErrBox) Clear() {
	e.Message = ""
}

// View renders the error box. Returns empty string if no error.
func (e *ErrBox) View() string {
	if e.Message == "" {
		return ""
	}
	return ErrorStyle.Width(e.Width).Render(e.Message)
}

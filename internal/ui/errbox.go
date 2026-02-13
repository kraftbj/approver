package ui

// ErrBox displays a single-line error or info message.
type ErrBox struct {
	Message string
	IsInfo  bool
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
	e.IsInfo = false
}

// SetInfo sets an info (success) message.
func (e *ErrBox) SetInfo(msg string) {
	e.Message = msg
	e.IsInfo = true
}

// Clear removes the error message.
func (e *ErrBox) Clear() {
	e.Message = ""
	e.IsInfo = false
}

// View renders the error box. Returns empty string if no error.
func (e *ErrBox) View() string {
	if e.Message == "" {
		return ""
	}
	if e.IsInfo {
		return InfoStyle.Width(e.Width).Render(e.Message)
	}
	return ErrorStyle.Width(e.Width).Render(e.Message)
}

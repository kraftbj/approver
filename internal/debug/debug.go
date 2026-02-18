package debug

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
)

var (
	enabled bool
	logger  *log.Logger
	once    sync.Once
)

// Enable turns on debug logging. Call before any Log calls.
func Enable() {
	enabled = true
}

// Log writes a debug message to ~/.config/approver/debug.log.
// No-op unless Enable() has been called.
func Log(format string, args ...interface{}) {
	if !enabled {
		return
	}
	once.Do(func() {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		dir := filepath.Join(home, ".config", "approver")
		os.MkdirAll(dir, 0o755)
		f, err := os.OpenFile(filepath.Join(dir, "debug.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return
		}
		logger = log.New(f, "", log.Ltime|log.Lmicroseconds)
	})
	if logger != nil {
		logger.Output(2, fmt.Sprintf(format, args...))
	}
}

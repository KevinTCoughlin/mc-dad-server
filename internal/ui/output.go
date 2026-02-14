package ui

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/term"
)

// Color codes for terminal output.
const (
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[1;33m"
	colorBlue   = "\033[0;34m"
	colorCyan   = "\033[0;36m"
	colorBold   = "\033[1m"
	colorReset  = "\033[0m"
)

// UI provides colored terminal output for user-facing messages.
type UI struct {
	color bool
}

var (
	defaultUI   *UI
	defaultOnce sync.Once
)

// Default returns a shared UI instance with auto-detected color support.
func Default() *UI {
	defaultOnce.Do(func() {
		defaultUI = New(shouldColor())
	})
	return defaultUI
}

// New creates a UI with explicit color control.
func New(color bool) *UI {
	return &UI{color: color}
}

func shouldColor() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func (u *UI) colorize(color, s string) string {
	if !u.color {
		return s
	}
	return color + s + colorReset
}

// Info prints an informational message.
func (u *UI) Info(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(u.colorize(colorBlue, "[INFO]") + " " + msg)
}

// Success prints a success message.
func (u *UI) Success(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(u.colorize(colorGreen, "[OK]") + " " + msg)
}

// Warn prints a warning message.
func (u *UI) Warn(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Println(u.colorize(colorYellow, "[WARN]") + " " + msg)
}

// Error prints an error message to stderr.
func (u *UI) Error(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintln(os.Stderr, u.colorize(colorRed, "[ERROR]")+" "+msg)
}

// Step prints a section header.
func (u *UI) Step(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	header := fmt.Sprintf("\n━━━ %s ━━━\n", msg)
	fmt.Println(u.colorize(colorCyan+colorBold, header))
}

// Bold returns text wrapped in bold codes (if color enabled).
func (u *UI) Bold(s string) string {
	return u.colorize(colorBold, s)
}

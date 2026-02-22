package console

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/KevinTCoughlin/mc-dad-server/internal/platform"
)

// Options holds the values the console needs from the CLI globals.
type Options struct {
	Dir     string
	Session string
}

// Styles.
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("62")).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)
)

// cmdDoneMsg is sent when a dispatched command finishes.
type cmdDoneMsg struct {
	input  string
	output string
	quit   bool
}

type model struct {
	viewport viewport.Model
	input    textinput.Model
	lines    []string
	history  []string
	histIdx  int
	opts     *Options
	runner   platform.CommandRunner
	width    int
	height   int
	ready    bool
	quitting bool
	cancel   context.CancelFunc
	ctx      context.Context
	logPath  string
	// logOffset tracks file position for the log tailer.
	logOffset int64
	// running indicates whether a command is currently executing.
	running bool
}

func newModel(opts *Options, runner platform.CommandRunner) model {
	ti := textinput.New()
	ti.Prompt = promptStyle.Render("> ")
	ti.Focus()
	ti.CharLimit = 256

	ctx, cancel := context.WithCancel(context.Background())

	return model{
		input:   ti,
		opts:    opts,
		runner:  runner,
		histIdx: -1,
		ctx:     ctx,
		cancel:  cancel,
		logPath: filepath.Join(opts.Dir, "logs", "latest.log"),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		tailLog(m.ctx, m.logPath),
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Reserve lines: 1 title bar + 1 input + 1 status bar.
		viewHeight := m.height - 3
		if viewHeight < 1 {
			viewHeight = 1
		}

		if !m.ready {
			m.viewport = viewport.New(m.width, viewHeight)
			m.viewport.SetContent(strings.Join(m.lines, "\n"))
			m.ready = true
		} else {
			m.viewport.Width = m.width
			m.viewport.Height = viewHeight
		}
		m.input.Width = m.width - 4

	case logReadMsg:
		m.logOffset = msg.offset
		m.lines = append(m.lines, msg.line)
		if m.ready {
			m.viewport.SetContent(strings.Join(m.lines, "\n"))
			m.viewport.GotoBottom()
		}
		cmds = append(cmds, nextLogLine(m.ctx, m.logPath, m.logOffset))

	case cmdDoneMsg:
		m.running = false
		if msg.quit {
			m.quitting = true
			m.cancel()
			return m, tea.Quit
		}
		if msg.output == clearSentinel {
			m.lines = nil
			if m.ready {
				m.viewport.SetContent("")
			}
		} else {
			// Show the command that was run.
			m.lines = append(m.lines, promptStyle.Render("> ")+msg.input)
			if msg.output != "" {
				for _, line := range strings.Split(msg.output, "\n") {
					m.lines = append(m.lines, line)
				}
			}
			if m.ready {
				m.viewport.SetContent(strings.Join(m.lines, "\n"))
				m.viewport.GotoBottom()
			}
		}

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.quitting = true
			m.cancel()
			return m, tea.Quit

		case tea.KeyEnter:
			if m.running {
				break
			}
			input := m.input.Value()
			m.input.SetValue("")
			if strings.TrimSpace(input) == "" {
				break
			}
			// Add to history.
			m.history = append(m.history, input)
			m.histIdx = len(m.history)
			// Run command asynchronously.
			m.running = true
			cmds = append(cmds, m.runCommand(input))

		case tea.KeyUp:
			if len(m.history) > 0 && m.histIdx > 0 {
				m.histIdx--
				m.input.SetValue(m.history[m.histIdx])
				m.input.CursorEnd()
			}

		case tea.KeyDown:
			if m.histIdx < len(m.history)-1 {
				m.histIdx++
				m.input.SetValue(m.history[m.histIdx])
				m.input.CursorEnd()
			} else if m.histIdx == len(m.history)-1 {
				m.histIdx = len(m.history)
				m.input.SetValue("")
			}

		case tea.KeyPgUp, tea.KeyPgDown:
			var vpCmd tea.Cmd
			m.viewport, vpCmd = m.viewport.Update(msg)
			cmds = append(cmds, vpCmd)
		}
	}

	// Update text input.
	var tiCmd tea.Cmd
	m.input, tiCmd = m.input.Update(msg)
	cmds = append(cmds, tiCmd)

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	title := titleStyle.Render(" MC Dad Server Console ")
	statusText := statusBarStyle.Render(
		fmt.Sprintf(" %s | Ctrl+C to exit | PgUp/PgDn to scroll", m.opts.Dir))

	// Pad title bar to full width.
	titleBar := title + strings.Repeat(" ", max(0, m.width-lipgloss.Width(title)))

	return fmt.Sprintf("%s\n%s\n%s\n%s",
		titleBar,
		m.viewport.View(),
		m.input.View(),
		statusText,
	)
}

func (m model) runCommand(input string) tea.Cmd {
	ctx := m.ctx
	opts := m.opts
	runner := m.runner
	return func() tea.Msg {
		output, quit := dispatch(ctx, input, opts, runner)
		return cmdDoneMsg{input: input, output: output, quit: quit}
	}
}

// Run starts the interactive console TUI.
func Run(opts *Options, runner platform.CommandRunner) error {
	p := tea.NewProgram(
		newModel(opts, runner),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	_, err := p.Run()
	return err
}

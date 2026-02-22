package console

import (
	"bufio"
	"context"
	"io"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// logReadMsg carries a log line and the file offset for the next read.
type logReadMsg struct {
	line   string
	offset int64
}

// scanBufInitial is the initial size of the bufio.Scanner buffer for log lines.
const scanBufInitial = 64 * 1024

// scanBufMax is the maximum token size accepted from a log line. Lines longer
// than this are split across multiple reads.
const scanBufMax = 1024 * 1024

// tailLog returns a tea.Cmd that waits for the log file to appear, then reads
// the first new line from the end. Subsequent lines are read via nextLogLine.
func tailLog(ctx context.Context, path string) tea.Cmd {
	return func() tea.Msg {
		// Wait for the file to appear.
		var startOffset int64
		for {
			info, err := os.Stat(path)
			if err == nil {
				startOffset = info.Size()
				break
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(500 * time.Millisecond):
			}
		}

		// Now poll from end using the offset-based reader.
		return readFromOffset(ctx, path, startOffset)
	}
}

// nextLogLine returns a tea.Cmd that waits for the next log line at the given
// offset. It handles log rotation by resetting the offset when the file
// shrinks.
func nextLogLine(ctx context.Context, path string, offset int64) tea.Cmd {
	return func() tea.Msg {
		return readFromOffset(ctx, path, offset)
	}
}

// readFromOffset polls the file at the given offset until a complete line is
// available. Returns a logReadMsg with the line and updated offset.
func readFromOffset(ctx context.Context, path string, offset int64) tea.Msg {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		f, err := os.Open(path)
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// Check if file was rotated (smaller than our offset).
		info, err := f.Stat()
		if err != nil {
			_ = f.Close()
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if info.Size() < offset {
			offset = 0
		}

		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			_ = f.Close()
			time.Sleep(100 * time.Millisecond)
			continue
		}

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, scanBufInitial), scanBufMax)
		if scanner.Scan() {
			line := scanner.Text()
			newOffset, err := f.Seek(0, io.SeekCurrent)
			if err != nil {
				_ = f.Close()
				time.Sleep(100 * time.Millisecond)
				continue
			}
			_ = f.Close()
			return logReadMsg{line: line, offset: newOffset}
		}

		if err := scanner.Err(); err != nil {
			_ = f.Close()
			time.Sleep(100 * time.Millisecond)
			continue
		}

		_ = f.Close()
		time.Sleep(100 * time.Millisecond)
	}
}

package vote

import (
	"bufio"
	"context"
	"io"
	"os"
	"regexp"
	"time"
)

// chatRegex matches Minecraft server log chat lines:
// [HH:MM:SS] [Server thread/INFO]: <PlayerName> message
var chatRegex = regexp.MustCompile(`\[[\d:]+\] \[Server thread/INFO\]: <(\w+)> (.+)$`)

// ParseChatMessage extracts player name and message from a server log line.
func ParseChatMessage(line string) (player, message string, ok bool) {
	m := chatRegex.FindStringSubmatch(line)
	if m == nil {
		return "", "", false
	}
	return m[1], m[2], true
}

// TailLog opens a log file, seeks to the end, and sends new lines to the
// returned channel. It polls for new data and stops when ctx is cancelled.
func TailLog(ctx context.Context, path string) (<-chan string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// Seek to end so we only read new lines.
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		_ = f.Close()
		return nil, err
	}

	ch := make(chan string, 64)
	go func() {
		defer func() { _ = f.Close() }()
		defer close(ch)

		scanner := bufio.NewScanner(f)
		for {
			for scanner.Scan() {
				select {
				case ch <- scanner.Text():
				case <-ctx.Done():
					return
				}
			}
			// No more data — wait and retry.
			select {
			case <-ctx.Done():
				return
			case <-time.After(250 * time.Millisecond):
			}
		}
	}()

	return ch, nil
}

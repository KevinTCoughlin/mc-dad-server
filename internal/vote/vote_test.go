package vote

import (
	"testing"
)

func TestParseChatMessage(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantPlayer string
		wantMsg    string
		wantOk     bool
	}{
		{
			name:       "valid vote",
			line:       "[12:34:56] [Server thread/INFO]: <Steve> 1",
			wantPlayer: "Steve",
			wantMsg:    "1",
			wantOk:     true,
		},
		{
			name:       "valid chat message",
			line:       "[09:00:00] [Server thread/INFO]: <Alex> hello world",
			wantPlayer: "Alex",
			wantMsg:    "hello world",
			wantOk:     true,
		},
		{
			name:       "multi digit vote",
			line:       "[12:34:56] [Server thread/INFO]: <Player123> 5",
			wantPlayer: "Player123",
			wantMsg:    "5",
			wantOk:     true,
		},
		{
			name:   "server message not chat",
			line:   "[12:34:56] [Server thread/INFO]: Steve joined the game",
			wantOk: false,
		},
		{
			name:   "empty line",
			line:   "",
			wantOk: false,
		},
		{
			name:   "different thread",
			line:   "[12:34:56] [Async Chat Thread/INFO]: <Steve> 1",
			wantOk: false,
		},
		{
			name:       "underscore in name",
			line:       "[12:34:56] [Server thread/INFO]: <Cool_Kid99> 3",
			wantPlayer: "Cool_Kid99",
			wantMsg:    "3",
			wantOk:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			player, msg, ok := ParseChatMessage(tt.line)
			if ok != tt.wantOk {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOk)
			}
			if !ok {
				return
			}
			if player != tt.wantPlayer {
				t.Errorf("player = %q, want %q", player, tt.wantPlayer)
			}
			if msg != tt.wantMsg {
				t.Errorf("msg = %q, want %q", msg, tt.wantMsg)
			}
		})
	}
}

func TestPickCandidates(t *testing.T) {
	t.Run("all maps fit", func(t *testing.T) {
		maps := []string{"a", "b", "c"}
		got := pickCandidates(maps, 5)
		if len(got) != 3 {
			t.Fatalf("got %d candidates, want 3", len(got))
		}
	})

	t.Run("subset selected", func(t *testing.T) {
		maps := []string{"a", "b", "c", "d", "e", "f", "g"}
		got := pickCandidates(maps, 3)
		if len(got) != 3 {
			t.Fatalf("got %d candidates, want 3", len(got))
		}
		// All results should be from the pool.
		pool := make(map[string]bool)
		for _, m := range maps {
			pool[m] = true
		}
		for _, c := range got {
			if !pool[c] {
				t.Errorf("candidate %q not in pool", c)
			}
		}
	})

	t.Run("no duplicates", func(t *testing.T) {
		maps := []string{"a", "b", "c", "d", "e"}
		got := pickCandidates(maps, 5)
		seen := make(map[string]bool)
		for _, c := range got {
			if seen[c] {
				t.Errorf("duplicate candidate: %q", c)
			}
			seen[c] = true
		}
	})

	t.Run("empty pool", func(t *testing.T) {
		got := pickCandidates(nil, 5)
		if len(got) != 0 {
			t.Fatalf("got %d candidates from empty pool", len(got))
		}
	})
}

func TestPickWinner(t *testing.T) {
	candidates := []string{"a", "b", "c"}

	t.Run("clear winner", func(t *testing.T) {
		tally := map[string]int{"a": 1, "b": 5, "c": 2}
		winner := pickWinner(candidates, tally)
		if winner != "b" {
			t.Errorf("winner = %q, want %q", winner, "b")
		}
	})

	t.Run("no votes picks random candidate", func(t *testing.T) {
		winner := pickWinner(candidates, map[string]int{})
		found := false
		for _, c := range candidates {
			if winner == c {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("winner %q not in candidates", winner)
		}
	})

	t.Run("single voter", func(t *testing.T) {
		tally := map[string]int{"c": 1}
		winner := pickWinner(candidates, tally)
		if winner != "c" {
			t.Errorf("winner = %q, want %q", winner, "c")
		}
	})

	t.Run("tie picks from tied candidates", func(t *testing.T) {
		tally := map[string]int{"a": 3, "b": 3}
		// Run multiple times to verify it picks from tied set.
		for range 20 {
			winner := pickWinner(candidates, tally)
			if winner != "a" && winner != "b" {
				t.Errorf("winner = %q, should be a or b", winner)
			}
		}
	})
}

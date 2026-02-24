package vote

import (
	"context"
	"fmt"
	"math/rand/v2"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KevinTCoughlin/mc-dad-server/internal/management"
	"github.com/KevinTCoughlin/mc-dad-server/internal/ui"
)

// Config configures a map vote session.
type Config struct {
	Maps       []string      // candidate map pool
	Duration   time.Duration // vote window
	MaxChoices int           // maps shown per vote
	ServerDir  string
	Screen     management.ServerManager
	Output     *ui.UI
}

// Result holds the outcome of a completed vote.
type Result struct {
	Winner string
	Votes  map[string]int
	Voters int
}

// RunVote runs a complete map vote: broadcast options, collect votes from the
// server log, tally results, and announce the winner.
func RunVote(ctx context.Context, cfg *Config) (*Result, error) {
	candidates := pickCandidates(cfg.Maps, cfg.MaxChoices)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no maps available for voting")
	}

	cfg.Output.Info("Starting map vote with %d candidates for %s", len(candidates), cfg.Duration)

	// Broadcast vote options.
	if err := broadcastVoteStart(ctx, cfg.Screen, candidates, int(cfg.Duration.Seconds())); err != nil {
		return nil, fmt.Errorf("broadcasting vote: %w", err)
	}

	// Start tailing the log.
	logPath := filepath.Join(cfg.ServerDir, "logs", "latest.log")
	voteCtx, cancel := context.WithTimeout(ctx, cfg.Duration)
	defer cancel()

	lines, err := TailLog(voteCtx, logPath)
	if err != nil {
		return nil, fmt.Errorf("tailing log: %w", err)
	}

	// Collect votes.
	var mu sync.Mutex
	playerVotes := make(map[string]int) // player -> choice index (1-based)

	// Schedule reminders.
	go sendReminders(voteCtx, cfg.Screen, candidates, cfg.Duration)

	// Read votes until timeout.
	for line := range lines {
		player, msg, ok := ParseChatMessage(line)
		if !ok {
			continue
		}
		msg = strings.TrimSpace(msg)
		choice, err := strconv.Atoi(msg)
		if err != nil || choice < 1 || choice > len(candidates) {
			continue
		}
		mu.Lock()
		playerVotes[player] = choice
		mu.Unlock()
		cfg.Output.Info("%s voted for [%d] %s", player, choice, candidates[choice-1])
	}

	// Tally.
	mu.Lock()
	tally := make(map[string]int)
	for _, choice := range playerVotes {
		tally[candidates[choice-1]]++
	}
	mu.Unlock()

	winner := pickWinner(candidates, tally)
	result := &Result{
		Winner: winner,
		Votes:  tally,
		Voters: len(playerVotes),
	}

	cfg.Output.Success("Vote complete: %s wins with %d votes (%d voters)",
		winner, tally[winner], result.Voters)

	// Announce results and teleport.
	if err := broadcastResults(ctx, cfg.Screen, candidates, tally, winner); err != nil {
		return result, fmt.Errorf("broadcasting results: %w", err)
	}

	// Countdown then teleport.
	if err := countdownAndTeleport(ctx, cfg.Screen, winner, cfg.Output); err != nil {
		return result, fmt.Errorf("teleporting: %w", err)
	}

	return result, nil
}

// pickCandidates selects up to maxChoices maps from the pool via random sampling.
func pickCandidates(pool []string, maxChoices int) []string {
	if len(pool) <= maxChoices {
		out := make([]string, len(pool))
		copy(out, pool)
		return out
	}
	perm := rand.Perm(len(pool))
	out := make([]string, maxChoices)
	for i := range maxChoices {
		out[i] = pool[perm[i]]
	}
	return out
}

// pickWinner returns the map with the most votes. Ties are broken randomly.
// If no votes were cast, a random candidate is chosen.
func pickWinner(candidates []string, tally map[string]int) string {
	if len(tally) == 0 {
		return candidates[rand.IntN(len(candidates))]
	}

	maxVotes := 0
	for _, v := range tally {
		if v > maxVotes {
			maxVotes = v
		}
	}

	var tied []string
	for _, c := range candidates {
		if tally[c] == maxVotes {
			tied = append(tied, c)
		}
	}
	return tied[rand.IntN(len(tied))]
}

// broadcastVoteStart sends the vote options to all players via tellraw.
func broadcastVoteStart(ctx context.Context, screen management.ServerManager, candidates []string, durationSec int) error {
	lines := []string{
		`["",{"text":"==========================","color":"gold"}]`,
		`["",{"text":"   VOTE FOR NEXT MAP!","color":"gold","bold":true}]`,
		`["",{"text":"==========================","color":"gold"}]`,
		`["",{"text":"Type a number to vote:","color":"white"}]`,
	}
	for i, m := range candidates {
		lines = append(lines, fmt.Sprintf(
			`["",{"text":"  [%d] ","color":"green","bold":true},{"text":%q,"color":"white"}]`,
			i+1, m))
	}
	lines = append(lines,
		`["",{"text":"==========================","color":"gold"}]`,
		fmt.Sprintf(`["",{"text":"  Voting ends in %d seconds","color":"yellow"}]`, durationSec),
		`["",{"text":"==========================","color":"gold"}]`,
	)

	for _, l := range lines {
		if err := screen.SendCommand(ctx, "tellraw @a "+l); err != nil {
			return err
		}
	}
	return nil
}

// sendReminders sends periodic vote reminders at halfway and 5s remaining.
func sendReminders(ctx context.Context, screen management.ServerManager, candidates []string, duration time.Duration) {
	half := duration / 2
	fiveSec := duration - 5*time.Second

	select {
	case <-ctx.Done():
		return
	case <-time.After(half):
		msg := fmt.Sprintf(`["",{"text":"Vote reminder! %d seconds left. Type 1-%d to vote!","color":"yellow"}]`,
			int(duration.Seconds())-int(half.Seconds()), len(candidates))
		_ = screen.SendCommand(ctx, "tellraw @a "+msg)
	}

	if fiveSec > half {
		remaining := fiveSec - half
		select {
		case <-ctx.Done():
			return
		case <-time.After(remaining):
			msg := `["",{"text":"5 seconds left to vote!","color":"red","bold":true}]`
			_ = screen.SendCommand(ctx, "tellraw @a "+msg)
		}
	}
}

// broadcastResults announces the vote results to all players.
func broadcastResults(ctx context.Context, screen management.ServerManager, candidates []string, tally map[string]int, winner string) error {
	lines := []string{
		`["",{"text":"==========================","color":"gold"}]`,
		`["",{"text":"   RESULTS","color":"gold","bold":true}]`,
		`["",{"text":"==========================","color":"gold"}]`,
	}
	for _, c := range candidates {
		count := tally[c]
		label := "votes"
		if count == 1 {
			label = "vote"
		}
		marker := ""
		if c == winner {
			marker = " ***"
		}
		lines = append(lines, fmt.Sprintf(
			`["",{"text":"  %s: %d %s%s","color":%q}]`,
			c, count, label, marker, mapResultColor(c, winner)))
	}
	lines = append(lines,
		`["",{"text":"==========================","color":"gold"}]`,
	)

	for _, l := range lines {
		if err := screen.SendCommand(ctx, "tellraw @a "+l); err != nil {
			return err
		}
	}
	return nil
}

func mapResultColor(candidate, winner string) string {
	if candidate == winner {
		return "green"
	}
	return "gray"
}

// countdownAndTeleport announces a 5-second countdown then teleports all players.
func countdownAndTeleport(ctx context.Context, screen management.ServerManager, mapName string, output *ui.UI) error {
	msg := fmt.Sprintf(`["",{"text":"  Loading %s in 5...","color":"yellow","bold":true}]`, mapName)
	if err := screen.SendCommand(ctx, "tellraw @a "+msg); err != nil {
		return err
	}

	for i := 4; i >= 1; i-- {
		if err := management.Sleep(ctx, 1); err != nil {
			return err
		}
		msg := fmt.Sprintf(`["",{"text":"%d...","color":"yellow"}]`, i)
		if err := screen.SendCommand(ctx, "tellraw @a "+msg); err != nil {
			return err
		}
	}

	if err := management.Sleep(ctx, 1); err != nil {
		return err
	}

	output.Info("Teleporting all players to %s", mapName)
	return management.RotateToMap(ctx, mapName, screen, output)
}

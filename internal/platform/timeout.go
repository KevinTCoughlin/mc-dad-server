package platform

import (
	"context"
	"time"
)

// WithTimeout returns ctx when it already has a deadline. Otherwise it
// returns a child that expires after d, so an HTTP call made with
// context.Background() cannot hang. cancel is always safe to call.
func WithTimeout(ctx context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, ok := ctx.Deadline(); ok {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, d)
}

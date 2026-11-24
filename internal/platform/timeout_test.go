package platform

import (
	"context"
	"testing"
	"time"
)

func TestWithTimeoutAddsDeadline(t *testing.T) {
	t.Parallel()

	ctx, cancel := WithTimeout(context.Background(), time.Second)
	defer cancel()

	if _, ok := ctx.Deadline(); !ok {
		t.Fatal("expected a deadline on a context that had none")
	}
}

func TestWithTimeoutPreservesDeadline(t *testing.T) {
	t.Parallel()

	parent, stop := context.WithTimeout(context.Background(), time.Hour)
	defer stop()
	want, ok := parent.Deadline()
	if !ok {
		t.Fatal("parent deadline missing")
	}

	ctx, cancel := WithTimeout(parent, time.Millisecond)
	defer cancel()

	got, ok := ctx.Deadline()
	if !ok {
		t.Fatal("child deadline missing")
	}
	if !got.Equal(want) {
		t.Fatalf("deadline = %s, want the parent deadline %s", got, want)
	}
}

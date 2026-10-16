package search_test

import (
	"context"
	"testing"
	"time"

	"github.com/gohandson/goroutineleak-ja/search"
)

func backend(d time.Duration, v string) search.Backend {
	return func(ctx context.Context, query string) (string, error) {
		time.Sleep(d)
		return v, nil
	}
}

func TestFirst(t *testing.T) {
	// TODO: goleak.VerifyNone 関数を defer で呼び出す

	backends := []search.Backend{
		backend(0, "fast"),
		backend(50*time.Millisecond, "slow1"),
		backend(50*time.Millisecond, "slow2"),
	}

	got, err := search.First(context.Background(), "gopher", backends)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if got != "fast" {
		t.Errorf("got %q, want %q", got, "fast")
	}
}

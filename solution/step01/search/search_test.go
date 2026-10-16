package search_test

import (
	"context"
	"runtime"
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
	backends := []search.Backend{
		backend(0, "fast"),
		backend(50*time.Millisecond, "slow1"),
		backend(50*time.Millisecond, "slow2"),
	}

	before := runtime.NumGoroutine()

	got, err := search.First(context.Background(), "gopher", backends)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if got != "fast" {
		t.Errorf("got %q, want %q", got, "fast")
	}

	// 遅いバックエンドの処理が終わるのに十分な時間を待つ
	time.Sleep(200 * time.Millisecond)

	after := runtime.NumGoroutine()
	t.Logf("ゴールーチン数: 実行前=%d, 実行後=%d", before, after)
}

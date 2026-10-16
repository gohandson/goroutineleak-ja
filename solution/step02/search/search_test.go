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

// waitNumGoroutine は、ゴールーチン数がwant以下になるまで、timeoutを上限に待つ。
// want以下になったらtrueを、ならなかったらfalseを返す。
func waitNumGoroutine(want int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if runtime.NumGoroutine() <= want {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return runtime.NumGoroutine() <= want
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

	// 遅いバックエンドの処理が終わり、ゴールーチン数が元に戻るのを待つ
	if !waitNumGoroutine(before, time.Second) {
		t.Errorf("ゴールーチンがリークしています: 実行前=%d, 実行後=%d", before, runtime.NumGoroutine())
	}
}

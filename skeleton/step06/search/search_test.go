package search_test

import (
	"context"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/gohandson/goroutineleak-ja/search"
)

// backend はd後にvを返すバックエンドを返す。
// ctxがキャンセルされたら、待つのをやめて戻る。
func backend(d time.Duration, v string) search.Backend {
	return func(ctx context.Context, query string) (string, error) {
		select {
		case <-time.After(d):
			return v, nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

func TestFirst(t *testing.T) {
	defer goleak.VerifyNone(t)

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

func TestFirst_cancel(t *testing.T) {
	defer goleak.VerifyNone(t)

	// ctxがキャンセルされるまで戻らないバックエンド
	cancelled := make(chan struct{}, 2)
	blocking := func(ctx context.Context, query string) (string, error) {
		<-ctx.Done()
		cancelled <- struct{}{}
		return "", ctx.Err()
	}

	backends := []search.Backend{
		backend(0, "fast"),
		blocking,
		blocking,
	}

	got, err := search.First(context.Background(), "gopher", backends)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if got != "fast" {
		t.Errorf("got %q, want %q", got, "fast")
	}

	// 残りのバックエンドにキャンセルが伝わっていること
	for range 2 {
		select {
		case <-cancelled:
		case <-time.After(time.Second):
			t.Fatal("バックエンドにキャンセルが伝わっていません")
		}
	}
}

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

// waitNumGoroutine は、ゴールーチン数がwant以下になるまで、timeoutを上限に待つ。
// want以下になったらtrueを、ならなかったらfalseを返す。
func waitNumGoroutine(want int, timeout time.Duration) bool {
	// TODO: timeoutになるまで10ミリ秒ごとに runtime.NumGoroutine 関数でゴールーチン数を調べ、
	// want以下になったらtrueを返す。timeoutになったらfalseを返す。
	return false
}

func TestFirst(t *testing.T) {
	backends := []search.Backend{
		backend(0, "fast"),
		backend(50*time.Millisecond, "slow1"),
		backend(50*time.Millisecond, "slow2"),
	}

	// TODO: First を呼び出す前のゴールーチン数を取得する

	got, err := search.First(context.Background(), "gopher", backends)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if got != "fast" {
		t.Errorf("got %q, want %q", got, "fast")
	}

	// TODO: waitNumGoroutine 関数で、ゴールーチン数が実行前に戻るのを1秒まで待つ。
	// 戻らなかったら、実行前と実行後のゴールーチン数を示して t.Errorf でテストを失敗させる。
}

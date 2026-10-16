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
	backends := []search.Backend{
		backend(0, "fast"),
		backend(50*time.Millisecond, "slow1"),
		backend(50*time.Millisecond, "slow2"),
	}

	// TODO: First を呼び出す前のゴールーチン数を runtime.NumGoroutine 関数で取得する

	got, err := search.First(context.Background(), "gopher", backends)
	if err != nil {
		t.Fatal("unexpected error:", err)
	}
	if got != "fast" {
		t.Errorf("got %q, want %q", got, "fast")
	}

	// TODO: 遅いバックエンドの処理が終わるのに十分な時間（200ミリ秒）待つ

	// TODO: ゴールーチン数を再び取得し、実行前と実行後の数を t.Logf で表示する
}

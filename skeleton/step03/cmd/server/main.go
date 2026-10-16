package main

import (
	"context"
	"log"
	"time"

	"github.com/gohandson/goroutineleak-ja/search"
)

func backend(d time.Duration, v string) search.Backend {
	return func(ctx context.Context, query string) (string, error) {
		time.Sleep(d)
		return v, nil
	}
}

func main() {
	// TODO: net/http/pprof パッケージを空白インポートする
	// TODO: pprofのエンドポイントを公開するため、別のゴールーチンで localhost:6060 を http.ListenAndServe 関数で待ち受ける

	backends := []search.Backend{
		backend(10*time.Millisecond, "fast"),
		backend(50*time.Millisecond, "slow1"),
		backend(50*time.Millisecond, "slow2"),
	}

	// 検索を繰り返し呼び出す
	for {
		if _, err := search.First(context.Background(), "gopher", backends); err != nil {
			log.Println(err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

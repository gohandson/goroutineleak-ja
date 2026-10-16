package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
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
	// pprofのエンドポイントを公開する
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

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

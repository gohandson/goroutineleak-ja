package main

import (
	"bufio"
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/gohandson/goroutineleak-ja/search"
)

func main() {
	// pprofのエンドポイントを公開する
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	// 何も書き込まれないパイプ
	r, w, err := os.Pipe()
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()

	backends := []search.Backend{
		func(ctx context.Context, q string) (string, error) {
			return "fast", nil
		},
		func(ctx context.Context, q string) (string, error) {
			// 書き込まれないので、ここで永遠にブロックする
			return bufio.NewReader(r).ReadString('\n')
		},
	}

	// 検索を繰り返し呼び出す
	for {
		if _, err := search.First(context.Background(), "gopher", backends); err != nil {
			log.Println(err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

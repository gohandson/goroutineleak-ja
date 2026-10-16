// Package globalsearch は、searchパッケージのFirstのチャネルをパッケージ変数にしたものである。
// リークの検出に失敗する例として使う。
package globalsearch

import "context"

// Backend は検索を行うバックエンドである。
type Backend func(ctx context.Context, query string) (string, error)

type result struct {
	val string
	err error
}

// ch をパッケージ変数にしたため、プログラムが動いている間は常に参照できる。
var ch = make(chan result)

// First はすべてのバックエンドに並行して問い合わせ、
// 最初に返ってきた結果を返す。
func First(ctx context.Context, query string, backends []Backend) (string, error) {
	for _, b := range backends {
		go func() {
			v, err := b(ctx, query)
			ch <- result{val: v, err: err}
		}()
	}

	r := <-ch
	return r.val, r.err
}

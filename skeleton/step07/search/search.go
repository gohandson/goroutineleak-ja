// Package search は複数のバックエンドに並行して問い合わせる検索処理を提供する。
package search

import "context"

// Backend は検索を行うバックエンドである。
type Backend func(ctx context.Context, query string) (string, error)

type result struct {
	val string
	err error
}

// First はすべてのバックエンドに並行して問い合わせ、
// 最初に返ってきた結果を返す。
func First(ctx context.Context, query string, backends []Backend) (string, error) {
	// TODO: ctx から context.WithCancel 関数でキャンセル可能なコンテキストを作り、
	// First が戻るときにキャンセルするように defer で cancel を呼び出す

	ch := make(chan result, len(backends))
	for _, b := range backends {
		go func() {
			v, err := b(ctx, query)
			ch <- result{val: v, err: err}
		}()
	}

	r := <-ch
	return r.val, r.err
}

// Package search は複数のバックエンドに並行して問い合わせる検索処理を提供する。
package search

import "context"

// Backend は検索を行うバックエンドである。
// 実装は、ctxがキャンセルされたら速やかに処理を中断して戻ることが期待される。
type Backend func(ctx context.Context, query string) (string, error)

type result struct {
	val string
	err error
}

// First はすべてのバックエンドに並行して問い合わせ、
// 最初に返ってきた結果を返す。
// 結果を返したあと、残りのバックエンドにはキャンセルを伝える。
func First(ctx context.Context, query string, backends []Backend) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// キャンセルされたバックエンドも結果を送るため、
	// 受信されなくても送信側がブロックしないようにバッファを持たせる。
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

# STEP 07: コンテキストでキャンセルしよう

## このステップで学ぶこと

STEP 06では、チャネルにバッファを持たせることでリークを直しました。
しかし、まだ改善できる点があります。
最初の結果が返ったあとも、遅いバックエンドへの問い合わせは、最後まで続いているのです。
結果が捨てられるだけの処理に、CPUやネットワークなどのリソースを使い続けるのは無駄です。

STEP 07では、`context.Context`を使って、残りのバックエンドの処理を中断できるようにします。
その過程で、次のことを学びます。

* `context.WithCancel`関数によるキャンセルの伝え方
* キャンセルだけではリークが防げない理由
* キャンセルを受けて戻るバックエンドの書き方

## 最初の結果が返ったらキャンセルする

`Backend`型は、`context.Context`を受け取ります。
`First`関数が戻るときに、このコンテキストをキャンセルすれば、残りのバックエンドに「もう結果はいらない」と伝えられます。

```go
func First(ctx context.Context, query string, backends []Backend) (string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan result, len(backends))
	// ...
}
```

`context.WithCancel`関数は、引数のコンテキストの子になるコンテキストと、それをキャンセルする`cancel`関数を返します。
`defer cancel()`としておくと、`First`関数がどのように戻っても、その時点で子のコンテキストがキャンセルされます。
呼び出し元から渡された`ctx`がキャンセルされた場合も、子のコンテキストはキャンセルされます。

`TODO`コメントに従って、`search/search.go`を修正してください。

## キャンセルを受けて戻るバックエンド

`First`関数がキャンセルしても、バックエンドの側がキャンセルに対応していなければ、処理は止まりません。
`context.Context`は、「キャンセルされたことを知らせる」しくみであって、ゴールーチンを強制的に止めるしくみではないからです。

STEP 06までのバックエンドは、`time.Sleep`関数で待っていたので、キャンセルを無視していました。
このステップのテストでは、`select`文で、時間が経つのと、キャンセルされるのを同時に待つようにしています。

```go
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
```

`ctx.Done()`は、コンテキストがキャンセルされると閉じられるチャネルです。
キャンセルされたら、`ctx.Err()`を返して、速やかに戻ります。
実際のバックエンドでは、`http.NewRequestWithContext`関数や、`database/sql`パッケージの`QueryContext`メソッドなど、コンテキストを受け取るAPIに`ctx`を渡すことで、同じことを実現します。

`cmd/server/main.go`のバックエンドも、同じようにキャンセルに対応させてあります。

## バッファは必要なままか

キャンセルされたバックエンドは、`ctx.Err()`を返して終了しますが、その結果も`ch`に送信しようとします。
つまり、`ch`のバッファがなければ、キャンセルされたバックエンドの結果の送信が、再びブロックしてしまいます。

`First`関数が受信する回数は1回だけなので、`ch`のバッファはこのステップでも必要です。
キャンセルは「無駄な処理を早く終わらせる」ためのもので、リークを防ぐのはバッファの役割です。
2つの修正は、役割が違い、どちらも必要です。

## 動かす

テストを実行します。
このステップのテストには、`TestFirst`に加えて、`TestFirst_cancel`が追加されています。
`TestFirst_cancel`は、`ctx`がキャンセルされるまで戻らないバックエンドを2つ渡して、`First`関数が戻ったあとに、そのバックエンドにキャンセルが伝わることを確かめます。

```
$ go test -race ./...
```

次のように、`ok`と表示されれば成功です。

```
ok  	github.com/gohandson/goroutineleak-ja/search	1.0s
```

サーバも、これまでと同じように起動して、プロファイルを確認できます。

```
$ go run ./cmd/server
```

```
$ curl 'http://localhost:6060/debug/pprof/goroutineleak?debug=1'
goroutineleak profile: total 0
```

## 考えてみよう

* `defer cancel()`を書かなかった場合、何が起きるか。`go vet`は何を報告するか
* バッファを`make(chan result)`に戻すと、`TestFirst`と`TestFirst_cancel`はそれぞれどうなるか（試してみましょう）
* バックエンドがキャンセルを無視する実装だったら、`First`関数をどう書けばリークを防げるか
* `First`関数は、最初に返ってきた結果がエラーでも、それを返してしまう。最初の成功した結果を返すようにするには、どう書き換えればよいか

## トラブルシューティング

### `TestFirst_cancel`が「バックエンドにキャンセルが伝わっていません」で失敗する

`First`関数の中で、`context.WithCancel`関数で作った子のコンテキストを、バックエンドに渡しているか確認してください。
`ctx, cancel := context.WithCancel(ctx)`のように、同じ名前の変数で上書きすると、以降の`b(ctx, query)`に子のコンテキストが渡されます。

### `go vet`が`lostcancel`を報告する

`context.WithCancel`関数が返す`cancel`関数は、必ず呼び出す必要があります。
`defer cancel()`が書かれているか確認してください。

## 参考文献

* [Go Concurrency Patterns: Context - The Go Blog](https://go.dev/blog/context)
* [context - pkg.go.dev](https://pkg.go.dev/context)
* [Goroutine Leak Profiles - The Go Blog](https://go.dev/blog/goroutine-leak-profiles)

---

- 問題: [`skeleton/step07`](../../skeleton/step07)
- 前のステップ: [STEP 06](../step06)
- 次のステップ: [STEP 08](../step08)

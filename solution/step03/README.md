# STEP 03: pprofで見つけよう

## このステップで学ぶこと

STEP 02では、goleakを使ってテストの中でリークを見つけました。
しかし、テストでは再現しなかったリークが、本番で動いているプログラムでだけ起きることもあります。

Go 1.27から、リークしたゴールーチンだけを報告するプロファイル、`goroutineleak`プロファイルが使えるようになりました。
STEP 03では、リークするサーバを動かし、このプロファイルでリークを見つけます。
その過程で、次のことを学びます。

* `net/http/pprof`パッケージによるpprofエンドポイントの公開
* `goroutineleak`プロファイルの取得と、`go tool pprof`での読み方
* 通常の`goroutine`プロファイルとの違い

## goroutineleakプロファイルの仕組み

これまでも、`goroutine`プロファイルを使えば、プログラムが動いている最中のゴールーチンの一覧を見ることができました。
しかし、`goroutine`プロファイルには、リークしているゴールーチンと、たまたま一時的にブロックしているだけのゴールーチンの区別がありません。
たとえば、アクセスが増えて、大量のリクエストが一時的にDBの応答待ちでブロックしているだけの場合でも、同じように並びます。

`goroutineleak`プロファイルは、ガベージコレクション（GC）の仕組みを使って、リークしているゴールーチンだけを見つけます。
考え方は次のとおりです。

* ブロックしていないゴールーチンは、動き続けられるので、リークしていない
* ブロックしているゴールーチンでも、それが待っているチャネルなどを、動き続けられるゴールーチンが参照していれば、いずれ待ちが解消される可能性があるので、リークしていない
* 上のどちらにも当てはまらないゴールーチン、つまり、待っているチャネルなどを動けるゴールーチンから誰も参照できなくなったものは、永遠に待ち続けるのでリークしている

誰も参照していないチャネルを待っているゴールーチンは、確実にリークしているといえます。
そのため、`goroutineleak`プロファイルでは、リークしていないものがリークとして報告される誤検出は、ほとんど起きないよう設計されています。

一方で、検出できるリークの種類は限られています。
どのようなリークが検出できて、どのようなリークが検出できないかは、STEP 06で試します。

## pprofのエンドポイントを公開する

`net/http/pprof`パッケージをインポートすると、`http.DefaultServeMux`に`/debug/pprof/`以下のハンドラが登録されます。
`goroutineleak`プロファイルも、Go 1.27以降では、このパッケージをインポートするだけで`/debug/pprof/goroutineleak`から取得できるようになります。

`cmd/server/main.go`は、`search.First`関数を100ミリ秒ごとに繰り返し呼び出すサーバです。
バックエンドは10ミリ秒、50ミリ秒、50ミリ秒かかる3つで、`First`関数を呼び出すたびに、遅い2つのゴールーチンがリークします。

`TODO`コメントに従って、次の処理を追加してください。

1. `net/http/pprof`パッケージを、`_`をつけて空白インポートする（`net/http`パッケージのインポートも必要です）
2. 別のゴールーチンで、`http.ListenAndServe`関数を使って`localhost:6060`で待ち受ける

`http.ListenAndServe`関数は、第2引数に`nil`を渡すと`http.DefaultServeMux`を使います。
エラーが返ってきたときは、`log.Println`関数で出力してください。

## 動かす

サーバを起動します。

```
$ go run ./cmd/server
```

起動したまま、別のターミナルを開いて、`goroutineleak`プロファイルを取得します。

```
$ curl -o leak.prof http://localhost:6060/debug/pprof/goroutineleak
```

取得したプロファイルは、`go tool pprof`で調べられます。
`-list`には正規表現を渡し、一致する関数のソースコードに、サンプル数を重ねて表示できます。

```
$ go tool pprof -list 'search.First' leak.prof
```

次のように、`ch <- result{val: v, err: err}`の行にサンプルが集まっていれば成功です。
`Total`の数は、取得した時点までにリークしたゴールーチン数なので、実行するたびに変わります。

```
Total: 54
ROUTINE ======================== github.com/gohandson/goroutineleak-ja/search.First.func1 in .../search/search.go
         0         54 (flat, cum)   100% of Total
         .          .     19:		go func() {
         .          .     20:			v, err := b(ctx, query)
         .         54     21:			ch <- result{val: v, err: err}
         .          .     22:		}()
```

この結果から、リークの原因が`ch`への送信であることが分かります。

テキスト形式で内容を確認したいときは、`debug=1`をつけて取得します。

```
$ curl 'http://localhost:6060/debug/pprof/goroutineleak?debug=1'
```

`goroutineleak profile: total 54`のような見出しに続いて、スタックトレースごとにゴールーチン数が並びます。

ブラウザで http://localhost:6060/debug/pprof/ を開いて、`goroutineleak`のリンクから確認することもできます。

## 通常のgoroutineプロファイルと比べる

同じサーバから、通常の`goroutine`プロファイルも取得して、比べてみましょう。

```
$ curl 'http://localhost:6060/debug/pprof/goroutine?debug=1' | head -20
```

`goroutine`プロファイルには、リークしたゴールーチンのほかに、pprofのHTTPサーバのゴールーチンや、`main`関数のゴールーチンなど、リークしていないゴールーチンも含まれます。
一方、`goroutineleak`プロファイルには、リークしたゴールーチンだけが含まれます。

しばらく待ってから、再度`goroutineleak`プロファイルを取得すると、`Total`の数が増えていることも確認できます。
`First`関数は100ミリ秒ごとに呼び出されて、そのたびに2つずつリークするためです。
このように、リークは時間が経つほど増えていきます。

## 考えてみよう

* どの行でリークしているか
* 時間が経つと数はどう変わるか
* 通常の`goroutine`プロファイルと何が違うか
* pprofのエンドポイントを本番環境で公開するときに、気をつけるべきことは何か

## トラブルシューティング

### ポート6060がすでに使われている

`address already in use`というログが出る場合は、前に起動したサーバが残っているか、別のプログラムがポート6060を使っています。
前のステップで起動したサーバを`Ctrl+C`で止め忘れていないか確認してください。

### goroutineleakプロファイルが空になる

起動した直後は、まだリークが発生していません。
数秒待ってから取得してください。
また、Goのバージョンが1.27以上であることを`go version`で確認してください。

### `go test ./...`が失敗する

STEP 02で追加した`goleak`のテストは、まだリークを直していないため、このステップでも失敗します。
リークの修正は、次のSTEP 04で行います。

### サーバが止まらない

`go run ./cmd/server`で起動したサーバは、`Ctrl+C`で止められます。

## 参考文献

* [Goroutine Leak Profiles - The Go Blog](https://go.dev/blog/goroutine-leak-profiles)
* [net/http/pprof - pkg.go.dev](https://pkg.go.dev/net/http/pprof)
* [runtime/pprof - pkg.go.dev](https://pkg.go.dev/runtime/pprof)
* [Profiling Go Programs - The Go Blog](https://go.dev/blog/pprof)
* [pprof - GitHub](https://github.com/google/pprof)

---

- 問題: [`skeleton/step03`](../../skeleton/step03)
- 前のステップ: [STEP 02](../step02)
- 次のステップ: [STEP 04](../step04)

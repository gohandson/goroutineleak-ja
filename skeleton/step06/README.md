# STEP 06: リークを直そう

## このステップで学ぶこと

STEP 03とSTEP 05で、`search.First`関数がゴールーチンをリークしていることを、goleakと`goroutineleak`プロファイルの両方で確認しました。
STEP 06では、このリークを直し、直ったことを同じ2つの方法で確かめます。
その過程で、次のことを学びます。

* バッファ付きチャネルによるリークの防ぎ方
* 修正後の確認に、テストとプロファイルを組み合わせる方法

## 原因をおさらいする

`First`関数のリークは、次の2つの条件が重なって起きていました。

* `ch`がバッファなしのチャネルなので、送信は受信する側が現れるまでブロックする
* `First`関数は1回しか受信せずに戻るので、2つ目以降の送信を受信する側がいなくなる

どちらかの条件を崩せば、リークは起きなくなります。
`First`関数が結果を受信する回数を増やすという方法もありますが、それでは、結局すべてのバックエンドを待つことになり、「最初に返ってきた結果を返す」という目的を果たせません。

そこで、チャネルにバッファを持たせます。

## 修正する

`search/search.go`の`First`関数の`ch`を、バックエンドの数と同じ大きさのバッファを持つチャネルにします。

```go
ch := make(chan result, len(backends))
```

バッファ付きのチャネルへの送信は、バッファに空きがあればブロックしません。
バッファの大きさがバックエンドの数と同じなので、すべてのゴールーチンが、`First`関数が戻ったかどうかにかかわらず、ブロックせずに送信を終えて終了できます。
バッファに残った結果は、`ch`がどこからも参照されなくなったときに、GCによって回収されます。

`TODO`コメントに従って、`search/search.go`を修正してください。

## 確かめる

### テストで確かめる

STEP 03で追加したテストを実行します。

```
$ go test -race ./...
```

次のように、`ok`と表示されれば、goleakがリークを検出しなかったということです。

```
ok  	github.com/gohandson/goroutineleak-ja/search	1.0s
```

### プロファイルで確かめる

サーバを起動して、しばらく動かしたあとに、`goroutineleak`プロファイルを取得します。

```
$ go run ./cmd/server
```

別のターミナルで、次のように取得します。

```
$ curl 'http://localhost:6060/debug/pprof/goroutineleak?debug=1'
```

次のように、リークしたゴールーチンが0であれば成功です。

```
goroutineleak profile: total 0
```

STEP 05では、数秒でリークしたゴールーチンが数十個に増えていたはずです。
同じ時間が経っても0のままであることを確かめてください。

通常の`goroutine`プロファイルも見てみましょう。

```
$ curl 'http://localhost:6060/debug/pprof/goroutine?debug=1' | head -1
```

こちらも、STEP 05のときと比べて、ゴールーチン数が増え続けないことが分かります。

## 考えてみよう

* バッファの大きさを`1`にすると、リークはどうなるか。3つのバックエンドで考えてみましょう（試してみて、goleakの結果を確かめてもよいでしょう）
* バッファの大きさを`len(backends)`より大きくすると、何が変わるか
* `First`関数が返したあと、バッファに残った結果はどうなるか

## トラブルシューティング

### テストがまだ失敗する

`search/search.go`の`First`関数のチャネルが、`make(chan result, len(backends))`になっているか確認してください。
また、`search_test.go`の`defer goleak.VerifyNone(t)`を消して成功させていないか確認してください。

### サーバのプロファイルでまだリークが報告される

前のステップで起動したサーバが、止まらずに残っていないか確認してください。
`go run ./cmd/server`し直す前に、`Ctrl+C`で止めておく必要があります。

## 参考文献

* [Goroutine Leak Profiles - The Go Blog](https://go.dev/blog/goroutine-leak-profiles)
* [The Go Programming Language Specification - Channel types](https://go.dev/ref/spec#Channel_types)
* [Effective Go - Channels](https://go.dev/doc/effective_go#channels)

---

- 解答例: [`solution/step06`](../../solution/step06)
- 前のステップ: [STEP 05](../step05)
- 次のステップ: [STEP 07](../step07)

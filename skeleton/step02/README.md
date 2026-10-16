# STEP 02: goleakで見つけよう

## このステップで学ぶこと

STEP 01では、`runtime.NumGoroutine`関数でゴールーチン数を比べて、リークを確認しました。
しかしこの方法では、リークしたことは分かっても、どのゴールーチンがどこでリークしているのかまでは分かりません。
また、他のゴールーチンの増減に影響されるため、テストの自動チェックには向きません。

STEP 02では、[goleak](https://github.com/uber-go/goleak)というライブラリを使って、テストの中でゴールーチンリークを検出します。
その過程で、次のことを学びます。

* goleakの使い方
* goleakが検出する仕組みと、検出できるタイミング
* テストで見つけられるリークと、本番で動くプログラムのリークの違い

## goleakを取得する

goleakは、テストが終わったあとに終了していないゴールーチンが残っていないかを調べるライブラリです。
Uberが公開しています。

まず、goleakを取得します。

```
$ go get go.uber.org/goleak
```

`go.mod`に`go.uber.org/goleak`が追加されます。

## テストにgoleakを追加する

`search/search_test.go`に`go.uber.org/goleak`のインポートを追加します。
そして、`TestFirst`関数の先頭に、次の1行を追加します。

```go
defer goleak.VerifyNone(t)
```

`goleak.VerifyNone`関数は、呼び出された時点で動いているゴールーチンを調べ、テストの実行中に起動されて、まだ終了していないゴールーチンがあれば、テストを失敗させます。
`defer`で呼び出しているため、`TestFirst`関数が終わるときに検査が行われます。

なお、STEP 01で追加した`runtime.NumGoroutine`関数による確認は、このステップでは使いません。
このステップの`TestFirst`関数は、ゴールーチン数の確認の代わりに`goleak.VerifyNone`関数を使います。

## 動かす

次のコマンドでテストを実行します。

```
$ go test -race ./...
```

テストが失敗し、次のように、残っているゴールーチンの状態とスタックトレースが表示されれば成功です。

```
--- FAIL: TestFirst (0.44s)
    search_test.go:36: found unexpected goroutines:
        [Goroutine 38 in state chan send, with github.com/gohandson/goroutineleak-ja/search.First.func1 on top of the stack:
        github.com/gohandson/goroutineleak-ja/search.First.func1()
        	/.../search/search.go:21 +0x96
        created by github.com/gohandson/goroutineleak-ja/search.First in goroutine 36
        	/.../search/search.go:19 +0x6b
         Goroutine 39 in state chan send, with ...
        ]
FAIL
```

表示された内容から、次のことが分かります。

* `chan send`の状態、つまりチャネルへの送信でブロックしている
* ブロックしているのは`search.First`関数の中で起動された関数である
* `search.go`の21行目（`ch <- result{val: v, err: err}`）でブロックしている
* 2つのゴールーチンがリークしている

このステップの解答例では、テストが失敗する状態のままにしています。
リークを直すのはSTEP 04です。

## goleakの仕組み

`goleak.VerifyNone`関数は、`runtime.Stack`関数で全ゴールーチンのスタックトレースを取得し、テストに関係のないゴールーチン（テストを動かすためのゴールーチンなど）を除いたものを、リークしている疑いのあるゴールーチンとして報告します。

検査を行うのは、`VerifyNone`関数が呼び出されたとき、つまり、このテストでは`TestFirst`関数が終わるときです。
ゴールーチンはすぐには終了しないことがあるため、goleakは残っているゴールーチンがなくなるまで、間隔を空けて何度か調べ直します。
それでも残っているものを、リークとして報告します。
テストが失敗するまでに0.4秒ほどかかったのは、このリトライを行っているためです。

この仕組みには、次の特徴があります。

* リークしたゴールーチンがブロックしている理由は問わない（チャネルでも、ロックでも、I/Oでも、スリープでも、終了していなければ報告される）
* 終了を待つだけなので、本当にリークしているのか、単に終了が遅いだけなのかは区別できない
* テストの終わりという決まったタイミングで調べるため、テストが終わらないと検査できない

## 本番で動くプログラムにも使えるか

goleakは、テストコードの中で使うためのライブラリです。
「このテストが終わった時点では、ゴールーチンは1つも残っていないはずだ」という前提で検査するからです。

一方、本番で動いているサーバでは、リクエストを処理するゴールーチンなど、終了していなくて当たり前のゴールーチンがたくさんあります。
そのため、「終了していないゴールーチンがある」ことだけでは、リークかどうかを判断できません。

また、テストでは再現しなかったリークが、本番の特定の入力やタイミングでだけ起きることもあります。
本番で動いているプログラムのリークを調べるための方法を、次のSTEP 03で学びます。

## 考えてみよう

* goleakはいつ、何をチェックしているか
* 本番環境で動いているプログラムには使えるか
* 遅いバックエンドの待ち時間を長くすると、テストの結果はどう変わるか

## トラブルシューティング

### `goleak`が見つからないというエラーが出る

`go get go.uber.org/goleak`を、`go.mod`のあるディレクトリ（`skeleton/step02`）で実行してください。
インポートのパスは`go.uber.org/goleak`です。

### テストが成功してしまう

`defer goleak.VerifyNone(t)`が`TestFirst`関数の先頭に書かれているか確認してください。
また、`search/search.go`を修正していないか確認してください。
このステップでは、`search/search.go`はSTEP 01から変えません。

## 参考文献

* [goleak - GitHub](https://github.com/uber-go/goleak)
* [goleak - pkg.go.dev](https://pkg.go.dev/go.uber.org/goleak)
* [Goroutine Leak Profiles - The Go Blog](https://go.dev/blog/goroutine-leak-profiles)

---

- 解答例: [`solution/step02`](../../solution/step02)
- 前のステップ: [STEP 01](../step01)
- 次のステップ: [STEP 03](../step03)

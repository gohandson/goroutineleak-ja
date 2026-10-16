# STEP 08: 見つからないリークを知ろう

## このステップで学ぶこと

goleakも`goroutineleak`プロファイルも、万能ではありません。
特に`goroutineleak`プロファイルは、検出できるリークの種類が限られています。
STEP 08では、さまざまなリークのパターンを実際に起こして、それぞれのツールで検出できるか、できないかを確かめます。
その過程で、次のことを学びます。

* `goroutineleak`プロファイルが検出できるリークと、できないリーク
* 検出できなくなる理由
* goleakと`goroutineleak`プロファイルの使い分け

## goroutineleakプロファイルの限界

`goroutineleak`プロファイルが検出できるのは、次のいずれかでブロックしているゴールーチンです。

* チャネルの送信と受信（`nil`チャネルを含む）
* `select`文（`default`がないもの。`case`がない`select {}`を含む）
* `sync`パッケージの`Mutex`・`RWMutex`・`WaitGroup`・`Cond`

次のような場合は、検出できません。

* ファイルやネットワークのI/O、システムコールでブロックしている
* ブロックしているのではなく、`time.Sleep`関数などで待っている
* ブロックしている原因のチャネルなどを、グローバル変数や、動き続けているゴールーチンが参照している

最後のものは、STEP 05で説明した仕組みから理解できます。
GCは、動き続けられるゴールーチンが参照しているチャネルを待っているゴールーチンは、いずれ待ちが解消されるかもしれないと考えます。
グローバル変数は、プログラムが動いている間は常に参照できるので、そのチャネルを待っているゴールーチンは、実際には永遠に待ち続けるとしても、リークとは判定されません。

## リークのパターン集

`leaks`パッケージには、ゴールーチンを1つ起動して、そのゴールーチンを二度と動けなくする関数が、10個用意されています。

* `LocalChanSend`: 誰も受信しないチャネルへの送信
* `LocalChanRecv`: 誰も送信しないチャネルからの受信
* `RangeNotClosed`: `close`されないチャネルを`range`で受信し続ける
* `NilChan`: `nil`チャネルからの受信
* `EmptySelect`: `case`のない`select {}`
* `WaitGroup`: `Done`が呼ばれない`sync.WaitGroup`の`Wait`
* `Mutex`: `Unlock`されない`sync.Mutex`の`Lock`
* `GlobalChan`: グローバル変数のチャネルへの送信
* `Sleep`: 1時間の`time.Sleep`
* `PipeRead`: 書き込まれない`os.Pipe`の読み込み

`leaks/leaks_test.go`のテストは、それぞれの関数を呼び出して、`goroutineleak`プロファイルにリークが現れるか、goleakがリークを検出するかを確かめます。

`goroutineleak`プロファイルは、`runtime/pprof`パッケージの`Lookup`関数からも取得できます。
テストでは、`runtime.GC`関数を呼び出してから、`pprof.Lookup("goroutineleak").WriteTo`でプロファイルを書き出し、先頭行の`goroutineleak profile: total N`から、リークしたゴールーチンの数を読み取っています。

`TODO`コメントに従って、各ケースの`wantProfile`を、`goroutineleak`プロファイルで検出できると思うなら`true`、できないと思うなら`false`にしてください。
STEP 05の説明と、ここまでの内容をもとに予想してみましょう。

## 動かす

テストを実行します。
10個のパターンを順番に試すので、実行には数秒かかります。

```
$ go test -race -v ./leaks
```

予想が外れていると、そのケースのテストが失敗します。
すべてのケースが`PASS`するまで、`wantProfile`を直してみてください。

## 予想と結果

答え合わせです。

* `goroutineleak`プロファイルで検出できる: `LocalChanSend`、`LocalChanRecv`、`RangeNotClosed`、`NilChan`、`EmptySelect`、`WaitGroup`、`Mutex`
* `goroutineleak`プロファイルでは検出できない: `GlobalChan`、`Sleep`、`PipeRead`
* goleakは、どのパターンも検出できる

goleakは、ブロックしている理由に関係なく、テストの終了後に残っているゴールーチンをすべて報告するため、10個すべてを検出できます。
`goroutineleak`プロファイルは、確実にリークといえるものだけを報告するため、検出できないパターンがあります。
その代わり、本番で動いているプログラムでも、誤検出をほとんど気にせずに使えます。

2つのツールは補い合う関係にあります。

* テストでは、goleakで、ブロックの理由を問わずに残ったゴールーチンを見つける
* 本番では、`goroutineleak`プロファイルで、確実にリークといえるものを見つける
* `goroutineleak`プロファイルで見つからなくても、通常の`goroutine`プロファイルでゴールーチン数が増え続けていないかを、あわせて見る

## 見つからないリークを動かしてみる

`search.First`関数に似た例で、検出できないリークを、サーバとして動かしてみましょう。

### チャネルをグローバル変数にした場合

`internal/globalsearch/search.go`は、`First`関数の`ch`を、パッケージ変数にしたものです。

```go
// ch をパッケージ変数にしたため、プログラムが動いている間は常に参照できる。
var ch = make(chan result)
```

`cmd/global/main.go`は、このパッケージの`First`関数を繰り返し呼び出すサーバです。

```
$ go run ./cmd/global
```

別のターミナルで、2つのプロファイルを取得して比べます。

```
$ curl 'http://localhost:6060/debug/pprof/goroutineleak?debug=1' | head -1
goroutineleak profile: total 0

$ curl 'http://localhost:6060/debug/pprof/goroutine?debug=1' | head -1
goroutine profile: total 64
```

`goroutineleak`プロファイルでは0なのに、`goroutine`プロファイルのゴールーチン数は増え続けています。
`ch`にはグローバル変数からいつでもアクセスできるので、GCは、いつか誰かが`ch`から受信するかもしれないと考え、リークとは判定しないのです。

### I/Oでブロックする場合

`cmd/io/main.go`は、2つ目のバックエンドが、何も書き込まれないパイプから読み込む例です。

```go
r, w, err := os.Pipe() // 書き込まれないパイプ
```

```
$ go run ./cmd/io
```

```
$ curl 'http://localhost:6060/debug/pprof/goroutineleak?debug=1' | head -1
goroutineleak profile: total 0

$ curl 'http://localhost:6060/debug/pprof/goroutine?debug=1' | head -1
goroutine profile: total 34
```

こちらも、リークしているのは確かですが、ブロックしている理由がI/Oなので、`goroutineleak`プロファイルには現れません。
`goroutine`プロファイルを見ると、`IO wait`状態のゴールーチンが増え続けていることが分かります。

```
$ curl 'http://localhost:6060/debug/pprof/goroutine?debug=2' | grep -A3 'IO wait' | head
```

## 考えてみよう

* 次のようにコードを変えると、goleakと`goroutineleak`プロファイルの結果はどう変わるか
  * `ch`をパッケージ変数にする
  * バックエンドが`time.Sleep`ではなく、標準入力の読み込みでブロックする
* グローバル変数にしたチャネルのリークを、見つけやすくするには、どうすればよいか
* `goroutineleak`プロファイルで何も見つからなかったとき、リークはないと言い切れるか

## トラブルシューティング

### `PipeRead`のケースで、goleakが検出しない

`os.Pipe`が返す書き込み側の`*os.File`が、GCで回収されると、ファイナライザによって閉じられます。
すると、読み込み側が`EOF`で戻ってしまい、ゴールーチンは終了します。
`leaks`パッケージでは、`pipeWriters`変数に書き込み側を保持して、これを防いでいます。

### テストが不安定になる

ゴールーチンがブロックして、リークとして判定されるまでには、少し時間がかかります。
テストでは、500ミリ秒まで、10ミリ秒ごとにプロファイルを取得し直しています。
マシンが非常に遅い環境では、この時間を長くしてみてください。

### ポート6060がすでに使われている

前のステップのサーバを止め忘れていないか確認してください。
`cmd/global`と`cmd/io`も、同じポート6060を使うので、同時には起動できません。

## 参考文献

* [Goroutine Leak Profiles - The Go Blog](https://go.dev/blog/goroutine-leak-profiles)
* [goleak - GitHub](https://github.com/uber-go/goleak)
* [runtime/pprof - pkg.go.dev](https://pkg.go.dev/runtime/pprof)
* [os.Pipe - pkg.go.dev](https://pkg.go.dev/os#Pipe)

---

- 解答例: [`solution/step08`](../../solution/step08)
- 前のステップ: [STEP 07](../step07)
- 次のステップ: [STEP 09](../step09)

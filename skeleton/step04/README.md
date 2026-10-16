# STEP 04: goleakの仕組みを調べよう

## このステップで学ぶこと

STEP 03では、`goleak.VerifyNone`関数を1行書くだけで、リークしたゴールーチンが見つかりました。
このステップでは、goleakの中で何が行われているのかを、自分で実装しながら調べます。
その過程で、次のことを学びます。

* ゴールーチンのスタックトレースを取得する方法
* スタックトレースの読み方と、ゴールーチンの状態
* goleakがリークを見つける仕組み

## スタックトレースを見る

Goのプログラムでは、`runtime.Stack`関数で、ゴールーチンのスタックトレースを取得できます。
第2引数に`true`を渡すと、すべてのゴールーチンのスタックトレースが取得できます。

```go
buf := make([]byte, 1<<16)
n := runtime.Stack(buf, true)
fmt.Printf("%s", buf[:n])
```

`cmd/stack/main.go`は、いろいろな理由でブロックするゴールーチンを起動して、すべてのゴールーチンのスタックトレースを表示するプログラムです。
実行してみましょう。

```
$ go run ./cmd/stack
```

次のように表示されます（ゴールーチンのIDやアドレス、パスは環境によって異なります）。

```
goroutine 1 [running]:
main.main()
	/.../cmd/stack/main.go:40 +0x16e

goroutine 19 [chan send]:
main.main.func1()
	/.../cmd/stack/main.go:14 +0x1d
created by main.main in goroutine 1
	/.../cmd/stack/main.go:13 +0xa5

goroutine 20 [chan receive]:
main.main.func2()
	/.../cmd/stack/main.go:20 +0x1d
created by main.main in goroutine 1
	/.../cmd/stack/main.go:19 +0xc8

goroutine 21 [sleep]:
time.Sleep(0x34630b8a000)
	/.../src/runtime/time.go:368 +0x165
main.main.func3()
	/.../cmd/stack/main.go:25 +0x1d
created by main.main in goroutine 1
	/.../cmd/stack/main.go:24 +0xc8

goroutine 22 [sync.WaitGroup.Wait]:
sync.runtime_SemacquireWaitGroup(0x0?, 0x0?)
	/.../src/runtime/sema.go:114 +0x2e
sync.(*WaitGroup).Wait(0x3c877a316120)
	/.../src/sync/waitgroup.go:206 +0x85
main.main.func4()
	/.../cmd/stack/main.go:32 +0x17
created by main.main in goroutine 1
	/.../cmd/stack/main.go:31 +0x136
```

ゴールーチンごとに、空行で区切られたブロックになっています。
それぞれのブロックは、次のように読めます。

* 1行目: `goroutine ID [状態]:`という見出し。IDはゴールーチンを区別するための番号で、状態はそのゴールーチンがいま何をしているかを表す
* 2行目以降: スタックトレース。いちばん上が、いま実行している（または止まっている）関数で、その呼び出し元が順に下に並ぶ
* 最後の2行: `created by`で始まる、そのゴールーチンを起動した場所

状態には、たとえば次のようなものがあります。

* `running`: 実行中。スタックトレースを取得しているゴールーチン自身は、この状態になる
* `runnable`: 実行できる状態で、CPUが空くのを待っている
* `chan send`: チャネルへの送信でブロックしている
* `chan receive`: チャネルからの受信でブロックしている
* `select (no cases)`: `case`のない`select {}`でブロックしている
* `sync.WaitGroup.Wait`: `WaitGroup`の`Wait`でブロックしている
* `sleep`: `time.Sleep`関数でスリープしている
* `IO wait`: ファイルやネットワークのI/Oの完了を待っている

STEP 03で表示された`Goroutine 38 in state chan send, with ... on top of the stack`は、このスタックトレースから読み取った情報だったのです。

この書式は、`runtime/pprof`パッケージの`Lookup("goroutine")`で`debug=2`を指定して書き出した場合や、`/debug/pprof/goroutine?debug=2`、プログラムがパニックしたときの出力でも同じです。

## goleakの仕組み

goleakの`VerifyNone`関数は、おおまかに次のことをしています。

1. `runtime.Stack`関数で、すべてのゴールーチンのスタックトレースを取得する
2. 空行で区切って、ゴールーチンごとに分ける
3. 見出しの行から、IDや状態を読み取る
4. 検査を呼び出したゴールーチンや、テストを動かすためのゴールーチンなど、リークとは関係のないものを除く
5. 残ったゴールーチンがあれば、少し待ってから、もう一度1から調べ直す
6. 何度か繰り返しても残っているゴールーチンがあれば、リークとして報告する

STEP 03で、テストが失敗するまでに0.4秒ほどかかったのは、5の調べ直しをしていたためです。

つまり、goleakは、ゴールーチンが何を待っているのかを理解しているわけではなく、「終了していないゴールーチンが残っていないか」を、スタックトレースから調べているだけです。
この点が、STEP 05で扱う`goroutineleak`プロファイルとの大きな違いです。

## mygoleakを実装する

`internal/mygoleak`パッケージに、goleakの仕組みを真似た`mygoleak`パッケージがあります。
このうち、次の3か所が`TODO`になっているので、実装してください。

* `stacks`関数: `runtime.Stack`関数で、スタックトレースを文字列で取得する。バッファが小さくて収まらなかったときは、バッファを大きくして取得し直す
* `parse`関数: 取得した文字列を空行で区切り、ゴールーチンごとに`Goroutine`型の値を作る。見出しの行は、用意された`header`変数の正規表現で読み取れる。先頭の関数名は、スタックトレースの2行目の、引数の`(`より前の部分である
* `Find`関数: リークの疑いがあるゴールーチンから、`Find`を呼び出したゴールーチン自身、`IgnoreCurrent`で除外されたもの、テストを動かしているもの（先頭の関数が`testing.(*T).Run`のもの）を除く

`Find`を呼び出したゴールーチン自身のIDは、用意されている`currentID`関数で取得できます。

`IgnoreCurrent`関数は、goleakにある同名のオプションと同じ考え方で、呼び出した時点で存在するゴールーチンを、あとの検査の対象から外します。
テストの実行前から動いているゴールーチンを、リークと間違えないようにするために使います。

## 動かす

テストを実行します。

```
$ go test -race -v ./internal/...
```

`TestFind`では、チャネルの送信、受信、`select {}`、スリープ、`WaitGroup`の5つのリークを起こして、それぞれのゴールーチンの状態が正しく読み取れることを確かめます。
続く`TestFind_noLeak`と`TestVerifyNone`も含めて、すべて`PASS`すれば成功です。

```
--- PASS: TestFind (0.00s)
    --- PASS: TestFind/chan_send (0.00s)
    --- PASS: TestFind/chan_receive (0.00s)
    --- PASS: TestFind/empty_select (0.00s)
    --- PASS: TestFind/sleep (0.00s)
    --- PASS: TestFind/wait_group (0.00s)
--- PASS: TestFind_noLeak (0.00s)
--- PASS: TestVerifyNone (1.45s)
```

## searchパッケージに使ってみる

作った`mygoleak`を、`search.First`関数のリークの検出に使ってみましょう。
`search/search_test.go`のgoleakを、`mygoleak`に一時的に差し替えます。

```go
import "github.com/gohandson/goroutineleak-ja/internal/mygoleak"

func TestFirst(t *testing.T) {
	defer mygoleak.VerifyNone(t)
	// ...
}
```

```
$ go test -race ./search
```

次のように、STEP 03のときと同じゴールーチンが報告されれば成功です。

```
--- FAIL: TestFirst (1.44s)
    search_test.go:36: found 2 unexpected goroutines:
        goroutine 9 [chan send]:
        github.com/gohandson/goroutineleak-ja/search.First.func1()
        	/.../search/search.go:21 +0x96
        created by github.com/gohandson/goroutineleak-ja/search.First in goroutine 7
        	/.../search/search.go:19 +0x6b
        ...
```

確認できたら、`search/search_test.go`は元の`goleak`を使う状態に戻しておいてください。

## 本物のgoleakとの違い

`mygoleak`は、仕組みを理解するための簡易版です。
本物のgoleakは、これに加えて次のようなことをしています。

* Goのランタイムや、シグナルを待つゴールーチンなど、ユーザのコードとは関係なく存在するゴールーチンを、あらかじめ除く
* 特定の関数のゴールーチンを除くオプションを持つ
* `TestMain`関数から使い、パッケージ内のすべてのテストが終わったあとに、まとめて検査できる（`goleak.VerifyTestMain`関数）

## 考えてみよう

* `VerifyNone`関数が、1回で諦めずに調べ直すのはなぜか。調べ直さないと、どのような誤検出が起きるか
* `sleep`状態のゴールーチンは、リークだといえるか。goleakはこれをどう扱うか
* スタックトレースだけを見る方法では、ゴールーチンが「一時的にブロックしている」のか「永遠にブロックしている」のかを区別できないのはなぜか

## トラブルシューティング

### `TestFind`が失敗する

`parse`関数で、ゴールーチンどうしを区切る空行（`"\n\n"`）で分けているか確認してください。
また、見出しの行の状態が`chan send, 3 minutes`のように、カンマ以降に待ち時間がつくことがあります。
`header`変数の正規表現は、カンマより前だけを状態として取り出します。

### `Find`がテストを動かしているゴールーチンまで報告する

`testing.(*T).Run`で待っているゴールーチンを除いているか確認してください。
`Goroutine`型の`TopFunc`フィールドに、スタックの先頭の関数名が入ります。

### `go test ./...`が失敗する

`./search`のテストは、リークを直す前なので失敗します。
リークの修正は、STEP 06で行います。
このステップでは、`go test -race ./internal/...`で`mygoleak`のテストだけを実行してください。

## 参考文献

* [runtime.Stack - pkg.go.dev](https://pkg.go.dev/runtime#Stack)
* [goleak - GitHub](https://github.com/uber-go/goleak)
* [Goroutine Leak Profiles - The Go Blog](https://go.dev/blog/goroutine-leak-profiles)

---

- 解答例: [`solution/step04`](../../solution/step04)
- 前のステップ: [STEP 03](../step03)
- 次のステップ: [STEP 05](../step05)

# STEP 09: 本番で監視しよう

## このステップで学ぶこと

STEP 05では、手動で`curl`して`goroutineleak`プロファイルを取得しました。
しかし本番では、リークがいつ起きるか分からないので、一度取得するだけでは足りません。
プロファイルを定期的に取得し、リークの有無を継続的に監視する必要があります。

STEP 09では、継続的プロファイラであるDatadogを例に、その考え方を学びます。
このステップにはコードを書く演習はありません。
その過程で、次のことを学びます。

* 継続的プロファイリングでリークを監視する考え方
* dd-trace-goで`goroutineleak`プロファイルを有効にする方法
* `goroutineleak`プロファイルを定期的に取得するときの注意点

## 継続的プロファイリング

継続的プロファイリングとは、本番で動いているプログラムから、CPUやヒープなどのプロファイルを、一定の間隔で自動的に取得して、集めて見られるようにしておくしくみです。
リークは、時間が経つほど悪化するため、定期的に取得することで次のことが分かります。

* いつからリークが始まったか
* どのデプロイが原因か
* リークしたゴールーチンの数が、どのように増えているか

`goroutineleak`プロファイルは、本番でも、誤検出をほとんど気にせずに使えるよう設計されています。
そのため、継続的プロファイラで取得して監視するのに向いています。

## DatadogのGoroutineLeakProfile

Datadogの継続的プロファイラ（dd-trace-goの`profiler`パッケージ）にも、`GoroutineLeakProfile`というプロファイルタイプが追加されています。
次のように、`profiler.WithProfileTypes`関数に指定して有効にします。

```go
import "github.com/DataDog/dd-trace-go/v2/profiler"

func main() {
	err := profiler.Start(
		profiler.WithProfileTypes(
			profiler.CPUProfile,
			profiler.HeapProfile,
			profiler.GoroutineLeakProfile,
		),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer profiler.Stop()

	// ...
}
```

注意点は次のとおりです。

* デフォルトで有効なのはCPUとヒープなどに限られるので、`GoroutineLeakProfile`は、`WithProfileTypes`関数で明示的に指定する必要がある
* `WithProfileTypes`関数に指定したプロファイルだけが有効になるので、これまで有効にしていたプロファイルも、あわせて指定する
* Go 1.27以降が必要（Go 1.26でも、`GOEXPERIMENT=goroutineleakprofile`をつけてビルドすれば取得できる）
* 本書の執筆時点では、`GoroutineLeakProfile`はdd-trace-goのv2.11.0から利用できる。それより前のバージョン（v2.10.1など）には含まれていない

## 取得する頻度

`goroutineleak`プロファイルの取得は、通常のGCよりもコストがかかる場合があります。
一方で、リークは一度起きたら、その後も観測できるため、毎秒のように取得する必要はありません。
Goのブログでは、たとえば4時間に1回のように、頻度を下げても、リークを検出する能力はほとんど落ちないと説明されています。

## ここまでの使い分け

このハンズオンで学んだツールを、まとめます。

* `go test`とgoleak: テストの中で、ブロックの理由を問わず、残ったゴールーチンを見つける
* `goroutineleak`プロファイルを手動で取得: 動いているサーバの、確実にリークといえるゴールーチンを、その場で調べる
* 継続的プロファイラ: `goroutineleak`プロファイルを定期的に取得し、リークの発生や増加を継続的に監視する
* `goroutine`プロファイル: `goroutineleak`プロファイルでは見つからない、I/Oやグローバル変数に由来するリークを、ゴールーチン数の推移から疑う

## 考えてみよう

* 自分のプロダクトで、`goroutineleak`プロファイルを定期的に取得するとしたら、どのくらいの頻度にするか
* リークを検知したときに、通知するしきい値は、どう決めるか
* リークを見つけた後、修正の確認にはどのツールを使うか

## 参考文献

* [Goroutine Leak Profiles - The Go Blog](https://go.dev/blog/goroutine-leak-profiles)
* [Datadog Continuous Profiler - Datadog Docs](https://docs.datadoghq.com/profiler/)
* [dd-trace-go profiler - pkg.go.dev](https://pkg.go.dev/github.com/DataDog/dd-trace-go/v2/profiler)

---

- 問題: [`skeleton/step09`](../../skeleton/step09)
- 前のステップ: [STEP 08](../step08)

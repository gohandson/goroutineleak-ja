// Package mygoleak は、goleakの仕組みを真似た簡易的なゴールーチンリーク検出器である。
// 学習用のため、goleakが持つ細かい機能は実装していない。
package mygoleak

import (
	"regexp"
	"strings"
	"testing"
	"time"
)

// Goroutine はスタックトレースから読み取ったゴールーチンの情報である。
type Goroutine struct {
	// ID はゴールーチンのIDである。
	ID int
	// State はゴールーチンの状態である。"chan send"や"sleep"など。
	State string
	// TopFunc はスタックの先頭、つまり実行中（または停止中）の関数名である。
	TopFunc string
	// Stack はスタックトレースの全文である。
	Stack string
}

// stacks は、現在のゴールーチンのスタックトレースを返す。
// all が true のときは、すべてのゴールーチンのスタックトレースを返す。
func stacks(all bool) string {
	// TODO: runtime.Stack関数でスタックトレースを取得して文字列で返す。
	// スタックトレースがバッファに収まらなかったときは、バッファを大きくしてやり直す。
	return ""
}

// header は"goroutine 7 [chan send, 3 minutes]:"のような見出しの行にマッチする。
// 1つ目のグループがID、2つ目のグループが状態である。
var header = regexp.MustCompile(`^goroutine (\d+) \[([^\],]+)`)

// parse は、runtime.Stack関数が返す文字列を、ゴールーチンごとに分解する。
func parse(s string) []Goroutine {
	// TODO: sを空行で区切ってゴールーチンごとのブロックに分け、
	// 見出しの行からID・状態を、2行目から先頭の関数名を読み取ってGoroutineを作る。
	// 見出しの行は、header変数の正規表現で読み取れる。
	return nil
}

// currentID は、この関数を呼び出したゴールーチンのIDを返す。
func currentID() int {
	return parse(stacks(false))[0].ID
}

type options struct {
	ignoreIDs map[int]bool
}

// Option は検出の挙動を変えるオプションである。
type Option func(*options)

// IgnoreCurrent は、IgnoreCurrent関数を呼び出した時点で存在するゴールーチンを、
// 検出の対象から外す。
func IgnoreCurrent() Option {
	ids := make(map[int]bool)
	for _, g := range parse(stacks(true)) {
		ids[g.ID] = true
	}
	return func(o *options) {
		for id := range ids {
			o.ignoreIDs[id] = true
		}
	}
}

// Find は、いま存在するゴールーチンのうち、リークの疑いがあるものを返す。
// 疑いがあるのは、次のどれにも当てはまらないゴールーチンである。
//
//   - Findを呼び出したゴールーチン
//   - IgnoreCurrentで除外されたゴールーチン
//   - テストを動かしているゴールーチン
func Find(opts ...Option) []Goroutine {
	o := &options{ignoreIDs: make(map[int]bool)}
	for _, opt := range opts {
		opt(o)
	}

	var found []Goroutine
	for _, g := range parse(stacks(true)) {
		// TODO: 次のゴールーチンを、リークの疑いがあるものから除く。
		//   - Findを呼び出したゴールーチン（currentID関数でIDを取得できる）
		//   - オプションで除外されたゴールーチン（o.ignoreIDs）
		//   - テストを動かしているゴールーチン（先頭の関数が testing.(*T).Run のもの）
		found = append(found, g)
	}
	return found
}

// VerifyNone は、リークの疑いがあるゴールーチンが残っていればテストを失敗させる。
// ゴールーチンはすぐには終了しないことがあるため、
// 待ち時間を延ばしながら何度か調べ直し、それでも残っているものを報告する。
func VerifyNone(t testing.TB, opts ...Option) {
	t.Helper()

	var found []Goroutine
	wait := time.Millisecond
	for range 20 {
		found = Find(opts...)
		if len(found) == 0 {
			return
		}
		time.Sleep(wait)
		wait = min(wait*2, 100*time.Millisecond)
	}

	var sb strings.Builder
	for _, g := range found {
		sb.WriteString(g.Stack)
		sb.WriteString("\n\n")
	}
	t.Errorf("found %d unexpected goroutines:\n%s", len(found), sb.String())
}

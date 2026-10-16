package leaks_test

import (
	"bytes"
	"fmt"
	"runtime"
	"runtime/pprof"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/gohandson/goroutineleak-ja/leaks"
)

// leakedGoroutines はgoroutineleakプロファイルに載っているゴールーチンの数を返す。
func leakedGoroutines(t *testing.T) int {
	t.Helper()

	runtime.GC()

	var buf bytes.Buffer
	if err := pprof.Lookup("goroutineleak").WriteTo(&buf, 1); err != nil {
		t.Fatal("failed to write profile:", err)
	}

	var n int
	if _, err := fmt.Sscanf(buf.String(), "goroutineleak profile: total %d", &n); err != nil {
		t.Fatal("failed to parse profile:", err)
	}
	return n
}

// profileDetects はfを実行し、goroutineleakプロファイルにリークが現れるかを返す。
func profileDetects(t *testing.T, f func()) bool {
	t.Helper()

	before := leakedGoroutines(t)
	f()

	// ゴールーチンがブロックし、リークと判定されるまで少し待つ
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if leakedGoroutines(t) > before {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// leak は、nameに対応する関数を呼び出してリークを起こす。
func leak(t *testing.T, name string) {
	t.Helper()

	switch name {
	case "LocalChanSend":
		leaks.LocalChanSend()
	case "LocalChanRecv":
		leaks.LocalChanRecv()
	case "RangeNotClosed":
		leaks.RangeNotClosed()
	case "NilChan":
		leaks.NilChan()
	case "EmptySelect":
		leaks.EmptySelect()
	case "WaitGroup":
		leaks.WaitGroup()
	case "Mutex":
		leaks.Mutex()
	case "GlobalChan":
		leaks.GlobalChan()
	case "Sleep":
		leaks.Sleep()
	case "PipeRead":
		leaks.PipeRead()
	default:
		t.Fatal("unknown leak:", name)
	}
}

func TestLeaks(t *testing.T) {
	cases := []struct {
		name string
		// goroutineleakプロファイルで検出できるか
		wantProfile bool
	}{
		{name: "LocalChanSend", wantProfile: true},
		{name: "LocalChanRecv", wantProfile: true},
		{name: "RangeNotClosed", wantProfile: true},
		{name: "NilChan", wantProfile: true},
		{name: "EmptySelect", wantProfile: true},
		{name: "WaitGroup", wantProfile: true},
		{name: "Mutex", wantProfile: true},
		{name: "GlobalChan", wantProfile: false},
		{name: "Sleep", wantProfile: false},
		{name: "PipeRead", wantProfile: false},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// このテストの開始時点で存在するゴールーチンは、goleakの対象外にする
			ignore := goleak.IgnoreCurrent()

			f := func() { leak(t, tt.name) }
			if got := profileDetects(t, f); got != tt.wantProfile {
				t.Errorf("goroutineleakプロファイルで検出できるか: got %v, want %v", got, tt.wantProfile)
			}

			// goleakは、どのパターンも検出できる
			if err := goleak.Find(ignore); err == nil {
				t.Error("goleakでリークを検出できませんでした")
			}
		})
	}
}

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

func TestLeaks(t *testing.T) {
	// TODO: 各ケースの wantProfile を、goroutineleakプロファイルで検出できるなら true、できないなら false にする
	cases := []struct {
		name string
		leak func()
		// goroutineleakプロファイルで検出できるか
		wantProfile bool
	}{
		{name: "LocalChanSend", leak: leaks.LocalChanSend},
		{name: "LocalChanRecv", leak: leaks.LocalChanRecv},
		{name: "RangeNotClosed", leak: leaks.RangeNotClosed},
		{name: "NilChan", leak: leaks.NilChan},
		{name: "EmptySelect", leak: leaks.EmptySelect},
		{name: "WaitGroup", leak: leaks.WaitGroup},
		{name: "Mutex", leak: leaks.Mutex},
		{name: "GlobalChan", leak: leaks.GlobalChan},
		{name: "Sleep", leak: leaks.Sleep},
		{name: "PipeRead", leak: leaks.PipeRead},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			// このテストの開始時点で存在するゴールーチンは、goleakの対象外にする
			ignore := goleak.IgnoreCurrent()

			if got := profileDetects(t, tt.leak); got != tt.wantProfile {
				t.Errorf("goroutineleakプロファイルで検出できるか: got %v, want %v", got, tt.wantProfile)
			}

			// goleakは、どのパターンも検出できる
			if err := goleak.Find(ignore); err == nil {
				t.Error("goleakでリークを検出できませんでした")
			}
		})
	}
}

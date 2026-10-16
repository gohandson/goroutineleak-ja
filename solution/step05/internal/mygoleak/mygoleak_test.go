package mygoleak_test

import (
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/gohandson/goroutineleak-ja/internal/mygoleak"
)

func leakSend() {
	ch := make(chan int)
	go func() {
		ch <- 1
	}()
}

func leakRecv() {
	ch := make(chan int)
	go func() {
		<-ch
	}()
}

func leakEmptySelect() {
	go func() {
		select {}
	}()
}

func leakSleep() {
	go func() {
		time.Sleep(time.Hour)
	}()
}

func leakWaitGroup() {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		wg.Wait()
	}()
}

// waitFind は、リークの疑いがあるゴールーチンが現れてブロックするまで待って、それを返す。
// 起動した直後のゴールーチンは、まだブロックしておらずrunnableなどの状態であるため、
// 状態が落ち着くのを待つ必要がある。
func waitFind(opt mygoleak.Option) []mygoleak.Goroutine {
	var gs []mygoleak.Goroutine
	for range 100 {
		gs = mygoleak.Find(opt)
		if len(gs) > 0 && !slices.ContainsFunc(gs, running) {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	return gs
}

func running(g mygoleak.Goroutine) bool {
	return g.State == "runnable" || g.State == "running"
}

// leak は、nameに対応するリークを起こす。
func leak(t *testing.T, name string) {
	t.Helper()

	switch name {
	case "chan send":
		leakSend()
	case "chan receive":
		leakRecv()
	case "empty select":
		leakEmptySelect()
	case "sleep":
		leakSleep()
	case "wait group":
		leakWaitGroup()
	default:
		t.Fatal("unknown leak:", name)
	}
}

func TestFind(t *testing.T) {
	cases := []struct {
		name      string
		wantState string
	}{
		{name: "chan send", wantState: "chan send"},
		{name: "chan receive", wantState: "chan receive"},
		{name: "empty select", wantState: "select (no cases)"},
		{name: "sleep", wantState: "sleep"},
		{name: "wait group", wantState: "sync.WaitGroup.Wait"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			opt := mygoleak.IgnoreCurrent()

			leak(t, tt.name)

			gs := waitFind(opt)
			if len(gs) != 1 {
				t.Fatalf("got %d goroutines, want 1: %v", len(gs), gs)
			}
			if gs[0].State != tt.wantState {
				t.Errorf("got state %q, want %q", gs[0].State, tt.wantState)
			}
		})
	}
}

func TestFind_noLeak(t *testing.T) {
	opt := mygoleak.IgnoreCurrent()

	done := make(chan struct{})
	go func() {
		close(done)
	}()
	<-done

	if gs := mygoleak.Find(opt); len(gs) != 0 {
		t.Errorf("got %d goroutines, want 0: %v", len(gs), gs)
	}
}

// fakeT はErrorfが呼ばれたかどうかを記録する。
type fakeT struct {
	testing.TB
	failed bool
}

func (f *fakeT) Helper() {}

func (f *fakeT) Errorf(format string, args ...any) {
	f.failed = true
}

func TestVerifyNone(t *testing.T) {
	cases := []struct {
		name       string
		leak       bool
		wantFailed bool
	}{
		{name: "no leak", leak: false, wantFailed: false},
		{name: "leak", leak: true, wantFailed: true},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			opt := mygoleak.IgnoreCurrent()

			if tt.leak {
				leakRecv()
			}

			ft := &fakeT{TB: t}
			mygoleak.VerifyNone(ft, opt)
			if ft.failed != tt.wantFailed {
				t.Errorf("failed = %v, want %v", ft.failed, tt.wantFailed)
			}
		})
	}
}

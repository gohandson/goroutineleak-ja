// Package leaks はゴールーチンリークを起こす関数を集めたパッケージである。
// 各関数は、呼び出すとゴールーチンを1つ起動し、そのゴールーチンは二度と動けなくなる。
package leaks

import (
	"os"
	"sync"
	"time"
)

// LocalChanSend は、誰も受信しないチャネルへの送信でブロックする。
func LocalChanSend() {
	ch := make(chan int)
	go func() {
		ch <- 1
	}()
}

// LocalChanRecv は、誰も送信しないチャネルからの受信でブロックする。
func LocalChanRecv() {
	ch := make(chan int)
	go func() {
		<-ch
	}()
}

// RangeNotClosed は、closeされないチャネルをrangeで受信し続ける。
func RangeNotClosed() {
	ch := make(chan int)
	go func() {
		for range ch {
		}
	}()
}

// NilChan は、nilチャネルから受信しようとする。
func NilChan() {
	var ch chan int
	go func() {
		<-ch
	}()
}

// EmptySelect は、caseを持たないselect文でブロックする。
func EmptySelect() {
	go func() {
		select {}
	}()
}

// WaitGroup は、Doneが呼ばれないWaitGroupをWaitする。
func WaitGroup() {
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		wg.Wait()
	}()
}

// Mutex は、Unlockされないsync.MutexのLockを待つ。
func Mutex() {
	mu := new(sync.Mutex)
	mu.Lock()
	go func() {
		mu.Lock()
	}()
}

// globalCh はパッケージ変数なので、プログラムが動いている間は常に参照できる。
var globalCh = make(chan int)

// GlobalChan は、パッケージ変数のチャネルへの送信でブロックする。
func GlobalChan() {
	go func() {
		globalCh <- 1
	}()
}

// Sleep は、非常に長い時間スリープする。
func Sleep() {
	go func() {
		time.Sleep(time.Hour)
	}()
}

// pipeWriters は、PipeReadで作ったパイプの書き込み側を保持する。
// 書き込み側がGCされると閉じられて読み込みがEOFで終わってしまうため、
// 参照を持ち続ける。
var pipeWriters []*os.File

// PipeRead は、書き込まれないパイプからの読み込みでブロックする。
func PipeRead() {
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	pipeWriters = append(pipeWriters, w)
	go func() {
		buf := make([]byte, 1)
		r.Read(buf)
	}()
}

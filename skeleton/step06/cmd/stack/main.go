package main

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

func main() {
	// チャネルへの送信でブロックするゴールーチン
	sendCh := make(chan int)
	go func() {
		sendCh <- 1
	}()

	// チャネルからの受信でブロックするゴールーチン
	recvCh := make(chan int)
	go func() {
		<-recvCh
	}()

	// スリープするゴールーチン
	go func() {
		time.Sleep(time.Hour)
	}()

	// WaitGroupを待つゴールーチン
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Wait()
	}()

	// ゴールーチンがブロックするまで少し待つ
	time.Sleep(100 * time.Millisecond)

	// すべてのゴールーチンのスタックトレースを取得して表示する
	buf := make([]byte, 1<<16)
	n := runtime.Stack(buf, true)
	fmt.Printf("%s", buf[:n])
}

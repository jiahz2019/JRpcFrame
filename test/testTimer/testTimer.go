package main

import (
	"JRpcFrame/jlog"
	"JRpcFrame/jtimer"
	"fmt"
	"time"
)

func foo(args ...interface{}) {
	fmt.Printf("I am No. %d function, delay %d ms\n", args[0].(int), args[1].(int))
}

func main() {

	autoTS := jtimer.NewAutoTimerScheduler()
	//给调度器添加Timer
	for i := 1; i < 10; i++ {
		f := jtimer.NewDelayFunc(foo, []interface{}{i, i * 3000})
		tID, err := autoTS.CreateTimerAfter(f, time.Duration(3000*i)*time.Millisecond, 3, int64(time.Duration(3000*i)*time.Millisecond))

		if err != nil {
			jlog.StdLogger.Error("create timer error", tID, err)
			break
		}
	}

	//阻塞等待
	select {}
}

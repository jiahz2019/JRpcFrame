package main

import (
	"JRpcFrame/jlog"
	"os"
)

func main() {

	jlog.StdLogger.Debug("hello world")
	jlog.StdLogger.Info("hello world")
	dlog := jlog.NewLogger(os.Stdout, "test log", jlog.Date|jlog.Time|jlog.LogLevel|jlog.LongFile, jlog.LogDebug)
	dlog.Debug("hello world")
	dlog.Info("hello world")
	dlog.SetLevel(jlog.LogInfo)
	//设置日志写入文件
	dlog.SetLogFile("./log", "testfile.log")
	dlog.Debug("it should exist ")
	dlog.Error("===> Error!!! ~~888")
	dlog.Errorf("===> Error!!!! ~~~%d~~~", 666)

}

package jlog

import(
	"os"
)
//全局日志器，输出到控制台
var StdLogger = NewLogger(os.Stdout,"std log",DefaultFlag,LogDebug)

func init() {
	StdLogger.calldDepth = 2
}
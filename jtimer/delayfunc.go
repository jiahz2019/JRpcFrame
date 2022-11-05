package jtimer

import (
	"JRpcFrame/jlog"
	"fmt"
	"reflect"
	"runtime"
)

/*
   @brief:定义一个延迟调用函数,即时间定时器超时的时候，触发的事先注册好的
*/
type DelayFunc struct {
	f    func(...interface{}) //f : 延迟函数调用原型
	args []interface{}        //args: 延迟调用函数传递的形参
}

/*
   @brief:创建一个延迟调用函数
*/
func NewDelayFunc(f func(v ...interface{}), args []interface{}) *DelayFunc {
	return &DelayFunc{
		f:    f,
		args: args,
	}
}

/*
   @brief:打印当前延迟函数的信息，用于日志记录
*/
func (df *DelayFunc) String() string {
	return fmt.Sprintf("{DelayFun:%s, args:%v}", reflect.TypeOf(df.f).Name(), df.args)
}

/*
   @brief: 执行延迟函数---如果执行失败，抛出异常
*/
func (df *DelayFunc) Call() {
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 4096)
			l := runtime.Stack(buf, false)
			errString := fmt.Sprint(err)
			jlog.StdLogger.Error(df.String(), " Core dump info[", errString, "]\n", string(buf[:l]))

		}
	}()

	//调用定时器超时函数
	df.f(df.args...)
}

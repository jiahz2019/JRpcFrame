package main

import (
	//"JRpcFrame/jlog"
	"JRpcFrame/jtimer"
	"JRpcFrame/node"
	"JRpcFrame/service"
	"fmt"
	"time"
)

func init() {
	node.GetNode().Setup(&TestService1{})
	node.GetNode().Setup(&TestService2{})
	node.GetNode().Setup(&TestService3{})

}

type TestService1 struct {
	service.Service
}

func (slf *TestService1) OnInit() error {
	return nil
}

type InputData struct {
	A int
	B int
}

func (slf *TestService1) RPC_Sum(input *InputData, output *int) error {
	*output = input.A + input.B
	return nil
}

type TestService2 struct {
	service.Service
}

func (slf *TestService2) OnInit() error {
	f1 := jtimer.NewDelayFunc(slf.CallTest, []interface{}{})
	jtimer.GlobelTimer.CreateTimerAfter(f1, 4*time.Second, 1, int64(4*time.Second))
	f2 := jtimer.NewDelayFunc(slf.AsyncCallTest, []interface{}{})
	jtimer.GlobelTimer.CreateTimerAfter(f2, 4*time.Second, 1, int64(4*time.Second))
	f3 := jtimer.NewDelayFunc(slf.GoTest, []interface{}{})
	jtimer.GlobelTimer.CreateTimerAfter(f3, 4*time.Second, 1, int64(4*time.Second))
	return nil
}

func (slf *TestService2) CallTest(...interface{}) {
	var input InputData
	input.A = 300
	input.B = 600
	var output int

	//同步调用其他服务的rpc,input为传入的rpc,output为输出参数
	err := slf.Call("TestService3.RPC_Sub", &input, &output)
	if err != nil {
		fmt.Printf("Call error :%+v\n", err)
	} else {
		fmt.Printf("Call output %d\n", output)
	}
}

func (slf *TestService2) AsyncCallTest(...interface{}) {
	var input InputData
	input.A = 300
	input.B = 600
	/*slf.AsyncCallNode(1,"TestService1.RPC_Sum",&input,func(output *int,err error){
	})*/
	//异步调用，在数据返回时，会回调传入函数
	//注意函数的第一个参数一定是RPC_Sum函数的第二个参数，err error为RPC_Sum返回值
	slf.AsyncCall("TestService3.RPC_Sub", &input, func(output *int, err error) {
		if err != nil {
			fmt.Printf("AsyncCall error :%+v\n", err)
		} else {
			fmt.Printf("AsyncCall output %d\n", *output)
		}
	})
}

func (slf *TestService2) GoTest(...interface{}) {
	var input InputData
	input.A = 300
	input.B = 600

	//在某些应用场景下不需要数据返回可以使用Go，它是不阻塞的,只需要填入输入参数
	err := slf.Go("TestService3.RPC_Sub", &input)
	if err != nil {
		fmt.Printf("Go error :%+v\n", err)
	} else {
		fmt.Printf("Go output success\n")
	}

	//以下是广播方式，如果在同一个子网中有多个同名的服务名，CastGo将会广播给所有的node
	//slf.CastGo("TestService6.RPC_Sum",&input)
}

type TestService3 struct {
	service.Service
}

func (slf *TestService3) OnInit() error {
	return nil
}

func (slf *TestService3) RPC_Sub(input *InputData, output *int) error {
	*output = input.A - input.B
	return nil
}

func main() {
	node.Start()

}

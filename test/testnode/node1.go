package main

import (
	"JRpcFrame/node"
	"JRpcFrame/service"
)

func init() {
	node.GetNode().Setup(&TestService1{})
	node.GetNode().Setup(&TestService2{})
}

//新建自定义服务TestService1
type TestService1 struct {
	service.Service
}

//服务初始化函数，在安装服务时，服务将自动调用OnInit函数
func (slf *TestService1) OnInit() error {
	return nil
}

type TestService2 struct {
	service.Service
}

func (slf *TestService2) OnInit() error {
	return nil
}

func main() {

	node.Start()

}

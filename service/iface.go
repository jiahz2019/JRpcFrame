package service

import (
	"JRpcFrame/event"
	"JRpcFrame/jrpc"
	"JRpcFrame/profiler"
)

//存放service包中的所有接口
//module的接口
type IModule interface {
	SetModuleId(moduleId uint32) bool
	GetModuleId() uint32
	AddModule(module IModule) (uint32, error)
	GetModule(moduleId uint32) IModule
	GetAncestor() IModule
	ReleaseModule(moduleId uint32)
	NewModuleId() uint32
	GetParent() IModule
	OnInit() error
	OnRelease()
	getBaseModule() IModule
	GetService() IService
	GetModuleName() string
	PublishEvent(ev event.IEvent, s IService)
	BroadCastEvent(ev event.IEvent)
	GetEventPublisher() event.IEventPublisher
}

//service的接口
type IService interface {
	InitService(iService IService, getClientFun jrpc.FuncRpcClient, getServerFun jrpc.FuncRpcServer, serviceCfg interface{})
	Wait()
	Start()

	OnSetup(iService IService) //服务安装时调用的hook函数
	OnInit() error
	OnStart()
	OnRelease()

	SetName(serviceName string)
	GetName() string
	GetRpcHandler() jrpc.IRpcHandler
	GetServiceCfg() interface{}
	GetProfiler() *profiler.Profiler
	GetEventListener() event.IEventListener
	SetEventChannelNum(num int)
	OpenProfiler()

	PublishEvent(ev event.IEvent, s IService)
	BroadCastEvent(ev event.IEvent)
	GetEventPublisher() event.IEventPublisher
}

/*
type INodeConnectListener interface{
	OnNodeConnected(nodeId int)  //与节点nodeid建立起连接
	OnNodeDisconnect(nodeId int)  //与节点nodeid断开连接
}*/

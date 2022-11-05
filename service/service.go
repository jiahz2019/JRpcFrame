package service

import (
	"JRpcFrame/event"
	"JRpcFrame/jlog"
	"JRpcFrame/jrpc"
	mempool "JRpcFrame/memorypool"
	"JRpcFrame/profiler"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
)

var closeSig chan bool

//最大Event缓存数量
var maxServiceEventChannel = 2000000
var eventPool = mempool.NewPoolEx(make(chan mempool.IPoolData, maxServiceEventChannel),
	func() mempool.IPoolData {
		return &event.Event{}
	})

type Service struct {
	Module
	jrpc.BaseRpcHandler        //rpc
	name                string //service name
	wg                  sync.WaitGroup
	serviceCfg          interface{}
	goroutineNum        int32
	startStatus         bool

	eventListener event.IEventListener
	chanEvent     chan event.IEvent

	profiler *profiler.Profiler //性能分析器

	OnNodeConnectFunc    func(nodeId int)
	OnNodeDisconnectFunc func(nodeId int)
}
type NodeConnectEvent struct {
	event.Event
	NodeId int
}

func NewNodeConnectEvent(nodeId int, publisher event.IEventPublisher) *NodeConnectEvent {
	n := &NodeConnectEvent{}
	n.Data = nil
	n.Type = event.ServiceNodeConnectEvent
	n.Publisher = publisher
	n.NodeId = nodeId
	return n
}
func NewNodeDisconnectEvent(nodeId int, publisher event.IEventPublisher) *NodeConnectEvent {
	n := &NodeConnectEvent{}
	n.Data = nil
	n.Type = event.ServiceNodeDisconnectEvent
	n.Publisher = publisher
	n.NodeId = nodeId
	return n
}

func (s *Service) InitService(iService IService, getClientFun jrpc.FuncRpcClient, getServerFun jrpc.FuncRpcServer, serviceCfg interface{}) {

	if s.chanEvent == nil {
		s.chanEvent = make(chan event.IEvent, maxServiceEventChannel)
	}
	sb := jrpc.NewBaseRpcHandler(iService.(jrpc.IRpcHandlerChannel), s.name, getClientFun, getServerFun)
	sb.RegisterRpcMethod(iService.(jrpc.IRpcHandler))
	s.BaseRpcHandler = *sb
	//初始化祖先
	s.self = iService.(IModule)
	s.ancestor = iService.(IModule)
	s.seedModuleId = InitModuleId
	s.descendants = map[uint32]IModule{}
	s.eventPublisher = event.NewEventPublisher()

	s.startStatus = false
	s.serviceCfg = serviceCfg
	s.goroutineNum = 1
	s.eventListener = event.NewEventListener(s)
	//把每个service加入servicemgr中
	serviceMgr.Add(iService)
}

/*
   @brief:service启动
*/
func (s *Service) Start() {
	s.startStatus = true
	for i := int32(0); i < s.goroutineNum; i++ {
		s.wg.Add(1)
		go func() {
			s.Run()
		}()
	}
}

/*
   @brief:等待service执行结束
*/
func (s *Service) Wait() {
	s.wg.Wait()
}

func (s *Service) Run() {
	jlog.StdLogger.Debug("Start running Service ", s.GetName())
	defer s.wg.Done()
	var bStop = false
	s.self.(IService).OnStart()
	for {
		var analyzer *profiler.Analyzer
		select {
		case <-closeSig:
			bStop = true
		case ev := <-s.chanEvent:
			switch ev.GetEventType() {
			case event.ServiceRpcRequestEvent:
				cEvent, ok := ev.(*event.Event)
				if ok == false {
					jlog.StdLogger.Error("Type event conversion error")
					break
				}
				rpcRequest, ok := cEvent.Data.(*jrpc.RpcRequest)
				if ok == false {
					jlog.StdLogger.Error("Type *rpc.RpcRequest conversion error")
					break
				}
				if s.profiler != nil {
					analyzer = s.profiler.Push("[Req]" + rpcRequest.RpcRequestData.GetServiceMethod())
				}

				s.HandlerRpcRequest(rpcRequest)
				if analyzer != nil {
					analyzer.Pop()
					analyzer = nil
				}
				eventPool.Put(cEvent)
			case event.ServiceRpcResponseEvent:
				cEvent, ok := ev.(*event.Event)
				if ok == false {
					jlog.StdLogger.Error("Type event conversion error")
					break
				}
				rpcResponseCB, ok := cEvent.Data.(*jrpc.Call)
				if ok == false {
					jlog.StdLogger.Error("Type *rpc.Call conversion error")
					break
				}
				if s.profiler != nil {
					analyzer = s.profiler.Push("[Res]" + rpcResponseCB.ServiceMethod)
				}
				s.HandlerRpcResponseCB(rpcResponseCB)
				if analyzer != nil {
					analyzer.Pop()
					analyzer = nil
				}
				eventPool.Put(cEvent)

			default:
				//监测该event的执行状态
				if s.profiler != nil {
					analyzer = s.profiler.Push("[SEvent]" + strconv.Itoa(int(ev.GetEventType())))
				}

				s.eventListener.EventHandler(ev)
				//监测完毕，保存监测报告
				if analyzer != nil {
					analyzer.Pop()
					analyzer = nil
				}
			}

		}
		if bStop == true {
			if atomic.AddInt32(&s.goroutineNum, -1) <= 0 {
				s.startStatus = false
				s.Release()
			}
			break
		}
	}
}

/*
   @brief:释放service
*/
func (s *Service) Release() {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			l := runtime.Stack(buf, false)
			errString := fmt.Sprint(r)
			jlog.StdLogger.Error("core dump info[", errString, "]\n", string(buf[:l]))
		}
	}()
	s.self.OnRelease()
	jlog.StdLogger.Debug("Release Service ", s.GetName())
}

/*
   @brief:开启性能报告
*/
func (s *Service) OpenProfiler() {
	s.profiler = profiler.RegProfiler(s.GetName())
	if s.profiler == nil {
		jlog.StdLogger.Fatal("rofiler.RegProfiler ", s.GetName(), " fail.")
	}
}

/*
    @brief:把event压入service中，等待执行
	@param [in] ev:需要压入的事件
*/
func (s *Service) PushEvent(ev event.IEvent) error {
	return s.pushEvent(ev)
}

func (s *Service) pushEvent(ev event.IEvent) error {
	if len(s.chanEvent) >= maxServiceEventChannel {
		err := errors.New("The event channel in the service is full")
		jlog.StdLogger.Error(err.Error())
		return err
	}
	s.chanEvent <- ev
	return nil
}

/*
    @brief:把rpc请求压入service中，等待执行
	@param [in] ev:需要压入的rpc请求
*/
func (s *Service) PushRpcRequest(rpcRequest *jrpc.RpcRequest) error {
	ev := eventPool.Get().(*event.Event)
	ev.Type = event.ServiceRpcRequestEvent
	ev.Data = rpcRequest

	return s.pushEvent(ev)
}

/*
    @brief:把rpc回应请求压入service中，等待执行
	@param [in] ev:需要压入的rpc回应，即call
*/
func (s *Service) PushRpcResponse(call *jrpc.Call) error {
	ev := eventPool.Get().(*event.Event)
	ev.Type = event.ServiceRpcResponseEvent
	ev.Data = call

	return s.pushEvent(ev)
}

/*
    @brief:service监听publisher发出的类型为eventType的事件及其回调函数
	@param [in] ev:注册事件的类型
	@param [in] publisher:事件发出者
	@param [in] callback:事件的回调函数
*/
func (s *Service) RegEventReceiverCb(eventType event.EventType, publisher event.IEventPublisher, callback event.EventCallBack) {
	s.eventListener.RegEventCb(eventType, publisher, callback)
}

/*
    @brief:service取消监听publisher发出的类型为eventType的事件
	@param [in] ev:注销事件的类型
	@param [in] publisher:事件发出者
*/
func (s *Service) UnRegEventReceiverFunc(eventType event.EventType, publisher event.IEventPublisher) {
	s.eventListener.UnRegEventCb(eventType, publisher)
}

func (s *Service) RegNodeConnectCb() {
	s.RegEventReceiverCb(event.ServiceNodeConnectEvent, s.eventPublisher, s.OnNodeConnectEvent)
}
func (s *Service) RegNodeDisconnectCb() {
	s.RegEventReceiverCb(event.ServiceNodeDisconnectEvent, s.eventPublisher, s.OnNodeDisconnectEvent)
}
func (s *Service) OnNodeConnectEvent(ev event.IEvent) {
	conEv, ok := ev.(*NodeConnectEvent)
	if !ok || conEv.GetEventType() != event.ServiceNodeConnectEvent {
		return
	}
	s.OnNodeConnectFunc(conEv.NodeId)
}
func (s *Service) OnNodeDisconnectEvent(ev event.IEvent) {
	conEv, ok := ev.(*NodeConnectEvent)
	if !ok || conEv.GetEventType() != event.ServiceNodeDisconnectEvent {
		return
	}
	s.OnNodeDisconnectFunc(conEv.NodeId)
}

/*
    @brief:service设置本地连接到nodeid节点时的hook函数
	@param [in] f:需要设置的func
*/
func (s *Service) SetOnNodeConnectFunc(f func(nodeId int)) {
	s.OnNodeConnectFunc = f
}

/*
    @brief:service设置本地与nodeid节点断开连接时的hook函数
	@param [in] f:需要设置的func
*/
func (s *Service) SetOnNodeDisconnectFunc(f func(nodeId int)) {
	s.OnNodeDisconnectFunc = f
}

func (s *Service) IsSingleCoroutine() bool {
	return s.goroutineNum == 1
}

//所有的hookfunc
//---------------------------------------------
func (s *Service) OnSetup(iService IService) {
	if iService.GetName() == "" {
		s.name = reflect.Indirect(reflect.ValueOf(iService)).Type().Name()
	}
}

func (s *Service) OnRelease() {
}

func (s *Service) OnStart() {
}

func (s *Service) OnInit() error {
	return nil
}

func (s *Service) GetName() string {
	return s.name
}

func (s *Service) SetName(serviceName string) {
	s.name = serviceName
}

func (s *Service) GetServiceCfg() interface{} {
	return s.serviceCfg
}
func (s *Service) GetRpcHandler() jrpc.IRpcHandler {
	return &s.BaseRpcHandler
}
func (s *Service) GetEventListener() event.IEventListener {
	return s.eventListener
}
func (s *Service) GetProfiler() *profiler.Profiler {
	return s.profiler
}

func (s *Service) GetServiceEventChannelNum() int {
	return len(s.chanEvent)
}

func (s *Service) SetEventChannelNum(num int) {
	if s.chanEvent == nil {
		s.chanEvent = make(chan event.IEvent, num)
	} else {
		panic("this stage cannot be set")
	}
}

func (s *Service) SetGoRoutineNum(goroutineNum int32) bool {
	//已经开始状态不允许修改协程数量,打开性能分析器不允许开多线程
	if s.startStatus == true || s.profiler != nil {
		jlog.StdLogger.Error("open profiler mode is not allowed to set Multi-coroutine.")
		return false
	}

	s.goroutineNum = goroutineNum
	return true
}

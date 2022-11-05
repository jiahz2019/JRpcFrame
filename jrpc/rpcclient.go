package jrpc

import (
	"JRpcFrame/jlog"
	"JRpcFrame/jnet"
	"JRpcFrame/jtimer"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

var clientSeq uint32

type TriggerNodeConnectEvent func(bConnect bool, clientSeq uint32, nodeId int)

//rpcclient
type RpcClient struct {
	clientSeq uint32 //客户端序号，由clientSeq自增
	nodeId    int    //连接的节点的id
	isLocal   bool   //是否是本地服务器的客户端
	tcpClient *jnet.TcpClient
	conn      *jnet.TcpConn

	pendingLock          sync.RWMutex
	startSeq             uint64
	pending              map[uint64]*Call  //seq-call，未完成的call
	pendingTimer         map[uint64]uint32 //seq-timerid
	callRpcTimeout       time.Duration
	maxCheckCallRpcCount int

	TriggerNodeConnectEvent
}

//rpclient的agent
type RpcClientAgent struct {
	conn      *jnet.TcpConn
	rpcClient *RpcClient
}

/*
   brief:rpcclient的构造函数
*/
func NewRpcClient(nodeId int, clientName string, remoteAddr string, MinMsgLen uint32, MaxMsgLen uint32, LittleEndian bool) *RpcClient {
	c := &RpcClient{
		nodeId:               nodeId,
		maxCheckCallRpcCount: 1000,
		callRpcTimeout:       15 * time.Second,
		pending:              make(map[uint64]*Call),
		pendingTimer:         make(map[uint64]uint32),
		tcpClient:            jnet.NewTcpClient(clientName, remoteAddr, MinMsgLen, MaxMsgLen, LittleEndian),
	}
	c.tcpClient.ConnectInterval = 2 * time.Second
	c.tcpClient.AutoReconnect = true
	c.tcpClient.NewAgent = c.NewAgent
	c.clientSeq = atomic.AddUint32(&clientSeq, 1)
	if remoteAddr == "" {
		c.isLocal = true
	} else {
		c.isLocal = false
	}
	return c
}

/*
    brief:生成一个rpcclient的agent
	param [in] c:agent服务的连接
*/
func (client *RpcClient) NewAgent(conn *jnet.TcpConn) jnet.Agent {
	client.conn = conn
	agent := &RpcClientAgent{conn: conn, rpcClient: client}
	return agent

}

/*
   brief:rpcclient开始
*/
func (client *RpcClient) Start() {
	if client.isLocal {
		return
	}
	client.tcpClient.Start()
}

/*
   brief:RpcClientAgent开启服务
*/
func (agent *RpcClientAgent) Run() {

	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			l := runtime.Stack(buf, false)
			errString := fmt.Sprint(r)
			jlog.StdLogger.Errorf("core dump info[", errString, "]\n", string(buf[:l]))
		}
	}()

	//触发与nodeid连接事件
	agent.rpcClient.TriggerNodeConnectEvent(true, agent.rpcClient.GetClientSeq(), agent.rpcClient.GetNodeId())
	for {
		protocolType, body, err := agent.conn.ReadMsg()
		if err != nil {
			jlog.StdLogger.Error("rpcClient ", " ReadMsg error:", err.Error(), "rpcClient name: ", agent.rpcClient.tcpClient.GetName())
			return
		}

		processor := GetProcessor(protocolType)
		if processor == nil {
			agent.conn.ReleaseReadMsg(body)
			jlog.StdLogger.Error("rpcClient ", " ReadMsg head error:", err.Error(), "rpcClient name: ", agent.rpcClient.tcpClient.GetName())
			return
		}
		//解析收到的rpc响应
		response := RpcResponse{}
		response.RpcResponseData = processor.MakeRpcResponse(0, "", nil)

		err = processor.Unmarshal(body, response.RpcResponseData)
		agent.conn.ReleaseReadMsg(body)
		if err != nil {
			processor.ReleaseRpcResponse(response.RpcResponseData)
			jlog.StdLogger.Error("rpcClient Unmarshal head error:", err.Error())
			continue
		}
		//收到了rsp,call已经完成，client移除seq对应的call，将这个call返回
		v := agent.rpcClient.RemovePending(response.RpcResponseData.GetSeq())
		if v == nil {
			jlog.StdLogger.Error("rpcClient cannot find seq ", response.RpcResponseData.GetSeq(), " in pending")
		} else {
			v.Err = nil
			//方法有返回值，将其解码到call中
			if len(response.RpcResponseData.GetReply()) > 0 {
				err = processor.Unmarshal(response.RpcResponseData.GetReply(), v.Reply)
				if err != nil {
					jlog.StdLogger.Error("rpcClient Unmarshal body error:", err.Error())
					v.Err = err
				}
			}

			if response.RpcResponseData.GetErr() != nil {
				v.Err = response.RpcResponseData.GetErr()
			}
			//有回调函数，将其交给handler
			if v.callback != nil && v.callback.IsValid() {
				v.rpcHandler.PushRpcResponse(v)
			} else {
				v.done <- v
			}
		}
	}

}

/*
   @brief:异步发送rpc请求
*/
func (client *RpcClient) AsyncCall(rpcHandler IRpcHandler, callback reflect.Value, serviceMethod string, args interface{}, replyParam interface{}) error {
	//获取编码类型
	processorType, processor := GetProcessorType(args)
	//编码方法参数
	InParam, herr := processor.Marshal(args)
	if herr != nil {
		return herr
	}

	seq := client.generateSeq()
	//生成rpc请求
	request := NewRpcRequest(processor, seq, 0, serviceMethod, false, InParam)
	bytes, err := processor.Marshal(request.RpcRequestData)
	//编码完成，释放请求
	ReleaseRpcRequest(request)
	if err != nil {
		return err
	}

	if client.conn == nil {
		return errors.New("Rpc server is disconnect,call " + serviceMethod)
	}

	//为client添加rpc call，对应发出的rpc req
	call := NewCall()
	call.Reply = replyParam
	call.callback = &callback
	call.rpcHandler = rpcHandler
	call.ServiceMethod = serviceMethod
	call.Seq = seq
	client.AddPending(call)

	err = client.conn.WriteMsg(byte(processorType), bytes)
	if err != nil {
		client.RemovePending(call.Seq)
		ReleaseCall(call)
		return err
	}

	return nil

}

/*
   brief:原生的rpc调用，方法的输入参数不进行编码
*/
func (client *RpcClient) RawGo(processor IRpcProcessor, noReply bool, rpcMethodId uint32, serviceMethod string, args []byte, reply interface{}) *Call {
	call := NewCall()
	call.ServiceMethod = serviceMethod
	call.Reply = reply
	call.Seq = client.generateSeq()

	request := NewRpcRequest(processor, call.Seq, rpcMethodId, serviceMethod, noReply, args)
	bytes, err := processor.Marshal(request.RpcRequestData)
	ReleaseRpcRequest(request)
	if err != nil {
		call.Seq = 0
		call.Err = err
		return call
	}

	if client.conn == nil {
		call.Seq = 0
		call.Err = errors.New(serviceMethod + "  was called failed,rpc client is disconnect")
		return call
	}

	if noReply == false {
		client.AddPending(call)
	}

	err = client.conn.WriteMsg(byte(processor.GetProcessorType()), bytes)
	if err != nil {
		client.RemovePending(call.Seq)
		call.Seq = 0
		call.Err = err
	}

	return call
}

/*
   brief:发送rpc请求，无回调函数
*/
func (client *RpcClient) Go(noReply bool, serviceMethod string, args interface{}, reply interface{}) *Call {
	_, processor := GetProcessorType(args)
	InParam, err := processor.Marshal(args)
	if err != nil {
		call := NewCall()
		call.Err = err
		return call
	}

	return client.RawGo(processor, noReply, 0, serviceMethod, InParam, reply)
}

/*
    brief:pending中添加未完成的call
	param [in] call:需要添加的call
*/
func (client *RpcClient) AddPending(call *Call) {
	client.pendingLock.Lock()
	call.callTime = time.Now()
	client.pending[call.Seq] = call //如果下面发送失败，将会一一直存在这里
	//把call加入pending的同时，定时一个超时处理rpc call的函数
	f := jtimer.NewDelayFunc(client.handleRpcCallTimeout, []interface{}{call.Seq})
	jtimer.GlobelTimer.CreateTimerAfter(f, client.callRpcTimeout, 1, int64(client.callRpcTimeout))
	client.pendingLock.Unlock()
}

/*
    brief:pending中移除完成的call
	param [in] call:需要移除的call
	return: 移除的call
*/
func (client *RpcClient) RemovePending(seq uint64) *Call {
	if seq == 0 {
		return nil
	}
	client.pendingLock.Lock()

	call := client.removePending(seq)
	client.pendingLock.Unlock()
	return call
}

func (client *RpcClient) removePending(seq uint64) *Call {
	call, ok := client.pending[seq]
	if ok == false {
		return nil
	}
	timerId, ok := client.pendingTimer[seq]
	if ok == true {
		jtimer.GlobelTimer.RomoveTimer(timerId)
	}
	delete(client.pending, seq)
	return call
}

func (client *RpcClient) generateSeq() uint64 {
	return atomic.AddUint64(&client.startSeq, 1)
}

func (client *RpcClient) handleRpcCallTimeout(v ...interface{}) {
	seq := v[0].(uint64)
	client.pendingLock.Lock()
	call, ok := client.pending[seq]
	//seq对应的call不在说明，call已经完成，直接返回
	if !ok {
		client.pendingLock.Unlock()
		return
	}
	//call超时
	strTimeout := strconv.FormatInt(int64(client.callRpcTimeout/time.Second), 10)
	call.Err = errors.New("RPC call takes more than " + strTimeout + " seconds")
	client.removePending(call.Seq)
	if call.callback != nil && call.callback.IsValid() {
		call.rpcHandler.PushRpcResponse(call)
	} else {
		call.done <- call
	}
	client.pendingLock.Unlock()
}

func (agent *RpcClientAgent) OnClose() {
	//触发与nodeid断开连接事件
	agent.rpcClient.TriggerNodeConnectEvent(false, agent.rpcClient.GetClientSeq(), agent.rpcClient.GetNodeId())
}

func (client *RpcClient) IsConnected() bool {
	return client.isLocal || (client.conn != nil && client.conn.IsClose() == false)
}

func (client *RpcClient) GetNodeId() int {
	return client.nodeId
}

func (client *RpcClient) Close(waitDone bool) {
	client.tcpClient.Close(waitDone)
}

func (client *RpcClient) GetClientSeq() uint32 {
	return client.clientSeq
}

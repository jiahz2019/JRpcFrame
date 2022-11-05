package jrpc

import (
	mempool "JRpcFrame/memorypool"
	"math"
	"reflect"
	"time"
)

//默认rpc消息的最长和最短
var DefaultMaxMsgLen = uint32(math.MaxUint32)
var DefaultMinMsgLen = uint32(2)

var maxRpcRequestPool = 10240
var rpcRequestPool = mempool.NewPoolEx(
	make(chan mempool.IPoolData, maxRpcRequestPool),
	func() mempool.IPoolData {
		return &RpcRequest{}
	})

var maxRpcCallPool = 10240
var rpcCallPool = mempool.NewPoolEx(
	make(chan mempool.IPoolData, maxRpcCallPool),
	func() mempool.IPoolData {
		return &Call{done: make(chan *Call, 1)}
	})

//回复处理函数
type HandleReplyFunc func(Returns interface{}, Err RpcError)

func handleReplyNull(Returns interface{}, Err RpcError) {
}

var handleReplyNullFunc = handleReplyNull

type RpcRequest struct {
	ref            bool
	RpcRequestData IRpcRequestData

	inParam    interface{}
	localReply interface{}

	handleReply  HandleReplyFunc
	callback     *reflect.Value
	rpcProcessor IRpcProcessor
}

type RpcResponse struct {
	RpcResponseData IRpcResponseData
}
type Call struct {
	ref           bool
	Seq           uint64
	ServiceMethod string
	Reply         interface{}
	Response      *RpcResponse
	Err           error
	done          chan *Call // Strobes when call is complete.
	connId        int
	callback      *reflect.Value
	rpcHandler    IRpcHandler
	callTime      time.Time
}

func NewRpcRequest(rpcProcessor IRpcProcessor, seq uint64, rpcMethodId uint32, serviceMethod string, noReply bool, inParam []byte) *RpcRequest {
	rpcRequest := rpcRequestPool.Get().(*RpcRequest)
	rpcRequest.rpcProcessor = rpcProcessor
	rpcRequest.RpcRequestData = rpcRequest.rpcProcessor.MakeRpcRequest(seq, rpcMethodId, serviceMethod, noReply, inParam)

	return rpcRequest
}

func ReleaseRpcRequest(rpcRequest *RpcRequest) {
	rpcRequest.rpcProcessor.ReleaseRpcRequest(rpcRequest.RpcRequestData)
	rpcRequestPool.Put(rpcRequest)
}

func NewCall() *Call {
	return rpcCallPool.Get().(*Call)
}

func ReleaseCall(call *Call) {
	rpcCallPool.Put(call)
}
func (slf *RpcRequest) Clear() *RpcRequest {
	slf.RpcRequestData = nil
	slf.localReply = nil
	slf.inParam = nil
	slf.handleReply = nil
	slf.callback = nil
	slf.rpcProcessor = nil
	return slf
}

func (slf *RpcRequest) Reset() {
	slf.Clear()
}

func (slf *RpcRequest) IsRef() bool {
	return slf.ref
}

func (slf *RpcRequest) Ref() {
	slf.ref = true
}

func (slf *RpcRequest) UnRef() {
	slf.ref = false
}

func (rpcResponse *RpcResponse) Clear() *RpcResponse {
	rpcResponse.RpcResponseData = nil
	return rpcResponse
}

func (call *Call) Clear() *Call {
	call.Seq = 0
	call.ServiceMethod = ""
	call.Reply = nil
	call.Response = nil
	if len(call.done) > 0 {
		call.done = make(chan *Call, 1)
	}

	call.Err = nil
	call.connId = 0
	call.callback = nil
	call.rpcHandler = nil
	return call
}

func (call *Call) Reset() {
	call.Clear()
}

func (call *Call) IsRef() bool {
	return call.ref
}

func (call *Call) Ref() {
	call.ref = true
}

func (call *Call) UnRef() {
	call.ref = false
}

func (call *Call) Done() *Call {
	return <-call.done
}

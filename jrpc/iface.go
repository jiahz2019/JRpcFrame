package jrpc

type IRpcRequestData interface {
	GetSeq() uint64
	GetServiceMethod() string
	GetInParam() []byte
	IsNoReply() bool
	GetRpcMethodId() uint32
}

type IRpcResponseData interface {
	GetSeq() uint64
	GetErr() *RpcError
	GetReply() []byte
}

type IRpcHandlerChannel interface {
	PushRpcResponse(call *Call) error
	PushRpcRequest(rpcRequest *RpcRequest) error
}

type IRpcHandler interface {
	IRpcHandlerChannel
	GetName() string
	//CallMethod(ServiceMethod string, param interface{}, reply interface{}) error
	UnmarshalInParam(rpcProcessor IRpcProcessor, serviceMethod string, rawRpcMethodId uint32, inParam []byte) (interface{}, error)
	
}

type IRpcProcessor interface {
	Marshal(v interface{}) ([]byte, error) //b表示自定义缓冲区，可以填nil，由系统自动分配
	Unmarshal(data []byte, v interface{}) error
	MakeRpcRequest(seq uint64,rpcMethodId uint32,serviceMethod string,noReply bool,inParam []byte) IRpcRequestData
	MakeRpcResponse(seq uint64,err RpcError,reply []byte) IRpcResponseData

	ReleaseRpcRequest(rpcRequestData IRpcRequestData)
	ReleaseRpcResponse(rpcRequestData IRpcResponseData)
	IsParse(param interface{}) bool //是否可解析
	GetProcessorType() RpcProcessorType
}

type RpcHandleFinder interface {
	FindRpcHandler(serviceMethod string) IRpcHandler
}

type RawRpcCallBack interface {
	Unmarshal(data []byte) (interface{}, error)
	CB(data interface{})
}

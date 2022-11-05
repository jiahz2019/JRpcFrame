package jrpc

import (
	mempool "JRpcFrame/memorypool"

	"github.com/gogo/protobuf/proto"
)

//rpc，probuff编码器
type GoPBProcessor struct {
}

//rpc请求池
var rpcGoPbResponseDataPool = mempool.NewPool(make(chan interface{}, 10240), func() interface{} {
	return &GoPBRpcResponseData{}
})

//rpc响应池
var rpcGoPbRequestDataPool = mempool.NewPool(make(chan interface{}, 10240), func() interface{} {
	return &GoPBRpcRequestData{}
})

func (slf *GoPBRpcRequestData) MakeRequest(seq uint64, rpcMethodId uint32, serviceMethod string, noReply bool, inParam []byte) *GoPBRpcRequestData {
	slf.Seq = seq
	slf.RpcMethodId = rpcMethodId
	slf.ServiceMethod = serviceMethod
	slf.NoReply = noReply
	slf.InParam = inParam

	return slf
}

func (slf *GoPBRpcResponseData) MakeRespone(seq uint64, err RpcError, reply []byte) *GoPBRpcResponseData {
	slf.Seq = seq
	slf.Error = err.Error()
	slf.Reply = reply

	return slf
}

/*
    @brief:编码
	@param [in] v:需要编码的数据
*/
func (slf *GoPBProcessor) Marshal(v interface{}) ([]byte, error) {
	return proto.Marshal(v.(proto.Message))
}

/*
    @brief:解码
	@param [in] v:需要解码的数据
*/
func (slf *GoPBProcessor) Unmarshal(data []byte, msg interface{}) error {
	protoMsg := msg.(proto.Message)
	return proto.Unmarshal(data, protoMsg)
}

/*
    @brief:生成一个rpc请求
	@param [in] seq:请求序号
	@param [in] rpcMethodId :调用方法id
	@param [in] serviceMethod:调用方法
	@param [in] noReply:是否回复
	@param [in] inParam:方法输入参数
	@return:一个rpc请求
*/
func (slf *GoPBProcessor) MakeRpcRequest(seq uint64, rpcMethodId uint32, serviceMethod string, noReply bool, inParam []byte) IRpcRequestData {
	pGoPbRpcRequestData := rpcGoPbRequestDataPool.Get().(*GoPBRpcRequestData)
	pGoPbRpcRequestData.MakeRequest(seq, rpcMethodId, serviceMethod, noReply, inParam)
	return pGoPbRpcRequestData
}

/*
    @brief:生成一个rpc响应
	@param [in] seq:响应对应的请求序号
	@param [in] err :rpc调用错误
	@param [in] reply:回复内容
    @return:一个rpc响应
*/
func (slf *GoPBProcessor) MakeRpcResponse(seq uint64, err RpcError, reply []byte) IRpcResponseData {
	pGoPBRpcResponseData := rpcGoPbResponseDataPool.Get().(*GoPBRpcResponseData)
	pGoPBRpcResponseData.MakeRespone(seq, err, reply)
	return pGoPBRpcResponseData
}

/*
   @brief:释放rpc请求
*/
func (slf *GoPBProcessor) ReleaseRpcRequest(rpcRequestData IRpcRequestData) {
	rpcGoPbRequestDataPool.Put(rpcRequestData)
}

/*
   @brief:释放rpc响应
*/
func (slf *GoPBProcessor) ReleaseRpcResponse(rpcResponseData IRpcResponseData) {
	rpcGoPbResponseDataPool.Put(rpcResponseData)
}

//----------------------------------------------
//goPbRpcRequestData实现IRpcRequestData接口
//goPbRpcResponseData实现IRpcResponseData
/*
    @brief:判断参数是否能被该解码器解码或编码
	@param [in]:需要判断的参数
	@return :判断结构
*/
func (slf *GoPBProcessor) IsParse(param interface{}) bool {
	_, ok := param.(proto.Message)
	return ok
}

func (slf *GoPBProcessor) GetProcessorType() RpcProcessorType {
	return RpcProcessorGoGoPB
}

func (slf *GoPBRpcRequestData) IsNoReply() bool {
	return slf.GetNoReply()
}

func (slf *GoPBRpcResponseData) GetErr() *RpcError {
	if slf.GetError() == "" {
		return nil
	}

	err := RpcError(slf.GetError())
	return &err
}

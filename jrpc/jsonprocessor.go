package jrpc

import (
	mempool "JRpcFrame/memorypool"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

//rpc，json编码器
type JsonProcessor struct {
}

//rpc请求数据，将会以json编码
type JsonRpcRequestData struct {
	//packhead
	Seq           uint64 // sequence number chosen by client
	rpcMethodId   uint32 //用于标识原始rpc请求（以回调函数的方式进行调用）
	ServiceMethod string // 用于标识普通rpc请求，format: "Service.Method"
	NoReply       bool   //是否需要返回
	//packbody
	InParam []byte
}

//rpc响应数据，将会以json编码
type JsonRpcResponseData struct {
	//head
	Seq uint64 // sequence number chosen by client
	Err string

	//returns
	Reply []byte
}

//rpc请求内存池
var rpcJsonResponseDataPool = mempool.NewPool(make(chan interface{}, 10240), func() interface{} {
	return &JsonRpcResponseData{}
})

//rpc响应内存池
var rpcJsonRequestDataPool = mempool.NewPool(make(chan interface{}, 10240), func() interface{} {
	return &JsonRpcRequestData{}
})

/*
    @brief:编码
	@param [in] v:需要编码的数据
*/
func (jsonProcessor *JsonProcessor) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

/*
    @brief:解码
	@param [in] v:需要解码的数据
*/
func (jsonProcessor *JsonProcessor) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
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
func (jsonProcessor *JsonProcessor) MakeRpcRequest(seq uint64, rpcMethodId uint32, serviceMethod string, noReply bool, inParam []byte) IRpcRequestData {
	jsonRpcRequestData := rpcJsonRequestDataPool.Get().(*JsonRpcRequestData)
	jsonRpcRequestData.Seq = seq
	jsonRpcRequestData.rpcMethodId = rpcMethodId
	jsonRpcRequestData.ServiceMethod = serviceMethod
	jsonRpcRequestData.NoReply = noReply
	jsonRpcRequestData.InParam = inParam
	return jsonRpcRequestData
}

/*
    @brief:生成一个rpc响应
	@param [in] seq:响应对应的请求序号
	@param [in] err :rpc调用错误
	@param [in] reply:回复内容
    @return:一个rpc响应
*/
func (jsonProcessor *JsonProcessor) MakeRpcResponse(seq uint64, err RpcError, reply []byte) IRpcResponseData {
	jsonRpcResponseData := rpcJsonResponseDataPool.Get().(*JsonRpcResponseData)
	jsonRpcResponseData.Seq = seq
	jsonRpcResponseData.Err = err.Error()
	jsonRpcResponseData.Reply = reply

	return jsonRpcResponseData
}

/*
   @brief:释放rpc请求
*/
func (jsonProcessor *JsonProcessor) ReleaseRpcRequest(rpcRequestData IRpcRequestData) {
	rpcJsonRequestDataPool.Put(rpcRequestData)
}

/*
   @brief:释放rpc响应
*/
func (jsonProcessor *JsonProcessor) ReleaseRpcResponse(rpcResponseData IRpcResponseData) {
	rpcJsonResponseDataPool.Put(rpcResponseData)
}

//----------------------------------------------
//jsonRpcRequestData实现IRpcRequestData接口
//jsonRpcResponseData实现IRpcResponseData
/*
    @brief:判断参数是否能被该解码器解码或编码
	@param [in]:需要判断的参数
	@return :判断结构
*/
func (jsonProcessor *JsonProcessor) IsParse(param interface{}) bool {
	_, err := json.Marshal(param)
	return err == nil
}

func (jsonProcessor *JsonProcessor) GetProcessorType() RpcProcessorType {
	return RpcProcessorJson
}
func (jsonRpcRequestData *JsonRpcRequestData) IsNoReply() bool {
	return jsonRpcRequestData.NoReply
}

func (jsonRpcRequestData *JsonRpcRequestData) GetSeq() uint64 {
	return jsonRpcRequestData.Seq
}

func (jsonRpcRequestData *JsonRpcRequestData) GetRpcMethodId() uint32 {
	return jsonRpcRequestData.rpcMethodId
}

func (jsonRpcRequestData *JsonRpcRequestData) GetServiceMethod() string {
	return jsonRpcRequestData.ServiceMethod
}

func (jsonRpcRequestData *JsonRpcRequestData) GetInParam() []byte {
	return jsonRpcRequestData.InParam
}

func (jsonRpcResponseData *JsonRpcResponseData) GetSeq() uint64 {
	return jsonRpcResponseData.Seq
}

func (jsonRpcResponseData *JsonRpcResponseData) GetErr() *RpcError {
	if jsonRpcResponseData.Err == "" {
		return nil
	}

	err := RpcError(jsonRpcResponseData.Err)
	return &err
}

func (jsonRpcResponseData *JsonRpcResponseData) GetReply() []byte {
	return jsonRpcResponseData.Reply
}

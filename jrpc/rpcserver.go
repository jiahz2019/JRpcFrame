package jrpc

import (
	"JRpcFrame/jlog"
	"JRpcFrame/jnet"
	"errors"
	"fmt"
	"reflect"
	"strings"
)

type RpcProcessorType uint8

const (
	UnknowProcessor    RpcProcessorType = 0
	RpcProcessorJson   RpcProcessorType = 1
	RpcProcessorGoGoPB RpcProcessorType = 2
)

//var processor IRpcProcessor = &JsonProcessor{}
var arrayProcessor = []IRpcProcessor{&JsonProcessor{}, &GoPBProcessor{}}
var arrayProcessorLen uint8 = 2

//rpcserver
type RpcServer struct {
	functions       map[interface{}]interface{}
	rpcHandleFinder RpcHandleFinder
	tcpServer       *jnet.TcpServer
}

//rpcserver的agent
type RpcServerAgent struct {
	conn      *jnet.TcpConn
	rpcServer *RpcServer
	userData  interface{}
}

/*
   brief:rpcserver的构造函数
*/
func NewRpcServer(rpcHandleFinder RpcHandleFinder, name string, addr string, MinMsgLen uint32, MaxMsgLen uint32, LittleEndian bool) *RpcServer {
	s := &RpcServer{
		rpcHandleFinder: rpcHandleFinder,
		tcpServer:       jnet.NewTcpServer(name, addr, MinMsgLen, MaxMsgLen, LittleEndian),
	}
	s.tcpServer.MaxConnNum = 10000
	s.tcpServer.NewAgent = s.NewAgent

	return s
}

/*
    brief:生成一个rpcserver的agent
	param [in] c:agent服务的连接
*/
func (s *RpcServer) NewAgent(c *jnet.TcpConn) jnet.Agent {
	agent := &RpcServerAgent{conn: c, rpcServer: s}
	return agent
}

/*
   brief:rpcserver开始
*/
func (s *RpcServer) Start() {
	s.tcpServer.Start()
}

/*
   brief:RpcServerAgent开启服务
*/
func (agent *RpcServerAgent) Run() {
	for {
		protocolType, body, err := agent.conn.ReadMsg()
		if err != nil {
			jlog.StdLogger.Error("remoteAddress:", agent.conn.RemoteAddr().String(), ",read message: ", err.Error())
			//will close tcpconnz
			break
		}

		processor := GetProcessor(protocolType)
		if processor == nil {
			agent.conn.ReleaseReadMsg(body)
			jlog.StdLogger.Error("remote rpc  ", agent.conn.RemoteAddr(), " cannot find processor:", protocolType)
			return
		}

		//解析head
		req := NewRpcRequest(processor, 0, 0, "", false, nil)
		err = processor.Unmarshal(body, req.RpcRequestData)
		agent.conn.ReleaseReadMsg(body)
		if err != nil {
			jlog.StdLogger.Error("rpc Unmarshal request is error:", err.Error())
			if req.RpcRequestData.GetSeq() > 0 {
				rpcError := RpcError(err.Error())
				if req.RpcRequestData.IsNoReply() == false {
					agent.WriteResponse(processor, req.RpcRequestData.GetServiceMethod(), req.RpcRequestData.GetSeq(), nil, rpcError)
				}
				ReleaseRpcRequest(req)
				continue
			} else {
				//will close tcpconn
				ReleaseRpcRequest(req)
				break
			}
		}

		//交给程序处理
		serviceMethod := strings.Split(req.RpcRequestData.GetServiceMethod(), ".")
		if len(serviceMethod) < 1 {
			rpcError := RpcError("rpc request req.ServiceMethod is error")
			if req.RpcRequestData.IsNoReply() == false {
				agent.WriteResponse(processor, req.RpcRequestData.GetServiceMethod(), req.RpcRequestData.GetSeq(), nil, rpcError)
			}
			ReleaseRpcRequest(req)
			jlog.StdLogger.Error("rpc request req.ServiceMethod is error")
			continue
		}
		//找到service method对应的rpchandler，交给他处理
		rpcHandler := agent.rpcServer.rpcHandleFinder.FindRpcHandler(serviceMethod[0])
		if rpcHandler == nil {
			rpcError := RpcError(fmt.Sprintf("service method %s not config!", req.RpcRequestData.GetServiceMethod()))
			if req.RpcRequestData.IsNoReply() == false {
				agent.WriteResponse(processor, req.RpcRequestData.GetServiceMethod(), req.RpcRequestData.GetSeq(), nil, rpcError)
			}

			jlog.StdLogger.Error("service method ", req.RpcRequestData.GetServiceMethod(), " not config!")
			ReleaseRpcRequest(req)
			continue
		}

		if req.RpcRequestData.IsNoReply() == false {
			req.handleReply = func(Returns interface{}, Err RpcError) {
				agent.WriteResponse(processor, req.RpcRequestData.GetServiceMethod(), req.RpcRequestData.GetSeq(), Returns, Err)
				ReleaseRpcRequest(req)
			}
		}
		//设置req中方法的输入参数
		req.inParam, err = rpcHandler.UnmarshalInParam(req.rpcProcessor, req.RpcRequestData.GetServiceMethod(), req.RpcRequestData.GetRpcMethodId(), req.RpcRequestData.GetInParam())
		if err != nil {
			rErr := "Call Rpc " + req.RpcRequestData.GetServiceMethod() + " Param error " + err.Error()
			if req.RpcRequestData.IsNoReply() {
				ReleaseRpcRequest(req)
			} else {
				agent.WriteResponse(req.rpcProcessor, req.RpcRequestData.GetServiceMethod(), req.RpcRequestData.GetSeq(), nil, RpcError(rErr))
				ReleaseRpcRequest(req)
			}

			jlog.StdLogger.Error(rErr)
			continue
		}
		err = rpcHandler.PushRpcRequest(req)
		if err != nil {
			rpcError := RpcError(err.Error())

			if req.RpcRequestData.IsNoReply() {
				agent.WriteResponse(processor, req.RpcRequestData.GetServiceMethod(), req.RpcRequestData.GetSeq(), nil, rpcError)
			}

			ReleaseRpcRequest(req)
		}
	}
}

/*
    @brief:RpcServerAgent发出rpc回应
	@param [in] processor:编码器
	@param [in] serviceMethod:方法名
	@param [in] seq:回应包的序号
	@param [in] reply:回复数据
	@param [in] rpcError:回复rpc请求执行过程中的错误

*/
func (agent *RpcServerAgent) WriteResponse(processor IRpcProcessor, serviceMethod string, seq uint64, reply interface{}, rpcError RpcError) {
	var mReply []byte
	var errM error

	if reply != nil {
		mReply, errM = processor.Marshal(reply)
		if errM != nil {
			rpcError = ConvertError(errM)
		}
	}
	//回复客户端
	var rpcResponse RpcResponse
	rpcResponse.RpcResponseData = processor.MakeRpcResponse(seq, rpcError, mReply)
	bytes, errM := processor.Marshal(rpcResponse.RpcResponseData)
	defer processor.ReleaseRpcResponse(rpcResponse.RpcResponseData)

	if errM != nil {
		jlog.StdLogger.Error("service method ", serviceMethod, " Marshal error:", errM.Error())
		return
	}

	errM = agent.conn.WriteMsg(byte(processor.GetProcessorType()), bytes)
	if errM != nil {
		jlog.StdLogger.Error("Rpc ", serviceMethod, " return is error:", errM.Error())
	}
}

/*
   @brief:在本节点的其他rpchandler上执行方法serviceMethod
*/
func (server *RpcServer) selfNodeRpcHandlerGo(processor IRpcProcessor, client *RpcClient, noReply bool, handlerName string, rpcMethodId uint32, serviceMethod string, args interface{}, reply interface{}, rawArgs []byte) *Call {
	//升成call
	pCall := NewCall()
	pCall.Seq = client.generateSeq()
	//根据服务名字找到rpchandler
	rpcHandler := server.rpcHandleFinder.FindRpcHandler(handlerName)
	if rpcHandler == nil {
		pCall.Seq = 0
		pCall.Err = errors.New("service method " + serviceMethod + " not config!")
		jlog.StdLogger.Error(pCall.Err.Error())
		pCall.done <- pCall

		return pCall
	}

	if processor == nil {
		_, processor = GetProcessorType(args)
	}
	//生成rpcrequest
	req := NewRpcRequest(processor, 0, rpcMethodId, serviceMethod, noReply, nil)
	req.inParam = args
	req.localReply = reply
	if rawArgs != nil {
		var err error
		req.inParam, err = rpcHandler.UnmarshalInParam(processor, serviceMethod, rpcMethodId, rawArgs)
		if err != nil {
			ReleaseRpcRequest(req)
			pCall.Err = err
			pCall.done <- pCall
			return pCall
		}
	}
	//需要返回的话，设置request的回复处理函数
	//如果不需要返回，那么client就不知道方法什么时候结束，也就是没有callback的asynccall
	if noReply == false {
		client.AddPending(pCall)
		req.handleReply = func(Returns interface{}, Err RpcError) {
			if reply != nil && Returns != reply && Returns != nil {
				byteReturns, err := req.rpcProcessor.Marshal(Returns)
				if err != nil {
					jlog.StdLogger.Error("returns data cannot be marshal ", pCall.Seq)
					ReleaseRpcRequest(req)
					return
				}

				err = req.rpcProcessor.Unmarshal(byteReturns, reply)
				if err != nil {
					jlog.StdLogger.Error("returns data cannot be Unmarshal ", pCall.Seq)
					ReleaseRpcRequest(req)
					return
				}
			}

			v := client.RemovePending(pCall.Seq)
			if v == nil {
				jlog.StdLogger.Error("rpcClient cannot find seq ", pCall.Seq, " in pending")
				ReleaseRpcRequest(req)
				return
			}
			if len(Err) == 0 {
				pCall.Err = nil
			} else {
				pCall.Err = Err
			}
			pCall.done <- pCall
			ReleaseRpcRequest(req)
		}
	}

	err := rpcHandler.PushRpcRequest(req)
	if err != nil {
		ReleaseRpcRequest(req)
		pCall.Err = err
		pCall.done <- pCall
	}

	return pCall
}

/*
   @brief:在本节点的其他rpchandler上异步执行方法serviceMethod
*/

func (server *RpcServer) selfNodeRpcHandlerAsyncGo(client *RpcClient, callerRpcHandler IRpcHandler, noReply bool, handlerName string, serviceMethod string, args interface{}, reply interface{}, callback reflect.Value) error {
	rpcHandler := server.rpcHandleFinder.FindRpcHandler(handlerName)
	if rpcHandler == nil {
		err := errors.New("service method " + serviceMethod + " not config!")
		jlog.StdLogger.Error(err.Error())
		return err
	}

	_, processor := GetProcessorType(args)
	req := NewRpcRequest(processor, 0, 0, serviceMethod, noReply, nil)
	req.inParam = args
	req.localReply = reply

	if noReply == false {
		callSeq := client.generateSeq()
		pCall := NewCall()
		pCall.Seq = callSeq
		pCall.rpcHandler = callerRpcHandler
		pCall.callback = &callback
		pCall.Reply = reply
		pCall.ServiceMethod = serviceMethod
		client.AddPending(pCall)
		req.handleReply = func(Returns interface{}, Err RpcError) {
			v := client.RemovePending(callSeq)
			if v == nil {
				jlog.StdLogger.Error("rpcClient cannot find seq ", pCall.Seq, " in pending")
				ReleaseCall(pCall)
				ReleaseRpcRequest(req)
				return
			}
			if len(Err) == 0 {
				pCall.Err = nil
			} else {
				pCall.Err = Err
			}

			if Returns != nil {
				pCall.Reply = Returns
			}
			pCall.rpcHandler.PushRpcResponse(pCall)
			ReleaseRpcRequest(req)
		}
	}

	err := rpcHandler.PushRpcRequest(req)
	if err != nil {
		ReleaseRpcRequest(req)
		return err
	}

	return nil
}

/*
    brief:获取参数的编码类型以及编码器
	param [in] param:经过编码后的参数
	return:编码类型,编码器
*/
func GetProcessorType(param interface{}) (RpcProcessorType, IRpcProcessor) {
	for i := uint8(0); i < arrayProcessorLen; i++ {
		if arrayProcessor[i].IsParse(param) == true {
			return RpcProcessorType(i + 1), arrayProcessor[i]
		}
	}

	return RpcProcessorJson, arrayProcessor[RpcProcessorJson]
}

/*
    brief:根据编码类型获取编码器
	param [in] processorType:编码类型
	return: 编码器
*/
func GetProcessor(processorType uint8) IRpcProcessor {

	if (processorType > arrayProcessorLen) || (processorType == 0) {
		return nil
	}

	return arrayProcessor[processorType-1]
}

func (agent *RpcServerAgent) OnClose() {
}

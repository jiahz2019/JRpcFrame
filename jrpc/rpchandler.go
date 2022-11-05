package jrpc

import (
	"JRpcFrame/jlog"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxClusterNode int = 128

type FuncRpcClient func(nodeId int, serviceMethod string, client []*RpcClient) (error, int)
type FuncRpcServer func() *RpcServer

type RpcMethodInfo struct {
	methodCaller     reflect.Value //方法所属结构体，它是method.func的第一个参数
	method           reflect.Method
	inParamValue     reflect.Value
	inParam          interface{}
	outParamValue    reflect.Value
	hasResponder     bool
	rpcProcessorType RpcProcessorType
}

type BaseRpcHandler struct {
	IRpcHandlerChannel
	serviceName     string
	mapFunctions    map[string]RpcMethodInfo  //methodname-RpcMethodInfo，普通rpc
	mapRawFunctions map[uint32]RawRpcCallBack //methodid-RawRpcCallBack，原始rpc

	funcRpcClient FuncRpcClient
	funcRpcServer FuncRpcServer
	rpcClientList []*RpcClient
}

/*
   @brief:构造函数
*/
func NewBaseRpcHandler(rpcHandlerChannel IRpcHandlerChannel, serviceName string, getClientFun FuncRpcClient, getServerFun FuncRpcServer) *BaseRpcHandler {
	r := &BaseRpcHandler{
		mapRawFunctions:    make(map[uint32]RawRpcCallBack),
		IRpcHandlerChannel: rpcHandlerChannel,
		mapFunctions:       map[string]RpcMethodInfo{},
		rpcClientList:      make([]*RpcClient, maxClusterNode),
		funcRpcClient:      getClientFun,
		funcRpcServer:      getServerFun,
		serviceName:        serviceName,
	}

	return r
}

/*
   @brief:处理rpc回应
   @param [in] call:需要处理的回应
*/
func (handler *BaseRpcHandler) HandlerRpcResponseCB(call *Call) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			l := runtime.Stack(buf, false)
			errString := fmt.Sprint(r)
			jlog.StdLogger.Errorf("core dump info[", errString, "]\n", string(buf[:l]))
		}
	}()

	if call.Err == nil {
		call.callback.Call([]reflect.Value{reflect.ValueOf(call.Reply), nilError})
	} else {
		call.callback.Call([]reflect.Value{reflect.ValueOf(call.Reply), reflect.ValueOf(call.Err)})
	}
	ReleaseCall(call)

}

/*
   @brief:处理rpc请求
   @param [in] request:需要处理的rpc请求
*/
func (handler *BaseRpcHandler) HandlerRpcRequest(request *RpcRequest) {
	if request.handleReply == nil {
		defer ReleaseRpcRequest(request)
	}
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			l := runtime.Stack(buf, false)
			errString := fmt.Sprint(r)
			jlog.StdLogger.Error("Handler Rpc ", request.RpcRequestData.GetServiceMethod(), " Core dump info[", errString, "]\n", string(buf[:l]))
			rpcErr := RpcError("call error : core dumps")
			if request.handleReply != nil {
				request.handleReply(nil, rpcErr)
			}
		}
	}()

	//如果是原始RPC请求
	rawRpcId := request.RpcRequestData.GetRpcMethodId()
	if rawRpcId > 0 {
		v, ok := handler.mapRawFunctions[rawRpcId]
		if ok == false {
			jlog.StdLogger.Error("RpcHandler cannot find request rpc id", rawRpcId)
			return
		}

		v.CB(request.inParam)
		return
	}
	//普通的rpc请求

	v, ok := handler.mapFunctions[request.RpcRequestData.GetServiceMethod()]
	if ok == false {
		err := "RpcHandler " + handler.GetName() + "cannot find " + request.RpcRequestData.GetServiceMethod()
		jlog.StdLogger.Error(err)
		if request.handleReply != nil {
			request.handleReply(nil, RpcError(err))
		}
		return
	}
	var paramList []reflect.Value
	var err error
	//生成method的参数
	//1.RpcHandler

	paramList = append(paramList, v.methodCaller) //接受者
	if v.hasResponder == true {
		//如果方法需要回复，第二参数就是handleReply函数
		if request.handleReply != nil {
			responder := reflect.ValueOf(request.handleReply)
			paramList = append(paramList, responder)
		} else {
			paramList = append(paramList, reflect.ValueOf(handleReplyNullFunc))
		}
	}
	//2.inParam，方法输入参数
	paramList = append(paramList, reflect.ValueOf(request.inParam))
	var oParam reflect.Value
	if v.outParamValue.IsValid() {
		//如果方法有返回值，第三或第四个参数就是oParam
		if request.localReply != nil {
			oParam = reflect.ValueOf(request.localReply) //输出参数
		} else {
			oParam = reflect.New(v.outParamValue.Type().Elem())
		}
		paramList = append(paramList, oParam) //输出参数
	} else if request.handleReply != nil && v.hasResponder == false { //调用方有返回值，但被调用函数没有返回参数
		rErr := "Call Rpc " + request.RpcRequestData.GetServiceMethod() + " without return parameter!"
		jlog.StdLogger.Error(rErr)
		request.handleReply(nil, RpcError(rErr))
		return
	}

	//调用方法
	returnValues := v.method.Func.Call(paramList)
	errInter := returnValues[0].Interface()
	if errInter != nil {
		err = errInter.(error)
	}

	if request.handleReply != nil && v.hasResponder == false {
		request.handleReply(oParam.Interface(), ConvertError(err))
	}
}

/*
   @brief:无回调函数的异步rpc调用
*/
func (handler *BaseRpcHandler) goRpc(processor IRpcProcessor, bCast bool, nodeId int, serviceMethod string, args interface{}) error {
	var pClientList [maxClusterNode]*RpcClient
	err, count := handler.funcRpcClient(nodeId, serviceMethod, pClientList[:])
	if count == 0 {
		if err != nil {
			jlog.StdLogger.Error("Call ", serviceMethod, " is error:", err.Error())
		} else {
			jlog.StdLogger.Error("Can not find ", serviceMethod)
		}
		return err
	}

	//2.rpcClient调用
	//如果调用本结点服务
	for i := 0; i < count; i++ {
		//不广播的话，只gorpc一次
		if !bCast && i > 0 {
			break
		}
		if pClientList[i].isLocal == true {
			pLocalRpcServer := handler.funcRpcServer()
			//判断是否是同一服务
			findIndex := strings.Index(serviceMethod, ".")
			if findIndex == -1 {
				sErr := errors.New("Call serviceMethod " + serviceMethod + " is error!")
				jlog.StdLogger.Error(sErr.Error())
				err = sErr

				continue
			}
			serviceName := serviceMethod[:findIndex]
			//找一个本节点的rpcHandler处理
			pCall := pLocalRpcServer.selfNodeRpcHandlerGo(processor, pClientList[i], true, serviceName, 0, serviceMethod, args, nil, nil)
			if pCall.Err != nil {
				err = pCall.Err
			}
			ReleaseCall(pCall)
			continue
		}

		//跨node调用
		pCall := pClientList[i].Go(true, serviceMethod, args, nil)
		if pCall.Err != nil {
			err = pCall.Err
		}
		ReleaseCall(pCall)
	}

	return err
}

/*
   @brief:同步rpc调用
*/
func (handler *BaseRpcHandler) callRpc(nodeId int, serviceMethod string, args interface{}, reply interface{}) error {
	var pClientList [maxClusterNode]*RpcClient
	err, count := handler.funcRpcClient(nodeId, serviceMethod, pClientList[:])
	if err != nil {
		jlog.StdLogger.Error("Call serviceMethod is error:", err.Error())
		return err
	} else if count <= 0 {
		err = errors.New("Call serviceMethod is error:cannot find " + serviceMethod)
		jlog.StdLogger.Error(err.Error())
		return err
	}

	//2.rpcClient调用
	//如果调用本结点服务
	pClient := pClientList[0]
	if pClient.isLocal == true {
		pLocalRpcServer := handler.funcRpcServer()
		//判断是否是同一服务
		findIndex := strings.Index(serviceMethod, ".")
		if findIndex == -1 {
			err := errors.New("Call serviceMethod " + serviceMethod + "is error!")
			jlog.StdLogger.Error(err.Error())
			return err
		}
		serviceName := serviceMethod[:findIndex]

		//本节点的rpcHandler处理
		pCall := pLocalRpcServer.selfNodeRpcHandlerGo(nil, pClient, false, serviceName, 0, serviceMethod, args, reply, nil)
		err = pCall.Done().Err
		ReleaseCall(pCall)
		return err
	}

	//跨node调用
	pCall := pClient.Go(false, serviceMethod, args, reply)
	if pCall.Err != nil {
		err = pCall.Err
		ReleaseCall(pCall)
		return err
	}
	err = pCall.Done().Err
	ReleaseCall(pCall)
	return err
}

/*
   @brief:有回调函数的异步rpc调用
*/
func (handler *BaseRpcHandler) asyncCallRpc(nodeId int, serviceMethod string, args interface{}, callback interface{}) error {
	fVal := reflect.ValueOf(callback)
	if fVal.Kind() != reflect.Func {
		err := errors.New("call " + serviceMethod + " input callback param is error!")
		jlog.StdLogger.Error(err.Error())
		return err
	}

	if fVal.Type().NumIn() != 2 {
		err := errors.New("call " + serviceMethod + " callback param function is error!")
		jlog.StdLogger.Error(err.Error())
		return err
	}

	if fVal.Type().In(0).Kind() != reflect.Ptr || fVal.Type().In(1).String() != "error" {
		err := errors.New("call " + serviceMethod + " callback param function is error!")
		jlog.StdLogger.Error(err.Error())
		return err
	}

	reply := reflect.New(fVal.Type().In(0).Elem()).Interface()
	var pClientList [maxClusterNode]*RpcClient
	err, count := handler.funcRpcClient(nodeId, serviceMethod, pClientList[:])
	if count == 0 || err != nil {
		strNodeId := strconv.Itoa(nodeId)
		if err == nil {
			err = errors.New("cannot find rpcClient from nodeId " + strNodeId + " " + serviceMethod)
		}
		fVal.Call([]reflect.Value{reflect.ValueOf(reply), reflect.ValueOf(err)})
		jlog.StdLogger.Error("Call serviceMethod is error:", err.Error())
		return nil
	}

	//2.rpcClient调用
	//如果调用本结点服务
	pClient := pClientList[0]

	if pClient.isLocal == true {
		pLocalRpcServer := handler.funcRpcServer()
		//判断是否是同一服务
		findIndex := strings.Index(serviceMethod, ".")
		if findIndex == -1 {
			err := errors.New("Call serviceMethod " + serviceMethod + " is error!")
			fVal.Call([]reflect.Value{reflect.ValueOf(reply), reflect.ValueOf(err)})
			jlog.StdLogger.Error(err.Error())
			return nil
		}
		serviceName := serviceMethod[:findIndex]

		//本节点的rpcHandler处理
		err = pLocalRpcServer.selfNodeRpcHandlerAsyncGo(pClient, handler, false, serviceName, serviceMethod, args, reply, fVal)
		if err != nil {
			fVal.Call([]reflect.Value{reflect.ValueOf(reply), reflect.ValueOf(err)})
		}
		return nil
	}

	//跨node调用
	err = pClient.AsyncCall(handler, fVal, serviceMethod, args, reply)
	if err != nil {
		fVal.Call([]reflect.Value{reflect.ValueOf(reply), reflect.ValueOf(err)})
	}
	return nil
}

/*
    @brief:有回调函数的异步rpc调用
	@param [in] serviceMethod:方法名
	@param [in] args:输入参数
	@param [in] callback:回调函数
*/
func (handler *BaseRpcHandler) AsyncCall(serviceMethod string, args interface{}, callback interface{}) error {
	return handler.asyncCallRpc(0, serviceMethod, args, callback)
}

/*
    @brief:同步rpc调用
	@param [in] serviceMethod:方法名
	@param [in] args:输入参数
	@param [in] reply:回复
*/
func (handler *BaseRpcHandler) Call(serviceMethod string, args interface{}, reply interface{}) error {
	return handler.callRpc(0, serviceMethod, args, reply)
}

/*
    @brief:无回调函数的异步rpc调用
	@param [in] serviceMethod:方法名
	@param [in] args:输入参数
*/
func (handler *BaseRpcHandler) Go(serviceMethod string, args interface{}) error {
	return handler.goRpc(nil, false, 0, serviceMethod, args)
}

/*
    @brief:指定节点执行有回调函数的异步rpc调用
	@param [in] nodeId:指定节点id
	@param [in] serviceMethod:方法名
	@param [in] args:输入参数
	@param [in] callback:回调函数
*/
func (handler *BaseRpcHandler) AsyncCallNode(nodeId int, serviceMethod string, args interface{}, callback interface{}) error {
	return handler.asyncCallRpc(nodeId, serviceMethod, args, callback)
}

/*
    @brief:指定节点执行同步rpc调用
	@param [in] nodeId:指定节点id
	@param [in] serviceMethod:方法名
	@param [in] args:输入参数
	@param [in] reply:回复
*/
func (handler *BaseRpcHandler) CallNode(nodeId int, serviceMethod string, args interface{}, reply interface{}) error {
	return handler.callRpc(nodeId, serviceMethod, args, reply)
}

/*
    @brief:指定节点执行无回调函数的异步rpc调用
	@param [in] nodeId:指定节点id
	@param [in] serviceMethod:方法名
	@param [in] args:输入参数
*/
func (handler *BaseRpcHandler) GoNode(nodeId int, serviceMethod string, args interface{}) error {
	return handler.goRpc(nil, false, nodeId, serviceMethod, args)
}

func (handler *BaseRpcHandler) CastGo(serviceMethod string, args interface{}) error {
	return handler.goRpc(nil, true, 0, serviceMethod, args)
}

/*
    @brief:把rpcHandler中的所有方法注册到这个handler中
	@param [in] rpcHandler:需要移植的rpchandler
*/
func (handler *BaseRpcHandler) RegisterRpcMethod(rpcHandler IRpcHandler) error {
	typ := reflect.TypeOf(rpcHandler)
	value := reflect.ValueOf(rpcHandler)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		err := handler.addMethods(value, method)
		if err != nil {
			panic(err)
		}
	}

	return nil
}

func (handler *BaseRpcHandler) GetName() string {
	return handler.serviceName
}

func (handler *BaseRpcHandler) addMethods(value reflect.Value, method reflect.Method) error {
	//只有RPC_开头的才能被调用
	if strings.Index(method.Name, "RPC_") != 0 {
		return nil
	}

	//取出输入参数类型
	var rpcMethodInfo RpcMethodInfo
	typ := method.Type
	if typ.NumOut() != 1 {
		return fmt.Errorf("%s The number of returned arguments must be 1", method.Name)
	}

	if typ.Out(0).String() != "error" {
		return fmt.Errorf("%s The return parameter must be of type error", method.Name)
	}
	//方法输入参数个数只能时2,3,4个
	if typ.NumIn() < 2 || typ.NumIn() > 4 {
		return fmt.Errorf("%s Unsupported parameter format", method.Name)
	}
	var parIdx = 0
	//第一个参数是方法所属的结构体
	if typ.In(parIdx) != value.Type() {
		return fmt.Errorf("%s method first parameter is error ,now first parameter is %s", value.Type(), typ.In(parIdx))
	}
	rpcMethodInfo.methodCaller = value
	parIdx++
	//判断第二个参数是否是HandleReplyFunc
	if typ.In(parIdx).String() == "jrpc.HandleReplyFunc" {
		parIdx += 1
		rpcMethodInfo.hasResponder = true
	}

	for i := parIdx; i < typ.NumIn(); i++ {
		if handler.isExportedOrBuiltinType(typ.In(i)) == false {
			return fmt.Errorf("%s Unsupported parameter types", method.Name)
		}
	}
	//设置methodinfo
	rpcMethodInfo.inParamValue = reflect.New(typ.In(parIdx).Elem())
	rpcMethodInfo.inParam = reflect.New(typ.In(parIdx).Elem()).Interface()
	pt, _ := GetProcessorType(rpcMethodInfo.inParamValue.Interface())
	rpcMethodInfo.rpcProcessorType = pt

	parIdx++
	if parIdx < typ.NumIn() {
		rpcMethodInfo.outParamValue = reflect.New(typ.In(parIdx).Elem())
	}

	rpcMethodInfo.method = method
	//serviceName=handler.GetName()+"."+method.Name
	handler.mapFunctions[handler.GetName()+"."+method.Name] = rpcMethodInfo
	return nil
}

func (handler *BaseRpcHandler) UnmarshalInParam(rpcProcessor IRpcProcessor, serviceMethod string, rawRpcMethodId uint32, inParam []byte) (interface{}, error) {
	if rawRpcMethodId > 0 {
		v, ok := handler.mapRawFunctions[rawRpcMethodId]
		if ok == false {
			strRawRpcMethodId := strconv.FormatUint(uint64(rawRpcMethodId), 10)
			err := errors.New("RpcHandler cannot find request rpc id " + strRawRpcMethodId)
			jlog.StdLogger.Error(err.Error())
			return nil, err
		}

		msg, err := v.Unmarshal(inParam)
		if err != nil {
			strRawRpcMethodId := strconv.FormatUint(uint64(rawRpcMethodId), 10)
			err := errors.New("RpcHandler cannot Unmarshal rpc id " + strRawRpcMethodId)
			jlog.StdLogger.Error(err.Error())
			return nil, err
		}

		return msg, err
	}

	v, ok := handler.mapFunctions[serviceMethod]
	if ok == false {
		return nil, errors.New("RpcHandler " + handler.GetName() + " cannot find " + serviceMethod)
	}

	var err error
	param := reflect.New(v.inParamValue.Type().Elem()).Interface()
	err = rpcProcessor.Unmarshal(inParam, param)
	return param, err

}

/*
    @brief:注册原始rpc请求方法
	@param [in] rpcMethodId:原始rpc方法id
	@param [in] rawRpcCB:原始rpc请求方法
*/
func (handler *BaseRpcHandler) RegRawRpc(rpcMethodId uint32, rawRpcCB RawRpcCallBack) {
	handler.mapRawFunctions[rpcMethodId] = rawRpcCB
}

// Is this an exported - upper case - name?
func isExported(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}

// Is this type exported or a builtin?
func (handler *BaseRpcHandler) isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

package cluster

import (
	"JRpcFrame/jlog"
	"JRpcFrame/jrpc"
	"JRpcFrame/service"
	"fmt"
	"strings"

	"sync"
)

var configDir = "./config/" //集群配置
//安装服务的函数
type SetupServiceFun func(s ...service.IService)

var cluster *Cluster

//节点状态
type NodeStatus int

const (
	Normal  NodeStatus = 0 //正常
	Discard NodeStatus = 1 //丢弃
)

//集群信息
type Cluster struct {
	localNodeInfo           NodeInfo    //本结点配置信息
	masterDiscoveryNodeList []NodeInfo  //配置服务发现Master结点
	globalCfg               interface{} //全局配置

	localServiceCfg  map[string]interface{} //本节点所有服务的配置，map[serviceName]配置数据*
	serviceDiscovery IServiceDiscovery      //服务发现接口 (本节点的服务发现)

	locker         sync.RWMutex                //结点与服务关系保护锁
	mapRpc         map[int]NodeRpcInfo         //map[nodeId]NodeRpcInfo (保存本节所有rpc client)
	mapIdNode      map[int]NodeInfo            //map[NodeId]NodeInfo
	mapServiceNode map[string]map[int]struct{} //map[serviceName]map[NodeId]

	rpcServer jrpc.RpcServer
}
type NodeRpcInfo struct {
	nodeInfo NodeInfo
	client   *jrpc.RpcClient
}
type NodeInfo struct {
	NodeId            int
	NodeName          string
	Private           bool
	ListenAddr        string
	MaxRpcParamLen    uint32   //最大Rpc参数长度
	ServiceList       []string //所有的服务列表
	PublicServiceList []string //对外公开的服务列表
	DiscoveryService  []string //筛选发现的服务，如果不配置，不进行筛选

	status NodeStatus
}

func init() {
	cluster = NewCluster()
}

/*
   @brief:构造函数
*/
func NewCluster() *Cluster {
	c := &Cluster{}
	c.localServiceCfg = map[string]interface{}{}
	c.mapRpc = map[int]NodeRpcInfo{}
	c.mapIdNode = map[int]NodeInfo{}
	c.mapServiceNode = map[string]map[int]struct{}{}
	return c
}
func (cls *Cluster) Start() {
	cls.rpcServer.Start()
}

func (cls *Cluster) Stop() {
	cls.serviceDiscovery.OnNodeStop()
}

/*
    @brief:本节点的集群初始化
	@param [in] locakNodeId:集群信息所在的本地节点
*/
func (cls *Cluster) InitCluster(localNodeId int, setupServiceFun SetupServiceFun) error {
	//1.初始化配置
	err := cls.LoadConfig(localNodeId)
	if err != nil {
		return err
	}
	cls.initLocalRpcServer()
	cls.initLocalRpcClient()

	//2.安装服务发现结点
	cls.SetupServiceDiscovery(localNodeId, setupServiceFun)

	err = cls.serviceDiscovery.InitDiscovery(localNodeId, cls.serviceDiscoveryDelNodeInfo, cls.serviceDiscoverySetNodeInfo)
	if err != nil {
		return err
	}

	return nil
}

func (cls *Cluster) SetupServiceDiscovery(localNodeId int, setupServiceFun SetupServiceFun) {
	if cls.serviceDiscovery != nil {
		return
	}

	//1.如果没有配置DiscoveryNode配置，则使用默认配置文件发现服务
	//检查localNodeId的节点是否是master，以及是否配置了master服务
	localMaster, hasMaster := cls.checkDynamicDiscovery(localNodeId)
	if hasMaster == false {
		cls.serviceDiscovery = &DefaultDiscovery{}
		return
	}
	//2.否则，如果为动态服务发现安装本地发现服务
	setupServiceFun(&masterService, &clientService)

	cls.serviceDiscovery = getDynamicDiscovery()
	if localMaster == true {
		//如果本节点是master，则添加DiscoveryMaster服务
		cls.appendService(DynamicDiscoveryMasterName, false)
	}
	cls.appendService(DynamicDiscoveryClientName, true)

}

/*
   @brief:获取服务发现节点的信息
   @param [in] nodeId:服务发现节点id
   @return :节点信息
*/
func (cls *Cluster) GetMasterDiscoveryNodeInfo(nodeId int) *NodeInfo {
	for i := 0; i < len(cls.masterDiscoveryNodeList); i++ {
		if cls.masterDiscoveryNodeList[i].NodeId == nodeId {
			return &cls.masterDiscoveryNodeList[i]
		}
	}
	return nil
}

/*
   @brief:获取监测节点列表
   @return :监测节点列表
*/
func (cls *Cluster) GetDiscoveryNodeList() []NodeInfo {
	return cls.masterDiscoveryNodeList
}

/*
   @brief:本地节点删除节点nodeid,并断开连接
   @param [in] nodeId:需要删除的nodeid
   @param [in] immediately:是否立刻删除，不论是否在连接中
*/
func (cls *Cluster) serviceDiscoveryDelNodeInfo(nodeId int, immediately bool) {
	if nodeId == 0 {
		return
	}

	//MasterDiscover结点与本地结点不删除
	if cls.GetMasterDiscoveryNodeInfo(nodeId) != nil || nodeId == cls.localNodeInfo.NodeId {
		return
	}
	cls.locker.Lock()
	defer cls.locker.Unlock()

	nodeInfo, ok := cls.mapIdNode[nodeId]
	if ok == false {
		return
	}

	rpc, ok := cls.mapRpc[nodeId]
	for {
		//立即删除
		if immediately || ok == false {
			break
		}

		//正在连接中不主动断开，只断开没有连接中的
		if rpc.client.IsConnected() {
			nodeInfo.status = Discard
			jlog.StdLogger.Info("Discard node ", nodeInfo.NodeId, " ", nodeInfo.ListenAddr)
			return
		}
		break
	}

	for _, serviceName := range nodeInfo.ServiceList {
		cls.delServiceNode(serviceName, nodeId)
	}

	delete(cls.mapIdNode, nodeId)
	delete(cls.mapRpc, nodeId)
	if ok == true {
		rpc.client.Close(false)
	}

	jlog.StdLogger.Info("remove node ", nodeInfo.NodeId, " ", nodeInfo.ListenAddr)
}

/*
   @brief:本地节点设置节点nodeinfo,并连接到该节点
   @param [in] nodeId:需要设置的nodeinfo
*/
func (cls *Cluster) serviceDiscoverySetNodeInfo(nodeInfo *NodeInfo) {
	//本地结点不加入
	if nodeInfo.NodeId == cls.localNodeInfo.NodeId {
		return
	}
	cls.locker.Lock()
	defer cls.locker.Unlock()

	//先清一次的NodeId对应的所有服务清理
	lastNodeInfo, ok := cls.mapIdNode[nodeInfo.NodeId]
	if ok == true {
		for _, serviceName := range lastNodeInfo.ServiceList {
			cls.delServiceNode(serviceName, nodeInfo.NodeId)
		}
	}

	//再重新组装
	mapDuplicate := map[string]interface{}{} //预防重复数据
	for _, serviceName := range nodeInfo.PublicServiceList {
		if _, ok := mapDuplicate[serviceName]; ok == true {
			//存在重复
			jlog.StdLogger.Error("Bad duplicate Service Cfg.")
			continue
		}

		mapDuplicate[serviceName] = nil
		if _, ok := cls.mapServiceNode[serviceName]; ok == false {
			cls.mapServiceNode[serviceName] = make(map[int]struct{}, 1)
		}
		cls.mapServiceNode[serviceName][nodeInfo.NodeId] = struct{}{}
	}
	cls.mapIdNode[nodeInfo.NodeId] = *nodeInfo

	jlog.StdLogger.Info("Discovery nodeId: ", nodeInfo.NodeId, " services:", nodeInfo.PublicServiceList)

	//已经存在连接，则不需要进行设置
	if _, rpcInfoOK := cls.mapRpc[nodeInfo.NodeId]; rpcInfoOK == true {
		return
	}
	rpcInfo := NodeRpcInfo{}
	rpcInfo.nodeInfo = *nodeInfo
	//生成与nodeid相连的client
	rpcInfo.client = jrpc.NewRpcClient(nodeInfo.NodeId, fmt.Sprint(cls.localNodeInfo.NodeId, "_rpcClient"), nodeInfo.ListenAddr, jrpc.DefaultMinMsgLen, nodeInfo.MaxRpcParamLen, true)
	rpcInfo.client.TriggerNodeConnectEvent = cls.triggerNodeConnectEvent
	rpcInfo.client.Start()
	cls.mapRpc[nodeInfo.NodeId] = rpcInfo

}

/*
    @brief:检查该节点是否为服务发现节点
	@param [in] localNodeId:节点ID
	@return: 是否为Master结点,是否配置Master服务
*/
func (cls *Cluster) checkDynamicDiscovery(localNodeId int) (bool, bool) {
	var localMaster bool //本结点是否为Master结点
	var hasMaster bool   //是否配置了Master服务

	//遍历所有结点
	for _, nodeInfo := range cls.masterDiscoveryNodeList {
		if nodeInfo.NodeId == localNodeId {
			localMaster = true
		}
		hasMaster = true
	}

	//返回查询结果
	return localMaster, hasMaster
}

/*
    @brief:为cluster所在的node添加服务
	@param [in] srviceName:添加的服务名字
	@param [in] bPublicService:服务是否公开
*/
func (cls *Cluster) appendService(serviceName string, bPublicService bool) {
	cls.localNodeInfo.ServiceList = append(cls.localNodeInfo.ServiceList, serviceName)
	if bPublicService {
		cls.localNodeInfo.PublicServiceList = append(cls.localNodeInfo.PublicServiceList, serviceName)
	}

	if _, ok := cls.mapServiceNode[serviceName]; ok == false {
		cls.mapServiceNode[serviceName] = map[int]struct{}{}
	}
	cls.mapServiceNode[serviceName][cls.localNodeInfo.NodeId] = struct{}{}
}

/*
    @brief:删除本地节点存储的节点nodeId的service
	@param [in] srviceName:需要删除的服务名
	@param [in] nodeid:服务所在nodeid
*/
func (cls *Cluster) delServiceNode(serviceName string, nodeId int) {
	if nodeId == cls.localNodeInfo.NodeId {
		return
	}

	mapNode := cls.mapServiceNode[serviceName]
	delete(mapNode, nodeId)
	if len(mapNode) == 0 {
		delete(cls.mapServiceNode, serviceName)
	}
}

/*
    @brief:检查本地节点是否配置了这个service
	@param [in] servcieName:需要检查的service name
	@return: bool

*/
func (cls *Cluster) IsConfigService(serviceName string) bool {
	cls.locker.RLock()
	defer cls.locker.RUnlock()
	mapNode, ok := cls.mapServiceNode[serviceName]
	if ok == false {
		return false
	}

	_, ok = mapNode[cls.localNodeInfo.NodeId]
	return ok
}

/*
   @brief:本节点是否是监测节点
*/
func (cls *Cluster) IsMasterDiscoveryNode() bool {
	return cls.GetMasterDiscoveryNodeInfo(cls.GetLocalNodeInfo().NodeId) != nil
}

/*
   @brief:丢弃节点
*/
func (cls *Cluster) DiscardNode(nodeId int) {
	nodeInfo, ok := cls.mapIdNode[nodeId]
	if ok == true && nodeInfo.status == Discard {
		cls.serviceDiscoveryDelNodeInfo(nodeId, true)
	}
}

/*
   @brief:生成一个连接本地节点rpc server的client，保存在mapRpc中
*/
func (cls *Cluster) initLocalRpcClient() {
	rpcInfo := NodeRpcInfo{}
	rpcInfo.nodeInfo = cls.localNodeInfo
	rpcInfo.client = jrpc.NewRpcClient(rpcInfo.nodeInfo.NodeId, "localRpcClient", "", jrpc.DefaultMinMsgLen, jrpc.DefaultMaxMsgLen, true)

	cls.mapRpc[cls.localNodeInfo.NodeId] = rpcInfo
}

/*
   @brief:初始化本地rpcserver
*/
func (cls *Cluster) initLocalRpcServer() {
	cls.rpcServer = *jrpc.NewRpcServer(cls, fmt.Sprint(cls.localNodeInfo.NodeId, "_rpcServer"), cls.localNodeInfo.ListenAddr, jrpc.DefaultMinMsgLen, cls.localNodeInfo.MaxRpcParamLen, true)
}

/*
   @brief:初始化连接至本地rpcserver的rpcclient
*/
func (cls *Cluster) GetLocalNodeInfo() *NodeInfo {
	return &cls.localNodeInfo
}

/*
    @brief:获取节点信息根据nodeid
	@param [in] nodeid:节点id
*/
func (cls *Cluster) GetNodeInfo(nodeId int) (NodeInfo, bool) {
	cls.locker.RLock()
	defer cls.locker.RUnlock()

	nodeInfo, ok := cls.mapIdNode[nodeId]
	return nodeInfo, ok
}

/*
    @brief:获取RpcHandler信息根据serviceName
	@param [in] nodeid:serviceName
*/
func (cls *Cluster) FindRpcHandler(serviceName string) jrpc.IRpcHandler {
	pService := service.GetServiceMgr().GetService(serviceName)
	if pService == nil {
		return nil
	}

	return pService.GetRpcHandler()
}

/*
   @brief:获取本地cluster
*/
func GetCluster() *Cluster {
	return cluster
}
func (cls *Cluster) getRpcClient(nodeId int) *jrpc.RpcClient {
	c, ok := cls.mapRpc[nodeId]
	if ok == false {
		return nil
	}

	return c.client
}

func (cls *Cluster) GetRpcClient(nodeId int) *jrpc.RpcClient {
	cls.locker.RLock()
	defer cls.locker.RUnlock()
	return cls.getRpcClient(nodeId)
}

func GetRpcClient(nodeId int, serviceMethod string, clientList []*jrpc.RpcClient) (error, int) {
	if nodeId > 0 {
		pClient := GetCluster().GetRpcClient(nodeId)
		if pClient == nil {
			return fmt.Errorf("cannot find  nodeid %d!", nodeId), 0
		}
		clientList[0] = pClient
		return nil, 1
	}

	findIndex := strings.Index(serviceMethod, ".")
	if findIndex == -1 {
		return fmt.Errorf("servicemethod param  %s is error!", serviceMethod), 0
	}
	serviceName := serviceMethod[:findIndex]

	//1.找到对应的rpcNodeid
	return GetCluster().GetNodeIdByService(serviceName, clientList, true)
}

func (cls *Cluster) GetNodeIdByService(serviceName string, rpcClientList []*jrpc.RpcClient, bAll bool) (error, int) {
	cls.locker.RLock()
	defer cls.locker.RUnlock()
	mapNodeId, ok := cls.mapServiceNode[serviceName]
	count := 0
	if ok == true {
		for nodeId, _ := range mapNodeId {
			pClient := GetCluster().getRpcClient(nodeId)
			if pClient == nil || (bAll == false && pClient.IsConnected() == false) {
				continue
			}
			rpcClientList[count] = pClient
			count++
			if count >= cap(rpcClientList) {
				break
			}
		}
	}

	return nil, count
}

func GetRpcServer() *jrpc.RpcServer {
	return &cluster.rpcServer
}

/*
   @brief:设置配置路径
*/
func SetConfigDir(cfgDir string) {
	configDir = cfgDir
}

/*
 */
func (cls *Cluster) triggerNodeConnectEvent(bConnect bool, clientSeq uint32, nodeId int) {
	cls.locker.Lock()
	//获取触发连接的节点信息
	nodeInfo, ok := cls.mapRpc[nodeId]
	if ok == false || nodeInfo.client == nil || nodeInfo.client.GetClientSeq() != clientSeq {
		cls.locker.Unlock()
		return
	}
	cls.locker.Unlock()

	for serviceName, ser := range service.GetServiceMap() {
		if ser == nil {
			jlog.StdLogger.Error("cannot find service name ", serviceName)
			continue
		}
		if bConnect {
			//触发连接hook

			ev := service.NewNodeConnectEvent(nodeId, ser.GetEventPublisher())
			ser.PublishEvent(ev, ser)
		} else {
			//触发断开连接hook
			ev := service.NewNodeDisconnectEvent(nodeId, ser.GetEventPublisher())
			ser.PublishEvent(ev, ser)
		}

	}
}

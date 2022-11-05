package cluster

import (
	"JRpcFrame/jlog"

	"JRpcFrame/service"
	"errors"
	"sync"
)

const DynamicDiscoveryMasterName = "DiscoveryMaster"
const DynamicDiscoveryClientName = "DiscoveryClient"
const RegServiceDiscover = DynamicDiscoveryMasterName + ".RPC_RegServiceDiscover"
const SubServiceDiscover = DynamicDiscoveryClientName + ".RPC_SubServiceDiscover"

var masterService DynamicDiscoveryMaster
var clientService DynamicDiscoveryClient

type DynamicDiscoveryMaster struct {
	service.Service
	sync.Mutex
	mapNodeIsReg map[int]bool         //当前已经发现的节点id
	nodeInfo     []*DiscoveryNodeInfo //当前已经发现的节点信息
}

type DynamicDiscoveryClient struct {
	service.Service
	sync.Mutex
	funDelService FunDelNode
	funSetService FunSetNodeInfo
	localNodeId   int

	mapDiscovery map[int32]map[int32]struct{} //map[masterNodeId]map[nodeId]struct{}
}

func init() {
	masterService.SetName(DynamicDiscoveryMasterName)
	clientService.SetName(DynamicDiscoveryClientName)
}

func getDynamicDiscovery() IServiceDiscovery {
	return &clientService
}

func (ds *DynamicDiscoveryMaster) OnInit() error {
	ds.mapNodeIsReg = make(map[int]bool, 20)
	ds.SetOnNodeConnectFunc(ds.OnNodeConnected)
	ds.SetOnNodeDisconnectFunc(ds.OnNodeDisconnect)
	ds.RegNodeConnectCb()
	ds.RegNodeDisconnectCb()
	return nil
}

func (ds *DynamicDiscoveryMaster) OnStart() {
	nodeInfo := &DiscoveryNodeInfo{}
	localNodeInfo := cluster.GetLocalNodeInfo()
	if localNodeInfo.Private == true {
		return
	}

	nodeInfo.NodeId = int32(localNodeInfo.NodeId)
	nodeInfo.NodeName = localNodeInfo.NodeName
	nodeInfo.ListenAddr = localNodeInfo.ListenAddr
	nodeInfo.PublicServiceList = localNodeInfo.PublicServiceList
	nodeInfo.MaxRpcParamLen = localNodeInfo.MaxRpcParamLen

	ds.addNodeInfo(nodeInfo)
}

/*
    @brief:注册节点
	@param [in] nodeInfo:节点的信息
*/
func (ds *DynamicDiscoveryMaster) addNodeInfo(nodeInfo *DiscoveryNodeInfo) {
	ds.Lock()
	defer ds.Unlock()
	if len(nodeInfo.PublicServiceList) == 0 {

		return
	}
	ds.mapNodeIsReg[int(nodeInfo.NodeId)] = true
	ds.nodeInfo = append(ds.nodeInfo, nodeInfo)
}

/*
    @brief:删除发现的节点信息
	@param [in] nodeId:删除节点id
*/
func (ds *DynamicDiscoveryMaster) removeNodeInfo(nodeId int) {
	ds.Lock()
	defer ds.Unlock()
	//本地结点不删除
	if nodeId == 0 || nodeId == cluster.localNodeInfo.NodeId {
		return
	}
	ds.mapNodeIsReg[nodeId] = false
	for pos := 0; pos < len(ds.nodeInfo); pos++ {
		if ds.nodeInfo[pos].NodeId == int32(nodeId) {
			ds.nodeInfo = append(ds.nodeInfo[:pos], ds.nodeInfo[pos+1:]...)
			pos--
		}
	}

}

/*
    @brief:节点是否注册
	@param [in] nodeId:节点id
*/
func (ds *DynamicDiscoveryMaster) isRegNode(nodeId int) bool {
	bReg, ok := ds.mapNodeIsReg[nodeId]
	if !ok {
		return false
	} else {
		return bReg
	}
}

/*
    @brief:nodeId节点连接到本节点的connecthook函数
	@param [in] nodeId:节点id
*/
func (ds *DynamicDiscoveryMaster) OnNodeConnected(nodeId int) {
	//没注册过结点不通知
	if ds.isRegNode(nodeId) == false {
		return
	}
	//向它发布所有服务列表信息
	var discoverMasterReq DiscoverMasterReq
	discoverMasterReq.IsFull = true
	discoverMasterReq.NodeInfo = ds.nodeInfo
	discoverMasterReq.MasterNodeId = int32(cluster.GetLocalNodeInfo().NodeId)

	ds.GoNode(nodeId, SubServiceDiscover, &discoverMasterReq)
}

/*
    @brief:nodeId节点与本节点断开连接的disconnecthook函数
	@param [in] nodeId:节点id
*/
func (ds *DynamicDiscoveryMaster) OnNodeDisconnect(nodeId int) {
	if ds.isRegNode(nodeId) == false {
		return
	}

	var discoverMasterReq DiscoverMasterReq
	discoverMasterReq.MasterNodeId = int32(cluster.GetLocalNodeInfo().NodeId)
	discoverMasterReq.DelNodeId = int32(nodeId)
	//无注册过的结点不广播，避免非当前Master网络中的连接断开时通知到本网络
	//ds.CastGo(SubServiceDiscover, &discoverMasterReq)
	ds.RpcBroadCastGo(SubServiceDiscover, &discoverMasterReq)

	ds.removeNodeInfo(nodeId)
	//删除结点
	cluster.serviceDiscoveryDelNodeInfo(nodeId, true)
}

// 收到注册过来的结点
func (ds *DynamicDiscoveryMaster) RPC_RegServiceDiscover(req *DiscoverClientReq, res *Empty) error {
	if req.NodeInfo == nil {
		err := errors.New("RPC_RegServiceDiscover req is error.")
		jlog.StdLogger.Error(err.Error())
		return err
	}

	//广播给其他所有结点
	var discoverMasterReq DiscoverMasterReq
	discoverMasterReq.MasterNodeId = int32(cluster.GetLocalNodeInfo().NodeId)
	discoverMasterReq.NodeInfo = append(discoverMasterReq.NodeInfo, req.NodeInfo)
	ds.RpcBroadCastGo(SubServiceDiscover, &discoverMasterReq)

	//存入本地
	ds.addNodeInfo(req.NodeInfo)

	//初始化结点信息
	var nodeInfo NodeInfo
	nodeInfo.NodeId = int(req.NodeInfo.NodeId)
	nodeInfo.NodeName = req.NodeInfo.NodeName
	nodeInfo.Private = req.NodeInfo.Private
	nodeInfo.ServiceList = req.NodeInfo.PublicServiceList
	nodeInfo.PublicServiceList = req.NodeInfo.PublicServiceList
	nodeInfo.ListenAddr = req.NodeInfo.ListenAddr
	nodeInfo.MaxRpcParamLen = req.NodeInfo.MaxRpcParamLen
	//主动删除已经存在的结点,确保先断开，再连接
	cluster.serviceDiscoveryDelNodeInfo(nodeInfo.NodeId, true)

	//加入到本地Cluster模块中，将连接该结点
	cluster.serviceDiscoverySetNodeInfo(&nodeInfo)

	return nil
}

/*
   @brief:向所有被检测的节点广播rpcgo
*/
func (ds *DynamicDiscoveryMaster) RpcBroadCastGo(serviceMethod string, args interface{}) {
	for nodeId, bReg := range ds.mapNodeIsReg {
		if bReg {
			ds.GoNode(int(nodeId), serviceMethod, args)
		}
	}
}

func (dc *DynamicDiscoveryClient) InitDiscovery(localNodeId int, funDelNode FunDelNode, funSetNodeInfo FunSetNodeInfo) error {
	dc.localNodeId = localNodeId
	dc.funDelService = funDelNode
	dc.funSetService = funSetNodeInfo

	return nil
}

func (dc *DynamicDiscoveryClient) OnNodeStop() {
}

func (dc *DynamicDiscoveryClient) OnInit() error {
	dc.mapDiscovery = map[int32]map[int32]struct{}{}
	dc.SetOnNodeConnectFunc(dc.OnNodeConnected)
	dc.SetOnNodeDisconnectFunc(dc.OnNodeDisconnect)
	dc.RegNodeConnectCb()
	dc.RegNodeDisconnectCb()

	return nil
}
func (dc *DynamicDiscoveryClient) OnStart() {
	//2.添加并连接发现主结点
	dc.addDiscoveryMaster()
}

func (dc *DynamicDiscoveryClient) addDiscoveryMaster() {
	discoveryNodeList := cluster.GetDiscoveryNodeList()
	for i := 0; i < len(discoveryNodeList); i++ {
		if discoveryNodeList[i].NodeId == cluster.GetLocalNodeInfo().NodeId {
			continue
		}
		dc.funSetService(&discoveryNodeList[i])
	}
}

/*
    @brief:服务发现客户端添加masterNodeId监测节点需要监测的node
	@param [in] masterNodeId:监测节点id
	@param [in] nodeId:需要被masterNodeId节点监测的id
*/
func (dc *DynamicDiscoveryClient) addMasterNode(masterNodeId int32, nodeId int32) {
	dc.Lock()
	defer dc.Unlock()
	_, ok := dc.mapDiscovery[masterNodeId]
	if ok == false {
		dc.mapDiscovery[masterNodeId] = map[int32]struct{}{}
	}
	dc.mapDiscovery[masterNodeId][nodeId] = struct{}{}
}

/*
    @brief:服务发现客户端移除masterNodeId监测节点需要监测的node
	@param [in] masterNodeId:监测节点id
	@param [in] nodeId:需要被masterNodeId节点监测的id
*/
func (dc *DynamicDiscoveryClient) removeMasterNode(masterNodeId int32, nodeId int32) {
	dc.Lock()
	defer dc.Unlock()
	mapNodeId, ok := dc.mapDiscovery[masterNodeId]
	if ok == false {
		return
	}

	delete(mapNodeId, nodeId)
}

/*
   @brief:节点id有没有被监测
*/
func (dc *DynamicDiscoveryClient) findNodeId(nodeId int32) bool {
	for _, mapNodeId := range dc.mapDiscovery {
		_, ok := mapNodeId[nodeId]
		if ok == true {
			return true
		}
	}
	return false
}

func (dc *DynamicDiscoveryClient) OnNodeConnected(masterNodeId int) {
	nodeInfo := cluster.GetMasterDiscoveryNodeInfo(masterNodeId)
	if nodeInfo == nil {

		return
	}

	var req DiscoverClientReq
	req.NodeInfo = &DiscoveryNodeInfo{}
	req.NodeInfo.NodeId = int32(cluster.localNodeInfo.NodeId)
	req.NodeInfo.NodeName = cluster.localNodeInfo.NodeName
	req.NodeInfo.ListenAddr = cluster.localNodeInfo.ListenAddr
	req.NodeInfo.MaxRpcParamLen = cluster.localNodeInfo.MaxRpcParamLen
	req.NodeInfo.PublicServiceList = cluster.localNodeInfo.PublicServiceList
	//向Master服务同步本Node服务信息
	err := dc.AsyncCallNode(masterNodeId, RegServiceDiscover, &req, func(res *Empty, err error) {
		if err != nil {
			jlog.StdLogger.Error("call ", RegServiceDiscover, " is fail :", err.Error())
			return
		}
	})
	if err != nil {
		jlog.StdLogger.Error("call ", RegServiceDiscover, " is fail :", err.Error())
	}
}

func (dc *DynamicDiscoveryClient) OnNodeDisconnect(nodeId int) {
	//将Discard结点清理
	cluster.DiscardNode(nodeId)
}

//订阅发现的服务通知
func (dc *DynamicDiscoveryClient) RPC_SubServiceDiscover(req *DiscoverMasterReq) error {
	mapNodeInfo := map[int32]*DiscoveryNodeInfo{}
	for _, nodeInfo := range req.NodeInfo {
		//不对本地结点或者不存在任何公开服务的结点
		if int(nodeInfo.NodeId) == dc.localNodeId {
			continue
		}

		if cluster.IsMasterDiscoveryNode() == false && len(nodeInfo.PublicServiceList) == 1 &&
			nodeInfo.PublicServiceList[0] == DynamicDiscoveryClientName {
			continue
		}

		//遍历所有的公开服务，并筛选之
		for _, serviceName := range nodeInfo.PublicServiceList {
			nInfo := mapNodeInfo[nodeInfo.NodeId]
			if nInfo == nil {
				nInfo = &DiscoveryNodeInfo{}
				nInfo.NodeId = nodeInfo.NodeId
				nInfo.NodeName = nodeInfo.NodeName
				nInfo.ListenAddr = nodeInfo.ListenAddr
				nInfo.MaxRpcParamLen = nodeInfo.MaxRpcParamLen
				mapNodeInfo[nodeInfo.NodeId] = nInfo
			}
			nInfo.PublicServiceList = append(nInfo.PublicServiceList, serviceName)
		}

	}
	//如果为完整同步，则找出差异的结点
	var willDelNodeId []int32
	//如果不是邻居结点，则做筛选
	if req.IsFull == true {
		diffNode := dc.fullCompareDiffNode(req.MasterNodeId, mapNodeInfo)
		if len(diffNode) > 0 {
			willDelNodeId = append(willDelNodeId, diffNode...)
		}
	}

	//指定删除结点
	if req.DelNodeId > 0 && req.DelNodeId != int32(dc.localNodeId) {
		willDelNodeId = append(willDelNodeId, req.DelNodeId)
	}
	//删除不必要的结点
	for _, nodeId := range willDelNodeId {
		//nodeInfo, _ := cluster.GetNodeInfo(int(nodeId))
		//cluster.TriggerDiscoveryEvent(false, int(nodeId), nodeInfo.PublicServiceList)
		dc.removeMasterNode(req.MasterNodeId, int32(nodeId))
		if dc.findNodeId(nodeId) == false {
			dc.funDelService(int(nodeId), false)
		}
	}

	//设置新结点
	for _, nodeInfo := range mapNodeInfo {
		dc.addMasterNode(req.MasterNodeId, nodeInfo.NodeId)
		dc.setNodeInfo(nodeInfo)
		//cluster.TriggerDiscoveryEvent(true, int(nodeInfo.NodeId), nodeInfo.PublicServiceList)
	}

	return nil
}

func (dc *DynamicDiscoveryClient) fullCompareDiffNode(masterNodeId int32, mapNodeInfo map[int32]*DiscoveryNodeInfo) []int32 {
	if mapNodeInfo == nil {
		return nil
	}

	diffNodeIdSlice := make([]int32, 0, len(mapNodeInfo))
	mapNodeId := map[int32]struct{}{}
	mapNodeId, ok := dc.mapDiscovery[masterNodeId]
	if ok == false {
		return nil
	}

	//本地任何Master都不存在的，放到diffNodeIdSlice
	for nodeId, _ := range mapNodeId {
		_, ok := mapNodeInfo[nodeId]
		if ok == false {
			diffNodeIdSlice = append(diffNodeIdSlice, nodeId)
		}
	}

	return diffNodeIdSlice
}

func (dc *DynamicDiscoveryClient) setNodeInfo(nodeInfo *DiscoveryNodeInfo) {
	if nodeInfo == nil || nodeInfo.Private == true || int(nodeInfo.NodeId) == dc.localNodeId {
		return
	}

	//筛选关注的服务
	localNodeInfo := cluster.GetLocalNodeInfo()
	if len(localNodeInfo.DiscoveryService) > 0 {
		var discoverServiceSlice = make([]string, 0, 24)
		for _, pubService := range nodeInfo.PublicServiceList {
			for _, discoverService := range localNodeInfo.DiscoveryService {
				if pubService == discoverService {
					discoverServiceSlice = append(discoverServiceSlice, pubService)
				}
			}
		}
		nodeInfo.PublicServiceList = discoverServiceSlice
	}

	if len(nodeInfo.PublicServiceList) == 0 {
		return
	}

	var nInfo NodeInfo
	nInfo.ServiceList = nodeInfo.PublicServiceList
	nInfo.PublicServiceList = nodeInfo.PublicServiceList
	nInfo.NodeId = int(nodeInfo.NodeId)
	nInfo.NodeName = nodeInfo.NodeName
	nInfo.ListenAddr = nodeInfo.ListenAddr
	nInfo.MaxRpcParamLen = nodeInfo.MaxRpcParamLen
	dc.funSetService(&nInfo)
}

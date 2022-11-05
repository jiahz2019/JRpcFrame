package cluster

type DefaultDiscovery struct {
	funDelService FunDelNode
	funSetService FunSetNodeInfo
	localNodeId int
}
/*
    brief:默认的服务发现初始化
*/
func (discovery *DefaultDiscovery) InitDiscovery(localNodeId int,funDelNode FunDelNode,funSetNodeInfo FunSetNodeInfo) error{
	discovery.localNodeId = localNodeId
	discovery.funDelService = funDelNode
	discovery.funSetService = funSetNodeInfo

	//解析本地其他服务配置
	_,nodeInfoList,err := GetCluster().readLocalClusterConfig(0)
	if err != nil {
		return err
	}

	for _,nodeInfo := range nodeInfoList {
		if nodeInfo.NodeId == localNodeId {
			continue
		}

		discovery.funSetService(&nodeInfo)
	}

	return nil
}

func (discovery *DefaultDiscovery) OnNodeStop(){
}
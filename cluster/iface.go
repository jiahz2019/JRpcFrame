package cluster

type OperType int
//删除节点
type FunDelNode func (nodeId int,immediately bool)
//设置节点信息
type FunSetNodeInfo func(nodeInfo *NodeInfo)

//服务发现接口
type IServiceDiscovery interface {
	InitDiscovery(localNodeId int,funDelNode FunDelNode,funSetNodeInfo FunSetNodeInfo) error
	OnNodeStop()
}
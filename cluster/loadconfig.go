package cluster

import (
	"JRpcFrame/jlog"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type NodeInfoList struct {
	MasterDiscoveryNode []NodeInfo //用于服务发现Node
	NodeList            []NodeInfo
}

func (cls *Cluster) LoadConfig(localNodeId int) error {
	//加载本地结点的NodeList配置
	discoveryNode, nodeInfoList, err := cls.readLocalClusterConfig(localNodeId)
	if err != nil {
		return err
	}
	cls.localNodeInfo = nodeInfoList[0]
	if cls.checkDiscoveryNodeList(discoveryNode) == false {
		return fmt.Errorf("DiscoveryNode config is error!")
	}
	cls.masterDiscoveryNodeList = discoveryNode

	//读取本地服务配置
	err = cls.readLocalService(localNodeId)
	if err != nil {
		return err
	}

	//本地配置服务加到全局map(包括mapServiceNode和mapIdNode)信息中
	cls.parseLocalCfg()
	return nil
}

/*
    @brief:加载本地结点的NodeList配置
	@param [in] nodeId:本地节点id
	@return: MasterDiscoveryNode，NodeList
*/
func (cls *Cluster) readLocalClusterConfig(nodeId int) ([]NodeInfo, []NodeInfo, error) {
	var nodeInfoList []NodeInfo
	var masterDiscoverNodeList []NodeInfo
	//读取./config/cluter/ 文件夹下所有的文件
	clusterCfgPath := strings.TrimRight(configDir, "/") + "/cluster"
	fileInfoList, err := os.ReadDir(clusterCfgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("Read dir %s is fail :%+v", clusterCfgPath, err)
	}

	//读取任何文件,只读符合格式的配置,目录下的文件可以自定义分文件
	for _, f := range fileInfoList {
		if f.IsDir() == false {
			filePath := strings.TrimRight(strings.TrimRight(clusterCfgPath, "/"), "\\") + "/" + f.Name()
			localNodeInfoList, err := cls.readClusterConfig(filePath)
			if err != nil {
				return nil, nil, fmt.Errorf("read file path %s is error:%+v", filePath, err)
			}

			masterDiscoverNodeList = append(masterDiscoverNodeList, localNodeInfoList.MasterDiscoveryNode...)
			//只加载本地节点
			for _, nodeInfo := range localNodeInfoList.NodeList {
				if nodeInfo.NodeId == nodeId || nodeId == 0 {
					nodeInfoList = append(nodeInfoList, nodeInfo)
				}
			}
		}
	}
	if nodeId != 0 && (len(nodeInfoList) != 1) {
		return nil, nil, fmt.Errorf("%d configurations were found for the configuration with node ID %d!", len(nodeInfoList), nodeId)
	}

	for i, _ := range nodeInfoList {
		for j, s := range nodeInfoList[i].ServiceList {

			if strings.HasPrefix(s, "_") == false && nodeInfoList[i].Private == false {
				nodeInfoList[i].PublicServiceList = append(nodeInfoList[i].PublicServiceList, strings.TrimLeft(s, "_"))
			} else {
				//私有结点或是有“_”前缀的是私有服务不加入到Public服务列表中
				nodeInfoList[i].ServiceList[j] = strings.TrimLeft(s, "_")
			}
		}
	}

	return masterDiscoverNodeList, nodeInfoList, nil

}

/*
    @brief:加载本地结点的service配置
	@param [in] nodeId:本地节点id
*/
func (cls *Cluster) readLocalService(localNodeId int) error {
	clusterCfgPath := strings.TrimRight(configDir, "/") + "/cluster"
	fileInfoList, err := os.ReadDir(clusterCfgPath)
	if err != nil {
		return fmt.Errorf("Read dir %s is fail :%+v", clusterCfgPath, err)
	}

	//读取任何文件,只读符合格式的配置,目录下的文件可以自定义分文件
	for _, f := range fileInfoList {
		if f.IsDir() == false {
			filePath := strings.TrimRight(strings.TrimRight(clusterCfgPath, "/"), "\\") + "/" + f.Name()
			currGlobalCfg, serviceConfig, mapNodeService, err := cls.readServiceConfig(filePath)
			if err != nil {
				continue
			}

			if currGlobalCfg != nil {
				cls.globalCfg = currGlobalCfg
			}
			for _, s := range cls.localNodeInfo.ServiceList {
				for {
					//取公共服务配置
					pubCfg, ok := serviceConfig[s]
					if ok == true {
						cls.localServiceCfg[s] = pubCfg
					}

					//如果结点也配置了该服务，则覆盖之
					nodeService, ok := mapNodeService[localNodeId]
					if ok == false {
						break
					}
					sCfg, ok := nodeService[s]
					if ok == false {
						break
					}

					cls.localServiceCfg[s] = sCfg
					break
				}
			}
		}

	}
	return nil
}

/*
    @brief:读取单个配置文件中的所有节点的list和所有服务发现节点的list
	@param [in] filepath:文件绝对路径
	@return: NodeInfoList
*/
func (cls *Cluster) readClusterConfig(filepath string) (*NodeInfoList, error) {
	c := &NodeInfoList{}
	d, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(d, c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

/*
    @brief:读取单个配置文件中Global,Service(公共服务)与NodeService(各个节点的服务)
	@param [in] filepath:文件绝对路径
*/
func (cls *Cluster) readServiceConfig(filepath string) (interface{}, map[string]interface{}, map[int]map[string]interface{}, error) {
	c := map[string]interface{}{}
	//读取配置
	d, err := os.ReadFile(filepath)
	if err != nil {
		return nil, nil, nil, err
	}
	err = json.Unmarshal(d, &c)
	if err != nil {
		return nil, nil, nil, err
	}

	GlobalCfg, ok := c["Global"]
	serviceConfig := map[string]interface{}{}
	serviceCfg, ok := c["Service"]
	if ok == true {
		serviceConfig = serviceCfg.(map[string]interface{})
	}

	mapNodeService := map[int]map[string]interface{}{}
	nodeServiceCfg, ok := c["NodeService"]
	if ok == true {
		nodeServiceList := nodeServiceCfg.([]interface{})
		for _, v := range nodeServiceList {
			serviceCfg := v.(map[string]interface{})
			nodeId, ok := serviceCfg["NodeId"]
			if ok == false {
				jlog.StdLogger.Fatal("NodeService list not find nodeId field")
			}
			mapNodeService[int(nodeId.(float64))] = serviceCfg
		}
	}
	return GlobalCfg, serviceConfig, mapNodeService, nil
}

func (cls *Cluster) parseLocalCfg() {
	cls.mapIdNode[cls.localNodeInfo.NodeId] = cls.localNodeInfo

	for _, sName := range cls.localNodeInfo.ServiceList {
		if _, ok := cls.mapServiceNode[sName]; ok == false {
			cls.mapServiceNode[sName] = make(map[int]struct{})
		}
		//jlog.StdLogger.Debug("Add mapServiceNode:",sName,"-",cls.localNodeInfo.NodeId)
		cls.mapServiceNode[sName][cls.localNodeInfo.NodeId] = struct{}{}
	}
}

/*
    @brief:确保服务发现节点列表中节点的id及监听地址各不同
	@param [in] discoverMasterNode:需要检查的服务发现节点列表
	@return: 服务发现节点列表是否合理
*/
func (cls *Cluster) checkDiscoveryNodeList(discoverMasterNode []NodeInfo) bool {
	for i := 0; i < len(discoverMasterNode)-1; i++ {
		for j := i + 1; j < len(discoverMasterNode); j++ {
			if discoverMasterNode[i].NodeId == discoverMasterNode[j].NodeId ||
				discoverMasterNode[i].ListenAddr == discoverMasterNode[j].ListenAddr {
				return false
			}
		}
	}

	return true
}

/*
    @brief:查询本地节点的服务配置
	@param [in] serviceName:需要查询的service Name
	@return: 服务配置
*/
func (cls *Cluster) GetServiceCfg(serviceName string) interface{} {
	serviceCfg, ok := cls.localServiceCfg[serviceName]
	if ok == false {
		return nil
	}

	return serviceCfg
}

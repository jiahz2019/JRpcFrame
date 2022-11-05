package service

var serviceMgr *ServiceMgr
//service管理器，每个node有一个service管理器
type ServiceMgr struct{
	mapService map[string]IService
}
func init(){
	serviceMgr = NewServiceMgr()
}
/*
    @brief:serviceMgr的构造函数
*/
func NewServiceMgr() *ServiceMgr{
	s:=&ServiceMgr{
        mapService : map[string]IService{},
	} 
	return s
}
/*
    @brief:初始化serviceMgr中的所有service，并执行相应的hookfunc
	@param [in] chanCloseSig:service的关闭信号
*/
func (sm *ServiceMgr)InitServiceMgr(chanCloseSig chan bool){

	closeSig=chanCloseSig

	for _,s := range sm.mapService{
		err := s.OnInit()
		if err != nil {
			panic(err)
		}
	}
}

/*
    @brief:SeviceMgr添加一个服务
	@param [in] s:需要添加的服务
*/
func (sm *ServiceMgr)Add(s IService) bool {
	_,ok := sm.mapService[s.GetName()]
	if ok == true {
		return false
	}

	sm.mapService[s.GetName()] = s
	return true
}
/*
    @brief:根据名字获取service
	@param [in] s:service Name
*/
func  (sm *ServiceMgr)GetService(serviceName string) IService {
	s,ok := sm.mapService[serviceName]
	if ok == false {
		return nil
	}

	return s
}
/*
    @brief:ServiceMgr启动所有service
*/
func (sm *ServiceMgr)StartAll(){
	for _,s := range sm.mapService {
		s.Start()
	}
}
/*
    @brief:ServiceMgr停止等待所有service
*/
func (sm *ServiceMgr)WaitStopAll(){
	for _,s := range sm.mapService {
		if s!=nil{
			s.Wait()
		}
	}

}


func GetServiceMgr()*ServiceMgr{
	return serviceMgr
}

func GetServiceMap() map[string]IService{
	return serviceMgr.mapService
}
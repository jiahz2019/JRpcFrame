package service

import (
	"JRpcFrame/event"
	"JRpcFrame/jlog"
	"fmt"
	"reflect"
)

const InitModuleId = 1e9

type Module struct {
	moduleId   uint32             //模块Id
	moduleName string             //模块名称
	parent     IModule            //父亲
	self       IModule            //自己
	child      map[uint32]IModule //孩子们

	//根结点
	ancestor     IModule            //始祖
	seedModuleId uint32             //模块id种子
	descendants  map[uint32]IModule //始祖的后裔们

	//事件管道
	eventPublisher event.IEventPublisher
}

/*
   @brief:生成模块id
*/
func (m *Module) NewModuleId() uint32 {
	m.ancestor.getBaseModule().(*Module).seedModuleId += 1
	return m.ancestor.getBaseModule().(*Module).seedModuleId
}

/*
    @brief:添加模块
	@param [in] moudle:需要添加的模块
	@return:添加的模块的id
*/
func (m *Module) AddModule(module IModule) (uint32, error) {
	//没有事件监听器不允许加入其他模块
	if m.GetService().GetEventListener() == nil {
		return 0, fmt.Errorf("module %+v Event listener is nil", m.self)
	}

	pAddModule := module.getBaseModule().(*Module)
	if pAddModule.GetModuleId() == 0 {
		pAddModule.moduleId = m.NewModuleId()
	}

	if m.child == nil {
		m.child = map[uint32]IModule{}
	}
	_, ok := m.child[module.GetModuleId()]
	if ok == true {
		return 0, fmt.Errorf("exists module id %d", module.GetModuleId())
	}
	pAddModule.self = module
	pAddModule.parent = m.self

	pAddModule.ancestor = m.ancestor
	pAddModule.moduleName = reflect.Indirect(reflect.ValueOf(module)).Type().Name()
	pAddModule.eventPublisher = event.NewEventPublisher()

	err := module.OnInit()
	if err != nil {
		return 0, err
	}

	m.child[module.GetModuleId()] = module
	m.ancestor.getBaseModule().(*Module).descendants[module.GetModuleId()] = module

	jlog.StdLogger.Info("Add module ", module.GetModuleName(), " completed")
	return module.GetModuleId(), nil
}

/*
    @brief:释放模块
	@param [in] moudleId:需要释放的模块id

*/
func (m *Module) ReleaseModule(moduleId uint32) {
	pModule := m.GetModule(moduleId).getBaseModule().(*Module)

	//释放子孙
	for id := range pModule.child {
		m.ReleaseModule(id)
	}

	pModule.self.OnRelease()
	pModule.GetEventPublisher().Destroy()
	jlog.StdLogger.Debug("Release module ", pModule.GetModuleName())

	delete(m.child, moduleId)
	delete(m.ancestor.getBaseModule().(*Module).descendants, moduleId)

	//清理被删除的Module
	pModule.self = nil
	pModule.parent = nil
	pModule.child = nil

	pModule.ancestor = nil
	pModule.descendants = nil

}

func (m *Module) OnInit() error {
	return nil
}
func (m *Module) OnRelease() {
}
func (m *Module) SetModuleId(moduleId uint32) bool {
	if m.moduleId > 0 {
		return false
	}

	m.moduleId = moduleId
	return true
}

func (m *Module) GetModuleId() uint32 {
	return m.moduleId
}

func (m *Module) GetModuleName() string {
	return m.moduleName
}

func (m *Module) GetAncestor() IModule {
	return m.ancestor
}

func (m *Module) GetModule(moduleId uint32) IModule {
	iModule, ok := m.GetAncestor().getBaseModule().(*Module).descendants[moduleId]
	if ok == false {
		return nil
	}
	return iModule
}

func (m *Module) getBaseModule() IModule {
	return m
}

func (m *Module) GetParent() IModule {
	return m.parent
}

func (m *Module) GetService() IService {

	return m.GetAncestor().(IService)
}

func (m *Module) GetEventPublisher() event.IEventPublisher {
	return m.eventPublisher

}

func (m *Module) PublishEvent(ev event.IEvent, s IService) {
	m.eventPublisher.PublishEvent(ev, s.GetEventListener())
}

func (m *Module) BroadCastEvent(ev event.IEvent) {
	m.eventPublisher.BroadCastEvent(ev)
}

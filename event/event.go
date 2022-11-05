package event

import (
	"JRpcFrame/jlog"
	"fmt"
	"runtime"
	"sync"
)

var emptyEvent Event

type EventCallBack func(event IEvent)

type Event struct {
	Type      EventType       //事件类型
	Data      interface{}     //事件数据
	Publisher IEventPublisher //事件发出者
	ref       bool
}

//事件监听者
type EventListener struct {
	IEventChannel

	locker sync.RWMutex
	//记录listener的已经监听的publisher，及相应的回调函数
	mapBindPublisherCb map[EventType]map[IEventPublisher]EventCallBack
}

//事件发布者
type EventPublisher struct {
	locker sync.RWMutex
	//记录该publisher的已经注册的listener
	mapRegListener map[EventType]map[IEventListener]interface{}
}

/*
   @brief:构造函数
*/
func NewEventListener(eventChannel IEventChannel) IEventListener {
	e := EventListener{}
	e.IEventChannel = eventChannel
	e.mapBindPublisherCb = map[EventType]map[IEventPublisher]EventCallBack{}

	return &e
}

/*
   @brief:构造函数
*/
func NewEventPublisher() IEventPublisher {
	e := EventPublisher{}
	e.mapRegListener = map[EventType]map[IEventListener]interface{}{}

	return &e
}

/*
    @brief:注册listener监听publisher的类型eventType事件
	@param [in] eventType:监听事件类型
	@param [in] publisher:将要发出事件的publisher
	@param [in] callBack:回调函数
*/
func (listener *EventListener) RegEventCb(eventType EventType, publisher IEventPublisher, callback EventCallBack) {
	publisher.addRegListenerInfo(eventType, listener)
	listener.addBindEvent(eventType, publisher, callback)
}

/*
    @brief:注销listener监听publisher的类型eventType事件
	@param [in] eventType:注销的事件类型
	@param [in] publisher:发出事件的publisher
*/
func (listener *EventListener) UnRegEventCb(eventType EventType, publisher IEventPublisher) {
	publisher.removeRegListenerInfo(eventType, listener)
	listener.removeBindEvent(eventType, publisher)
}

/*
    @brief:listener处理事件
	@param [in] ev:处理的事件
*/
func (listener *EventListener) EventHandler(ev IEvent) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 4096)
			l := runtime.Stack(buf, false)
			errString := fmt.Sprint(r)
			jlog.StdLogger.Error("core dump info[", errString, "]\n", string(buf[:l]))
		}
	}()

	callback, ok := listener.mapBindPublisherCb[ev.GetEventType()][ev.GetPublisher()]
	if ok == false {
		return
	}
	callback(ev)

}

/*
   @brief:将事件与其监听者绑定
*/
func (listener *EventListener) addBindEvent(eventType EventType, publisher IEventPublisher, callback EventCallBack) {
	listener.locker.Lock()
	defer listener.locker.Unlock()

	if _, ok := listener.mapBindPublisherCb[eventType]; ok == false {
		listener.mapBindPublisherCb[eventType] = map[IEventPublisher]EventCallBack{}
	}

	listener.mapBindPublisherCb[eventType][publisher] = callback
}

/*
   @brief:移除事件与其监听者的绑定
*/
func (listener *EventListener) removeBindEvent(eventType EventType, publisher IEventPublisher) {
	listener.locker.Lock()
	defer listener.locker.Unlock()
	if _, ok := listener.mapBindPublisherCb[eventType]; ok == true {
		delete(listener.mapBindPublisherCb[eventType], publisher)
	}
}

/*
    @brief:publisher向listener发布事件
	@param [in] ev:发布的事件
	@param [in] listener:接受事件的listener
*/
func (publisher *EventPublisher) PublishEvent(ev IEvent, listener IEventListener) {
	listener.PushEvent(ev)
}

/*
    @brief:publisher广播事件
	@param [in] ev:广播的事件

*/
func (publisher *EventPublisher) BroadCastEvent(ev IEvent) {
	publisher.locker.Lock()
	defer publisher.locker.Unlock()

	for eventTyp, mapListener := range publisher.mapRegListener {
		if eventTyp == ev.GetEventType() && mapListener != nil {
			for listener, _ := range mapListener {
				publisher.PublishEvent(ev, listener)
			}
		}
	}
}

/*
   brief:取消publisher，所有listener不在监听publisher
*/
func (publisher *EventPublisher) Destroy() {
	publisher.locker.Lock()
	defer publisher.locker.Unlock()
	for eventTyp, mapListener := range publisher.mapRegListener {
		if mapListener == nil {
			continue
		}

		for listener := range mapListener {
			listener.UnRegEventCb(eventTyp, publisher)
		}
	}
}

/*
   @brief:publisher添加注册的listener
*/
func (publisher *EventPublisher) addRegListenerInfo(eventType EventType, listener IEventListener) {
	publisher.locker.Lock()
	defer publisher.locker.Unlock()
	if publisher.mapRegListener == nil {
		publisher.mapRegListener = map[EventType]map[IEventListener]interface{}{}
	}

	if _, ok := publisher.mapRegListener[eventType]; ok == false {
		publisher.mapRegListener[eventType] = map[IEventListener]interface{}{}
	}
	publisher.mapRegListener[eventType][listener] = nil

}

/*
   @brief:publisher移除注册的listener
*/
func (publisher *EventPublisher) removeRegListenerInfo(eventType EventType, listener IEventListener) {
	if _, ok := publisher.mapRegListener[eventType]; ok == true {
		delete(publisher.mapRegListener[eventType], listener)
	}
}

func (e *Event) Reset() {
	*e = emptyEvent
}

func (e *Event) IsRef() bool {
	return e.ref
}

func (e *Event) Ref() {
	e.ref = true
}

func (e *Event) UnRef() {
	e.ref = false
}

func (e *Event) GetEventType() EventType {
	return e.Type
}

func (e *Event) GetPublisher() IEventPublisher {
	return e.Publisher
}

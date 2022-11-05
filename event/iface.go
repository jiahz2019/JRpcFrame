package event


type IEvent interface {
	GetEventType() EventType
	GetPublisher() IEventPublisher
}
type IEventChannel interface {
	PushEvent(ev IEvent) error
}

type IEventListener interface {
	IEventChannel

	EventHandler(ev IEvent)
	RegEventCb(eventType EventType, publisher IEventPublisher,callback EventCallBack)
	UnRegEventCb(eventType EventType, publisher IEventPublisher)
	addBindEvent(eventType EventType, publisher IEventPublisher,callback EventCallBack)
	removeBindEvent(eventType EventType, publisher IEventPublisher)
}

type IEventPublisher interface {
	PublishEvent(e IEvent,listener IEventListener)
    BroadCastEvent(e IEvent)
	Destroy()

	addRegListenerInfo(eventType EventType,listener  IEventListener)
	removeRegListenerInfo(eventType EventType,listener  IEventListener)
}


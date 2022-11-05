package event

type EventType int

const(
	ServiceRpcRequestEvent 		EventType = -1
	ServiceRpcResponseEvent  	EventType = -2
	ServiceNodeConnectEvent     EventType = -3
	ServiceNodeDisconnectEvent     EventType = -4
)
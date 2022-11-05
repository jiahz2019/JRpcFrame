package  jnet

import (
	"net"
)

type Agent interface {
	Run()
	OnClose()
}
type IConn interface{
	GetConnID() uint32
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Close()
	Destroy()

}


package jnet

import (
	"JRpcFrame/jlog"
	"errors"
	"sync"
)

/*
	连接管理模块
*/
type ConnManager struct {
	connections map[uint32]IConn //管理的连接信息，sessionID-conn
	connLock    sync.RWMutex     //读写连接的读写锁
}

/*
	brief:连接管理器构造函数
*/
func NewConnManager() *ConnManager {
	c := ConnManager{
		connections: make(map[uint32]IConn),
	}
	return &c
}

/*
	brief:添加连接
*/
func (connMgr *ConnManager) Add(conn IConn) {
	//保护共享资源Map 加写锁
	connMgr.connLock.Lock()
	defer connMgr.connLock.Unlock()

	//将conn连接添加到ConnMananger中
	connMgr.connections[conn.GetConnID()] = conn

}

/*
	brief:删除连接
*/
func (connMgr *ConnManager) Remove(conn IConn) {
	//保护共享资源Map 加写锁
	connMgr.connLock.Lock()
	defer connMgr.connLock.Unlock()

	//删除连接信息
	delete(connMgr.connections, conn.GetConnID())
}

/*
	brief:根据connid获取连接
	param [in] connID:连接id
*/
func (connMgr *ConnManager) Get(connID uint32) (IConn, error) {
	//保护共享资源Map 加读锁
	connMgr.connLock.RLock()
	defer connMgr.connLock.RUnlock()

	if conn, ok := connMgr.connections[connID]; ok {
		return conn, nil
	} else {
		return nil, errors.New("connection not found")
	}
}

/*
	brief:获取当前连接数
*/
func (connMgr *ConnManager) Len() int {
	return len(connMgr.connections)
}

/*
   清除并停止所有连接
*/
func (connMgr *ConnManager) ClearConn() {
	//保护共享资源Map 加写锁
	connMgr.connLock.Lock()
	defer connMgr.connLock.Unlock()

	needClear := false
	//停止并删除全部的连接信息
	for _, conn := range connMgr.connections {
		//停止
		conn.Close()
		needClear = true
	}

	if needClear {
		connMgr.connections = make(map[uint32]IConn)
	}

	jlog.StdLogger.Info("Clear All Connections successfully: conn num = ", connMgr.Len())
}

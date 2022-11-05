package jnet

import (
	"JRpcFrame/jlog"
	"net"
	"strings"
	"sync"
	"time"
)

//iServer 接口实现，定义一个Server服务类
type TcpServer struct {
	name      string //服务器的名称
	ipVersion string
	Addr      string
	ip        string
	port      string

	listener *net.TCPListener
	isClosed bool
	NewAgent func(*TcpConn) Agent

	wgLn    sync.WaitGroup
	wgConns sync.WaitGroup

	MaxConnNum int
	connMgr    *ConnManager //当前Server的链接管理器
	// msg parser
	MinMsgLen    uint32
	MaxMsgLen    uint32
	LittleEndian bool
	msgParser    *MsgParser

	onConnStart func(con *TcpConn)  //该Server的连接创建时Hook函数
	onConnClose func(conn *TcpConn) //该Server的连接断开时的Hook函数
}

/*
   brief:MServer的构造方法
*/
func NewTcpServer(name string, addr string, MinMsgLen uint32, MaxMsgLen uint32, LittleEndian bool) *TcpServer {
	splitAddr := strings.Split(addr, ":")
	if len(splitAddr) != 2 {
		jlog.StdLogger.Fatal("listen addr is error :", addr)
	}
	s := &TcpServer{
		name:      name,
		ipVersion: "tcp4",
		Addr:      addr,

		isClosed:   false,
		MaxConnNum: 1000,

		MinMsgLen:    MinMsgLen,
		MaxMsgLen:    MaxMsgLen,
		LittleEndian: LittleEndian,
		connMgr:      NewConnManager(),
	}
	mp := NewMsgParser(MinMsgLen, MaxMsgLen, LittleEndian)
	s.msgParser = mp
	s.ip = splitAddr[0]
	s.port = splitAddr[1]
	return s
}

/*
   brief:开启服务
*/
func (server *TcpServer) Start() {
	go server.run()
}

func (s *TcpServer) run() {
	s.wgLn.Add(1)
	defer s.wgLn.Done()

	addr, err := net.ResolveTCPAddr(s.ipVersion, s.Addr)
	if err != nil {
		jlog.StdLogger.Error("resolve tcp addr err: ", err.Error())
		return
	}

	//获取监听地址
	listener, err := net.ListenTCP(s.ipVersion, addr)
	if err != nil {
		jlog.StdLogger.Errorf("listen ", s.ipVersion, " err %s", err.Error())
		return
	} else {
		s.listener = listener
	}

	var cid uint32
	cid = 0
	var tempDelay time.Duration
	//开始监听
	for {
		if s.isClosed {
			break
		}
		conn, err := s.listener.AcceptTCP()
		if err != nil {
			//如果是超时错误，重新尝试
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				jlog.StdLogger.Error("accept error:", err.Error(), "; retrying in ", tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			jlog.StdLogger.Errorf("Accept err %s ", err.Error())
			return
		}
		//开启nagle
		conn.SetNoDelay(true)
		tempDelay = 0

		//连接数超过了最大连接数则断开连接
		if s.GetConnMgr().Len() >= s.MaxConnNum {
			jlog.StdLogger.Error("too much conn ", s.connMgr.Len())
			conn.Close()
			continue
		}

		//接受新的连接
		dealConn := NewTcpConn(conn, cid, s.msgParser)
		s.connMgr.Add(dealConn)
		if s.onConnStart != nil {
			s.onConnStart(dealConn)
		}
		cid++

		agent := s.NewAgent(dealConn)
		s.wgConns.Add(1)
		go func() {
			agent.Run()

			//该连接结束，清理资源
			if s.onConnClose != nil {
				s.onConnClose(dealConn)
			}
			dealConn.Close()
			s.connMgr.Remove(dealConn)
			agent.OnClose()

			s.wgConns.Done()
		}()
	}
}

/*
   brief:关闭server
*/
func (server *TcpServer) Close() {
	if server.isClosed {
		return
	}
	server.isClosed = true
	server.listener.Close()
	server.wgLn.Wait()

	server.connMgr.ClearConn()
	server.wgConns.Wait()
}

/*
   brief:获取MServer的名字
*/
func (s *TcpServer) GetName() string {
	return s.name
}

/*
   brief:获取MServer的ip
*/
func (s *TcpServer) GetHost() string {
	return s.ip
}

/*
   brief:获取MServer的端口
*/
func (s *TcpServer) GetPort() string {
	return s.port
}

/*
   brief:获取连接管理器
*/
func (s *TcpServer) GetConnMgr() *ConnManager {
	return s.connMgr
}

/*
    brief:设置该Server的连接创建时Hook函数
	param [in] hookfun:需要设置的Start hook函数
*/
func (s *TcpServer) SetOnConnStart(hookFunc func(*TcpConn)) {
	s.onConnStart = hookFunc
}

/*
    brief:设置该Server的连接断开时Hook函
	param [in] hookfun:需要设置的Stop hook函数
*/
func (s *TcpServer) SetOnConnClose(hookFunc func(*TcpConn)) {
	s.onConnClose = hookFunc
}

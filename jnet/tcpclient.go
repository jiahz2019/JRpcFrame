package jnet

import (
	"JRpcFrame/jlog"
	"net"
	"strings"
	"sync"
	"time"
)

//客户端，实现了IServer接口，用于本地服务器和其他服务器通信
type TcpClient struct {
	sync.Mutex
	name       string
	ipVersion  string //ip类型
	RemoteAddr string //ip:port
	remoteIP   string //远程服务器ip
	remotePort string //远程服务器port
	isClosed   bool
	wg         sync.WaitGroup

	NewAgent        func(*TcpConn) Agent
	connNum         int
	connMgr         *ConnManager //客户端的所有连接
	maxConnectTimes int          //最大尝试连接次数
	AutoReconnect   bool
	ConnectInterval time.Duration
	// msg parser
	MinMsgLen    uint32
	MaxMsgLen    uint32
	LittleEndian bool
	msgParser    *MsgParser

	onConnStart func(conn *TcpConn)
	onConnClose func(conn *TcpConn)
}

/*
   brief:client的构造函数
*/
func NewTcpClient(clientName string, remoteAddr string, MinMsgLen uint32, MaxMsgLen uint32, LittleEndian bool) *TcpClient {

	c := TcpClient{
		name:       clientName,
		ipVersion:  "tcp4",
		RemoteAddr: remoteAddr,
		isClosed:   false,

		maxConnectTimes: 5,
		ConnectInterval: 3 * time.Second,
		AutoReconnect:   false,

		MinMsgLen:    MinMsgLen,
		MaxMsgLen:    MaxMsgLen,
		LittleEndian: LittleEndian,
		connNum:      1,
		connMgr:      NewConnManager(), //作为客户端连接一般只有一个
	}
	if remoteAddr == "" {
		c.remoteIP = ""
		c.remotePort = ""

	} else {
		splitAddr := strings.Split(remoteAddr, ":")
		if len(splitAddr) != 2 {
			jlog.StdLogger.Fatal("client remote addr is error :", remoteAddr)
		}
		c.remoteIP = splitAddr[0]
		c.remotePort = splitAddr[1]
	}

	mp := NewMsgParser(MinMsgLen, MaxMsgLen, LittleEndian)
	c.msgParser = mp
	return &c
}

/*
   brief:client启动
*/
func (client *TcpClient) Start() {

	for i := 0; i < client.connNum; i++ {
		client.wg.Add(1)
		go client.connect()
	}
}

/*
   brief:client连接
*/
func (client *TcpClient) connect() {
	defer client.wg.Done()
	var cid uint32
	cid = 1
reconnect:
	conn := client.dial()
	if conn == nil {
		return
	}

	if client.isClosed {
		conn.Close()
		return
	}

	tcpConn := NewTcpConn(conn, cid, client.msgParser)
	client.connMgr.Add(tcpConn)
	if client.onConnStart != nil {
		client.onConnStart(tcpConn)
	}
	cid++

	agent := client.NewAgent(tcpConn)
	agent.Run()

	//连接结束，清理资源
	if client.onConnClose != nil {
		client.onConnClose(tcpConn)
	}
	tcpConn.Close()
	client.connMgr.Remove(tcpConn)
	agent.OnClose()

	//若设置了自动重连，则会重新开始
	if client.AutoReconnect {
		time.Sleep(client.ConnectInterval)
		goto reconnect
	}
}

/*
    brief:client关闭
	param [in] waitDone:是否等待client的所有conn的agent结束
*/
func (client *TcpClient) Close(waitDone bool) {
	if client.isClosed {
		return
	}
	client.isClosed = true
	client.connMgr.ClearConn()
	if waitDone {
		client.wg.Wait()
	}
}

func (client *TcpClient) dial() *net.TCPConn {
	//获得远程服务器的addr
	addr, err := net.ResolveTCPAddr(client.ipVersion, client.RemoteAddr)
	if err != nil {
		jlog.StdLogger.Error("resolve tcp addr err: ", err.Error())
		return nil
	}
	//不断重连，直至client关闭或是连接成功,或超过最大连接次数
	var connecttimes = client.maxConnectTimes
	for {
		conn, err := net.DialTCP(client.ipVersion, nil, addr)
		connecttimes--
		if client.isClosed {
			return conn
		} else if err == nil && conn != nil {
			conn.SetNoDelay(true)
			return conn
		} else if connecttimes <= 0 {
			jlog.StdLogger.Error("connect exceed maxConnectTimes ")
			return nil
		}

		jlog.StdLogger.Warn("connect to ", addr, " error:", err.Error())
		time.Sleep(client.ConnectInterval)
		continue
	}
}

/*
   brief:获取Client的连接管理器
*/
func (client *TcpClient) GetConnMgr() *ConnManager {
	return client.connMgr
}

func (client *TcpClient) SetOnConnStart(hookFunc func(*TcpConn)) {
	client.onConnStart = hookFunc
}

func (client *TcpClient) SetOnConnClose(hookFunc func(*TcpConn)) {
	client.onConnClose = hookFunc
}

func (client *TcpClient) GetName() string {
	return client.name
}

func (client *TcpClient) GetHost() string {
	return client.remoteIP
}

func (client *TcpClient) GetPort() string {
	return client.remotePort
}

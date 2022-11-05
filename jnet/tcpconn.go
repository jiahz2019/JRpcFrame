package jnet

import (
	"JRpcFrame/jlog"
	"errors"
	"net"
	"sync"
	"time"
	//"github.com/forgoer/openssl"
)

var tcpSeq uint32

//连接
type TcpConn struct {
	conn   *net.TCPConn //当前连接的socket TCP套接字
	connID uint32       //当前连接的ID ,ID全局唯一
	sync.Mutex
	writeChan chan []byte //有缓冲管道，用于读、写两个goroutine之间的消息通信

	isClosed bool

	msgParser *MsgParser

	seqMap     map[uint32]interface{} //为需要seq的提供额外的map
	seqMapLock sync.RWMutex

	property     map[string]interface{} //链接属性
	propertyLock sync.RWMutex           //保护链接属性修改的锁

}

/*
   brief:MConnection的构造方法
*/
func NewTcpConn(conn *net.TCPConn, connID uint32, msgParser *MsgParser) *TcpConn {
	//初始化MConn属性
	c := &TcpConn{
		conn:      conn,
		connID:    connID,
		isClosed:  false,
		msgParser: msgParser,
		writeChan: make(chan []byte, 1024),

		property: make(map[string]interface{}),
		seqMap:   make(map[uint32]interface{}),
	}
	tcpSeq = 0
	go func() {
		for b := range c.writeChan {
			if b == nil {
				break
			}
			_, err := conn.Write(b)
			c.msgParser.ReleaseByteSlice(b)

			if err != nil {
				break
			}
		}

		c.conn.Close()
		c.Lock()
		freeChannel(c)
		c.isClosed = true
		c.Unlock()

	}()
	return c
}

/*
    brief:向conn中写入数据
	param [in] b:需要写入的数据
*/
//b不能在其他协程被修改
func (conn *TcpConn) Write(b []byte) error {
	conn.Lock()
	defer conn.Unlock()
	if conn.isClosed || b == nil {
		conn.ReleaseByteSlice(b)
		return errors.New("conn is close")
	}

	return conn.doWrite(b)
}

/*
    brief:从conn中读出数据
	param [in][out] b:读出的数据
	return: 读出的字节数
*/
func (conn *TcpConn) Read(b []byte) (int, error) {
	return conn.conn.Read(b)
}

/*
    brief:以message的格式写入
	param [in] protocolType:协议类型
	param [in] body:写入的数据
*/
func (conn *TcpConn) WriteMsg(protocolType byte, body []byte) error {
	if conn.isClosed {
		return errors.New("conn is close")
	}
	err := conn.msgParser.WriteMsg(conn,
		&Message{
			BodyLen:      uint32(len(body)),
			Seq:          tcpSeq,
			ProtocolType: protocolType,
			Body:         body},
	)
	if err != nil {
		tcpSeq++
	}

	return err
}

/*
    brief:读出message
	return:协议类型,读出的数据

*/
func (conn *TcpConn) ReadMsg() (byte, []byte, error) {
	if conn.isClosed {
		return 0, nil, errors.New("conn is close")
	}
	msg, err := conn.msgParser.ReadMsg(conn)
	if err != nil {
		return 0, nil, err
	}
	return msg.GetProtocolType(), msg.GetBody(), nil
}

/*
   brief:强制关闭
*/
func (conn *TcpConn) Destroy() {
	conn.Lock()
	defer conn.Unlock()

	conn.doDestroy()
}

/*
   brief:关闭
*/
func (conn *TcpConn) Close() {
	conn.Lock()
	defer conn.Unlock()
	if conn.isClosed {
		return
	}
	conn.doWrite(nil)
	conn.isClosed = true

}

/*
    brief:释放经过readmsg读出的body
	param [in] body:经过readmsg读出的body
*/
func (conn *TcpConn) ReleaseReadMsg(body []byte) {
	conn.msgParser.ReleaseByteSlice(body)
}
func (conn *TcpConn) doDestroy() {
	//不管是否有数据，强制中断连接
	conn.conn.SetLinger(0)
	conn.conn.Close()

	if !conn.isClosed {
		close(conn.writeChan)
		conn.isClosed = true
	}
}

/*
   brief:释放write chan
*/
func freeChannel(conn *TcpConn) {
	for len(conn.writeChan) > 0 {
		byteBuff := <-conn.writeChan
		if byteBuff != nil {
			conn.ReleaseByteSlice(byteBuff)
		}
	}
}

func (conn *TcpConn) doWrite(b []byte) error {
	if len(conn.writeChan) == cap(conn.writeChan) {
		conn.ReleaseByteSlice(b)
		jlog.StdLogger.Error("close conn: channel full")
		conn.doDestroy()
		return errors.New("close conn: channel full")
	}

	conn.writeChan <- b
	return nil
}

/*
   brief:释放数组切片
*/
func (tcpConn *TcpConn) ReleaseByteSlice(byteBuff []byte) {
	tcpConn.msgParser.ReleaseByteSlice(byteBuff)
}

/*
   brief:从当前连接获取原始的socket TCPConn
*/
func (conn *TcpConn) GetTCPConnection() *net.TCPConn {
	return conn.conn
}

/*
   brief:获取当前连接的id
*/
func (conn *TcpConn) GetConnID() uint32 {
	return conn.connID
}

/*
   brief:获取远程客户端地址信息
*/
func (conn *TcpConn) RemoteAddr() net.Addr {
	return conn.conn.RemoteAddr()
}

/*
   brief:获取本地地址信息
*/
func (conn *TcpConn) LocalAddr() net.Addr {
	return conn.conn.LocalAddr()
}

/*
   brief:当前连接是否关闭
*/
func (conn *TcpConn) IsClose() bool {
	return conn.isClosed
}

/*
    brief:设置seqmap属性
	param [in] seq:包序号
	param [in] value:值
*/
func (conn *TcpConn) SetSeqMap(seq uint32, value interface{}) {
	conn.seqMapLock.Lock()
	defer conn.seqMapLock.Unlock()

	conn.seqMap[seq] = value
}

/*
    brief:获取seqmap属性
	param [in]  seq:包序号
	return:值
*/
func (conn *TcpConn) GetSeqMap(seq uint32) (interface{}, error) {
	conn.seqMapLock.Lock()
	defer conn.seqMapLock.Unlock()

	if value, ok := conn.seqMap[seq]; ok {
		return value, nil
	} else {
		return nil, errors.New("seqmap no seq found")
	}
}

/*
    brief:移除seqmap属性
	param [in] seq:需要移除的包序号
*/
func (conn *TcpConn) RemoveSeqMap(seq uint32) {
	conn.seqMapLock.Lock()
	defer conn.seqMapLock.Unlock()

	delete(conn.seqMap, seq)
}

/*
    brief:设置链接属性
	param [in] key:属性名字
	param [in] value:属性值
*/
func (conn *TcpConn) SetProperty(key string, value interface{}) {
	conn.propertyLock.Lock()
	defer conn.propertyLock.Unlock()

	conn.property[key] = value
}

/*
    brief:获取链接属性
	param [in] key:属性名字
	return:属性值
*/
func (conn *TcpConn) GetProperty(key string) (interface{}, error) {
	conn.propertyLock.RLock()
	defer conn.propertyLock.RUnlock()

	if value, ok := conn.property[key]; ok {
		return value, nil
	} else {
		return nil, errors.New("no property found")
	}
}

/*
    brief:移除链接属性
	param [in] key:需要移除属性
*/
func (conn *TcpConn) RemoveProperty(key string) {
	conn.propertyLock.Lock()
	defer conn.propertyLock.Unlock()

	delete(conn.property, key)
}

/*
   brief:设置读超时时间
*/
func (conn *TcpConn) SetReadDeadline(d time.Duration) {
	conn.conn.SetReadDeadline(time.Now().Add(d))
}

/*
   brief:设置写超时时间
*/
func (conn *TcpConn) SetWriteDeadline(d time.Duration) {
	conn.conn.SetWriteDeadline(time.Now().Add(d))
}

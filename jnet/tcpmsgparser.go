package jnet

import (
	mempool "JRpcFrame/memorypool"
	"encoding/binary"
	"errors"
	"io"
	//"github.com/forgoer/openssl"
)

//|-----------------------head--------- ---------------|--------body----------------|
//|---4 bytes---|----4 bytes----|----1 byte------------|--------bodyLen-------------|
//|--------------------------------------------------------------------------|
//|---bodyLen---|------seq------|-----protocoltype-----|-------body-------|
//|------------------------------------------------------------------------|
//DataPack 封包拆包类实例，暂时不需要成员
type MsgParser struct {
	minMsgLen    uint32
	maxMsgLen    uint32
	littleEndian bool
}

var msgSlicePoolList = mempool.NewSlicePoolList(3,
	mempool.NewSlicePool(1, 4096, 512),
	mempool.NewSlicePool(4097, 40960, 4096),
	mempool.NewSlicePool(40961, 417792, 16384),
)

/*
   brief:MsgParser的构造方法
*/
func NewMsgParser(minMsgLen uint32, maxMsgLen uint32, littleEndian bool) *MsgParser {
	return &MsgParser{
		minMsgLen:    minMsgLen,
		maxMsgLen:    maxMsgLen,
		littleEndian: littleEndian,
	}
}

/*
   brief:获取包头长度方法
*/
func (mp *MsgParser) GetHeadLen() uint32 {
	//dataLen uint32(4字节)  + seq uint32(4字节) + type byte(1字节)
	return 9
}

/*
    brief:连接中写入一个Message
	param [in] conn:需要写入的conn
	param [in] msg:需要写入的msg
*/
func (mp *MsgParser) WriteMsg(conn *TcpConn, msg *Message) error {
	// check bodyLen
	if msg.GetBodyLen() > mp.maxMsgLen {
		return errors.New("message too long")
	} else if msg.GetBodyLen() < mp.minMsgLen {
		return errors.New("message too short")
	}

	data := mp.MakeByteSlice(int(msg.GetBodyLen() + mp.GetHeadLen()))

	if mp.littleEndian {
		binary.LittleEndian.PutUint32(data[:4], msg.GetBodyLen())
		binary.LittleEndian.PutUint32(data[4:8], msg.GetSeq())
		data[8] = msg.GetProtocolType()
	} else {
		binary.BigEndian.PutUint32(data[:4], msg.GetBodyLen())
		binary.BigEndian.PutUint32(data[4:8], msg.GetSeq())
		data[8] = msg.GetProtocolType()
	}

	copy(data[mp.GetHeadLen():], msg.GetBody())
	conn.Write(data)
	return nil
}

/*
    brief:连接中读出一个Message
	param [in] conn:需要读出的conn
	return:读出的message
*/
func (mp *MsgParser) ReadMsg(conn *TcpConn) (*Message, error) {
	var msg = &Message{}
	head := make([]byte, 9)
	// read head
	if _, err := io.ReadFull(conn, head); err != nil {
		return nil, err
	}
	if mp.littleEndian {
		msg.SetBodyLen(binary.LittleEndian.Uint32(head[:4]))
		msg.SetSeq(binary.LittleEndian.Uint32(head[4:8]))
		msg.SetProtocolType(head[8])
	} else {
		msg.SetBodyLen(binary.BigEndian.Uint32(head[:4]))
		msg.SetSeq(binary.BigEndian.Uint32(head[4:8]))
		msg.SetProtocolType(head[8])
	}
	// check bodyLen
	if msg.GetBodyLen() > mp.maxMsgLen {
		return nil, errors.New("message too long")
	} else if msg.GetBodyLen() < mp.minMsgLen {
		return nil, errors.New("message too short")
	}

	// read body
	body := mp.MakeByteSlice(int(msg.GetBodyLen()))
	if _, err := io.ReadFull(conn, body); err != nil {
		mp.ReleaseByteSlice(body)
		return nil, err
	}
	msg.SetBody(body)
	return msg, nil
}

func (mp *MsgParser) MakeByteSlice(size int) []byte {
	return msgSlicePoolList.MakeByteSlice(size)
}
func (mp *MsgParser) ReleaseByteSlice(byteBuff []byte) bool {
	return msgSlicePoolList.ReleaseByteSlice(byteBuff)
}

func (p *MsgParser) SetByteOrder(littleEndian bool) {
	p.littleEndian = littleEndian
}

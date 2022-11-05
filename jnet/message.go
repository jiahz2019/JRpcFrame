package  jnet

import(
)

//自定义tcp基本消息结构
type Message struct {
	BodyLen uint32  //内容长度
	Seq  	uint32	//序列号
	ProtocolType    byte
	Body    []byte  //消息的内容

}
/*
    brief:NewMsgPackage 创建一个Message消息包
	param [in] ID:消息的ID
	param [in] data:消息的内容
*/
func NewMessage(protocol byte, data []byte) *Message {
	m := Message{}
    m.SetProtocolType(protocol)
	m.SetBody(data)
	m.SetBodyLen(uint32(len(data)))

	return &m
}


/*
    brief:获取消息数据段长度
*/
func (msg *Message) GetBodyLen() uint32 {
	return msg.BodyLen
}


/*
    brief:获取消息内容
*/
func (msg *Message) GetBody() []byte {
	return msg.Body
}

/*
    brief:设置消息数据段长度
*/
func (msg *Message) SetBodyLen(len uint32) {
	msg.BodyLen = len
}


/*
    brief:设计消息内容
	param [in] data:消息内容的字节
*/
func (msg *Message) SetBody(data []byte) {
	msg.BodyLen = uint32(len(data))
	msg.Body = data
}

/*
    brief:设计消息序号
*/
func (msg *Message) SetSeq(seq uint32) {
	msg.Seq = seq
}

/*
    brief:获得消息序号
*/
func (msg *Message) GetSeq() uint32  {
	return msg.Seq
}

/*
    brief:设计协议类型
*/
func (msg *Message) SetProtocolType(t byte){
	msg.ProtocolType = t
}

/*
    brief:获得协议类型
*/
func (msg *Message) GetProtocolType() byte {
	return msg.ProtocolType
}

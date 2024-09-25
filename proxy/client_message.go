package proxy

type ClientMessage struct {
	OpId    uint32 // 操作类型
	DataLen uint32 // 消息长度
	IncrId  uint64 // 唯一Id
	ClerkId int32  // 客户端Id
	Data    []byte // 消息内容
}

func NewCliMsgPackage(id uint32, clerkId int32, incrId uint64, data []byte) *ClientMessage {
	return &ClientMessage{
		OpId:    id,
		DataLen: uint32(len(data)),
		IncrId:  incrId,
		ClerkId: clerkId,
		Data:    data,
	}
}

// 获取消息OpId
func (m *ClientMessage) GetMsgId() uint32 {
	return m.OpId
}

// 获取消息长度
func (m *ClientMessage) GetDataLen() uint32 {
	return m.DataLen
}

// 获取客户端Id
func (m *ClientMessage) GetClerkId() int32 {
	return m.ClerkId
}

func (m *ClientMessage) GetIncrId() uint64 {
	return m.IncrId
}

// 获取数据
func (m *ClientMessage) GetData() []byte {
	return m.Data
}

func (m *ClientMessage) SetMsgId(id uint32) {
	m.OpId = id
}

func (m *ClientMessage) SetDataLen(datalen uint32) {
	m.DataLen = datalen
}

func (m *ClientMessage) SetClerkId(clerkId int32) {
	m.ClerkId = clerkId
}

func (m *ClientMessage) SetIncrId(incrId uint64) {
	m.IncrId = incrId
}

func (m *ClientMessage) SetData(data []byte) {
	m.Data = data
}

type Result struct {
	ErrId uint32
	Bbody Body
}

func (r *Result) GetValue() string {
	return r.Bbody.GetValue()
}

func (r *Result) GetKey() string {
	return r.Bbody.GetKey()
}

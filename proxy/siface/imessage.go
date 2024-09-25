package siface

// 将请求消息封装到一个Message中，定义抽象接口

type IMessage interface {
	// 获取消息Id
	GetMsgId() uint32
	// 获取消息长度
	GetDataLen() uint32
	// 获取数据
	GetData() []byte

	SetMsgId(uint32)

	SetDataLen(uint32)

	SetData([]byte)
}

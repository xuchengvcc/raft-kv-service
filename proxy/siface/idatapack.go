package siface

// 封包、拆包 模块
// 直接面向TCP连接中的数据流，用于处理TCP粘包问题

type IDataPack interface {
	GetHeadLen() uint32
	// 封包
	Pack(IMessage) ([]byte, error)
	// 拆包
	Unpack([]byte) (IMessage, error)
}

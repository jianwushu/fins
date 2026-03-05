package fins

// FINS协议常量定义

// 默认端口
const (
	DefaultPort = 9600
)

// TCP帧魔数
const (
	TCPMagic = "FINS"
)

// FINS/TCP 外层命令码（常见取值，部分 PLC 型号可能存在差异）
//
// 按本库约定：
// - 0x00000000：握手请求（Handshake Request，发送客户端 IPv4 或 0.0.0.0）
// - 0x00000001：握手响应（Handshake Response）
// - 0x00000002：正常读写帧（承载“内层”FINS 报文；请求/响应命令码相同）
const (
	// 握手：请求/响应
	TCPCommandHandshakeRequest  uint32 = 0x00000000
	TCPCommandHandshakeResponse uint32 = 0x00000001

	// 正常读写帧（请求/响应同为 0x00000002）
	TCPCommandFinsFrame uint32 = 0x00000002
)

// ICF (Information Control Field) 值
const (
	ICFNoResponse = 0x00 // 无响应请求
	ICFRequest    = 0x80 // 请求响应模式
)

// 命令码
const (
	CmdMemoryRead     = 0x0101 // 内存读取
	CmdMemoryWrite    = 0x0102 // 内存写入
	CmdMemoryBitWrite = 0x0103 // 内存位写入
	CmdParameterRead  = 0x0201 // 参数读取
	CmdParameterWrite = 0x0202 // 参数写入
	CmdControllerOp   = 0x0501 // 控制器操作
)

// 内存区域代码
const (
	MemAreaCIO = 0x30 // CIO区 - 输入输出继电器
	MemAreaWR  = 0x31 // WR区 - 工作继电器
	MemAreaHR  = 0x32 // HR区 - 保持继电器
	MemAreaTC  = 0x33 // TC区 - 定时器/计数器完成标志
	MemAreaA   = 0x34 // A区 - 辅助继电器
	MemAreaD   = 0x82 // D区 - 数据寄存器
	MemAreaT   = 0x89 // T区 - 定时器当前值
	MemAreaC   = 0x8C // C区 - 计数器当前值
)

// 数据类型
const (
	DataTypeBit  = 0x00 // 位（Bit）
	DataTypeWord = 0x01 // 字（Word）16位
)

// 错误码定义
const (
	ErrCodeSuccess           = 0x0000 // 正常
	ErrCodeLocalNodeError    = 0x0001 // 本地节点错误
	ErrCodeRemoteNodeError   = 0x0002 // 远程节点错误
	ErrCodeCommControllerErr = 0x0003 // 通信控制器错误
	ErrCodeResponseTimeout   = 0x0004 // 响应超时
	ErrCodeRequestCancelled  = 0x0005 // 请求取消
	ErrCodeNotExecutable     = 0x0101 // 不可执行
	ErrCodeAddressOutOfRange = 0x0102 // 地址越界
	ErrCodeAddressFormat     = 0x0103 // 地址格式错误
	ErrCodeDataLengthError   = 0x0104 // 数据长度错误
	ErrCodeDataNotWritable   = 0x0105 // 数据不可写入
	ErrCodeAccessModeError   = 0x0106 // 访问模式错误
	ErrCodeProtectionError   = 0x0201 // 保护错误
)

// 地址参数默认值
const (
	LocalNetwork  = 0x00 // 本地网络
	BroadcastAddr = 0x00 // 广播地址
	CPUUnit       = 0x00 // CPU单元
	EthernetPort  = 0xFE // 内置Ethernet端口
)

// SID模式
type SIDMode int

const (
	SIDFixed     SIDMode = 0 // 固定模式
	SIDIncrement SIDMode = 1 // 递增模式
)

// 帧头长度
const (
	TCPHeaderLength = 16 // FINS/TCP 外层头长度（Magic+Length+Command+ErrorCode）
	UDPHeaderLength = 10 // FINS/UDP（内层FINS）头长度
)

// 错误码到错误消息的映射
var ErrorMessages = map[uint16]string{
	ErrCodeSuccess:           "成功",
	ErrCodeLocalNodeError:    "本地节点错误",
	ErrCodeRemoteNodeError:   "远程节点错误",
	ErrCodeCommControllerErr: "通信控制器错误",
	ErrCodeResponseTimeout:   "响应超时",
	ErrCodeRequestCancelled:  "请求取消",
	ErrCodeNotExecutable:     "命令不可执行",
	ErrCodeAddressOutOfRange: "地址越界",
	ErrCodeAddressFormat:     "地址格式错误",
	ErrCodeDataLengthError:   "数据长度错误",
	ErrCodeDataNotWritable:   "数据不可写入",
	ErrCodeAccessModeError:   "访问模式错误",
	ErrCodeProtectionError:   "保护错误",
}

// GetErrorMessage 获取错误码对应的错误消息
func GetErrorMessage(code uint16) string {
	if msg, ok := ErrorMessages[code]; ok {
		return msg
	}
	return "未知错误"
}

// 内存区域名称映射
var MemoryAreaNames = map[byte]string{
	MemAreaCIO: "CIO区",
	MemAreaWR:  "WR区",
	MemAreaHR:  "HR区",
	MemAreaTC:  "TC区",
	MemAreaA:   "A区",
	MemAreaD:   "D区",
	MemAreaT:   "T区",
	MemAreaC:   "C区",
}

// GetMemoryAreaName 获取内存区域名称
func GetMemoryAreaName(code byte) string {
	if name, ok := MemoryAreaNames[code]; ok {
		return name
	}
	return "未知区域"
}

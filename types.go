package fins

import (
	"errors"
	"time"
)

// 自定义错误
var (
	ErrTimeout           = errors.New("操作超时")
	ErrInvalidFrame      = errors.New("无效的帧数据")
	ErrInvalidMagic      = errors.New("无效的魔数")
	ErrConnectionClosed  = errors.New("连接已关闭")
	ErrInvalidResponse   = errors.New("无效的响应")
	ErrInvalidSID        = errors.New("无效的SID")
	ErrInvalidAddress    = errors.New("无效的地址")
	ErrInvalidDataLength = errors.New("无效的数据长度")
)

// FinsClientConfig FINS客户端配置
type FinsClientConfig struct {
	IP             string        // PLC IP地址
	Port           int           // 端口号，默认9600
	LocalNode      byte          // 本地节点地址
	ServerNode     byte          // 服务器节点地址
	Timeout        time.Duration // 超时时间
	RetryCount     int           // 重试次数
	SIDMode        SIDMode       // SID模式
	FixedSID       byte          // 固定模式下的SID值
	StartSID       byte          // 递增模式的起始值
	MaxSID         byte          // SID循环最大值
	EnableAutoNode bool          // 是否自动获取节点地址
}

// DefaultConfig 返回默认配置
func DefaultConfig(ip string) *FinsClientConfig {
	return &FinsClientConfig{
		IP:             ip,
		Port:           DefaultPort,
		LocalNode:      0x01,
		ServerNode:     0x00, // 需要动态获取
		Timeout:        5 * time.Second,
		RetryCount:     3,
		SIDMode:        SIDFixed,
		FixedSID:       0x00,
		StartSID:       0x00,
		MaxSID:         0xFF,
		EnableAutoNode: true,
	}
}

// FinsUDPFrame FINS UDP帧结构
type FinsUDPFrame struct {
	ICF     byte   // 信息控制字段
	RSV     byte   // 保留字段
	GCT     byte   // 网关计数
	DNA     byte   // 目标网络地址
	DA1     byte   // 目标节点地址
	DA2     byte   // 目标单元地址
	SNA     byte   // 源网络地址
	SA1     byte   // 源节点地址
	SA2     byte   // 源单元地址
	SID     byte   // 服务ID
	Command uint16 // 命令码
	Data    []byte // 数据
}

// FinsTCPFrame FINS TCP帧结构
type FinsTCPFrame struct {
	Magic      [4]byte // 魔数 "FINS"
	Length     uint32  // 帧长度
	ErrorCode  uint32  // 错误码
	ClientNode uint32  // 客户端节点号
	ServerNode uint32  // 服务器节点号
	Command    uint16  // 命令码
	Data       []byte  // 数据
}

// FinsResponse FINS响应结构
type FinsResponse struct {
	SID        byte   // 服务ID
	StatusCode uint16 // 状态码
	Data       []byte // 响应数据
}

// IsSuccess 判断响应是否成功
func (r *FinsResponse) IsSuccess() bool {
	return r.StatusCode == ErrCodeSuccess
}

// GetErrorMessage 获取错误消息
func (r *FinsResponse) GetErrorMessage() string {
	return GetErrorMessage(r.StatusCode)
}

// MemoryAddress 内存地址结构
type MemoryAddress struct {
	AreaCode byte   // 内存区域代码
	Address  uint16 // 地址
	BitNo    byte   // 位号（位操作时使用）
}

// ReadRequest 读取请求参数
type ReadRequest struct {
	AreaCode byte   // 内存区域代码
	Address  uint16 // 起始地址
	BitNo    byte   // 位号
	DataType byte   // 数据类型
	Count    uint16 // 读取数量
}

// WriteRequest 写入请求参数
type WriteRequest struct {
	AreaCode byte   // 内存区域代码
	Address  uint16 // 起始地址
	BitNo    byte   // 位号
	DataType byte   // 数据类型
	Count    uint16 // 写入数量
	Data     []byte // 写入数据
}

// PendingRequest 待处理的请求
type PendingRequest struct {
	SID       byte              // 服务ID
	Request   []byte            // 请求数据
	CreatedAt time.Time         // 创建时间
	Response  chan *FinsResponse // 响应通道
}

// ConnectionStats 连接统计信息
type ConnectionStats struct {
	TotalRequests  uint64    // 总请求数
	SuccessCount   uint64    // 成功次数
	ErrorCount     uint64    // 错误次数
	TimeoutCount   uint64    // 超时次数
	LastRequestAt  time.Time // 最后请求时间
	LastResponseAt time.Time // 最后响应时间
}


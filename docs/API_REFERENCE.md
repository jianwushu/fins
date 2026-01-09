# FINS协议库 API参考文档

## 客户端创建和管理

### NewClient
```go
func NewClient(config *FinsClientConfig, useTCP bool) (*FinsClient, error)
```
创建FINS客户端。
- `config`: 客户端配置
- `useTCP`: true=TCP模式, false=UDP模式
- 返回: 客户端实例和错误

### Connect
```go
func (c *FinsClient) Connect() error
```
连接到PLC。

### Close
```go
func (c *FinsClient) Close() error
```
关闭连接。

### IsConnected
```go
func (c *FinsClient) IsConnected() bool
```
检查是否已连接。

### GetStats
```go
func (c *FinsClient) GetStats() ConnectionStats
```
获取连接统计信息。

## 字(Word)操作 - D区

### ReadDWord
```go
func (c *FinsClient) ReadDWord(address uint16) (uint16, error)
```
读取D区单个字(16位)。
- `address`: 寄存器地址
- 返回: 字值和错误

### ReadDWords
```go
func (c *FinsClient) ReadDWords(address uint16, count uint16) ([]uint16, error)
```
读取D区多个字。
- `address`: 起始地址
- `count`: 读取数量
- 返回: 字数组和错误

### WriteDWord
```go
func (c *FinsClient) WriteDWord(address uint16, value uint16) error
```
写入D区单个字。
- `address`: 寄存器地址
- `value`: 要写入的值

### WriteDWords
```go
func (c *FinsClient) WriteDWords(address uint16, values []uint16) error
```
写入D区多个字。
- `address`: 起始地址
- `values`: 要写入的值数组

## 字节数组操作 - D区

### ReadDBytes
```go
func (c *FinsClient) ReadDBytes(address uint16, byteCount uint16) ([]byte, error)
```
读取D区字节数组。
- `address`: 起始地址
- `byteCount`: 要读取的字节数
- 返回: 字节数组和错误

**说明**: 
- 自动处理字节对齐(PLC按字存储)
- 支持奇数字节数读取

### WriteDBytes
```go
func (c *FinsClient) WriteDBytes(address uint16, data []byte) error
```
写入D区字节数组。
- `address`: 起始地址
- `data`: 要写入的字节数组

**说明**:
- 自动处理字节对齐
- 奇数字节会自动补0对齐

## 字节数组操作 - CIO区

### ReadCIOBytes
```go
func (c *FinsClient) ReadCIOBytes(address uint16, byteCount uint16) ([]byte, error)
```
读取CIO区字节数组。

### WriteCIOBytes
```go
func (c *FinsClient) WriteCIOBytes(address uint16, data []byte) error
```
写入CIO区字节数组。

## 字节数组操作 - HR区

### ReadHRBytes
```go
func (c *FinsClient) ReadHRBytes(address uint16, byteCount uint16) ([]byte, error)
```
读取HR区字节数组。

### WriteHRBytes
```go
func (c *FinsClient) WriteHRBytes(address uint16, data []byte) error
```
写入HR区字节数组。

## 字节数组操作 - WR区

### ReadWRBytes
```go
func (c *FinsClient) ReadWRBytes(address uint16, byteCount uint16) ([]byte, error)
```
读取WR区字节数组。

### WriteWRBytes
```go
func (c *FinsClient) WriteWRBytes(address uint16, data []byte) error
```
写入WR区字节数组。

## 位操作 - CIO区

### ReadCIOBit
```go
func (c *FinsClient) ReadCIOBit(address uint16, bitNo byte) (bool, error)
```
读取CIO区位。
- `address`: 字地址
- `bitNo`: 位号(0-15)
- 返回: 位值和错误

### WriteCIOBit
```go
func (c *FinsClient) WriteCIOBit(address uint16, bitNo byte, value bool) error
```
写入CIO区位。
- `address`: 字地址
- `bitNo`: 位号(0-15)
- `value`: 位值

## 通用内存操作

### ReadMemoryArea
```go
func (c *FinsClient) ReadMemoryArea(areaCode byte, address uint16, count uint16) ([]byte, error)
```
读取通用内存区域。
- `areaCode`: 内存区域代码(MemAreaD, MemAreaCIO等)
- `address`: 起始地址
- `count`: 读取字数
- 返回: 字节数组和错误

### WriteMemoryArea
```go
func (c *FinsClient) WriteMemoryArea(areaCode byte, address uint16, values []uint16) error
```
写入通用内存区域。
- `areaCode`: 内存区域代码
- `address`: 起始地址
- `values`: 要写入的字数组

### ReadBytes
```go
func (c *FinsClient) ReadBytes(areaCode byte, address uint16, byteCount uint16) ([]byte, error)
```
读取任意内存区域的字节数组。
- `areaCode`: 内存区域代码
- `address`: 起始地址
- `byteCount`: 要读取的字节数
- 返回: 字节数组和错误

### WriteBytes
```go
func (c *FinsClient) WriteBytes(areaCode byte, address uint16, data []byte) error
```
写入字节数组到任意内存区域。
- `areaCode`: 内存区域代码
- `address`: 起始地址
- `data`: 要写入的字节数组

### ReadBit
```go
func (c *FinsClient) ReadBit(areaCode byte, address uint16, bitNo byte) (bool, error)
```
读取任意内存区域的位。
- `areaCode`: 内存区域代码
- `address`: 字地址
- `bitNo`: 位号(0-15)
- 返回: 位值和错误

### WriteBit
```go
func (c *FinsClient) WriteBit(areaCode byte, address uint16, bitNo byte, value bool) error
```
写入位到任意内存区域。
- `areaCode`: 内存区域代码
- `address`: 字地址
- `bitNo`: 位号(0-15)
- `value`: 位值

## 重试机制

### NewRetryableClient
```go
func NewRetryableClient(client *FinsClient, policy *RetryPolicy) *RetryableClient
```
创建支持重试的客户端包装器。
- `client`: FINS客户端
- `policy`: 重试策略(nil使用默认策略)

### RetryPolicy
```go
type RetryPolicy struct {
    MaxRetries      int           // 最大重试次数
    InitialDelay    time.Duration // 初始延迟
    MaxDelay        time.Duration // 最大延迟
    BackoffFactor   float64       // 退避因子
    RetryableErrors []error       // 可重试的错误类型
}
```

### DefaultRetryPolicy
```go
func DefaultRetryPolicy() *RetryPolicy
```
返回默认重试策略:
- MaxRetries: 3
- InitialDelay: 100ms
- MaxDelay: 5s
- BackoffFactor: 2.0

## 配置

### FinsClientConfig
```go
type FinsClientConfig struct {
    IP             string        // PLC IP地址
    Port           int           // 端口号(默认9600)
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
```

### DefaultConfig
```go
func DefaultConfig(ip string) *FinsClientConfig
```
返回默认配置。

## 常量

### 内存区域代码
```go
const (
    MemAreaCIO = 0x30  // CIO区 - 输入输出继电器
    MemAreaWR  = 0x31  // WR区 - 工作继电器
    MemAreaHR  = 0x32  // HR区 - 保持继电器
    MemAreaTC  = 0x33  // TC区 - 定时器/计数器完成标志
    MemAreaA   = 0x34  // A区 - 辅助继电器
    MemAreaD   = 0x82  // D区 - 数据寄存器
    MemAreaT   = 0x89  // T区 - 定时器当前值
    MemAreaC   = 0x8C  // C区 - 计数器当前值
)
```

### 命令码
```go
const (
    CmdMemoryRead     = 0x0101  // 内存读取
    CmdMemoryWrite    = 0x0102  // 内存写入
    CmdMemoryBitWrite = 0x0103  // 内存位写入
    CmdParameterRead  = 0x0201  // 参数读取
    CmdParameterWrite = 0x0202  // 参数写入
    CmdControllerOp   = 0x0501  // 控制器操作
)
```

### SID模式
```go
const (
    SIDFixed     SIDMode = 0  // 固定模式
    SIDIncrement SIDMode = 1  // 递增模式
)
```

## 错误

### 预定义错误
```go
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
```

### 错误码
```go
const (
    ErrCodeSuccess           = 0x0000  // 正常
    ErrCodeAddressOutOfRange = 0x0102  // 地址越界
    ErrCodeDataLengthError   = 0x0104  // 数据长度错误
    // ... 更多错误码见 constants.go
)
```

## 使用示例

### 基本读写
```go
client, _ := fins.NewClient(config, false)
client.Connect()

// 读取字
value, _ := client.ReadDWord(100)

// 写入字
client.WriteDWord(100, 1234)

// 读取字节
data, _ := client.ReadDBytes(100, 10)

// 写入字节
client.WriteDBytes(200, []byte{0x01, 0x02, 0x03})
```

### 字节数组与数据类型转换
```go
// 整数转字节
value := uint32(0x12345678)
buf := make([]byte, 4)
binary.BigEndian.PutUint32(buf, value)
client.WriteDBytes(100, buf)

// 字节转整数
data, _ := client.ReadDBytes(100, 4)
value = binary.BigEndian.Uint32(data)

// 字符串操作
text := "Hello"
client.WriteDBytes(200, []byte(text))
data, _ = client.ReadDBytes(200, uint16(len(text)))
readText := string(data)
```

### 重试机制
```go
policy := &fins.RetryPolicy{
    MaxRetries:    5,
    InitialDelay:  200 * time.Millisecond,
    BackoffFactor: 2.0,
}
retryClient := fins.NewRetryableClient(client, policy)
data, err := retryClient.ReadMemoryArea(fins.MemAreaD, 100, 1)
```


package fins

import (
	"encoding/binary"
)

// Client FINS客户端接口
type Client interface {
	Connect() error
	Close() error
	SendRequest(command uint16, data []byte) (*FinsResponse, error)
	IsConnected() bool
	GetStats() ConnectionStats
}

// FinsClient FINS客户端（统一接口）
type FinsClient struct {
	client Client
	config *FinsClientConfig
}

// NewClient 创建FINS客户端（根据配置自动选择TCP或UDP）
func NewClient(config *FinsClientConfig, useTCP bool) (*FinsClient, error) {
	var client Client
	var err error

	if useTCP {
		client, err = NewTCPClient(config)
	} else {
		client, err = NewUDPClient(config)
	}

	if err != nil {
		return nil, err
	}

	return &FinsClient{
		client: client,
		config: config,
	}, nil
}

// Connect 连接到PLC
func (c *FinsClient) Connect() error {
	return c.client.Connect()
}

// Close 关闭连接
func (c *FinsClient) Close() error {
	return c.client.Close()
}

// ReadMemoryArea 读取内存区域
func (c *FinsClient) ReadMemoryArea(areaCode byte, address uint16, count uint16) ([]byte, error) {
	req := &ReadRequest{
		AreaCode: areaCode,
		Address:  address,
		BitNo:    0,
		DataType: DataTypeWord,
		Count:    count,
	}

	data := BuildReadMemoryRequest(req)
	resp, err := c.client.SendRequest(CmdMemoryRead, data)
	if err != nil {
		return nil, err
	}

	return ParseReadMemoryResponse(resp)
}

// WriteMemoryArea 写入内存区域
func (c *FinsClient) WriteMemoryArea(areaCode byte, address uint16, values []uint16) error {
	// 构建写入数据
	data := make([]byte, len(values)*2)
	for i, v := range values {
		binary.BigEndian.PutUint16(data[i*2:], v)
	}

	req := &WriteRequest{
		AreaCode: areaCode,
		Address:  address,
		BitNo:    0,
		DataType: DataTypeWord,
		Count:    uint16(len(values)),
		Data:     data,
	}

	reqData := BuildWriteMemoryRequest(req)
	resp, err := c.client.SendRequest(CmdMemoryWrite, reqData)
	if err != nil {
		return err
	}

	return ParseWriteMemoryResponse(resp)
}

// ReadBit 读取单个位
func (c *FinsClient) ReadBit(areaCode byte, address uint16, bitNo byte) (bool, error) {
	req := &ReadRequest{
		AreaCode: areaCode,
		Address:  address,
		BitNo:    bitNo,
		DataType: DataTypeBit,
		Count:    1,
	}

	data := BuildReadMemoryRequest(req)
	resp, err := c.client.SendRequest(CmdMemoryRead, data)
	if err != nil {
		return false, err
	}

	result, err := ParseReadMemoryResponse(resp)
	if err != nil {
		return false, err
	}

	if len(result) < 1 {
		return false, ErrInvalidResponse
	}

	return result[0] != 0, nil
}

// WriteBit 写入单个位
func (c *FinsClient) WriteBit(areaCode byte, address uint16, bitNo byte, value bool) error {
	var bitValue byte
	if value {
		bitValue = 1
	}

	req := &WriteRequest{
		AreaCode: areaCode,
		Address:  address,
		BitNo:    bitNo,
		DataType: DataTypeBit,
		Count:    1,
		Data:     []byte{bitValue},
	}

	reqData := BuildWriteMemoryRequest(req)
	resp, err := c.client.SendRequest(CmdMemoryBitWrite, reqData)
	if err != nil {
		return err
	}

	return ParseWriteMemoryResponse(resp)
}

// ReadDWord 读取D区单个字
func (c *FinsClient) ReadDWord(address uint16) (uint16, error) {
	data, err := c.ReadMemoryArea(MemAreaD, address, 1)
	if err != nil {
		return 0, err
	}
	if len(data) < 2 {
		return 0, ErrInvalidResponse
	}
	return binary.BigEndian.Uint16(data[0:2]), nil
}

// ReadDWords 读取D区多个字
func (c *FinsClient) ReadDWords(address uint16, count uint16) ([]uint16, error) {
	data, err := c.ReadMemoryArea(MemAreaD, address, count)
	if err != nil {
		return nil, err
	}

	if len(data) < int(count)*2 {
		return nil, ErrInvalidResponse
	}

	result := make([]uint16, count)
	for i := uint16(0); i < count; i++ {
		result[i] = binary.BigEndian.Uint16(data[i*2 : (i+1)*2])
	}
	return result, nil
}

// WriteDWord 写入D区单个字
func (c *FinsClient) WriteDWord(address uint16, value uint16) error {
	return c.WriteMemoryArea(MemAreaD, address, []uint16{value})
}

// WriteDWords 写入D区多个字
func (c *FinsClient) WriteDWords(address uint16, values []uint16) error {
	return c.WriteMemoryArea(MemAreaD, address, values)
}

// ReadCIOBit 读取CIO区位
func (c *FinsClient) ReadCIOBit(address uint16, bitNo byte) (bool, error) {
	return c.ReadBit(MemAreaCIO, address, bitNo)
}

// WriteCIOBit 写入CIO区位
func (c *FinsClient) WriteCIOBit(address uint16, bitNo byte, value bool) error {
	return c.WriteBit(MemAreaCIO, address, bitNo, value)
}

// IsConnected 检查是否已连接
func (c *FinsClient) IsConnected() bool {
	return c.client.IsConnected()
}

// GetStats 获取统计信息
func (c *FinsClient) GetStats() ConnectionStats {
	return c.client.GetStats()
}

// GetConfig 获取配置
func (c *FinsClient) GetConfig() *FinsClientConfig {
	return c.config
}

// SendRawRequest 发送原始请求
func (c *FinsClient) SendRawRequest(command uint16, data []byte) (*FinsResponse, error) {
	return c.client.SendRequest(command, data)
}

// ReadBytes 读取内存区域为字节数组
// areaCode: 内存区域代码
// address: 起始地址
// byteCount: 要读取的字节数
// 返回: 字节数组
func (c *FinsClient) ReadBytes(areaCode byte, address uint16, byteCount uint16) ([]byte, error) {
	// 计算需要读取的字数(每个字2字节)
	wordCount := (byteCount + 1) / 2 // 向上取整

	req := &ReadRequest{
		AreaCode: areaCode,
		Address:  address,
		BitNo:    0,
		DataType: DataTypeWord,
		Count:    wordCount,
	}

	data := BuildReadMemoryRequest(req)
	resp, err := c.client.SendRequest(CmdMemoryRead, data)
	if err != nil {
		return nil, err
	}

	result, err := ParseReadMemoryResponse(resp)
	if err != nil {
		return nil, err
	}

	// 如果请求的字节数是奇数,只返回需要的字节数
	if uint16(len(result)) > byteCount {
		return result[:byteCount], nil
	}

	return result, nil
}

// WriteBytes 写入字节数组到内存区域
// areaCode: 内存区域代码
// address: 起始地址
// data: 要写入的字节数组
func (c *FinsClient) WriteBytes(areaCode byte, address uint16, data []byte) error {
	// 如果字节数是奇数,需要补齐到偶数
	writeData := data
	if len(data)%2 != 0 {
		writeData = make([]byte, len(data)+1)
		copy(writeData, data)
		writeData[len(data)] = 0 // 补0
	}

	// 计算字数
	wordCount := uint16(len(writeData) / 2)

	req := &WriteRequest{
		AreaCode: areaCode,
		Address:  address,
		BitNo:    0,
		DataType: DataTypeWord,
		Count:    wordCount,
		Data:     writeData,
	}

	reqData := BuildWriteMemoryRequest(req)
	resp, err := c.client.SendRequest(CmdMemoryWrite, reqData)
	if err != nil {
		return err
	}

	return ParseWriteMemoryResponse(resp)
}

// ReadDBytes 读取D区字节数组
func (c *FinsClient) ReadDBytes(address uint16, byteCount uint16) ([]byte, error) {
	return c.ReadBytes(MemAreaD, address, byteCount)
}

// WriteDBytes 写入D区字节数组
func (c *FinsClient) WriteDBytes(address uint16, data []byte) error {
	return c.WriteBytes(MemAreaD, address, data)
}

// ReadCIOBytes 读取CIO区字节数组
func (c *FinsClient) ReadCIOBytes(address uint16, byteCount uint16) ([]byte, error) {
	return c.ReadBytes(MemAreaCIO, address, byteCount)
}

// WriteCIOBytes 写入CIO区字节数组
func (c *FinsClient) WriteCIOBytes(address uint16, data []byte) error {
	return c.WriteBytes(MemAreaCIO, address, data)
}

// ReadHRBytes 读取HR区字节数组
func (c *FinsClient) ReadHRBytes(address uint16, byteCount uint16) ([]byte, error) {
	return c.ReadBytes(MemAreaHR, address, byteCount)
}

// WriteHRBytes 写入HR区字节数组
func (c *FinsClient) WriteHRBytes(address uint16, data []byte) error {
	return c.WriteBytes(MemAreaHR, address, data)
}

// ReadWRBytes 读取WR区字节数组
func (c *FinsClient) ReadWRBytes(address uint16, byteCount uint16) ([]byte, error) {
	return c.ReadBytes(MemAreaWR, address, byteCount)
}

// WriteWRBytes 写入WR区字节数组
func (c *FinsClient) WriteWRBytes(address uint16, data []byte) error {
	return c.WriteBytes(MemAreaWR, address, data)
}

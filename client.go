package fins

import (
	"encoding/binary"
	"fmt"
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

// ========== 对外统一 API（字符串地址） ==========

// ReadWord 按字符串地址读取 1 个 word（16bit）。
//
// 示例："D100"、"WR200"。
func (c *FinsClient) ReadWord(address string) (uint16, error) {
	pa, err := ParseAddress(address)
	if err != nil {
		return 0, err
	}
	if pa.IsBit {
		return 0, fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}

	data, err := c.readMemoryArea(pa.AreaCode, pa.Address, 1)
	if err != nil {
		return 0, err
	}
	if len(data) < 2 {
		return 0, ErrInvalidResponse
	}
	return binary.BigEndian.Uint16(data[0:2]), nil
}

// ReadWords 按字符串地址批量读取 word。
//
// address 为起始 word 地址，例如 "D100"；count 为 word 数量。
func (c *FinsClient) ReadWords(address string, count uint16) ([]uint16, error) {
	pa, err := ParseAddress(address)
	if err != nil {
		return nil, err
	}
	if pa.IsBit {
		return nil, fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}

	data, err := c.readMemoryArea(pa.AreaCode, pa.Address, count)
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

// WriteWord 按字符串地址写入 1 个 word（16bit）。
func (c *FinsClient) WriteWord(address string, value uint16) error {
	pa, err := ParseAddress(address)
	if err != nil {
		return err
	}
	if pa.IsBit {
		return fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}
	return c.writeMemoryArea(pa.AreaCode, pa.Address, []uint16{value})
}

// WriteWords 按字符串地址批量写入 word。
// address 为起始 word 地址，例如 "D200"；values 为 word 列表。
func (c *FinsClient) WriteWords(address string, values []uint16) error {
	pa, err := ParseAddress(address)
	if err != nil {
		return err
	}
	if pa.IsBit {
		return fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}
	return c.writeMemoryArea(pa.AreaCode, pa.Address, values)
}

// ReadBytes 按字符串地址读取字节数组。
//
// address 为起始 word 地址（例如 "D100"），byteCount 为要读取的字节数。
func (c *FinsClient) ReadBytes(address string, byteCount uint16) ([]byte, error) {
	pa, err := ParseAddress(address)
	if err != nil {
		return nil, err
	}
	if pa.IsBit {
		return nil, fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}
	return c.readBytes(pa.AreaCode, pa.Address, byteCount)
}

// WriteBytes 按字符串地址写入字节数组。
//
// address 为起始 word 地址（例如 "D200"）。
func (c *FinsClient) WriteBytes(address string, data []byte) error {
	pa, err := ParseAddress(address)
	if err != nil {
		return err
	}
	if pa.IsBit {
		return fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}
	return c.writeBytes(pa.AreaCode, pa.Address, data)
}

// ReadBit 按字符串 bit 地址读取 1 个 bit。
//
// 示例："CIO0.00"、"WR10.15"。
func (c *FinsClient) ReadBit(address string) (bool, error) {
	pa, err := ParseAddress(address)
	if err != nil {
		return false, err
	}
	if !pa.IsBit {
		return false, fmt.Errorf("%w: %q is not bit address", ErrInvalidAddress, address)
	}
	return c.readBit(pa.AreaCode, pa.Address, pa.BitNo)
}

// WriteBit 按字符串 bit 地址写入 1 个 bit。
func (c *FinsClient) WriteBit(address string, value bool) error {
	pa, err := ParseAddress(address)
	if err != nil {
		return err
	}
	if !pa.IsBit {
		return fmt.Errorf("%w: %q is not bit address", ErrInvalidAddress, address)
	}
	return c.writeBit(pa.AreaCode, pa.Address, pa.BitNo, value)
}

// ========== 内部底座（areaCode/address）- 不对外暴露 ==========

func (c *FinsClient) readMemoryArea(areaCode byte, address uint16, count uint16) ([]byte, error) {
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

func (c *FinsClient) writeMemoryArea(areaCode byte, address uint16, values []uint16) error {
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

func (c *FinsClient) readBit(areaCode byte, address uint16, bitNo byte) (bool, error) {
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

func (c *FinsClient) writeBit(areaCode byte, address uint16, bitNo byte, value bool) error {
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

func (c *FinsClient) readBytes(areaCode byte, address uint16, byteCount uint16) ([]byte, error) {
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

	if uint16(len(result)) > byteCount {
		return result[:byteCount], nil
	}
	return result, nil
}

func (c *FinsClient) writeBytes(areaCode byte, address uint16, data []byte) error {
	writeData := data
	if len(data)%2 != 0 {
		writeData = make([]byte, len(data)+1)
		copy(writeData, data)
		writeData[len(data)] = 0
	}

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

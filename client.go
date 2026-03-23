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
	GetConnectionStatus() ConnectionStatus
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

	return NewClientWithTransport(client, config)
}

// NewClientWithTransport 使用现有传输层客户端包装出统一的 FINS 客户端。
func NewClientWithTransport(transport Client, config *FinsClientConfig) (*FinsClient, error) {
	if transport == nil {
		return nil, fmt.Errorf("传输层客户端不能为空")
	}
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	return &FinsClient{
		client: transport,
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

// GetConnectionStatus 获取连接状态。
func (c *FinsClient) GetConnectionStatus() ConnectionStatus {
	return c.client.GetConnectionStatus()
}

// GetConfig 获取配置
func (c *FinsClient) GetConfig() *FinsClientConfig {
	return c.config
}

// ========== 对外统一 API（字符串地址） ==========

func (c *FinsClient) parseWordAddress(address string) (*ParsedAddress, error) {
	return c.parseAddress(address, false)
}

func (c *FinsClient) parseBitAddress(address string) (*ParsedAddress, error) {
	return c.parseAddress(address, true)
}

func (c *FinsClient) parseAddress(address string, wantBit bool) (*ParsedAddress, error) {
	pa, err := ParseAddress(address)
	if err != nil {
		return nil, err
	}
	if pa.IsBit != wantBit {
		if wantBit {
			return nil, fmt.Errorf("%w: %q is not bit address", ErrInvalidAddress, address)
		}
		return nil, fmt.Errorf("%w: %q is bit address", ErrInvalidAddress, address)
	}
	return pa, nil
}

func decodeWords(data []byte, count uint16) ([]uint16, error) {
	expected := int(count) * 2
	if len(data) < expected {
		return nil, ErrInvalidResponse
	}

	result := make([]uint16, count)
	for i := range result {
		offset := i * 2
		result[i] = binary.BigEndian.Uint16(data[offset : offset+2])
	}
	return result, nil
}

func decodeSingleWord(data []byte) (uint16, error) {
	words, err := decodeWords(data, 1)
	if err != nil {
		return 0, err
	}
	return words[0], nil
}

func decodeSingleByte(data []byte) (byte, error) {
	if len(data) < 1 {
		return 0, ErrInvalidResponse
	}
	return data[0], nil
}

func padBytesToWords(data []byte) []byte {
	if len(data)%2 == 0 {
		return data
	}

	padded := make([]byte, len(data)+1)
	copy(padded, data)
	return padded
}

// ReadWord 按字符串地址读取 1 个 word（16bit）。
//
// 示例："D100"、"WR200"。
func (c *FinsClient) ReadWord(address string) (uint16, error) {
	pa, err := c.parseWordAddress(address)
	if err != nil {
		return 0, err
	}

	data, err := c.readMemoryArea(pa.AreaCode, pa.Address, 1)
	if err != nil {
		return 0, err
	}
	return decodeSingleWord(data)
}

// ReadWords 按字符串地址批量读取 word。
//
// address 为起始 word 地址，例如 "D100"；count 为 word 数量。
func (c *FinsClient) ReadWords(address string, count uint16) ([]uint16, error) {
	if count == 0 {
		return nil, fmt.Errorf("%w: word count must be greater than 0", ErrInvalidAddress)
	}

	pa, err := c.parseWordAddress(address)
	if err != nil {
		return nil, err
	}

	data, err := c.readMemoryArea(pa.AreaCode, pa.Address, count)
	if err != nil {
		return nil, err
	}
	return decodeWords(data, count)
}

// WriteWord 按字符串地址写入 1 个 word（16bit）。
func (c *FinsClient) WriteWord(address string, value uint16) error {
	pa, err := c.parseWordAddress(address)
	if err != nil {
		return err
	}
	return c.writeMemoryArea(pa.AreaCode, pa.Address, []uint16{value})
}

// WriteWords 按字符串地址批量写入 word。
// address 为起始 word 地址，例如 "D200"；values 为 word 列表。
func (c *FinsClient) WriteWords(address string, values []uint16) error {
	if len(values) == 0 {
		return fmt.Errorf("%w: values must not be empty", ErrInvalidAddress)
	}

	pa, err := c.parseWordAddress(address)
	if err != nil {
		return err
	}
	return c.writeMemoryArea(pa.AreaCode, pa.Address, values)
}

// ReadBytes 按字符串地址读取字节数组。
//
// address 为起始 word 地址（例如 "D100"），byteCount 为要读取的字节数。
func (c *FinsClient) ReadBytes(address string, byteCount uint16) ([]byte, error) {
	if byteCount == 0 {
		return nil, fmt.Errorf("%w: byte count must be greater than 0", ErrInvalidAddress)
	}

	pa, err := c.parseWordAddress(address)
	if err != nil {
		return nil, err
	}
	return c.readBytes(pa.AreaCode, pa.Address, byteCount)
}

// WriteBytes 按字符串地址写入字节数组。
//
// address 为起始 word 地址（例如 "D200"）。
func (c *FinsClient) WriteBytes(address string, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("%w: data must not be empty", ErrInvalidAddress)
	}

	pa, err := c.parseWordAddress(address)
	if err != nil {
		return err
	}
	return c.writeBytes(pa.AreaCode, pa.Address, data)
}

// ReadBit 按字符串 bit 地址读取 1 个 bit。
//
// 示例："CIO0.00"、"WR10.15"。
func (c *FinsClient) ReadBit(address string) (bool, error) {
	pa, err := c.parseBitAddress(address)
	if err != nil {
		return false, err
	}
	return c.readBit(pa.AreaCode, pa.Address, pa.BitNo)
}

// WriteBit 按字符串 bit 地址写入 1 个 bit。
func (c *FinsClient) WriteBit(address string, value bool) error {
	pa, err := c.parseBitAddress(address)
	if err != nil {
		return err
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
		offset := i * 2
		binary.BigEndian.PutUint16(data[offset:offset+2], v)
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

	bitValue, err := decodeSingleByte(result)
	if err != nil {
		return false, err
	}
	return bitValue != 0, nil
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
	wordCount := (byteCount + 1) / 2

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
	if uint16(len(result)) <= byteCount {
		return result, nil
	}
	return result[:byteCount], nil
}

func (c *FinsClient) writeBytes(areaCode byte, address uint16, data []byte) error {
	writeData := padBytesToWords(data)
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

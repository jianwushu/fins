package fins

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// BuildTCPFrame 构建TCP帧
func BuildTCPFrame(frame *FinsTCPFrame) ([]byte, error) {
	buf := new(bytes.Buffer)

	// 写入魔数 "FINS"
	buf.Write(frame.Magic[:])

	// 写入长度（大端序）
	if err := binary.Write(buf, binary.BigEndian, frame.Length); err != nil {
		return nil, fmt.Errorf("写入长度失败: %w", err)
	}

	// 写入错误码（大端序）
	if err := binary.Write(buf, binary.BigEndian, frame.ErrorCode); err != nil {
		return nil, fmt.Errorf("写入错误码失败: %w", err)
	}

	// 写入客户端节点号（大端序）
	if err := binary.Write(buf, binary.BigEndian, frame.ClientNode); err != nil {
		return nil, fmt.Errorf("写入客户端节点号失败: %w", err)
	}

	// 写入服务器节点号（大端序）
	if err := binary.Write(buf, binary.BigEndian, frame.ServerNode); err != nil {
		return nil, fmt.Errorf("写入服务器节点号失败: %w", err)
	}

	// 写入命令码（大端序）
	if err := binary.Write(buf, binary.BigEndian, frame.Command); err != nil {
		return nil, fmt.Errorf("写入命令码失败: %w", err)
	}

	// 写入数据
	if len(frame.Data) > 0 {
		buf.Write(frame.Data)
	}

	return buf.Bytes(), nil
}

// ParseTCPFrame 解析TCP帧
func ParseTCPFrame(data []byte) (*FinsTCPFrame, error) {
	if len(data) < TCPHeaderLength {
		return nil, ErrInvalidFrame
	}

	frame := &FinsTCPFrame{}

	// 读取魔数
	copy(frame.Magic[:], data[0:4])
	if string(frame.Magic[:]) != TCPMagic {
		return nil, ErrInvalidMagic
	}

	// 读取长度
	frame.Length = binary.BigEndian.Uint32(data[4:8])

	// 读取错误码
	frame.ErrorCode = binary.BigEndian.Uint32(data[8:12])

	// 读取客户端节点号
	frame.ClientNode = binary.BigEndian.Uint32(data[12:16])

	// 读取服务器节点号
	frame.ServerNode = binary.BigEndian.Uint32(data[16:20])

	// 读取命令码
	if len(data) >= TCPHeaderLength+2 {
		frame.Command = binary.BigEndian.Uint16(data[20:22])
	}

	// 读取数据部分
	if len(data) > TCPHeaderLength+2 {
		frame.Data = make([]byte, len(data)-TCPHeaderLength-2)
		copy(frame.Data, data[TCPHeaderLength+2:])
	}

	return frame, nil
}

// NewTCPRequestFrame 创建TCP请求帧
func NewTCPRequestFrame(clientNode, serverNode uint32, command uint16, data []byte) *FinsTCPFrame {
	magic := [4]byte{'F', 'I', 'N', 'S'}
	
	// 计算帧长度：头部20字节 + 命令码2字节 + 数据长度
	length := uint32(TCPHeaderLength + 2 + len(data))

	return &FinsTCPFrame{
		Magic:      magic,
		Length:     length,
		ErrorCode:  0,
		ClientNode: clientNode,
		ServerNode: serverNode,
		Command:    command,
		Data:       data,
	}
}

// ParseTCPResponse 解析TCP响应
func ParseTCPResponse(data []byte) (*FinsResponse, error) {
	frame, err := ParseTCPFrame(data)
	if err != nil {
		return nil, err
	}

	// 响应数据至少包含2字节状态码
	if len(frame.Data) < 2 {
		return nil, ErrInvalidResponse
	}

	response := &FinsResponse{
		SID:        0, // TCP模式下SID通常为0
		StatusCode: binary.BigEndian.Uint16(frame.Data[0:2]),
	}

	// 提取响应数据（跳过状态码）
	if len(frame.Data) > 2 {
		response.Data = make([]byte, len(frame.Data)-2)
		copy(response.Data, frame.Data[2:])
	}

	return response, nil
}

// ReadTCPFrameFromConn 从连接中读取完整的TCP帧
// 返回完整的帧数据
func ReadTCPFrameFromConn(reader func([]byte) (int, error)) ([]byte, error) {
	// 1. 读取魔数（4字节）
	magic := make([]byte, 4)
	n, err := reader(magic)
	if err != nil {
		return nil, err
	}
	if n != 4 || string(magic) != TCPMagic {
		return nil, ErrInvalidMagic
	}

	// 2. 读取长度字段（4字节）
	lengthBuf := make([]byte, 4)
	n, err = reader(lengthBuf)
	if err != nil {
		return nil, err
	}
	if n != 4 {
		return nil, ErrInvalidFrame
	}

	length := binary.BigEndian.Uint32(lengthBuf)

	// 3. 读取剩余数据（长度 - 8字节已读取的魔数和长度字段）
	remaining := int(length) - 8
	if remaining < 0 {
		return nil, ErrInvalidFrame
	}

	remainingData := make([]byte, remaining)
	n, err = reader(remainingData)
	if err != nil {
		return nil, err
	}
	if n != remaining {
		return nil, ErrInvalidFrame
	}

	// 组合完整帧
	fullFrame := make([]byte, length)
	copy(fullFrame[0:4], magic)
	copy(fullFrame[4:8], lengthBuf)
	copy(fullFrame[8:], remainingData)

	return fullFrame, nil
}


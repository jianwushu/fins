package fins

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// BuildUDPFrame 构建UDP帧
func BuildUDPFrame(frame *FinsUDPFrame) ([]byte, error) {
	buf := new(bytes.Buffer)

	// 写入帧头（10字节）
	buf.WriteByte(frame.ICF)
	buf.WriteByte(frame.RSV)
	buf.WriteByte(frame.GCT)
	buf.WriteByte(frame.DNA)
	buf.WriteByte(frame.DA1)
	buf.WriteByte(frame.DA2)
	buf.WriteByte(frame.SNA)
	buf.WriteByte(frame.SA1)
	buf.WriteByte(frame.SA2)
	buf.WriteByte(frame.SID)

	// 写入命令码（2字节，大端序）
	if err := binary.Write(buf, binary.BigEndian, frame.Command); err != nil {
		return nil, fmt.Errorf("写入命令码失败: %w", err)
	}

	// 写入数据
	if len(frame.Data) > 0 {
		buf.Write(frame.Data)
	}

	return buf.Bytes(), nil
}

// ParseUDPFrame 解析UDP帧
func ParseUDPFrame(data []byte) (*FinsUDPFrame, error) {
	if len(data) < UDPHeaderLength+2 {
		return nil, ErrInvalidFrame
	}

	frame := &FinsUDPFrame{
		ICF: data[0],
		RSV: data[1],
		GCT: data[2],
		DNA: data[3],
		DA1: data[4],
		DA2: data[5],
		SNA: data[6],
		SA1: data[7],
		SA2: data[8],
		SID: data[9],
	}

	// 读取命令码（大端序）
	frame.Command = binary.BigEndian.Uint16(data[10:12])

	// 读取数据部分
	if len(data) > 12 {
		frame.Data = make([]byte, len(data)-12)
		copy(frame.Data, data[12:])
	}

	return frame, nil
}

// NewUDPRequestFrame 创建UDP请求帧
func NewUDPRequestFrame(localNode, serverNode, sid byte, command uint16, data []byte) *FinsUDPFrame {
	return &FinsUDPFrame{
		ICF:     ICFRequest,
		RSV:     0x00,
		GCT:     0x02,
		DNA:     LocalNetwork,
		DA1:     serverNode,
		DA2:     CPUUnit,
		SNA:     LocalNetwork,
		SA1:     localNode,
		SA2:     CPUUnit,
		SID:     sid,
		Command: command,
		Data:    data,
	}
}

// ParseUDPResponse 解析UDP响应
func ParseUDPResponse(data []byte) (*FinsResponse, error) {
	frame, err := ParseUDPFrame(data)
	if err != nil {
		return nil, err
	}

	// 响应数据至少包含2字节状态码
	if len(frame.Data) < 2 {
		return nil, ErrInvalidResponse
	}

	response := &FinsResponse{
		SID:        frame.SID,
		StatusCode: binary.BigEndian.Uint16(frame.Data[0:2]),
	}

	// 提取响应数据（跳过状态码）
	if len(frame.Data) > 2 {
		response.Data = make([]byte, len(frame.Data)-2)
		copy(response.Data, frame.Data[2:])
	}

	return response, nil
}

// BuildReadMemoryRequest 构建读取内存请求数据
func BuildReadMemoryRequest(req *ReadRequest) []byte {
	data := make([]byte, 6)
	data[0] = req.AreaCode
	binary.BigEndian.PutUint16(data[1:3], req.Address)
	data[3] = req.BitNo
	binary.BigEndian.PutUint16(data[4:6], req.Count)
	return data
}

// BuildWriteMemoryRequest 构建写入内存请求数据
func BuildWriteMemoryRequest(req *WriteRequest) []byte {
	headerLen := 6
	data := make([]byte, headerLen+len(req.Data))
	data[0] = req.AreaCode
	binary.BigEndian.PutUint16(data[1:3], req.Address)
	data[3] = req.BitNo
	binary.BigEndian.PutUint16(data[4:6], req.Count)
	copy(data[headerLen:], req.Data)
	return data
}

// ParseReadMemoryResponse 解析读取内存响应
func ParseReadMemoryResponse(resp *FinsResponse) ([]byte, error) {
	if !resp.IsSuccess() {
		return nil, fmt.Errorf("读取失败: %s (0x%04X)", resp.GetErrorMessage(), resp.StatusCode)
	}
	return resp.Data, nil
}

// ParseWriteMemoryResponse 解析写入内存响应
func ParseWriteMemoryResponse(resp *FinsResponse) error {
	if !resp.IsSuccess() {
		return fmt.Errorf("写入失败: %s (0x%04X)", resp.GetErrorMessage(), resp.StatusCode)
	}
	return nil
}


package fins

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// FinsTCPClient FINS TCP客户端
type FinsTCPClient struct {
	config     *FinsClientConfig
	conn       net.Conn
	mutex      sync.RWMutex
	stats      ConnectionStats
	closed     bool
	closeChan  chan struct{}
	responseCh chan *FinsResponse
}

// NewTCPClient 创建TCP客户端
func NewTCPClient(config *FinsClientConfig) (*FinsTCPClient, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	client := &FinsTCPClient{
		config:     config,
		closeChan:  make(chan struct{}),
		responseCh: make(chan *FinsResponse, 10),
	}

	return client, nil
}

// Connect 连接到服务器
func (c *FinsTCPClient) Connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		return fmt.Errorf("已经连接")
	}

	// 建立TCP连接
	addr := fmt.Sprintf("%s:%d", c.config.IP, c.config.Port)
	conn, err := net.DialTimeout("tcp", addr, c.config.Timeout)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}

	c.conn = conn
	c.closed = false

	// 启动接收协程
	go c.receiveLoop()

	return nil
}

// Close 关闭连接
func (c *FinsTCPClient) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.closeChan)

	if c.conn != nil {
		return c.conn.Close()
	}

	return nil
}

// SendRequest 发送请求
func (c *FinsTCPClient) SendRequest(command uint16, data []byte) (*FinsResponse, error) {
	c.mutex.RLock()
	if c.closed {
		c.mutex.RUnlock()
		return nil, ErrConnectionClosed
	}
	conn := c.conn
	c.mutex.RUnlock()

	// 创建请求帧
	frame := NewTCPRequestFrame(
		uint32(c.config.LocalNode),
		uint32(c.config.ServerNode),
		command,
		data,
	)

	frameData, err := BuildTCPFrame(frame)
	if err != nil {
		return nil, fmt.Errorf("构建帧失败: %w", err)
	}

	// 发送数据
	_, err = conn.Write(frameData)
	if err != nil {
		return nil, fmt.Errorf("发送失败: %w", err)
	}

	c.stats.TotalRequests++
	c.stats.LastRequestAt = time.Now()

	// 等待响应或超时
	select {
	case resp := <-c.responseCh:
		c.stats.LastResponseAt = time.Now()
		if resp.IsSuccess() {
			c.stats.SuccessCount++
		} else {
			c.stats.ErrorCount++
		}
		return resp, nil
	case <-time.After(c.config.Timeout):
		c.stats.TimeoutCount++
		return nil, ErrTimeout
	case <-c.closeChan:
		return nil, ErrConnectionClosed
	}
}

// receiveLoop 接收循环
func (c *FinsTCPClient) receiveLoop() {
	for {
		select {
		case <-c.closeChan:
			return
		default:
		}

		// 读取完整的TCP帧
		frameData, err := ReadTCPFrameFromConn(func(buf []byte) (int, error) {
			return io.ReadFull(c.conn, buf)
		})

		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return
			}
			if !c.closed {
				fmt.Printf("读取帧失败: %v\n", err)
			}
			return
		}

		// 解析响应
		resp, err := ParseTCPResponse(frameData)
		if err != nil {
			fmt.Printf("解析响应失败: %v\n", err)
			continue
		}

		// 发送响应到通道
		select {
		case c.responseCh <- resp:
		case <-time.After(100 * time.Millisecond):
			fmt.Printf("响应通道已满，丢弃响应\n")
		case <-c.closeChan:
			return
		}
	}
}

// GetStats 获取统计信息
func (c *FinsTCPClient) GetStats() ConnectionStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.stats
}

// IsConnected 检查是否已连接
func (c *FinsTCPClient) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.conn != nil && !c.closed
}

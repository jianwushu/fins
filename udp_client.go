package fins

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// FinsUDPClient FINS UDP客户端
type FinsUDPClient struct {
	config      *FinsClientConfig
	conn        *net.UDPConn
	serverAddr  *net.UDPAddr
	sequenceNo  uint16
	currentSID  byte
	pendingReqs map[byte]*PendingRequest
	mutex       sync.RWMutex
	stats       ConnectionStats
	closed      bool
	closeChan   chan struct{}
}

// NewUDPClient 创建UDP客户端
func NewUDPClient(config *FinsClientConfig) (*FinsUDPClient, error) {
	if config == nil {
		return nil, fmt.Errorf("配置不能为空")
	}

	// 解析服务器地址
	serverAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", config.IP, config.Port))
	if err != nil {
		return nil, fmt.Errorf("解析服务器地址失败: %w", err)
	}

	client := &FinsUDPClient{
		config:      config,
		serverAddr:  serverAddr,
		sequenceNo:  uint16(config.StartSID),
		currentSID:  config.FixedSID,
		pendingReqs: make(map[byte]*PendingRequest),
		closeChan:   make(chan struct{}),
	}

	return client, nil
}

// Connect 连接到服务器
func (c *FinsUDPClient) Connect() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		return fmt.Errorf("已经连接")
	}

	// 创建UDP连接
	conn, err := net.DialUDP("udp", nil, c.serverAddr)
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
func (c *FinsUDPClient) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	close(c.closeChan)

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}

	// 清理待处理请求
	for sid, req := range c.pendingReqs {
		close(req.Response)
		delete(c.pendingReqs, sid)
	}

	// 创建新的closeChan，允许重连
	c.closeChan = make(chan struct{})

	return nil
}

// getNextSID 获取下一个SID
func (c *FinsUDPClient) getNextSID() byte {
	switch c.config.SIDMode {
	case SIDFixed:
		return c.config.FixedSID
	case SIDIncrement:
		c.sequenceNo++
		if c.sequenceNo > uint16(c.config.MaxSID) {
			c.sequenceNo = uint16(c.config.StartSID)
		}
		return byte(c.sequenceNo)
	default:
		return c.config.FixedSID
	}
}

// SendRequest 发送请求
func (c *FinsUDPClient) SendRequest(command uint16, data []byte) (*FinsResponse, error) {
	c.mutex.Lock()
	if c.closed {
		c.mutex.Unlock()
		return nil, ErrConnectionClosed
	}

	sid := c.getNextSID()

	// 创建请求帧
	frame := NewUDPRequestFrame(c.config.LocalNode, c.config.ServerNode, sid, command, data)
	frameData, err := BuildUDPFrame(frame)
	if err != nil {
		c.mutex.Unlock()
		return nil, fmt.Errorf("构建帧失败: %w", err)
	}

	// 创建待处理请求
	req := &PendingRequest{
		SID:       sid,
		Request:   frameData,
		CreatedAt: time.Now(),
		Response:  make(chan *FinsResponse, 1),
	}
	c.pendingReqs[sid] = req
	c.mutex.Unlock()

	// 发送数据
	_, err = c.conn.Write(frameData)
	if err != nil {
		c.mutex.Lock()
		delete(c.pendingReqs, sid)
		c.mutex.Unlock()
		return nil, fmt.Errorf("发送失败: %w", err)
	}

	c.stats.TotalRequests++
	c.stats.LastRequestAt = time.Now()

	// 等待响应或超时
	select {
	case resp := <-req.Response:
		c.stats.LastResponseAt = time.Now()
		if resp.IsSuccess() {
			c.stats.SuccessCount++
		} else {
			c.stats.ErrorCount++
		}
		return resp, nil
	case <-time.After(c.config.Timeout):
		c.mutex.Lock()
		delete(c.pendingReqs, sid)
		c.mutex.Unlock()
		c.stats.TimeoutCount++
		return nil, ErrTimeout
	case <-c.closeChan:
		return nil, ErrConnectionClosed
	}
}

// receiveLoop 接收循环
func (c *FinsUDPClient) receiveLoop() {
	buffer := make([]byte, 2048)

	for {
		select {
		case <-c.closeChan:
			return
		default:
		}

		// 设置读取超时
		c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

		n, err := c.conn.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if !c.closed {
				fmt.Printf("接收数据错误: %v\n", err)
			}
			return
		}

		// 解析响应
		resp, err := ParseUDPResponse(buffer[:n])
		if err != nil {
			fmt.Printf("解析响应失败: %v\n", err)
			continue
		}

		// 查找对应的请求
		c.mutex.Lock()
		req, exists := c.pendingReqs[resp.SID]
		if exists {
			delete(c.pendingReqs, resp.SID)
		}
		c.mutex.Unlock()

		if exists {
			select {
			case req.Response <- resp:
			default:
			}
		}
	}
}

// GetStats 获取统计信息
func (c *FinsUDPClient) GetStats() ConnectionStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.stats
}

// IsConnected 检查是否已连接
func (c *FinsUDPClient) IsConnected() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.conn != nil && !c.closed
}

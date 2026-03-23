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
	pendingReqs map[byte]*PendingRequest
	mutex       sync.RWMutex
	closed      bool
	status      ConnectionStatus
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
		pendingReqs: make(map[byte]*PendingRequest),
		status:      ConnectionStatusDisconnected,
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

	c.closed = false
	c.status = ConnectionStatusConnecting

	// 创建UDP连接
	conn, err := net.DialUDP("udp", nil, c.serverAddr)
	if err != nil {
		c.status = ConnectionStatusDisconnected
		return fmt.Errorf("连接失败: %w", err)
	}

	c.conn = conn
	c.closed = false
	c.status = ConnectionStatusConnected

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
	c.status = ConnectionStatusClosed
	c.signalCloseLocked()

	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}

	c.clearPendingRequestsLocked()
	return nil
}

func (c *FinsUDPClient) clearPendingRequestsLocked() {
	for sid := range c.pendingReqs {
		delete(c.pendingReqs, sid)
	}
}

func (c *FinsUDPClient) signalCloseLocked() {
	select {
	case <-c.closeChan:
	default:
		close(c.closeChan)
	}
	c.closeChan = make(chan struct{})
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
	conn, req, closeChan, err := c.prepareRequest(command, data)
	if err != nil {
		return nil, err
	}

	if _, err = conn.Write(req.Request); err != nil {
		c.handleSendFailure(req.SID)
		return nil, fmt.Errorf("发送失败: %w", err)
	}

	return c.waitForResponse(req, closeChan)
}

func (c *FinsUDPClient) prepareRequest(command uint16, data []byte) (*net.UDPConn, *PendingRequest, chan struct{}, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return nil, nil, nil, ErrConnectionClosed
	}
	if c.status != ConnectionStatusConnected || c.conn == nil {
		return nil, nil, nil, ErrNotConnected
	}

	sid := c.getNextSID()
	frame := NewUDPRequestFrame(c.config.LocalNode, c.config.ServerNode, sid, command, data)
	frameData, err := BuildUDPFrame(frame)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("构建帧失败: %w", err)
	}

	req := &PendingRequest{
		SID:       sid,
		Request:   frameData,
		CreatedAt: time.Now(),
		Response:  make(chan *FinsResponse, 1),
	}
	c.pendingReqs[sid] = req
	return c.conn, req, c.closeChan, nil
}

func (c *FinsUDPClient) handleSendFailure(sid byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.pendingReqs, sid)
	if !c.closed {
		c.status = ConnectionStatusDisconnected
	}
}

func (c *FinsUDPClient) waitForResponse(req *PendingRequest, closeChan chan struct{}) (*FinsResponse, error) {
	timer := time.NewTimer(c.config.Timeout)
	defer timer.Stop()

	select {
	case resp := <-req.Response:
		return resp, nil
	case <-timer.C:
		c.removePendingRequest(req.SID)
		return nil, ErrTimeout
	case <-closeChan:
		c.removePendingRequest(req.SID)
		return nil, ErrConnectionClosed
	}
}

func (c *FinsUDPClient) removePendingRequest(sid byte) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.pendingReqs, sid)
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

		_ = c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))

		n, err := c.conn.Read(buffer)
		if err != nil {
			if c.handleReceiveError(err) {
				continue
			}
			return
		}

		resp, err := ParseUDPResponse(buffer[:n])
		if err != nil {
			fmt.Printf("解析响应失败: %v\n", err)
			continue
		}

		c.deliverResponse(resp)
	}
}

func (c *FinsUDPClient) handleReceiveError(err error) bool {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if !c.closed {
		c.status = ConnectionStatusDisconnected
		c.signalCloseLocked()
		fmt.Printf("接收数据错误: %v\n", err)
	}
	return false
}

func (c *FinsUDPClient) deliverResponse(resp *FinsResponse) {
	c.mutex.Lock()
	req, exists := c.pendingReqs[resp.SID]
	if exists {
		delete(c.pendingReqs, resp.SID)
	}
	c.mutex.Unlock()

	if !exists {
		return
	}

	select {
	case req.Response <- resp:
	default:
	}
}

// SetConnectionStatus 设置连接状态。
func (c *FinsUDPClient) SetConnectionStatus(status ConnectionStatus) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.status = status
	if status == ConnectionStatusClosed {
		c.closed = true
	}
}

// GetConnectionStatus 获取连接状态。
func (c *FinsUDPClient) GetConnectionStatus() ConnectionStatus {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.status
}

// IsConnected 检查是否已连接
func (c *FinsUDPClient) IsConnected() bool {
	return c.GetConnectionStatus() == ConnectionStatusConnected
}

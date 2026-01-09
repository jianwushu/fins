package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jianwushu/fins"
)

func main() {
	// 创建客户端配置
	config := fins.DefaultConfig("192.168.1.10")
	config.LocalNode = 0x01
	config.ServerNode = 0x64
	config.Timeout = 2 * time.Second

	// 创建UDP客户端
	client, err := fins.NewClient(config, false)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 连接到PLC
	if err := client.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}

	fmt.Println("成功连接到PLC")

	// 创建自定义重试策略
	retryPolicy := &fins.RetryPolicy{
		MaxRetries:    5,
		InitialDelay:  200 * time.Millisecond,
		MaxDelay:      3 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []error{
			fins.ErrTimeout,
			fins.ErrConnectionClosed,
		},
	}

	// 创建支持重试的客户端
	retryClient := fins.NewRetryableClient(client, retryPolicy)

	// 示例1: 带重试的读取操作
	fmt.Println("\n=== 示例1: 带重试的读取D100 ===")
	data, err := retryClient.ReadMemoryArea(fins.MemAreaD, 100, 1)
	if err != nil {
		log.Printf("读取失败（已重试）: %v", err)
	} else {
		fmt.Printf("成功读取数据: %v\n", data)
	}

	// 示例2: 带重试的写入操作
	fmt.Println("\n=== 示例2: 带重试的写入D200-D204 ===")
	values := []uint16{111, 222, 333, 444, 555}
	err = retryClient.WriteMemoryArea(fins.MemAreaD, 200, values)
	if err != nil {
		log.Printf("写入失败（已重试）: %v", err)
	} else {
		fmt.Println("成功写入数据")
	}

	// 示例3: 使用默认重试策略
	fmt.Println("\n=== 示例3: 使用默认重试策略 ===")
	defaultRetryClient := fins.NewRetryableClient(client, nil) // nil使用默认策略

	data, err = defaultRetryClient.ReadMemoryArea(fins.MemAreaD, 300, 5)
	if err != nil {
		log.Printf("读取失败: %v", err)
	} else {
		fmt.Printf("成功读取%d字节数据\n", len(data))
	}

	// 显示统计信息
	fmt.Println("\n=== 连接统计信息 ===")
	stats := client.GetStats()
	fmt.Printf("总请求数: %d\n", stats.TotalRequests)
	fmt.Printf("成功次数: %d\n", stats.SuccessCount)
	fmt.Printf("错误次数: %d\n", stats.ErrorCount)
	fmt.Printf("超时次数: %d\n", stats.TimeoutCount)

	fmt.Println("\n重试示例执行完成")
}

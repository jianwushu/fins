package main

import (
	"fmt"
	"log"
	"time"

	"github.com/jianwushu/fins"
)

func main() {
	config := fins.DefaultConfig("127.0.0.1")
	config.LocalNode = 0x01
	config.ServerNode = 0x64

	client, err := fins.NewClient(config, false)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	if err := client.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}

	retryPolicy := &fins.RetryPolicy{
		MaxRetries:    3,
		InitialDelay:  200 * time.Millisecond,
		MaxDelay:      3 * time.Second,
		BackoffFactor: 2.0,
	}

	retryClient := fins.NewRetryableClient(client, retryPolicy)

	fmt.Println("\n=== 示例1: 带重试的读取 D100 ===")
	value, err := retryClient.ReadWord("D100")
	if err != nil {
		log.Printf("读取失败: %v", err)
	} else {
		fmt.Printf("D100 = %d\n", value)
	}

	fmt.Println("\n=== 示例2: 带重试的批量写入 D200-D204 ===")
	values := []uint16{111, 222, 333, 444, 555}
	if err := retryClient.WriteWords("D200", values); err != nil {
		log.Printf("写入失败: %v", err)
	} else {
		fmt.Println("写入成功")
	}
}

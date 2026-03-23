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
	// config.Timeout = 0 * time.Millisecond

	client, err := fins.NewClient(config, false)
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	if err := client.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}

	retryPolicy := &fins.RetryPolicy{
		MaxRetries:    5,
		InitialDelay:  200 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []error{
			fins.ErrTimeout,
			fins.ErrConnectionClosed,
		},
	}

	fmt.Println("\n=== 示例1: 对 ReadWord(D100) 做函数级重试 ===")
	var value uint16
	err = fins.DoWithRetry(retryPolicy, func() error {
		var readErr error
		value, readErr = client.ReadWord("D100")
		fmt.Printf("%v,%v\n", time.Now().Format("2006-01-02 15:04:05.000"), readErr)
		return readErr
	})
	if err != nil {
		log.Printf("读取失败: %v", err)
	} else {
		fmt.Printf("D100 = %d\n", value)
	}

	fmt.Println("\n=== 示例2: 对 WriteWords(D200) 做函数级重试 ===")
	values := []uint16{111, 222, 333, 444, 555}
	err = fins.DoWithRetry(retryPolicy, func() error {
		writeErr := client.WriteWords("D200", values)
		fmt.Printf("%v,%v\n", time.Now().Format("2006-01-02 15:04:05.000"), writeErr)
		return writeErr

	})
	if err != nil {
		log.Printf("写入失败: %v", err)
	} else {
		fmt.Println("写入成功")
	}
}

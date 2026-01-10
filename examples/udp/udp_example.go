package main

import (
	"fmt"
	"log"

	"github.com/jianwushu/fins"
)

func main() {
	// 创建UDP客户端配置
	config := fins.DefaultConfig("127.0.0.1")
	config.LocalNode = 0x01
	config.ServerNode = 0x64 // PLC的FINS节点地址
	config.SIDMode = fins.SIDFixed
	config.FixedSID = 0x00

	// 创建UDP客户端
	client, err := fins.NewClient(config, false) // false表示使用UDP
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	// 连接到PLC
	if err := client.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}

	fmt.Println("成功连接到PLC")

	// 示例1: 读取D100寄存器
	fmt.Println("\n=== 示例1: 读取D100寄存器 ===")
	value, err := client.ReadDWord(100)
	if err != nil {
		log.Printf("读取D100失败: %v", err)
	} else {
		fmt.Printf("D100 = %d (0x%04X)\n", value, value)
	}

	// 示例3: 写入D100
	fmt.Println("\n=== 示例3: 写入D100 ===")
	if err := client.WriteDWord(100, 1234); err != nil {
		log.Printf("写入D100失败: %v", err)
	} else {
		fmt.Println("成功写入D100 = 1234")
	}

	client.Close()

	// 连接到PLC
	if err := client.Connect(); err != nil {
		log.Fatalf("连接失败: %v", err)
	}

	fmt.Println("成功连接到PLC")

	// 示例1: 读取D100寄存器
	fmt.Println("\n=== 示例1: 读取D100寄存器 ===")
	value, err = client.ReadDWord(100)
	if err != nil {
		log.Printf("读取D100失败: %v", err)
	} else {
		fmt.Printf("D100 = %d (0x%04X)\n", value, value)
	}

	fmt.Println("\n示例执行完成")
}

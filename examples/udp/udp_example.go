package main

import (
	"fmt"
	"log"

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

	fmt.Println("成功通过UDP连接到PLC")

	fmt.Println("\n=== 示例1: 读取 D100 ===")
	value, err := client.ReadWord("D100")
	if err != nil {
		log.Printf("读取失败: %v", err)
	} else {
		fmt.Printf("D100 = %d (0x%04X)\n", value, value)
	}

	fmt.Println("\n=== 示例2: 读取 CIO0.00 ===")
	bit, err := client.ReadBit("CIO0.00")
	if err != nil {
		log.Printf("读取 CIO0.00 失败: %v", err)
	} else {
		fmt.Printf("CIO0.00 = %v\n", bit)
	}

	fmt.Println("\n=== 示例3: 写入 D100 ===")
	if err := client.WriteWord("D100", 1234); err != nil {
		log.Printf("写入失败: %v", err)
	} else {
		fmt.Println("成功写入 D100 = 1234")
	}
}

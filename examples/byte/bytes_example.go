package main

import (
	"encoding/binary"
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

	fmt.Println("\n=== 示例1: 读取 D100 开始的 10 个字节 ===")
	data, err := client.ReadBytes("D100", 10)
	if err != nil {
		log.Printf("读取失败: %v", err)
	} else {
		fmt.Printf("读取数据: % X\n", data)
	}

	fmt.Println("\n=== 示例2: 写入字节数组到 D200 ===")
	writeData := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}
	if err := client.WriteBytes("D200", writeData); err != nil {
		log.Printf("写入失败: %v", err)
	} else {
		fmt.Println("写入成功")
	}

	fmt.Println("\n=== 示例3: 字节数组与整数转换 ===")
	value := uint32(0x12345678)
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, value)
	if err := client.WriteBytes("D300", buf); err != nil {
		log.Printf("写入32位整数失败: %v", err)
	} else if readData, err := client.ReadBytes("D300", 4); err == nil {
		readValue := binary.BigEndian.Uint32(readData)
		fmt.Printf("读取回的32位整数: 0x%08X\n", readValue)
	}

	fmt.Println("\n=== 示例4: 字符串写入/读取 ===")
	text := "FINS Protocol"
	if err := client.WriteBytes("D400", []byte(text)); err != nil {
		log.Printf("写入字符串失败: %v", err)
	} else if readData, err := client.ReadBytes("D400", uint16(len(text))); err == nil {
		fmt.Printf("读取字符串: %s\n", string(readData))
	}
}

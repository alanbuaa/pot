package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net"
)

// 定义接收的数据结构
type ParameterUpdate struct {
	Action     string      `json:"action"`
	Level      string      `json:"level"`
	Group      string      `json:"group"`
	ParamName  string      `json:"param_name"`
	ParamValue interface{} `json:"param_value"`
}

func main() {
	// 监听指定端口
	listener, err := net.Listen("tcp", ":22326")
	if err != nil {
		log.Fatalf("无法启动服务器: %v", err)
	}
	defer listener.Close()

	fmt.Println("服务器已启动，监听端口 22326...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("接受连接错误: %v", err)
			continue
		}
		// 为每个连接创建一个goroutine
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// 读取长度信息（4字节）
	lengthBuf := make([]byte, 4)
	_, err := conn.Read(lengthBuf)
	if err != nil {
		log.Printf("读取数据长度错误: %v", err)
		return
	}

	// 将4字节转换为整数
	length := binary.BigEndian.Uint32(lengthBuf)

	// 读取实际数据
	dataBuf := make([]byte, length)
	_, err = conn.Read(dataBuf)
	if err != nil {
		log.Printf("读取数据错误: %v", err)
		return
	}

	// 解析JSON数据
	var update ParameterUpdate
	err = json.Unmarshal(dataBuf, &update)
	if err != nil {
		log.Printf("JSON解析错误: %v", err)
		return
	}

	// 处理接收到的数据
	fmt.Printf("收到参数更新请求:\n")
	fmt.Printf("Action: %s\n", update.Action)
	fmt.Printf("Level: %s\n", update.Level)
	fmt.Printf("Group: %s\n", update.Group)
	fmt.Printf("Parameter Name: %s\n", update.ParamName)
	fmt.Printf("Parameter Value: %v\n", update.ParamValue)
}

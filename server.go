package main

import (
	"bufio"
	"fmt"
	"gochat/proto"
	"io"
	"net"
	"time"
)

// 用户结构
type Client struct {
	name     string
	infoChan chan string
}

// 消息结构
type Message struct {
	addr string
	info string
}

// 全局map，存储在线用户
var onlineMap = make(map[string]Client)

// 全局消息
var messageChan = make(chan Message)

func WriteToClient(infoChan chan string, conn net.Conn) {
	for msg := range infoChan {
		data, err := proto.Encode(msg)
		if err != nil {
			fmt.Println("encode err: ", err)
			return
		}
		conn.Write(data)
	}
}

func makeMessage(addr string, client Client, message string) Message {
	return Message{addr, "[" + addr + "]" + client.name + ": " + message}
}

func HandlerConnect(conn net.Conn) {
	defer conn.Close()
	// 获取网络地址：ip+port
	addr := conn.RemoteAddr().String()
	// 创建客户端结构吗，默认用户名为网络地址
	client := Client{addr, make(chan string)}
	// 添加到在线用户
	onlineMap[addr] = client
	// 启动协程负责给当前用户发送消息
	go WriteToClient(client.infoChan, conn)
	// 发送用户上线消息
	messageChan <- makeMessage(addr, client, "[login]")
	fmt.Println(addr + " [login]")
	// 判断用户退出
	isQuit := make(chan bool)
	// 判断用户活跃
	isActive := make(chan bool)

	// 启动协程接收用户消息
	go func() {
		reader := bufio.NewReader(conn)
		for {
			n, msg, err := proto.Decode(reader)
			if err == io.EOF || n == 0 {
				isQuit <- true
				return
			}
			if err != nil {
				fmt.Println("read err: ", err)
				return
			}
			if msg[0] == '>' {
				if msg[1:] == "who" {
					// 查询在线列表
					data, err := proto.Encode("[online list]")
					if err != nil {
						fmt.Println("encode err: ", err)
						return
					}
					conn.Write(data)
					for addr, client := range onlineMap {
						userInfo := "[" + addr + "]" + client.name
						data, err := proto.Encode(userInfo)
						if err != nil {
							fmt.Println("encode err: ", err)
							return
						}
						conn.Write(data)
					}
				} else if n > 8 && msg[1:8] == "rename " {
					// 修改用户名
					client.name = msg[8:]
					onlineMap[addr] = client
					data, err := proto.Encode("[rename successful]")
					if err != nil {
						fmt.Println("encode err: ", err)
						return
					}
					conn.Write(data)
				}
			} else {
				// 广播普通消息
				messageChan <- makeMessage(addr, client, msg)
			}
			isActive <- true
		}
	}()

	for {
		select {
		case <-isQuit:
			fmt.Println(addr + " [logout]")
			delete(onlineMap, addr)
			messageChan <- makeMessage(addr, client, "[logout]")
			return
		case <-isActive:
			// 什么都不做，重置超时计时器
			continue
		case <-time.After(time.Second * 60):
			delete(onlineMap, addr)
			messageChan <- makeMessage(addr, client, "[logout]")
			return
		}
	}
}

func Manager() {
	for message := range messageChan {
		//发送消息给除发送人以外的所有用户
		for addr, client := range onlineMap {
			if addr != message.addr {
				client.infoChan <- message.info
			}
		}
	}
}

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:15535")
	if err != nil {
		fmt.Println("listen err: ", err)
	}
	defer listener.Close()

	// 启动管理协程，管理全局map和message
	go Manager()

	// 循环监听客户端请求
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("accept err: ", err)
			return
		}

		// 启动一个协程处理客户端数据请求
		go HandlerConnect(conn)
	}
}

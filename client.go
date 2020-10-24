package main

import (
	"bufio"
	"fmt"
	"gochat/proto"
	"net"
	"os"
	"strings"
)

func read(conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		_, msg, err := proto.Decode(reader)
		if err != nil {
			fmt.Println("read err:  ", err)
			return
		}
		fmt.Println(msg)
	}
}

func main() {
	conn, err := net.Dial("tcp", "127.0.0.1:15535")
	if err != nil {
		fmt.Println("err: ", err)
		return
	}

	go read(conn)

	reader := bufio.NewReader(os.Stdin)
	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if strings.ToLower(input) == ">q" {
			return
		}
		data, err := proto.Encode(input)
		if err != nil {
			fmt.Println("encode err: ", err)
			return
		}
		conn.Write(data)
	}
}

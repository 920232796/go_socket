package main

import (
	"fmt"
	"net"
	"io" 
)

func main() {

	fmt.Println("hello world")

	localAddrString := ":7449" //监听本地浏览器请求
	serverAddrString := "47.100.10.8:7449" //服务端地址+端口

	localAddr, err := net.ResolveTCPAddr("tcp", localAddrString) //把字符串变量转换为 TCPAddr特殊变量

	if err != nil {
		fmt.Println("resolve addr err is ", err.Error())
		return
	}

	serverAddr, err := net.ResolveTCPAddr("tcp", serverAddrString)
	if err != nil {
		return 
	}


	//开始监听
	listener, err := net.ListenTCP("tcp", localAddr)
	if err != nil {
		fmt.Println("listener err is ", err.Error())
		return 
	}

	fmt.Println("listening......")
	for {

		localConn, err := listener.AcceptTCP()
		if err != nil {
			fmt.Println("conn err is ", err.Error())
			return 
		}
		//有了连接，便可以收发数据了

		go handleLocalFunc(localConn, serverAddr) //一个处理函数，把这个连接传进去. 我们需要把浏览器给客户端的消息
		//都转发给服务端！

	}

}

func handleLocalFunc(localConn *net.TCPConn, serverAddr *net.TCPAddr) {
	fmt.Println("connected!")
	defer localConn.Close()
	//先去拨通一下服务端
	proxyServer, err := net.DialTCP("tcp", nil, serverAddr)
	if err != nil {
		fmt.Println("dial tcp err is ", err.Error())
		return 
	}
	defer proxyServer.Close()
	//把数据直接不用动，转发给服务端, 这里我们先随便试试能不能互通
	// proxyServer.Write([]byte("hello server!"))//实验可以通！！哈哈 继续

	//不断的读浏览器给客户端的数据，然后转发给服务端
	buf := make([]byte, 256)

	go func() {
		for {
			readCount, err := localConn.Read(buf)
			if err != nil {
				if err == io.EOF {
					fmt.Println("read end")
					return 
				}
				fmt.Println("read err is ", err.Error())
				return 
			}

			//写给服务端
			if readCount > 0 {
				writeCount, err := proxyServer.Write(buf[:readCount])
				if err != nil {
					fmt.Println("write err is ", err.Error())
					return 
				}
				if readCount != writeCount {
					fmt.Println("read and write err")
					return 
				}
			}

		}
	}()

	//别忘了服务器返回的一些信息，也要再写回到客户端呀，客户端再告诉浏览器代理信息
	for {
		readCount, err := proxyServer.Read(buf)
			if err != nil {
				if err == io.EOF {
					fmt.Println("read end")
					return 
				}
				fmt.Println("read err is ", err.Error())
				return 
			}

			//写给客户端
			if readCount > 0 {
				writeCount, err := localConn.Write(buf[:readCount])
				if err != nil {
					fmt.Println("write err is ", err.Error())
					return 
				}
				if readCount != writeCount {
					fmt.Println("read and write err")
					return 
				}
			}
	}




	// n, err := localConn.Read(buf)


	// if err != nil {
	// 	if err == io.EOF {
	// 		fmt.Println("read end !")
	// 		return 
	// 	}
	// 	fmt.Println("read err is ", err.Error())
	// 	return 
	// }

	// fmt.Println(buf[:n]) //打印一下消息

	// //第一次浏览器应该给客户端发消息，请求验证 我们判断第一个字节是不是5 也就是表示socket5 
	// if buf[0] != 0x05 {
	// 	fmt.Println("socket err~")
	// 	return 
	// }

	// // 如果是5，客户端就要要告诉浏览器，可以不需要验证 你可以给我发需要请求的服务器的地址。
	// localConn.Write([]byte{0x05, 0x00})

	// // 浏览器知道了，那么现在要给我们发送需要请求的服务器地址了，我们需要接收一下
	// n, err = localConn.Read(buf) //
}
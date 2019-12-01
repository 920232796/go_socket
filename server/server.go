package main

import (
	"fmt"
	"net"
	"io"
	"encoding/binary"
)

func main() {

	fmt.Println("hello world")
	serverAddrString := ":7449" //监听地址+端口

	serverAddr, err := net.ResolveTCPAddr("tcp", serverAddrString)
	if err != nil {
		fmt.Println("resolve err is ", err.Error())
		return 
	}

	listener, err := net.ListenTCP("tcp", serverAddr) 
	if err != nil {
		fmt.Println("listen err is ", err.Error())
		return 
	}

	//开始监听
	for {
		serverConn, err := listener.AcceptTCP()
		if err != nil {
			fmt.Println("accept err is ", err.Error())
			return 
		}

		go handleServerFunc(serverConn)
	}
}

func handleServerFunc(serverConn *net.TCPConn) {
	buf := make([]byte, 256)
	defer serverConn.Close()
	//服务端要开始解析浏览器代理经过客户端传过来的数据包，解析socket协议
	//读一次，写一次，这样 就类似于握手，都没问题了再进行数据传输.

	//首先要读  客户端先来问服务器，用哪种验证方式连接，数据包有三个字段
	//1 ver 表示socket 类型是什么， 咱们是 socket5
	//2 nmethods 表示第三个字段methods的长度为多少
	//3 methods 表示客户端都支持哪些验证方式，可以好几种，比如 不用验证 用户名密码啥的 有几种就占几个字节
	_, err := serverConn.Read(buf)
	if err != nil || buf[0] != 0x05 {
		//说明不是socket5协议，放弃
		return 
	}


	//如果是socket5没问题，那么就告诉客户端 可以不用验证 直接发数据
	//我们直接选择，不需要验证的通信方式
	serverConn.Write([]byte{0x05, 0x00})// 0x00表示不用验证


	//继续读，不需要验证，那么客户端会发过来它请求的服务器地址是啥，那么我们就读出来发过来的请求的服务器地址是啥
	n, err := serverConn.Read(buf) 
	if err != nil || n < 7 {
		//最小也得7个字节的数据才行阿！
		//具体可以看协议内容
		fmt.Println("read err is ", err.Error())
		return 
	}
	//buf[0]这个位置字段是ver，代表socket版本，我们不需要去关心，因为肯定是5 上次已经握手完了

	//buf[1]这个位置字段是cmd
	//代表客户端请求的类型，值长度也是1个字节，有三种类型； connect, bind, udp 分别是 0x01, 0x02, 0x03
	// 我暂时估计为啥只支持0x01，是因为浏览器请求网页就是用这种方式。
	if buf[1] != 0x01 {
		return 
		//只支持这一种连接方式
	}
	//buf[2] rsv ，是保留字，没啥用，不管
	//buf[3] atyp，地址类型，表示了客户端所请求的服务器的地址类型
	//1. ipv4 0x01  2. domainname 0x03 （也就是域名，这个应该最常用,若为域名就得调用dns解析出来地址） 3. ipv6 0x04
	//buf[4] 解析出来的真正地址, 长度不定
	//buf[end] 2字节 请求服务器的端口是哪个
	var dIP []byte 
	switch buf[3] {
	case 0x01:
		dIP = buf[4:4+net.IPv4len]
	case 0x03:
		ipAddr, err := net.ResolveIPAddr("ip", string(buf[5:n-2]))//为啥从5开始，因为4的位置是个空格 打印看了已经
		if err != nil {
			return 
		}
		dIP = ipAddr.IP
		fmt.Println("addr is ", string(buf[4:n-2]))
	case 0x04:
		dIP = buf[4 : 4+net.IPv6len]
	default: 
		return 
	}
	dPort := buf[n-2:]
	dstAddr := &net.TCPAddr{
		IP:   dIP,
		Port: int(binary.BigEndian.Uint16(dPort)),
	}
	// fmt.Println(dstAddr)

	//有了地址，我们就可以去连接真正要请求的服务器了！
	targetServer, err := net.DialTCP("tcp", nil, dstAddr) 
	if err != nil {
		fmt.Println("dial err is ", err.Error())
		return 
	}

	defer targetServer.Close()
	//连接成功，这时候服务端要跟客户端说一声，连接成功了！
	serverConn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})

	//然后的话，客户端就要把请求数据发过来了！！
	go func() {
		//服务端不断接收客户端发来的数据，再转给真正的服务器
		for {
			readCount, err := serverConn.Read(buf)
			if err != nil {
				if err == io.EOF {
					fmt.Println("read end")
					return 
				}
				fmt.Println("read err is ", err.Error())
				return 
			}

			//写给服务器
			if readCount > 0 {
				writeCount, err := targetServer.Write(buf[:readCount])
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

	//同时，服务端接收到服务器返回的数据，也要写回给客户端！！！
	for {
		readCount, err := targetServer.Read(buf)
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
				writeCount, err := serverConn.Write(buf[:readCount])
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


	// for {
	// 	countRead, err := serverConn.Read(buf)
	// 	if err != nil {
	// 		if err == io.EOF {
	// 			fmt.Println("read end ")
	// 			return 
	// 		}
	// 		fmt.Println("err is ", err.Error())
	// 		return 
	// 	}
		
	// 	fmt.Println(string(buf[:countRead]))
	// }
}
/**
 * 基于Go语言的聊天室-客户端程序
 **/
package main

import (
	"fmt"
	"net"
	"os"
	//"strings"
)

// 用户名
var uname string

// 接受数据Channel
var recvData = make(chan string)

// 发送数据Channel
var sentData = make(chan string)

// 创建连接对象
var conn net.Conn

// 用户是否被屏蔽
var isSheild bool = false

/**
 * 错误检查
 * @param err   错误类型数据
 * @param info  错误信息提示内容
 **/
func checkError(err error, info string) {

	if err != nil {

		fmt.Println(info + "  " + err.Error())

		os.Exit(1)

	}

}

/**
 * 数据接收
 * @param  recvData  接收数据Channel
 * @param  conn      TCP连接对象指针
 **/
func recv(recvData chan string, conn net.Conn) {

	for {

		buf := make([]byte, 1024)

		n, err := conn.Read(buf)

		checkError(err, "Connection")

		recvData <- string(buf[:n])

	}

}

/**
 * 数据发送
 * @param  sentData  接收数据Channel
 * @param  conn      TCP连接对象
 **/
func sent(sentData chan string, conn net.Conn) {

	for {

		data := <-sentData

		_, err := conn.Write([]byte(data))

		checkError(err, "Connection")

	}
}

/**
 * 聊天内容显示
 **/
func showMassage() {

	for {

		message := <-recvData

		if message == "/shield" {

			isSheild = true

		} else {

			fmt.Println(message)

		}

	}

}

/**
 * 聊天数据输入
 **/
func inputMessage() {

	var input, objUser string

	for {

		fmt.Scanln(&input)

		switch input {

		// 用户退出
		case "/quit":
			fmt.Println("退出聊天室，欢迎下次使用！")
			os.Exit(0)

		// 向单个用户发送消息
		case "/to":
			fmt.Println("请输入聊天对象：")
			fmt.Scanln(&objUser)
			fmt.Println("请输入消息：")
			fmt.Scanln(&input)
			if len(input) != 0 {
				input = "To-" + uname + "-" + objUser + "-" + input
			}

			// 默认群发
		default:
			if len(input) != 0 {
				input = uname + " : " + input
			}
		}

		// 发送数据
		if len(input) != 0 {

			if !isSheild {

				sentData <- input

			} else {

				fmt.Println("你已被管理员屏蔽，无法发言！")
			}

			input = ""

		}

	}

}

/**
 * 登录程序
 **/
func clientLog() {

LOOP:
	for {
		var username, password string
		fmt.Println("请输入用户名：")
		fmt.Scanln(&username)
		if username == "" {
			goto LOOP
		}
		fmt.Println("请输入密码：")
		fmt.Scanln(&password)

		sentData <- "Log-" + username + "-" + password

		res := <-recvData

		if res == "success" {

			fmt.Println("登录成功，你已进入聊天室！")

			uname = username // 登录成功保存用户名

			go showMassage() // 启动聊天内容显示协程

			break

		} else {

			fmt.Println(res)
		}
	}

	inputMessage() // 启动聊天内容输入

}

/**
 * 注册程序
 **/
func clientReg() {

	for {

		var username, pwd, pwdCheck string
		fmt.Println("请输入用户名：")
		fmt.Scanln(&username)
		fmt.Println("请输入密码：")
		fmt.Scanln(&pwd)
		fmt.Println("请确认密码：")
		fmt.Scanln(&pwdCheck)

		if pwdCheck != pwd {

			fmt.Println("两次输入密码不一致，请重新注册！")

		} else {

			sentData <- ("Reg-" + username + "-" + pwd) // 发送用户名和密码

			res := <-recvData // 等待登录结果

			if res == "success" {
				fmt.Println("注册成功，请登录")
				break
			} else {
				fmt.Println(res)
			}
		}
	}

	clientLog() // 转至登录
}

/**
 * 创建TCP连接
 * @param  tcpAddr  TCP地址格式
 * return net.Conn
 **/
func createTCP(tcpaddr string) net.Conn {

	tcpAddr, err := net.ResolveTCPAddr("tcp4", tcpaddr)

	checkError(err, "ResolveTCPAddr")

	conn, err := net.DialTCP("tcp", nil, tcpAddr)

	checkError(err, "DialTCP")

	fmt.Println("连接服务器成功！请输入将要进行的操作：1、登录  2、注册")

	return conn
}

/**
 * 客户端启动程序
 * @param  tcpAddr  TCP地址格式
 **/
func startClient(tcpAddr string) {

	conn = createTCP(tcpAddr) // 创建TCP连接

	go recv(recvData, conn) // 启动数据接收协程

	go sent(sentData, conn) // 启动数据发送协程

LOOP:
	{

		var input string

		fmt.Scanln(&input)

		if input == "1" {

			clientLog() // 登录

		} else if input == "2" {

			clientReg() // 注册

		} else {

			fmt.Println("输入有误，请重新输入！")
			goto LOOP
		}

	}

}

/**
 * 客户端主程序
 * @param  ip:port   服务端ip地址以及端口
 **/
func main() {

	if len(os.Args) == 2 {

		startClient(os.Args[1]) // 启动客户端

	} else {

		fmt.Println("输入错误！")

	}

}

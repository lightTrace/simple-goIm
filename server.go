/**
 * 基于Go语言的聊天室-服务端程序
 **/
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
)

// 常量配置
const (
	dataFileName = "database.txt" // 用户注册数据文件保存名
	dataSize     = 5 * 1024       // 文件大小
	maxLog       = 30             // 最大同时登录人数
	maxReg       = 500            // 最大用户注册人数
	success      = "success"      // 登录注册返回给客户端成功标识
)

// 用户数据结构体
type userData struct {
	Password string
	Level    int
}

// 声明客户端的结构体
type clientInfo struct {
	conn net.Conn // 客户端的TCP连接对象

	sentData chan string // 服务器向客户端发送数据通道

}

var conns = make(map[string]clientInfo) // 声明成功登录之后的连接对象map

var uData = make(map[string]userData) // 声明登录用户数据map

var messages = make(chan string) // 声明消息channel

var adminList = make(map[string]string) // 声明管理员列表

var connection = make(chan net.Conn) // 声明连接管理Channel

var ipToUname = make(map[string]string) // ip地址与用户名队名

/**
 * 错误显示并退出
 * @param err   错误类型数据
 * @param info  错误信息提示内容
 **/
func errorExit(err error, info string) {

	if err != nil {

		fmt.Println(info + "  " + err.Error())

		os.Exit(1)

	}

}

/**
 * 错误检查
 * @param err   错误类型数据
 * @param info  错误信息提示内容
 **/
func checkError(err error, info string) bool {

	if err != nil {

		fmt.Println(info + "  " + err.Error())

		return false
	}

	return true
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

		if err != nil {

			connection <- conn

			return
		}

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

		if err != nil {

			connection <- conn

			return
		}

	}
}

/**
 * 客户端连接管理
 * @param  conn
 **/
func connManager(connection chan net.Conn) {
	for {
		conn := <-connection
		username := ipToUname[conn.RemoteAddr().String()]
		notesInfo(messages, "用户:"+username+"已退出聊天室！")
		conn.Close()
		delete(conns, username)
		delete(ipToUname, conn.RemoteAddr().String())
	}
}

/**
* 服务器向客户端发送消息数据解析与封装操作
* @param client  客户端连接信息
* @param messages 数据通道中的数据
         群发数据格式：普通字符串
         单对单发送格式："To-" + uname(发送用户) + "-" + objUser（目标用户） +"-" +input
         用户列表："List-"+objUser（目标用户）+"-"+Listinfo
         命令:objUser + "-" + command
**/
func dataSent(conns *map[string]clientInfo, messages chan string) {

	for {
		msg := <-messages

		fmt.Println(msg)

		data := strings.Split(msg, "-") // 聊天数据分析:
		length := len(data)

		if length == 2 { // 管理员单个用户发送控制命令

			(*conns)[data[0]].sentData <- data[1]

		} else if length == 3 { // 用户列表

			(*conns)[data[1]].sentData <- data[2]

		} else if length == 4 { // 向单个用户发送数据

			msg = data[1] + " say to you : " + data[3]

			(*conns)[data[2]].sentData <- msg

		} else {
			// 群发
			for _, value := range *conns {
				value.sentData <- msg
			}
		}

	}
}

/**
 * 用户登录之后将用户接受的数据放入公共channel
 * @param messages 数据通道
 **/
func dataRec(conns *map[string]clientInfo, username string, recvData, messages chan string) {
	for {
		data := <-recvData
		if _, ok := (*conns)[username]; !ok {
			return
		}
		if len(data) > 0 {
			messages <- data
		}

	}

}

/**
 * 发送系统通知消息
 * @param messages  channel
 * @param info   channel
 **/
func notesInfo(messages chan string, info string) {
	messages <- info
}

/**
 * 获取用户列表
 * @param string   用户列表
 **/
func userList() string {

	var userList string = "当前在线用户列表："
	for user := range conns {

		userList += "\r\n" + user

	}
	return userList
}

/**
 * 获取全部已注册用户的数据,用于登录注册时验证
 * @param filename  文件名
 * return  userData  用户数据map
 */
func getAllUser(filename string) map[string]userData {
	buf := make([]byte, dataSize)
	udata := make(map[string]userData)
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0766) //打开保存用户的文件
	defer file.Close()                                              //保证文件关闭
	errorExit(err, "getAllUser")
	n, _ := file.Read(buf)
	json.Unmarshal(buf[:n], &udata) //将流文件转成json并赋值给udata这个map
	return udata
}

/**
 * 添加新注册用户数据
 * @param filename  文件名
 * @param username  用户名
 * @param password  密码
 */
func insertNewUser(filename, username, password string) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0766)
	defer file.Close()
	checkError(err, "insertNewUser")
	uData[username] = userData{password, 1}
	data, _ := json.MarshalIndent(uData, "", "  ")
	file.WriteString(string(data))
}

/**
 * 用户登录处理
 * @param username  用户登录名
 * @param password  用户密码
 * return 返回处理状态以及对应信息
 **/
func logProcess(username, password string) (status bool, info string) {

	if len(conns) == maxLog {
		return false, "当前登录人数已满,请稍后登录"
	}
	if user, ok := uData[username]; !ok {
		return false, "用户名或密码错误!"
	} else {
		if user.Password == password {
			if _, ok := conns[username]; ok {
				return false, "用户已登录！"
			} else {
				return true, success
			}
		} else {
			return false, "用户名或密码错误!"
		}
	}
}

/**
 * 用户注册处理
 * @param username  用户登录名
 * @param password  用户密码
 * return 返回处理状态以及对应信息
 **/
func regProcess(username, password string) (status bool, info string) {
	if len(uData) == maxReg {
		return false, "注册人数已满！"
	}
	if _, ok := uData[username]; ok {
		return false, "用户已存在，请更换注册名!"
	} else {
		insertNewUser(dataFileName, username, password)
		return true, success
	}
}

/**
 * 用户进入聊天室资格认证
 * @param recvData,sentData 客户端连接接收和发送Channel
 * @param conn TCP连接对象
 **/
func userAuth(conn *net.Conn, recvData, sentData chan string) {

	for {

		data := strings.Split(<-recvData, "-") // 等待用户发送登录或注册数据
		flag, username, password := data[0], data[1], data[2]

		if flag == "Reg" {

			_, info := regProcess(username, password)
			sentData <- info

		} else {

			status, info := logProcess(username, password)
			sentData <- info

			if status == true {
				messages <- ("用户:" + username + "进入聊天室")            // 用户登录成功，发送系统通知
				conns[username] = clientInfo{*conn, sentData}       // 记录用户登录状态信息
				ipToUname[(*conn).RemoteAddr().String()] = username // 记录IP与用户名对应信息
				messages <- "List-" + username + "-" + userList()   // 向用户发送已在线用户列表
				go dataRec(&conns, username, recvData, messages)    // 开启服务器对该客户端数据接收线程
				break
			}
		}
	}
}

/**
 * 设置TCP连接
 * @param  tcpAddr  TCP地址格式
 * return net.TCPListener
 **/
func createTCP(tcpaddr string) *net.TCPListener {

	tcpAddr, err := net.ResolveTCPAddr("tcp4", tcpaddr)

	errorExit(err, "ResolveTCPAddr")

	l, err := net.ListenTCP("tcp", tcpAddr)

	errorExit(err, "DialTCP")

	return l
}

/**
 * 管理员登录、注册
 **/
func intoManager() {
	fmt.Println("请输入将要进行操作：1、管理员注册 2、管理员登录")
	var input string
LOOP:
	{
		fmt.Scanln(&input)
		switch input {
		case "1":
			adminReg()
		case "2":
			adminLog()
		default:
			goto LOOP
		}
	}
	admimManager(messages)
}

/**
 * 管理员登录程序
 **/
func adminLog() {

	for {
		var adminname, password string
		fmt.Println("请输入管理员用户名：")
		fmt.Scanln(&adminname)
		fmt.Println("请输入管理员密码：")
		fmt.Scanln(&password)
		if pwd, ok := adminList[adminname]; !ok {
			fmt.Println("用户名或者密码错误")
		} else {
			if pwd != password {
				fmt.Println("用户名或者密码错误！")
			} else {
				fmt.Println("登录成功！")
				break
			}
		}
	}

}

/**
 * 管理员注册程序
 **/
func adminReg() {
	var adminname, password string
	fmt.Println("请输入管理员用户名：")
	fmt.Scanln(&adminname)
	fmt.Println("请输入管理员密码：")
	fmt.Scanln(&password)
	adminList[adminname] = password //将注册的管理员姓名密码保存到adminList中，单次启动有效
	fmt.Println("注册成功！请登录")
	adminLog() //跳转到登录
}

/**
 * 管理员管理模块
 **/
func admimManager(messages chan string) {

	for {
		var input, objUser string
		fmt.Scanln(&input)
		switch input {
		case "/to":
			fmt.Println(userList())
			fmt.Println("请输入聊天对象：")
			fmt.Scanln(&objUser)
			if _, ok := conns[objUser]; !ok {
				fmt.Println("不存在此用户!")
			} else {
				fmt.Println("请输入消息：")
				fmt.Scanln(&input)
				notesInfo(messages, "To-Manager-"+objUser+"-"+input)
			}

		case "/all":
			fmt.Println("请输入消息：")
			fmt.Scanln(&input)
			notesInfo(messages, "Manager say : "+input)

		case "/shield":
			fmt.Println(userList())
			fmt.Println("请输入屏蔽用户名：")
			fmt.Scanln(&objUser)
			notesInfo(messages, objUser+"-/shield")
			notesInfo(messages, "用户："+objUser+"已被管理员禁言！")

		case "/remove":
			fmt.Println(userList())
			fmt.Println("请输入踢出用户名：")
			fmt.Scanln(&objUser)
			notesInfo(messages, "用户："+objUser+"已被管理员踢出聊天室！")
			if _, ok := conns[objUser]; !ok {
				fmt.Println("不存在此用户!")
			} else {
				conns[objUser].conn.Close() // 删除该用户的连接
				delete(conns, objUser)      // 从已登录的列表中删除该用户
			}
		}

	}
}

/**
 * 服务端启动程序
 * @param  port  设置监听端口
 **/
func StartServer(port string) {

	l := createTCP(":" + port)

	fmt.Println("服务端启动成功，正在监听端口！")

	go dataSent(&conns, messages) // 启动服务器广播线程
	go intoManager()              // 启动管理模块
	go connManager(connection)    // 启动连接管理线程

	for {

		conn, err := l.Accept()
		if checkError(err, "Accept") == false {
			continue
		}
		fmt.Println("客户端:", conn.RemoteAddr().String(), "连接服务器成功！")
		var recvData = make(chan string)
		var sentData = make(chan string)
		go recv(recvData, conn)                // 开启对客户端的接受数据线程
		go sent(sentData, conn)                // 开启对客户端的发送数据线程
		go userAuth(&conn, recvData, sentData) // 用户资格认证
	}

}

/**
 * 服务端主程序
 * @param  port  设置端口
 **/
func main() {

	if len(os.Args) == 2 {
		uData = getAllUser(dataFileName) // 用户登录、注册数据初始化

		StartServer(os.Args[1]) // 启动客户端

	} else {

		fmt.Println("输入错误！")
	}

}

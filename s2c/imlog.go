package s2c

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

//测试性能用
func ImPacketInTime(messages ...interface{}) { //
	var LogPath = "./"
	imuserlog, err := os.OpenFile(LogPath+"ImPacketInTime.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		fmt.Println(err)
	}
	defer imuserlog.Close()
	userlog := log.New(imuserlog, "\r\n", log.Ldate|log.Ltime)
	userlog.Println(messages)
}
func ImPacketOutTime(messages ...interface{}) { //
	var LogPath = "./"
	imuserlog, err := os.OpenFile(LogPath+"ImPacketOutTime.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		fmt.Println(err)
	}
	defer imuserlog.Close()
	userlog := log.New(imuserlog, "\r\n", log.Ldate|log.Ltime)
	userlog.Println(messages)
}

//10.4：日志保留（1.平常用户登录登出记录，表的变化；2.IM错误（server+session+packetio）;3.数据库操作（主要记录错误））
func ImUserLog(messages ...interface{}) { //记录用户的登入登出大操作。
	var year, month, day = time.Now().Date()
	var LogPath = "./log/" + strconv.Itoa(year) + "/" + month.String() + "/" + strconv.Itoa(day) + "/"
	err := os.MkdirAll(LogPath, 0777)
	if err != nil {
		fmt.Println(err)
		return
	}
	imuserlog, err := os.OpenFile(LogPath+"imUser.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		fmt.Println(err)
	}
	defer imuserlog.Close()
	userlog := log.New(imuserlog, "\r\n", log.Ldate|log.Ltime)
	userlog.Println(messages)
}

func ImMaxSession(messages ...interface{}) {
	var year, month, day = time.Now().Date()
	var LogPath = "./log/" + strconv.Itoa(year) + "/" + month.String() + "/" + strconv.Itoa(day) + "/"
	err := os.MkdirAll(LogPath, 0777)
	if err != nil {
		fmt.Println(err)
		return
	}
	impacketlog, err := os.OpenFile(LogPath+"imMaxSession.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer impacketlog.Close()
	packetlog := log.New(impacketlog, "\r\n", log.Ldate|log.Ltime)
	packetlog.Println(messages)
}

func ImCountbeatLog(messages ...interface{}) {
	var year, month, day = time.Now().Date()
	var LogPath = "./log/" + strconv.Itoa(year) + "/" + month.String() + "/" + strconv.Itoa(day) + "/"
	err := os.MkdirAll(LogPath, 0777)
	if err != nil {
		fmt.Println(err)
		return
	}
	imsessionlog, err := os.OpenFile(LogPath+"imBeat.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)

	if err != nil {
		fmt.Println(err)
		return
	}
	defer imsessionlog.Close()
	sessionlog := log.New(imsessionlog, "\r\n", log.Ldate|log.Ltime)
	sessionlog.Println(messages)
}

/*
func ImSessionLog(messages ...interface{}) {
	var year, month, day = time.Now().Date()
	var LogPath = "./log/" + strconv.Itoa(year) + "/" + month.String() + "/" + strconv.Itoa(day) + "/"
	err := os.MkdirAll(LogPath, 0777)
	if err != nil {
		fmt.Println(err)
		return
	}
	imsessionlog, err := os.OpenFile(LogPath+"imSession.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)

	if err != nil {
		fmt.Println(err)
		return
	}
		defer imsessionlog.Close()
	sessionlog := log.New(imsessionlog, "\r\n", log.Ldate|log.Ltime)
	sessionlog.Println(messages)
}

func ImSessionErrLog(messages ...interface{}) {
	year, month, day := time.Now().Date()
	var LogPath = "./log/" + strconv.Itoa(year) + "/" + month.String() + "/" + strconv.Itoa(day) + "/"
	err := os.MkdirAll(LogPath, 0777)
	if err != nil {
		fmt.Println(err)
		return
	}
	imsessionlog, err := os.OpenFile(LogPath+"imSessionErr.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)

	if err != nil {
		fmt.Println(err)
		return
	}
		defer imsessionlog.Close()
	sessionlog := log.New(imsessionlog, "\r\n", log.Ldate|log.Ltime)
	sessionlog.Println(messages)
}*/
func ImServerLog(messages ...interface{}) { //IM错误（server+session+packetio）
	var year, month, day = time.Now().Date()
	var LogPath = "./log/" + strconv.Itoa(year) + "/" + month.String() + "/" + strconv.Itoa(day) + "/"
	err := os.MkdirAll(LogPath, 0777)
	if err != nil {
		fmt.Println(err)
		return
	}
	imserverlog, err := os.OpenFile(LogPath+"imServer.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)

	if err != nil {
		fmt.Println(err)
		return
	}
	defer imserverlog.Close()
	serverlog := log.New(imserverlog, "\r\n", log.Ldate|log.Ltime)
	serverlog.Println(messages)
}

func ImBusinessLog(messages ...interface{}) { //业务服务器调用IM接口记录
	var year, month, day = time.Now().Date()
	var LogPath = "./log/" + strconv.Itoa(year) + "/" + month.String() + "/" + strconv.Itoa(day) + "/"
	err := os.MkdirAll(LogPath, 0777)
	if err != nil {
		fmt.Println(err)
		return
	}
	imbusinesslog, err := os.OpenFile(LogPath+"imBusiness.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)

	if err != nil {
		fmt.Println(err)
		return
	}
	defer imbusinesslog.Close()
	businesslog := log.New(imbusinesslog, "\r\n", log.Ldate|log.Ltime)
	businesslog.Println(messages)
}
func DatabaseLog(messages ...interface{}) { //记录数据库操作，以及问题
	var year, month, day = time.Now().Date()
	var LogPath = "./log/" + strconv.Itoa(year) + "/" + month.String() + "/" + strconv.Itoa(day) + "/"
	err := os.MkdirAll(LogPath, 0777)
	if err != nil {
		fmt.Println(err)
		return
	}
	databaselog, err := os.OpenFile(LogPath+"imDatabase.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)

	if err != nil {
		fmt.Println(err)
		return
	}
	defer databaselog.Close()
	dblog := log.New(databaselog, "\r\n", log.Ldate|log.Ltime)
	dblog.Println(messages)
}

/*
func main() {
	ImSessionLog("djsbfhg", "aehju", 7777777777)
	ImServerLog("kwfhenrfbg", 7777777777)
	ImPacketLog("djkadgbnv", 7777777777)
	ImUserLog("dkjdgb", 7777777777)
	ImBusinessLog("ekjfb", 7777777777)
	ImSessionErrLog("kdhfbjueagh", 7777777777)
}*/

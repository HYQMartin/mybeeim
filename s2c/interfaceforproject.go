package s2c

import (
	"beeim/imdatabase"
	"beeim/s2c/business"
	"beeim/s2c/protocol"
	"beeim/uitl/uniq"
	"code.google.com/p/goprotobuf/proto"
	"net"
	"strconv"
	"time"
)

var (
	feedbackCMD = &business.Command{ //回馈包
		Sender:            proto.Uint32(0), //0：IM；1：业务系统
		Result:            proto.Uint32(0), //0：正确，1：IM错误，2:业务系统语法错误，3：请求无法实现
		ResultDescription: proto.String(""),
	}
)

const (
	maxMessage     = 1024
	BusinessKey    = string("IamBusinessLayer") //作为AES加密Key
	allUserMessage = 1                          //查询消息记录时，所有用户的标志
)

var nilPacket = Packet{}

type InterfaceForProject struct {
	InterfaceDB imdatabase.ImDb
}

func (self *InterfaceForProject) CreateGroup(interfaceCMD *business.Command) (resultPacket Packet, err error) {
	//1.从interfaceCMD 中解析出所需要的信息
	//2.处理，并且得到结果
	//3.把结果写入CMD，并且做成Packet 返回。

	//1.解析...................................
	uids := interfaceCMD.GetUid()
	if uids == nil {
		feedbackCMD.Result = proto.Uint32(2) //业务系统没有输入uid，请求无法实现
		feedbackCMD.ResultDescription = proto.String("no uid")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 101)
		return resultPacket, err
	}
	//检测uid是否合法。。。待优化。。。
	var uidstring string
	for k, v := range uids {
		if k == len(uids)-1 {
			uidstring += strconv.Itoa(int(v))
		} else {
			uidstring += strconv.Itoa(int(v)) + ","
		}
	}
	checkUser, err := self.InterfaceDB.Db.Query("select count(uid) from user where uid in (" + uidstring + ")")
	if err != nil {
		ImBusinessLog("Interface  check uids err", err)
		feedbackCMD.Result = proto.Uint32(1)
		feedbackCMD.ResultDescription = proto.String("IM error.")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 101)
		return resultPacket, err
	}
	defer checkUser.Close()
	var sumUids int //在user表中的输入的uid总数
	if checkUser.Next() {
		err = checkUser.Scan(&sumUids)
		if err != nil {
			ImBusinessLog("Interface  check uids err", err)
			feedbackCMD.Result = proto.Uint32(1)
			feedbackCMD.ResultDescription = proto.String("IM error.")
			resultPacket = *MakeInterfacePacket(feedbackCMD, 101)
			return resultPacket, err
		}
	}
	if sumUids != len(uids) {
		feedbackCMD.Result = proto.Uint32(3) //业务系统输入的uid有误，请求无法实现
		feedbackCMD.ResultDescription = proto.String("uid error.")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 101)
		return resultPacket, err
	}
	//2.处理...................................
	//IM在数据库的dialog表（扫表？）建立一条信息，包含dialog_ID，所有uid。
	//将dialog_ID添加到act_dialog中。返回dialog_ID。

	//1.新建dialogId，
	dialogid := uniq.GetUniq() //不合理，应当唯一，且和uid区分开
	err = self.InterfaceDB.AddDialog(dialogid, uids)
	if err != nil {
		ImBusinessLog("Interface.CreateGroup.AddDialog err", err)
		feedbackCMD.Result = proto.Uint32(1)
		feedbackCMD.ResultDescription = proto.String("IM error.")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 101)
		return resultPacket, err
	}
	//2.将dialog_ID添加到act_dialog中
	err = self.InterfaceDB.AddActDialog(dialogid)
	if err != nil {
		ImBusinessLog("Interface.CreateGroup.AddActDialog err", err)
		feedbackCMD.Result = proto.Uint32(1)
		feedbackCMD.ResultDescription = proto.String("IM error.")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 101)
		return resultPacket, err
	}
	//3.打包...................................
	var returnCMD = &business.Command{
		Sender:            proto.Uint32(0),
		Groupid:           proto.Uint64(dialogid),
		Result:            proto.Uint32(0),
		ResultDescription: proto.String("OK"),
	}
	resultPacket = *MakeInterfacePacket(returnCMD, 101)
	return resultPacket, nil
}

func (self *InterfaceForProject) PushMessage(interfaceCMD *business.Command) (resultPacket Packet, err error) {
	//1.从interfaceCMD 中解析出所需要的信息
	//2.处理，并且得到结果
	//3.把结果写入CMD，并且做成Packet 返回。

	//1.解析...................................
	if interfaceCMD.GetUid() == nil {
		feedbackCMD.Result = proto.Uint32(2)
		feedbackCMD.ResultDescription = proto.String("no uid")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 102)
		return resultPacket, err
	}
	senderUid := interfaceCMD.GetUid()[0]
	dialogID := interfaceCMD.GetGroupid()
	if dialogID == 0 {
		feedbackCMD.Result = proto.Uint32(2)
		feedbackCMD.ResultDescription = proto.String("no dialogid")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 102)
		return resultPacket, err
	}
	if interfaceCMD.GetMessage() == nil {
		feedbackCMD.Result = proto.Uint32(2)
		feedbackCMD.ResultDescription = proto.String("no message")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 102)
		return resultPacket, err
	}
	message := interfaceCMD.GetMessage()[0]
	random := GenRandom()
	var (
		requestMess = &protocol.Message{
			Data:   proto.String(message),
			Random: proto.Uint32(random),
		}
		requestCMD = &protocol.Command{
			Sender:   proto.Uint64(senderUid), //sender's uid
			Receiver: proto.Uint64(dialogID),  //0=server
			Message:  requestMess,
		}
	)
	pushMessagePacket := *MakePacket(requestCMD, 2)
	_, fl := SessionTable[senderUid]
	if fl == false {
		//fmt.Println("uid no in")
		feedbackCMD.Result = proto.Uint32(1)
		feedbackCMD.ResultDescription = proto.String("sender isn't online")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 102)
		return resultPacket, nil
	}

	SessionTable[senderUid].incoming <- pushMessagePacket
	feedbackCMD.Result = proto.Uint32(0)
	feedbackCMD.ResultDescription = proto.String("success")
	resultPacket = *MakeInterfacePacket(feedbackCMD, 102)
	return resultPacket, nil
	//2.处理...................................
	//2.1 从dialogID （act_dialog表---dialog表）       中得到receiverUid,若不为senderUid，则转发，或者存离线
	//2.2 查找receiverUid失败，则返回要求服务器先建群
	//3.打包...................................
}

func (self *InterfaceForProject) LeaveGroup(interfaceCMD *business.Command) (resultPacket Packet, err error) {
	//1.从interfaceCMD 中解析出所需要的信息
	//2.处理，并且得到结果
	//3.把结果写入CMD，并且做成Packet 返回。

	//1.解析...................................
	dialogid := interfaceCMD.GetGroupid()
	if dialogid == 0 {
		feedbackCMD.Result = proto.Uint32(2)
		feedbackCMD.ResultDescription = proto.String("no dialogid")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 103)
		return resultPacket, err
	}
	//var uids []uint64
	//	uids = make([]uint64, 0, len(interfaceCMD.GetUid()))
	uids := interfaceCMD.GetUid()[:]
	if uids == nil {
		feedbackCMD.Result = proto.Uint32(2)
		feedbackCMD.ResultDescription = proto.String("no uid")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 103)
		return resultPacket, err
	}
	////检测uid,dialogid是否合法。。。未完成********************

	//2.处理...................................

	//查询数据库的dialog表，给对应的uid添加当前时间戳。修该chatMap的dialogId的成员。

	//1.修改数据库dialog表,添加时间戳
	dbUpdateDialog, err := self.InterfaceDB.Db.Prepare("update dialog set timestamp=? where dialog_id=? and uid=?")

	if err != nil {
		ImBusinessLog("Prepare to update dialog error:", err)
		feedbackCMD.Result = proto.Uint32(1)
		feedbackCMD.ResultDescription = proto.String("IM error.")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 103)
		return resultPacket, err
	}
	defer dbUpdateDialog.Close()
	for _, uid := range uids {
		_, err = dbUpdateDialog.Exec(time.Now().Format("2006-01-02 15:04:05"), dialogid, uid)
		if err != nil {
			ImBusinessLog("Update dialog error : ", err)
			feedbackCMD.Result = proto.Uint32(1)
			feedbackCMD.ResultDescription = proto.String("IM error.")
			resultPacket = *MakeInterfacePacket(feedbackCMD, 103)
			return resultPacket, err
		}
		//2.修改chatMap
		_, flag := SessionTable[uid]
		if flag == true {
			delete(SessionTable[uid].chatMap, dialogid)
		}
	}
	//3.打包...................................
	feedbackCMD.Result = proto.Uint32(0)
	feedbackCMD.ResultDescription = proto.String("OK")
	resultPacket = *MakeInterfacePacket(feedbackCMD, 103)
	return resultPacket, nil
}

func (self *InterfaceForProject) CloseDialog(interfaceCMD *business.Command) (resultPacket Packet, err error) {
	//1.从interfaceCMD 中解析出所需要的信息
	//2.处理，并且得到结果
	//3.把结果写入CMD，并且做成Packet 返回。

	//1.解析...................................
	dialogID := interfaceCMD.GetGroupid()
	if dialogID == 0 {
		feedbackCMD.Result = proto.Uint32(2)
		feedbackCMD.ResultDescription = proto.String("no dialogid.")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 104)
		return resultPacket, err
	}
	////检测uid,dialogid是否合法。。。未完成********************

	//2.处理...................................
	//删除数据库的act_dialog表对应的dialog_ID
	dbDeleteActdialog, err := self.InterfaceDB.Db.Prepare("delete from act_dialog where dialog_id=?")

	if err != nil {
		ImBusinessLog("Prepare to delete act_dialog error :", err)
		feedbackCMD.Result = proto.Uint32(1)
		feedbackCMD.ResultDescription = proto.String("IM error.")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 104)
		return resultPacket, err
	}
	defer dbDeleteActdialog.Close()
	_, err = dbDeleteActdialog.Exec(dialogID)
	if err != nil {
		ImBusinessLog("Delete act_dialog error :", err)
		feedbackCMD.Result = proto.Uint32(1)
		feedbackCMD.ResultDescription = proto.String("IM error.")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 104)
		return resultPacket, err
	}
	//失败返回错误，成功返回nil
	//3.打包...................................
	feedbackCMD.Result = proto.Uint32(0)
	feedbackCMD.ResultDescription = proto.String("OK")
	resultPacket = *MakeInterfacePacket(feedbackCMD, 104)
	return resultPacket, nil
}

func (self *InterfaceForProject) LogIn(interfaceCMD *business.Command) (resultPacket Packet, err error) {
	//1.从interfaceCMD 中解析出所需要的信息
	//2.处理，并且得到结果
	//3.把结果写入CMD，并且做成Packet 返回。

	//1.解析...................................
	if interfaceCMD.GetUid() == nil {
		feedbackCMD.Result = proto.Uint32(2)
		feedbackCMD.ResultDescription = proto.String("no uid.")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 105)
		return resultPacket, err
	}
	uid := interfaceCMD.GetUid()[0]
	//验证uid，未完成/********************

	//2.处理...................................
	//在IM的内存表user表添加用户ID和token，返回token

	//1.1 内存表添加user
	token := MakeToken()
	UserTable[uid] = Token{token, 1}
	//1.2 数据库添加user
	err = self.InterfaceDB.AddUser(uid)
	if err != nil {
		ImBusinessLog("Interface.LogIn.AddUser err", err)
		feedbackCMD.Result = proto.Uint32(1)
		feedbackCMD.ResultDescription = proto.String("IM error.")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 105)
		return resultPacket, err
	}
	//3.打包...................................
	feedbackCMD.Result = proto.Uint32(0)
	feedbackCMD.ResultDescription = proto.String(token)
	resultPacket = *MakeInterfacePacket(feedbackCMD, 105)
	return resultPacket, nil
}
func (self *InterfaceForProject) LogOut(interfaceCMD *business.Command) (resultPacket Packet, err error) {
	//1.从interfaceCMD 中解析出所需要的信息
	//2.处理，并且得到结果
	//3.把结果写入CMD，并且做成Packet 返回。

	//1.解析...................................
	if interfaceCMD.GetUid() == nil {
		feedbackCMD.Result = proto.Uint32(2)
		feedbackCMD.ResultDescription = proto.String("no uid.")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 106)
		return resultPacket, err
	}
	uid := interfaceCMD.GetUid()[0]
	//2.处理...................................
	//删除IM的user表对应的uid，得到session表chatMap的所有dialog_ID，
	//并删除session表
	delete(UserTable, uid)
	delete(SessionTable, uid)
	//3.打包...................................
	feedbackCMD.Result = proto.Uint32(0)
	feedbackCMD.ResultDescription = proto.String("OK")
	resultPacket = *MakeInterfacePacket(feedbackCMD, 106)
	return resultPacket, nil
}

func (self *InterfaceForProject) InqueryHistory(interfaceCMD *business.Command) (resultPacket Packet, err error) {

	//1.从interfaceCMD 中解析出所需要的信息
	//2.处理，并且得到结果
	//3.把结果写入CMD，并且做成Packet 返回。

	//1.解析...................................
	if interfaceCMD.GetUid() == nil {
		feedbackCMD.Result = proto.Uint32(2)
		feedbackCMD.ResultDescription = proto.String("no uid.")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 107)
		return resultPacket, err
	}
	uid := interfaceCMD.GetUid()[0]
	dialogid := interfaceCMD.GetGroupid()
	if dialogid == 0 {
		feedbackCMD.Result = proto.Uint32(2)
		feedbackCMD.ResultDescription = proto.String("no dialogid.")
		resultPacket = *MakeInterfacePacket(feedbackCMD, 107)
		return resultPacket, err
	}
	//2.处理...................................
	//如果dialogid为空，查询dialog表获取有uid的dialog_ID。
	//根据dialog_ID查询数据库的message表，对于dialog表的uid带有时间戳的，则返回时间戳之前dialog的message
	//其他的返回所有的message。

	var t string
	var info string
	messageTimestamp := make([]string, 0, maxMessage) //
	messageText := make([]string, 0, maxMessage)

	if dialogid != allUserMessage { ////1.查询特定用户的聊天信息//dialogid不为空，
		//1.1获取uid时间戳
		dbUserTimestamp, err := self.InterfaceDB.Db.Query("select timestamp from dialog where uid=? and dialog_id=?", uid, dialogid)
		if err != nil {
			ImBusinessLog("Get user timestamp fail 1 ：", err)
			feedbackCMD.Result = proto.Uint32(1)
			feedbackCMD.ResultDescription = proto.String("IM error.")
			resultPacket = *MakeInterfacePacket(feedbackCMD, 107)
			return resultPacket, err
		}
		defer dbUserTimestamp.Close()
		var uidTimes string
		if dbUserTimestamp.Next() {
			err = dbUserTimestamp.Scan(&uidTimes)
			if err != nil {
				ImBusinessLog("Get user timestamp fail 2:", err)
				feedbackCMD.Result = proto.Uint32(1)
				feedbackCMD.ResultDescription = proto.String("IM error.")
				resultPacket = *MakeInterfacePacket(feedbackCMD, 107)
				return resultPacket, err
			}
		}
		//1.2查询message表
		if uidTimes != "0000-00-00 00:00:00" { //用户带时间戳，查找时间戳之前的message
			dbSelectMessages, err := self.InterfaceDB.Db.Query("select timestamp, information from message where dialog_id=? and timestamp<? order by timestamp desc", dialogid, uidTimes)
			if err != nil {
				ImBusinessLog("get message fail 1:", err)
				feedbackCMD.Result = proto.Uint32(1)
				feedbackCMD.ResultDescription = proto.String("IM error.")
				resultPacket = *MakeInterfacePacket(feedbackCMD, 107)
				return resultPacket, err
			}
			defer dbSelectMessages.Close()
			for dbSelectMessages.Next() {
				err := dbSelectMessages.Scan(&t, &info)
				if err != nil {
					ImBusinessLog("get message fail 2", err)
					feedbackCMD.Result = proto.Uint32(1)
					feedbackCMD.ResultDescription = proto.String("IM error.")
					resultPacket = *MakeInterfacePacket(feedbackCMD, 107)
					return resultPacket, err
				}
				messageTimestamp = append(messageTimestamp, t)
				messageText = append(messageText, info)
			}
			//fmt.Println("用户带时间戳", messageTimestamp)
		} else { //1.3.用户不带时间戳，查找所有dialogid的message,不考虑时间戳
			dbSelectMessages, err := self.InterfaceDB.Db.Query("select timestamp, information from message where dialog_id=? order by timestamp desc", dialogid)
			if err != nil {
				ImBusinessLog("find all dialogid message query err", err)
				feedbackCMD.Result = proto.Uint32(1)
				feedbackCMD.ResultDescription = proto.String("IM error.")
				resultPacket = *MakeInterfacePacket(feedbackCMD, 107)
				return resultPacket, err
			}
			defer dbSelectMessages.Close()
			for dbSelectMessages.Next() {
				err := dbSelectMessages.Scan(&t, &info)
				if err != nil {
					ImBusinessLog("find all dialogid message err", err)
					feedbackCMD.Result = proto.Uint32(1)
					feedbackCMD.ResultDescription = proto.String("IM error.")
					resultPacket = *MakeInterfacePacket(feedbackCMD, 107)
					return resultPacket, err
				}
				messageTimestamp = append(messageTimestamp, t)
				messageText = append(messageText, info)
			}
		}

	} else { ////2.查询所有的聊天信息 。dialogid为空,查询dialog表获取有uid1的dialog_ID
		//2.1获取的所有uid时间戳为0的dialogid的消息
		partMsgTimestamp := make([]string, 0, maxMessage)
		partMsgText := make([]string, 0, maxMessage)
		//fmt.Println("查询所有的聊天信息************")
		dbSelectAllMsgs, err := self.InterfaceDB.Db.Query("select timestamp, information from message where dialog_id in (select dialog_id from dialog where uid=? and timestamp in (0) ) order by timestamp desc ", uid)

		if err != nil {
			imdatabase.DatabaseLog(" Select from messageTable error:", err)
			ImBusinessLog(" Select from messageTable error:", err)
			feedbackCMD.Result = proto.Uint32(1)
			feedbackCMD.ResultDescription = proto.String("IM error.")
			resultPacket = *MakeInterfacePacket(feedbackCMD, 107)
			return resultPacket, err
		}
		defer dbSelectAllMsgs.Close()
		for dbSelectAllMsgs.Next() {
			err := dbSelectAllMsgs.Scan(&t, &info)
			if err != nil {
				ImBusinessLog(" Scan from messageTable error:", err)
				imdatabase.DatabaseLog(" Scan from messageTable error:", err)
				feedbackCMD.Result = proto.Uint32(1)
				feedbackCMD.ResultDescription = proto.String("IM error.")
				resultPacket = *MakeInterfacePacket(feedbackCMD, 107)
				return resultPacket, err
			}
			partMsgTimestamp = append(partMsgTimestamp, t)
			partMsgText = append(partMsgText, info)
		}
		part1Length := len(partMsgTimestamp)
		//fmt.Println("所有记录11", partMsgTimestamp, "partlenght:", part1Length)
		//第一部分的长度，即第二部分的起始位置
		//2.2获取uid带有时间戳的dialogid，以及时间戳
		dbSelectUidTimestamp, err := self.InterfaceDB.Db.Query("select dialog_id,timestamp from dialog where uid=? and timestamp not in (0) order by timestamp", uid)

		if err != nil {
			imdatabase.DatabaseLog(" Select from dialogTable error:", err)
			ImBusinessLog(" Select from dialogTable error:", err)
			feedbackCMD.Result = proto.Uint32(1)
			feedbackCMD.ResultDescription = proto.String("IM error.")
			resultPacket = *MakeInterfacePacket(feedbackCMD, 107)
			return resultPacket, err
		}
		defer dbSelectUidTimestamp.Close()
		var userDial uint64
		var userTimes string
		for dbSelectUidTimestamp.Next() {
			err := dbSelectUidTimestamp.Scan(&userDial, &userTimes)
			if err != nil {
				ImBusinessLog(" Scan from dialogTable error:", err)
				feedbackCMD.Result = proto.Uint32(1)
				feedbackCMD.ResultDescription = proto.String("IM error.")
				resultPacket = *MakeInterfacePacket(feedbackCMD, 107)
				return resultPacket, err
			}
			dbSelectMsgs, err := self.InterfaceDB.Db.Query("select timestamp, information from message where dialog_id=? and timestamp<? order by timestamp ", userDial, userTimes)

			if err != nil {
				ImBusinessLog("Select from message error:", err)
				feedbackCMD.Result = proto.Uint32(1)
				feedbackCMD.ResultDescription = proto.String("IM error.")
				resultPacket = *MakeInterfacePacket(feedbackCMD, 107)
				return resultPacket, err
			}
			defer dbSelectMsgs.Close()
			for dbSelectMsgs.Next() {
				err = dbSelectMsgs.Scan(&t, &info)
				if err != nil {
					ImBusinessLog("Scan from message error:", err)
					feedbackCMD.Result = proto.Uint32(1)
					feedbackCMD.ResultDescription = proto.String("IM error.")
					resultPacket = *MakeInterfacePacket(feedbackCMD, 107)
					return resultPacket, err
				}
				partMsgTimestamp = append(partMsgTimestamp, t)
				partMsgText = append(partMsgText, info)
			}
			//fmt.Println(...)
			QuickSort(partMsgTimestamp, partMsgText, part1Length, len(partMsgTimestamp)-1)
			MergeSort(partMsgTimestamp, partMsgText, &messageTimestamp, &messageText, part1Length, len(partMsgTimestamp)-1)
			///	messageTimestamp = append(messageTimestamp, partMsgTimestamp...)
			//	messageText = append(messageText, partMsgText...)
		}
	}
	//3.打包...................................
	returnCMD := &business.Command{
		Sender:            proto.Uint32(0),
		Result:            proto.Uint32(0),
		ResultDescription: proto.String("OK"),
	}
	returnCMD.Message = make([]string, 0, maxMessage)
	returnCMD.Timestamp = make([]string, 0, maxMessage)
	returnCMD.Message = append(returnCMD.Message, messageText...)
	returnCMD.Timestamp = append(returnCMD.Timestamp, messageTimestamp...)
	resultPacket = *MakeInterfacePacket(returnCMD, 107)
	return resultPacket, nil
}

func interfaceHandler(interfaceNumber uint32, interfaceCMD *business.Command) (resultPacket Packet, err error) {
	var interfaceForProject InterfaceForProject
	//打开数据库
	err = interfaceForProject.InterfaceDB.ConnectDb()
	if err != nil {
		ImBusinessLog("Open imdatabase error :", err)
		feedbackCMD.Result = proto.Uint32(1)
		feedbackCMD.ResultDescription = proto.String("IM error.")
		resultPacket = *MakeInterfacePacket(feedbackCMD, interfaceNumber)
		return resultPacket, err
	}
	switch interfaceNumber {
	case 101:
		resultPacket, err = interfaceForProject.CreateGroup(interfaceCMD)
		//fmt.Println("/////", resultPacket)
		//pa, _ := GetInterfaceCMD(resultPacket.GetData())
		//fmt.Println("****.", pa, resultPacket.GetType())
		if err != nil {
			ImBusinessLog("interfaceHandler err", err)
		}
	case 102:
		resultPacket, err = interfaceForProject.PushMessage(interfaceCMD)
		if err != nil {
			ImBusinessLog("interfaceHandler err", err)
		}
	case 103:
		resultPacket, err = interfaceForProject.LeaveGroup(interfaceCMD)
		if err != nil {
			ImBusinessLog("interfaceHandler err", err)
		}
	case 104:
		resultPacket, err = interfaceForProject.CloseDialog(interfaceCMD)
		if err != nil {
			ImBusinessLog("interfaceHandler err", err)
		}
	case 105:
		resultPacket, err = interfaceForProject.LogIn(interfaceCMD)
		if err != nil {
			ImBusinessLog("interfaceHandler err", err)
		}
	case 106:
		resultPacket, err = interfaceForProject.LogOut(interfaceCMD)
		if err != nil {
			ImBusinessLog("interfaceHandler err", err)
		}
	case 107:
		resultPacket, err = interfaceForProject.InqueryHistory(interfaceCMD)
		if err != nil {
			ImBusinessLog("interfaceHandler err", err)
		}
	}

	interfaceForProject.InterfaceDB.CloseDb()
	return resultPacket, nil
}
func MakeInterfacePacket(cmd *business.Command, Type uint32) *Packet {
	//这里需要改
	data, err := proto.Marshal(cmd)
	if err != nil {
		ImBusinessLog("Protobuf : make packet error.", err)
		return &nilPacket
	}
	packet := NewPacket()
	packet.SetType(Type)
	packet.SetData(data)
	return packet
}
func GetInterfaceCMD(data []byte) (cmd *business.Command, err error) {
	//这里需要改
	Cmd := &business.Command{}
	err = proto.Unmarshal(data, Cmd)
	if err != nil {
		ImBusinessLog("Protobuf : get CMD error.", err)
		return nil, err
	}
	return Cmd, nil
}
func acceptInterface(conns net.Conn) {
	//1.接受一个包
	//2.解析包，得到type和data
	//3.调用处理函数
	//4.把处理函数写成包丢过去
	//5.关闭conn并return
	//1.读包.................................
	//fmt.Println("************", conns)
	packetReader := NewPacketReader(conns)
	packetReader.AESKEY = []byte(BusinessKey)
	var interfacePacket Packet
	if packet, err := packetReader.ReadPacket(); err == nil {
		interfacePacket = *packet
		//fmt.Println("pppp", packet)
	} else {
		ImBusinessLog("acceptInterface readpacket err", err)
		//发送回馈包
		//关闭conn并结束协程
		conns.Close()
		return
	}
	//2.解析包.................................
	interfaceNumber := interfacePacket.GetType()
	interfaceCMD, err := GetInterfaceCMD(interfacePacket.GetData())
	//fmt.Println(interfaceCMD, "///", interfaceNumber)
	if err != nil {
		ImBusinessLog("interfaceCMD GetCMD() error", err)
		//发送回馈包
		//关闭conn并结束协程
		conns.Close()
		return
	}
	//3.处理.................................
	var resultPacket Packet
	resultPacket, err = interfaceHandler(interfaceNumber, interfaceCMD)
	//	ii, _ := GetInterfaceCMD(resultPacket.GetData())
	//fmt.Println(ii, "++++++++++++++++++++++")
	if err != nil {
		//发送回馈包
		//关闭conn并结束协程
		conns.Close()
		return
	}
	if resultPacket.GetData() == nil {
		//发送回馈包,告知未正确调用接口
		//关闭conn并结束协程
		conns.Close()
		return
	}
	//4.写包.................................
	packetWriter := NewPacketWriter(conns)
	packetWriter.AESKEY = []byte(BusinessKey)
	if err := packetWriter.WritePacket(&resultPacket, nil); err != nil {
		ImBusinessLog("Write resultPacket err", err)
		conns.Close()
		return
	}
	/*if err := packetWriter.Flush(); err != nil {
		ImBusinessLog("Flush resultPacket err", err)
		conns.Close()
		return
	}*/
	//5.关闭.................................
	if conns != nil {
		//	conns.Close()
	}
	//fmt.Println(conns)
}

func QuickSort(s1 []string, s2 []string, low int, high int) {
	if low < high {
		privokey := s1[low]
		lo, hi := low, high
		for low < high {
			for low < high && s1[high] <= privokey {
				high--
			}
			s1[low], s1[high] = s1[high], s1[low]
			s2[low], s2[high] = s2[high], s2[low]
			for low < high && s1[low] >= privokey {
				low++
			}
			s1[low], s1[high] = s1[high], s1[low]
			s2[low], s2[high] = s2[high], s2[low]
		}
		pivotLocat := low
		QuickSort(s1, s2, lo, pivotLocat-1)
		QuickSort(s1, s2, pivotLocat+1, hi)
	}
}
func MergeSort(s1 []string, s2 []string, res1 *[]string, res2 *[]string, secondBegin int, end int) {
	i, j := 0, secondBegin
	for i < secondBegin && j <= end {
		if s1[i] > s1[j] {
			*res1 = append(*res1, s1[i])
			*res2 = append(*res2, s2[i])
			i++
		} else {
			*res1 = append(*res1, s1[j])
			*res2 = append(*res2, s2[j])
			j++
		}
	}
	if i < secondBegin {
		*res1 = append(*res1, s1[i:secondBegin]...)
		*res2 = append(*res2, s2[i:secondBegin]...)
	}
	if j <= end {
		*res1 = append(*res1, s1[j:end+1]...)
		*res2 = append(*res2, s2[i:end+1]...)
	}
	//	fmt.Print("\nMerge:", res1)
}

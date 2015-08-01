package s2c

import (
	"beeim/imdatabase"
	//"fmt"
	r "math/rand"
	_ "mysql"
	"net"
	"time"
)

const (
	MaxDelayTime  = 60                       //服务器发出公钥  30s内没收到正确的回馈包则会关闭conn
	APPKEY        = string("appkeyismartin") //和客户端统一好的appkey
	MaxOutLineMsg = 200
)

var MaxMemberInGroup int //这里设置一个群的成员最大数
var TokenTime int        //令牌有效时间，单位秒
var MaxRecendTimes int   //最大重发次数

//********Session 属性
type Session struct {
	uid          uint64              //用户id
	conn         net.Conn            //Session.clientIP = conn.LocalAddr().String()
	beatCount    int32               //心跳
	incoming     chan Packet         //输入通道
	outgoing     chan Packet         //输出通道
	packetReader *PacketReader       //读包
	packetWriter *PacketWriter       //写包
	quiting      chan byte           //quiting通道用来调用Session.quit()，进而调用server.leave()
	chatMap      map[uint64][]uint64 //用户花名册，群号+群成员
	quitfor      chan byte           //用于关闭转发通道
	ifQuit       bool
}

//********Session 方法
func (self *Session) GetUid() uint64 {
	return self.uid
}
func (self *Session) SetUid(uid uint64) {
	self.uid = uid
}
func (self *Session) GetConn() net.Conn {
	return self.conn
}
func (self *Session) SetConn(conn net.Conn) {
	self.conn = conn
}
func (self *Session) GetBeatCount() int32 {
	return self.beatCount
}
func (self *Session) SetBeatCount(beatCount int32) {
	self.beatCount = beatCount
}
func (self *Session) Listen() {
	go self.Read()
	go self.Write()
}
func (self *Session) quit() {
	if self.ifQuit == false {
		/*_, flag := UserTable[self.GetUid()] //如果在线用户表有，则给令牌加有效时间
		if flag == true {
			if UserTable[self.GetUid()].Situation == 1 { //状态1改为状态2.
				var i = int(0)
				UserTable[self.GetUid()] = Token{UserTable[self.GetUid()].AToken, 2}
				go func(int) {
					for {
						time.Sleep(time.Second * 1)
						if i >= TokenTime {
							delete(UserTable, self.GetUid())
							break
						}
						if UserTable[self.GetUid()].Situation == 1 {
							break
						}
						i++
					}
				}(i)
			}
		}*/
		self.ifQuit = true
		_, fl := SessionTable[self.uid]
		if fl == true {
			self.quitfor <- 1 //转发信息的协程       ????
			self.quiting <- 0 //删session 关conn 用的

		}
	}
}

//random生成器
func GenRandom() uint32 {
	ran := r.New(r.NewSource(time.Now().UnixNano()))
	randnum := uint32(ran.Intn(4000000) + 100) //前100号留着备用
	return randnum
}

//读入tcp里面的包到Session.incoming
func (self *Session) Read() {
	for {
		if packet, err := self.packetReader.ReadPacket(); err == nil {
			//测试断点
			//fmt.Println("-------------readreadread--------------", packet.GetType())
			self.incoming <- *packet
		} else {
			ImServerLog("Read a packet error:" + err.Error())
			self.quit()
			break
		}
	}
}

//把Session.outgoing里面的内容写出去
func (self *Session) Write() {
	/*infor:
	for {
		select {
		case packet := <-self.outgoing:
			{
				apacket := packet
				//测试断点
				fmt.Println("----------------------writewrite-----------------------", apacket.GetType())
				if err := self.packetWriter.WritePacket(&packet, nil); err != nil {
					ImServerLog("Write a packet error:" + err.Error())
					self.quit()
					break infor
				}
				//对写出去的2/4 包记录入回馈表
				if apacket.GetType() == 2 || apacket.GetType() == 4 {
					var rand uint32
					packet2and4CMD, err := GetCommand(&apacket)
					if err != nil {
						ImServerLog("In Write().getcommand " + err.Error())
						break //反protobuf出错则等待客户端重发
					}
					rand = packet2and4CMD.Message.GetRandom()
					message := packet2and4CMD.Message.GetData()
					ImPacketOutTime(self.GetUid(), message, time.Now().UnixNano()) //------------------------------------------------for test delay time
					_, err = FeedBack.GetOne(rand)
					if err != nil { //未存过的则设置MaxRecendTimes次重发
						packetAndTime := PacketAndTime{apacket, self.uid, time.Now().Unix(), MaxRecendTimes}
						FeedBack.AddOne(rand, packetAndTime)
					}
				}
			}
		}
	}*/
	for {
		packet := <-self.outgoing
		apacket := packet
		//测试断点
		//fmt.Println("----------------------writewrite-----------------------", apacket.GetType())
		if err := self.packetWriter.WritePacket(&packet, nil); err != nil {
			ImServerLog("Write a packet error:" + err.Error())
			self.quit()
			continue
		}
		//对写出去的2/4 包记录入回馈表
		go func(Packet) {
			if apacket.GetType() == 2 || apacket.GetType() == 4 {
				//var rand uint32
				packet2and4CMD, err := GetCommand(&apacket)
				if err != nil {
					ImServerLog("In Write().getcommand " + err.Error())
					//break //反protobuf出错则等待客户端重发
					return
				}
				rand := packet2and4CMD.Message.GetRandom()
				message := packet2and4CMD.Message.GetData()
				ImPacketOutTime(self.GetUid(), message, time.Now().UnixNano()) //-------for test delay time
				_, err = FeedBack.GetOne(rand)
				if err != nil { //未存过的则设置MaxRecendTimes次重发
					packetAndTime := PacketAndTime{apacket, self.uid, time.Now().Unix(), MaxRecendTimes}
					FeedBack.AddOne(rand, packetAndTime)
				}
			}
		}(apacket)
	}

}

//收到心跳包则心跳+1
func Beat(uid uint64) {
	_, flag := SessionTable[uid]
	if flag == true {
		SessionTable[uid].beatCount = SessionTable[uid].beatCount + 1
	}
}

//********初始化 Session********
func setSession(conns net.Conn) *Session {
	packetReaders := NewPacketReader(conns)
	packetWriters := NewPacketWriter(conns)
	ClientSession := &Session{
		uid:          uint64(0),
		conn:         conns,
		beatCount:    int32(2),
		incoming:     make(chan Packet),
		outgoing:     make(chan Packet),
		quiting:      make(chan byte),
		packetReader: packetReaders,
		packetWriter: packetWriters,
		chatMap:      make(map[uint64][]uint64),
		quitfor:      make(chan byte, 5),
		ifQuit:       bool(false),
	}
	ClientSession.Listen()
	return ClientSession
}

func receiveMessagePacket(uid uint64, imDB imdatabase.ImDb) {
	userUid := uid
qfor:
	for {
		ClientSession, flag := SessionTable[userUid]
		if flag == false {
			ImServerLog("user session doesn't exist.", userUid) /////////
			return
		}
		select {
		case pp := <-ClientSession.incoming: //提取出来的是 protobuf对象   从对象里面提取出所要的信息改session层的 name,id...
			{
				ttype := pp.GetType() //对型号为 2,4,5 进行处理。
				pcmd, err := GetCommand(&pp)
				if err != nil {
					ImServerLog("GetCommand err when read type 1 packet" + err.Error())
					break
				}
				random2and4p := pcmd.Message.GetRandom()
				dialogId := pcmd.GetReceiver() // dialogUid
				message := pcmd.Message.GetData()
				//ImPacketInTime(userUid, message, time.Now().UnixNano()) //纪录读到的包
				Beat(userUid)
				if ttype == 1 {
					*returnCMD.Sender = 0
					*returnCMD.Receiver = userUid
					*returnCMD.Message.Random = random2and4p
					*returnCMD.Message.Data = string(ttype) + ":had login before!"
					logagainreturnPacket := MakePacket(returnCMD, 3)
					ClientSession.outgoing <- *logagainreturnPacket
					break
				}
				if ttype == 5 { // 心跳包，读取name就ok
				}
				if ttype == 3 { //客户端收到信息后返回服务端4号包表示确认收到
					FeedBack.DeleteOne(random2and4p)
				}
				if ttype == 2 || ttype == 4 { // 信息包或者文件包
					*returnCMD.Sender = 0
					*returnCMD.Receiver = userUid
					*returnCMD.Message.Random = random2and4p
					*returnCMD.Message.Data = string(ttype) + ":Server received successfully."
					receivesucessreturnPacket := MakePacket(returnCMD, 3)
					ClientSession.outgoing <- *receivesucessreturnPacket

					/*err := imDB.AddMessage(userUid, dialogId, pp.GetType(), message, time.Now().Format("2006-01-02 15:04:05"))
					if err != nil {
						fmt.Println("AddMessage to mysql err", err)
						*returnCMD.Sender = 0
						*returnCMD.Receiver = userUid
						*returnCMD.Message.Random = 1
						*returnCMD.Message.Data = "IM Server Error!"
						sqlerrreturnPacket := MakePacket(returnCMD, 3)
						ClientSession.outgoing <- *sqlerrreturnPacket
						ImServerLog("IM Server Error! Because read outlinemessage err")
						break
					}*/
					ImPacketInTime(userUid, message, time.Now().UnixNano())
					//记录消息和在线则转发，不在线则存离线.如果dialog是活跃的才发送;转发给Dialog 里面其他所有人.
					//ImServerLog("Receive a message as", ttype, "User : ", userUid, ". DialogId : ", dialogId)
					receivers, flag := ClientSession.chatMap[dialogId]
					if flag == true { //在chatMap里面能够找到对应的表，则发送给表内所有成员
						for _, receiverUid := range receivers {
							sess, fl := SessionTable[receiverUid] //在线
							if fl == true {
								//ImServerLog("Write feedback to user : ", userUid, ". Server received successfully") ///////////////////////////////////////
								//重新制作包发给另一端
								*messageCMD.Sender = dialogId
								*messageCMD.Receiver = receiverUid
								*messageCMD.Message.Data = message
								*messageCMD.Message.Random = GenRandom()
								sendPacket := MakePacket(messageCMD, ttype)
								sess.outgoing <- *sendPacket
								continue
								//写入数据库message表 message
							} //用户不在线，存离线outlinemessage
							*messageCMD.Sender = dialogId
							*messageCMD.Receiver = receiverUid
							*messageCMD.Message.Data = message
							*messageCMD.Message.Random = GenRandom()
							outLineMsg := MakePacket(messageCMD, ttype)

							err := imDB.AddOutlineMsg(receiverUid, uint32(ttype), outLineMsg.GetData(), time.Now().Format("2006-01-02 15:04:05"))
							if err != nil {
								*returnCMD.Sender = 0
								*returnCMD.Receiver = userUid
								*returnCMD.Message.Random = 1
								*returnCMD.Message.Data = "IM Server Error!"
								mysqlerrreturnPacket := MakePacket(returnCMD, 3)
								ClientSession.outgoing <- *mysqlerrreturnPacket
								//ImServerLog("IM Server Error! Because read outlinemessage err")
								continue
							}
							//*returnCMD.Sender = 0
							//*returnCMD.Receiver = userUid
							//*returnCMD.Message.Random = random2and4p
							//*returnCMD.Message.Data = string(ttype) + ":receiver isn't online!"
							//returnPacket := MakePacket(returnCMD, 3)
							//ClientSession.outgoing <- *returnPacket
							//ImServerLog("Write feedback to user : ", userUid, ". Receiver isn't online!") //////////////////////////////////////////////////
						}
						break
					}
					//没能在chatMap里面找到，则先扫描activedialog表，若没有则告诉客户端
					if ClientSession.chatMap[dialogId] == nil { //初始化chatmap对应的切片
						ClientSession.chatMap[dialogId] = make([]uint64, 0, MaxMemberInGroup)
					}
					actDialogTable, err := imDB.Db.Query("select * from act_dialog where dialog_id=? order by dialog_id", dialogId)
					if err != nil {
						*returnCMD.Sender = 0
						*returnCMD.Receiver = userUid
						*returnCMD.Message.Random = 1
						*returnCMD.Message.Data = "IM Server Error!"
						queryerrreturnPacket := MakePacket(returnCMD, 3)
						ClientSession.outgoing <- *queryerrreturnPacket
						break
					}
					defer actDialogTable.Close()
					if actDialogTable.Next() { //这里由于数据库定好了act_dialog 的唯一性，所以存在也只会存在一次，用if足够
						//查询dialog表//查询条件：dialog，时间戳为0，返回没有被踢群的用户
						activeUserTable, err := imDB.Db.Query("select uid from dialog where dialog_Id=? and timestamp=?", dialogId, "0000-00-00 00:00:00")
						if err != nil {
							*returnCMD.Sender = 0
							*returnCMD.Receiver = userUid
							*returnCMD.Message.Random = 1
							*returnCMD.Message.Data = "IM Server Error!"
							errorreturnPacket := MakePacket(returnCMD, 3)
							ClientSession.outgoing <- *errorreturnPacket
							break
						}
						defer activeUserTable.Close()
						var uid uint64
						var flaguidin bool //用户在群内
						for activeUserTable.Next() {
							err = activeUserTable.Scan(&uid) //没有被踢群的用户
							if err != nil {
								*returnCMD.Sender = 0
								*returnCMD.Receiver = userUid
								*returnCMD.Message.Random = 1
								*returnCMD.Message.Data = "IM Server Error!"
								errreturnPacket := MakePacket(returnCMD, 3)
								ClientSession.outgoing <- *errreturnPacket
								DatabaseLog("Database error in  scaning  dialog:" + err.Error())
								break
							}
							//****************************成员加入到chatMap
							if uid == userUid {
								flaguidin = true //自己在群内，则flaguid为true
								continue
							}
							ClientSession.chatMap[dialogId] = append(ClientSession.chatMap[dialogId], uid)
						}
						if flaguidin != true { //自己已被踢出群
							delete(ClientSession.chatMap, dialogId) //删除chatMap
							//这里发给用户一个命令包 type6  用户那边显示并且不能往该dialog 发送内容
							*returnCMD.Sender = 0
							*returnCMD.Receiver = userUid
							*returnCMD.Message.Random = random2and4p
							*returnCMD.Message.Data = string(ttype) + ":you are out of group!"
							outofgroupreturnPacket := MakePacket(returnCMD, 3)
							ClientSession.outgoing <- *outofgroupreturnPacket
							break
						}
						//自己在群内，向chatMap里的uid转发消息
						for _, receiverUid := range ClientSession.chatMap[dialogId] {
							sess, fl := SessionTable[receiverUid]
							if fl == true {
								//重新制作包发给另一端
								*messageCMD.Sender = dialogId
								*messageCMD.Receiver = receiverUid
								*messageCMD.Message.Data = message
								*messageCMD.Message.Random = GenRandom()
								sendaPacket := MakePacket(messageCMD, ttype)
								sess.outgoing <- *sendaPacket
								continue
							}
							//用户不在线，存离线outlinemessage
							*messageCMD.Sender = dialogId
							*messageCMD.Receiver = receiverUid
							*messageCMD.Message.Data = message
							*messageCMD.Message.Random = GenRandom()
							outLineMsg2 := MakePacket(messageCMD, ttype)
							err := imDB.AddOutlineMsg(receiverUid, uint32(ttype), outLineMsg2.GetData(), time.Now().Format("2006-01-02 15:04:05"))
							if err != nil {
								//fmt.Println("AddOutlinemessage to mysql err", err)
								*returnCMD.Sender = 0
								*returnCMD.Receiver = userUid
								*returnCMD.Message.Random = 1
								*returnCMD.Message.Data = "IM Server Error!"
								returnPacket := MakePacket(returnCMD, 3)
								ClientSession.outgoing <- *returnPacket
								ImServerLog("IM Server Error! Because read outlinemessage err", err)
								continue
							}
						}
						break
					}
					//act_dialog找不到
					*returnCMD.Sender = 0
					*returnCMD.Receiver = userUid
					*returnCMD.Message.Random = random2and4p
					*returnCMD.Message.Data = string(ttype) + ":dialog isn't active!"
					returnPacket3 := MakePacket(returnCMD, 3)
					ClientSession.outgoing <- *returnPacket3
				}
				_, fla := SessionTable[userUid]
				if fla == true {
					SessionTable[userUid] = ClientSession
				}
			}
		case <-ClientSession.quitfor:
			{
				//回馈包？？？
				break qfor
			}
		}
	}
}

func CreateSession(conns net.Conn, imDB imdatabase.ImDb) *Session {
	//sqlerr := make(chan (error))             //当数据库读写出错，则返回nil，ErrorMysql
	ClientSession := setSession(conns)       //初始化session
	publicPacket := MakePacket(PublicCMD, 0) //type=0的包 也就是发公钥的包
	ClientSession.outgoing <- *publicPacket  //首先把RSA的Public Key 发过去
	for {                                    //等待一定时间 回馈包，里面要求 username，appkey，aeskey。时间超过，或者里面内容不符合直接关掉conn
		select {
		case requestPacket := <-ClientSession.incoming:
			{ //对进来的包进行检验
				//isloged, userUid := receiveLoginPacket(ClientSession, requestPacket)
				if requestPacket.GetType() != 1 {
					*returnCMD.Sender = 0
					*returnCMD.Receiver = 1
					*returnCMD.Message.Data = "1:Please login first!"
					returnPacket := MakePacket(returnCMD, 3)
					ClientSession.outgoing <- *returnPacket
					//ImServerLog("A user didn't use a type 1 packet to login.")
					break
				}
				//if requestPacket.GetType() == 1 { //mean that this is a requestPacket
				requestCMD, err := GetCommand(&requestPacket) //Type=1的包，包含uid+APPKEY+AESKEY
				if err != nil {
					//回馈??
					ImServerLog("RequestCMD  get login command error ", err) ///////////////////////////////////////////
					break
				}
				//验证名字是否登陆了，验证APPKEY是否正确，如果都正确，纪录AESKEY，返回AESKEY加密的回馈包：uid
				appKey := requestCMD.Message.GetPassword()
				userUid := requestCMD.GetSender()
				token := requestCMD.Message.GetName()
				random := requestCMD.Message.GetRandom()
				aesKey := requestCMD.Message.GetData()
				//先看有没token
				_, fla := UserTable[userUid]
				if fla == false {
					//fmt.Println("*****************************用户不存在")
					//用户未先登录业务服务器
					ClientSession.packetReader.AESKEY = []byte(aesKey)
					ClientSession.packetWriter.AESKEY = []byte(aesKey)
					*returnCMD.Sender = 0
					*returnCMD.Receiver = userUid
					*returnCMD.Message.Random = random
					*returnCMD.Message.Data = "1:User don't login successfully!"
					notokenreturnPacket := MakePacket(returnCMD, 3)
					ClientSession.outgoing <- *notokenreturnPacket
					break
				}
				var userToken string
				userToken = UserTable[userUid].AToken
				if appKey != APPKEY || token != userToken {
					*returnCMD.Sender = 0
					*returnCMD.Receiver = userUid
					*returnCMD.Message.Random = random
					*returnCMD.Message.Data = "1:Wrong AppKey or Token."
					wrongkeyreturnPacket := MakePacket(returnCMD, 3)
					ClientSession.outgoing <- *wrongkeyreturnPacket
					break
				}
				//if appKey == APPKEY && token == userToken { //have the same appkey and online and have the same token
				//从这里开始可以对imDB操作
				ClientSession.packetReader.AESKEY = []byte(aesKey)
				ClientSession.packetWriter.AESKEY = []byte(aesKey)
				ClientSession.uid = userUid
				//如果存在旧的session tcp 则先关掉旧的，释放tcp资源
				oldSession, flags := SessionTable[userUid]
				if flags == true {
					*returnCMD.Sender = 0
					*returnCMD.Receiver = userUid
					*returnCMD.Message.Random = random
					*returnCMD.Message.Data = "1:Old session is replacing"
					placingreturnPacket := MakePacket(returnCMD, 3)
					oldSession.outgoing <- *placingreturnPacket
					SessionTable[userUid].quit()
					ImServerLog("User: ", userUid, " has logined before. Quit the earlier session.") //////////////////////////////////////////////
				}
				SessionTable[userUid] = ClientSession
				UserTable[userUid] = Token{UserTable[userUid].AToken, 1}
				*returnCMD.Sender = 0
				*returnCMD.Receiver = userUid
				*returnCMD.Message.Random = random
				*returnCMD.Message.Data = "1:Connect successfully."
				successreturnPacket := MakePacket(returnCMD, 3)
				ClientSession.outgoing <- *successreturnPacket

				//*********************************************************************************
				//goThroughOutLineMessage(userUid, imDB)
				outLineMessageTableFromSql, err := imDB.Db.Query("select type,packet,timestamp from outlinemessage where receiver_id=? order by timestamp asc", userUid)
				if err != nil {
					DatabaseLog("database error in prepare reading outlinemesssage:" + err.Error())
				}
				defer outLineMessageTableFromSql.Close()
				if err == nil {
					//DatabaseLog("select from outlinemessage receiver_id = ", userUid)
					//在离线表找到数据
					var outLineMsgType uint32
					var outLineMsgePacket []byte
					var outLineMsgTime string
					//切片存储所有扫描出来的离线包信息
					var allOutLineMsgType []uint32
					var allOutLineMsgePacket []string //string类型
					//var allOutLineMsgTime []string
					allOutLineMsgType = make([]uint32, 0, MaxOutLineMsg)
					allOutLineMsgePacket = make([]string, 0, MaxOutLineMsg)
					//allOutLineMsgTime = make([]string, 0, MaxOutLineMsg)

					for outLineMessageTableFromSql.Next() {
						err := outLineMessageTableFromSql.Scan(&outLineMsgType, &outLineMsgePacket, &outLineMsgTime)
						if err != nil {
							DatabaseLog("database error in reading outlinemesssage:" + err.Error())
							continue
						}
						allOutLineMsgType = append(allOutLineMsgType, outLineMsgType)
						allOutLineMsgePacket = append(allOutLineMsgePacket, string(outLineMsgePacket))
					}
					outLineMessageTableFromSql.Close()
					for i, _ := range allOutLineMsgType {
						var outLinePacket Packet
						outLinePacket.SetType(allOutLineMsgType[i])
						outLinePacket.SetData([]byte(allOutLineMsgePacket[i]))
						outLineCMD, err := GetCommand(&outLinePacket)
						if err != nil {
							ImServerLog("Outlinemessage GetCommand err : ", err)
							continue
						}
						*outLineCMD.Message.Random = GenRandom()
						outPacket := MakePacket(outLineCMD, outLinePacket.GetType())
						_, fl := SessionTable[userUid]
						if fl {
							SessionTable[userUid].outgoing <- *outPacket
						}
					}
					err = imDB.DeleteDbTable("delete from outlinemessage where receiver_id=?", userUid)
					if err != nil {
						DatabaseLog("Delete from outlinemessage error", err)
					}
				}
				//ImBusinessLog("Delete outlinemessage receiver_id = ", userUid)
				//return true

				//if isGetOutLineMessage == false {
				//	SessionTable[userUid].quit()
				//	return nil, ErrorMysql
				//} else {
				SessionTable[userUid] = ClientSession
				go receiveMessagePacket(userUid, imDB)
				return ClientSession
				//}

			}
			//case <-time.After(time.Second * MaxDelayTime):
			//{
			//	_, flag := SessionTable[userUid]
			//	if flag {
			//		SessionTable[userUid].quit()
			//		ImServerLog("Client didn't login after receiving rsa key in MaxDelayTime.") ///////////////////////////////////////////////////
			//	}
			//		return nil, nil
			//}
			//case caseerr := <-sqlerr:
			//	{
			//		if caseerr == ErrorMysql {
			//			return nil, ErrorMysql
			//		}
			//	}
		}
	}
	return nil
}

//登陆包处理，成功返回true和uid，失败返回false和0
/*func receiveLoginPacket(ClientSession *Session, requestPacket Packet) (bool, uint64) {
	if requestPacket.GetType() == 1 { //mean that this is a requestPacket
		//fmt.Println("*****************************得到1号包")

		requestCMD, err := GetCommand(&requestPacket) //Type=1的包，包含uid+APPKEY+AESKEY
		if err != nil {
			ImServerLog("RequestCMD  get login command error ", err) ///////////////////////////////////////////
			return false, 0
		}
		//fmt.Println("*****************************得到1号包为：", requestCMD)

		//验证名字是否登陆了，验证APPKEY是否正确，如果都正确，纪录AESKEY，返回AESKEY加密的回馈包：uid
		appKey := requestCMD.Message.GetPassword()
		userUid := requestCMD.GetSender()
		token := requestCMD.Message.GetName()
		random := requestCMD.Message.GetRandom()
		aesKey := requestCMD.Message.GetData()
		//先看有没token
		_, fla := UserTable[userUid]
		if fla == false {
			//fmt.Println("*****************************用户不存在")
			//用户未先登录业务服务器
			ClientSession.packetReader.AESKEY = []byte(aesKey)
			ClientSession.packetWriter.AESKEY = []byte(aesKey)
			*returnCMD.Sender = 0
			*returnCMD.Receiver = userUid
			*returnCMD.Message.Random = random
			*returnCMD.Message.Data = "1:User don't login successfully!"
			returnPacket := MakePacket(returnCMD, 3)
			ClientSession.outgoing <- *returnPacket
			return false, 0
		}
		var userToken string
		userToken = UserTable[userUid].AToken
		// isRegister mean that user has registered.isOnLine
		if appKey == APPKEY && token == userToken { //have the same appkey and online and have the same token
			//从这里开始可以对imDB操作

			if len(aesKey) != 16 {
				//fmt.Println("*****************************aeskey长度不对")

				*returnCMD.Sender = 0
				*returnCMD.Receiver = userUid
				*returnCMD.Message.Random = random
				*returnCMD.Message.Data = "1:Wrong aesKey!"
				returnPacket := MakePacket(returnCMD, 3)
				ClientSession.outgoing <- *returnPacket
				//ImServerLog("AES key from  ", userUid, " is error.")
				return false, 0
			}
			ClientSession.packetReader.AESKEY = []byte(aesKey)
			ClientSession.packetWriter.AESKEY = []byte(aesKey)
			ClientSession.uid = userUid
			//如果存在旧的session tcp 则先关掉旧的，释放tcp资源
			_, flags := SessionTable[userUid]
			if flags == true {
				//fmt.Println("*****************************已经存在session，关掉旧的ing")

				*returnCMD.Sender = 0
				*returnCMD.Receiver = userUid
				*returnCMD.Message.Random = random
				*returnCMD.Message.Data = "1:Old session is replacing"
				returnPacket := MakePacket(returnCMD, 3)
				SessionTable[userUid].outgoing <- *returnPacket

				SessionTable[userUid].quit()
				ImServerLog("User: ", userUid, " has logined before. Quit the earlier session.") //////////////////////////////////////////////
			}
			//fmt.Println("*****************************成功登陆")

			SessionTable[userUid] = ClientSession
			SessionTable[userUid].packetReader.AESKEY = []byte(aesKey)
			SessionTable[userUid].packetWriter.AESKEY = []byte(aesKey)
			UserTable[userUid] = Token{UserTable[userUid].AToken, 1}
			//return  type=3 packet
			*returnCMD.Sender = 0
			*returnCMD.Receiver = userUid
			*returnCMD.Message.Random = random
			*returnCMD.Message.Data = "1:Connect successfully."
			returnPacket := MakePacket(returnCMD, 3)
			ClientSession.outgoing <- *returnPacket
			//	fmt.Println("8888888888888888888888888888,---------", userUid)
			return true, userUid

		} else {
			//fmt.Println("*****************************token，appkey不对")

			*returnCMD.Sender = 0
			*returnCMD.Receiver = userUid
			*returnCMD.Message.Random = random
			*returnCMD.Message.Data = "1:Wrong AppKey or Token."
			returnPacket := MakePacket(returnCMD, 3)
			ClientSession.outgoing <- *returnPacket
			//ClientSession.quit()
			//ImSessionLog("Illegal user :", userUid)
			return false, 0
		}
	} else {
		*returnCMD.Sender = 0
		*returnCMD.Receiver = 1
		*returnCMD.Message.Data = "1:Please login first!"
		returnPacket := MakePacket(returnCMD, 3)
		ClientSession.outgoing <- *returnPacket
		//ImServerLog("A user didn't use a type 1 packet to login.")
		return false, 0
	}
}

//扫描离线包
func goThroughOutLineMessage(userUid uint64, imDB imdatabase.ImDb) {
	_, flag := SessionTable[userUid]
	if flag == false {
		return
	}
	outLineMessageTableFromSql, err := imDB.Db.Query("select type,packet,timestamp from outlinemessage where receiver_id=? order by timestamp asc", userUid)
	if err != nil {
		DatabaseLog("database error in prepare reading outlinemesssage:" + err.Error())
		//fmt.Println("database error in prepare reading outlinemesssage:" + err.Error())
		return
	}
	defer outLineMessageTableFromSql.Close()
	//DatabaseLog("select from outlinemessage receiver_id = ", userUid)
	//在离线表找到数据
	var outLineMsgType uint32
	var outLineMsgePacket []byte
	var outLineMsgTime string
	//切片存储所有扫描出来的离线包信息
	var allOutLineMsgType []uint32
	var allOutLineMsgePacket []string //string类型
	//var allOutLineMsgTime []string
	allOutLineMsgType = make([]uint32, 0, MaxOutLineMsg)
	allOutLineMsgePacket = make([]string, 0, MaxOutLineMsg)
	//allOutLineMsgTime = make([]string, 0, MaxOutLineMsg)

	for outLineMessageTableFromSql.Next() {
		err := outLineMessageTableFromSql.Scan(&outLineMsgType, &outLineMsgePacket, &outLineMsgTime)
		if err != nil {
			DatabaseLog("database error in reading outlinemesssage:" + err.Error())
			//fmt.Println("database error in reading outlinemesssage:" + err.Error())
			return
		}
		allOutLineMsgType = append(allOutLineMsgType, outLineMsgType)
		allOutLineMsgePacket = append(allOutLineMsgePacket, string(outLineMsgePacket))
		//allOutLineMsgTime = append(allOutLineMsgTime, outLineMsgTime)
	}
	//////////显示关闭数据表
	outLineMessageTableFromSql.Close() //////
	for i, _ := range allOutLineMsgType {
		var outLinePacket Packet
		outLinePacket.SetType(allOutLineMsgType[i])
		outLinePacket.SetData([]byte(allOutLineMsgePacket[i]))
		outLineCMD, err := GetCommand(&outLinePacket)
		if err != nil {
			ImServerLog("Outlinemessage GetCommand err : ", err)
			//fmt.Println("Outlinemessage GetCommand err :", err)
			continue
		}
		*outLineCMD.Message.Random = GenRandom()
		outPacket := MakePacket(outLineCMD, outLinePacket.GetType())
		_, fl := SessionTable[userUid]
		if fl {
			SessionTable[userUid].outgoing <- *outPacket
		}
	}
	err = imDB.DeleteDbTable("delete from outlinemessage where receiver_id=?", userUid)
	if err != nil {
		//fmt.Println("删除离线包出错：", err)
		DatabaseLog("Delete from outlinemessage error", err)
		return
	}
	//ImBusinessLog("Delete outlinemessage receiver_id = ", userUid)
	//return true
}

//消息包处理
func receiveMessagePacket(userUid uint64, imDB imdatabase.ImDb, sqlerr chan (error)) {
	//var forwordRandom = uint32(0)

qfor:
	for {
		_, flag := SessionTable[userUid]
		if flag == false {
			return
		}
		select {
		case pp := <-SessionTable[userUid].incoming: //提取出来的是 protobuf对象   从对象里面提取出所要的信息改session层的 name,id...
			{

				ttype := pp.GetType() //对型号为 2,4,5 进行处理。
				if ttype == 1 {
					pcmd, err := GetCommand(&pp)
					if err != nil {
						ImServerLog("GetCommand err when read type 1 packet" + err.Error())
						break
					}
					random2and4p := pcmd.Message.GetRandom()
					*returnCMD.Sender = 0
					*returnCMD.Receiver = userUid
					*returnCMD.Message.Random = random2and4p
					*returnCMD.Message.Data = string(ttype) + ":had login before!"
					returnPacket := MakePacket(returnCMD, 3)
					//***********************
					_, flag := SessionTable[userUid]
					if flag == false {
						return
					}
					SessionTable[userUid].outgoing <- *returnPacket
					//ImServerLog("Receive a packet of type 1 ,from user:", userUid)
				}
				if ttype == 5 { // 心跳包，读取name就ok
					Beat(userUid)
				}
				if ttype == 3 { //客户端收到信息后返回服务端4号包表示确认收到
					Beat(userUid)
					//returnPacketHandle(pp)
					//ImServerLog("Receive a packet of type 3 ,from user:", userUid)
					cmdd, err := GetCommand(&pp)
					if err != nil {
						ImServerLog("returnPacket getCommand err", err)
						break //回馈包反proto失败则等待服务器重发，重新收回馈包
					}
					rand := cmdd.Message.GetRandom()
					FeedBack.DeleteOne(rand)
					//_, flag := FeedBackInServerTable[rand]
					//if flag == true {
					//	delete(FeedBackInServerTable, rand)
					//}
				}
				if ttype == 2 || ttype == 4 { // 信息包或者文件包
					Beat(userUid)
					cmd2and4, err := GetCommand(&pp)
					if err != nil {
						ImServerLog("GetCommand type=2,4 err", err, " User :", userUid)
						break
					}
					random2and4 := cmd2and4.Message.GetRandom()
					dialogId := cmd2and4.GetReceiver() // dialogUid
					message := cmd2and4.Message.GetData()
					//-----------------------------for test delay time
					ImPacketInTime(userUid, message, time.Now().UnixNano())

					//if forwordRandom != random2and4 {
					//forwordRandom = random2and4
					//记录消息和在线则转发，不在线则存离线.如果dialog是活跃的才发送;转发给Dialog 里面其他所有人.
					//ImServerLog("Receive a message as", ttype, "User : ", userUid, ". DialogId : ", dialogId)
					_, flag := SessionTable[userUid]
					if flag == false {
						return
					}
					receivers, flag := SessionTable[userUid].chatMap[dialogId]
					if flag == true { //在chatMap里面能够找到对应的表，则发送给表内所有成员
						*returnCMD.Sender = 0
						*returnCMD.Receiver = userUid
						*returnCMD.Message.Random = random2and4
						*returnCMD.Message.Data = string(ttype) + ":Server received successfully."
						returnPacket := MakePacket(returnCMD, 3)
						SessionTable[userUid].outgoing <- *returnPacket
						for _, receiverUid := range receivers {
							sess, fl := SessionTable[receiverUid] //在线
							if fl {
								//ImServerLog("Write feedback to user : ", userUid, ". Server received successfully") ///////////////////////////////////////
								//重新制作包发给另一端
								*messageCMD.Sender = dialogId
								*messageCMD.Receiver = receiverUid
								*messageCMD.Message.Data = message
								*messageCMD.Message.Random = GenRandom()
								sendPacket := MakePacket(messageCMD, ttype)
								sess.outgoing <- *sendPacket
								//写入数据库message表 message
							} else { //用户不在线，存离线outlinemessage
								*messageCMD.Sender = dialogId
								*messageCMD.Receiver = receiverUid
								*messageCMD.Message.Data = message
								*messageCMD.Message.Random = GenRandom()
								outLineMsg := MakePacket(messageCMD, ttype)

								err := imDB.AddOutlineMsg(receiverUid, uint32(ttype), outLineMsg.GetData(), time.Now().Format("2006-01-02 15:04:05"))
								if err != nil {
									//fmt.Println("AddOutlinemessage to mysql err", err)
									*returnCMD.Sender = 0
									*returnCMD.Receiver = userUid
									*returnCMD.Message.Random = 1
									*returnCMD.Message.Data = "IM Server Error!"
									returnPacket := MakePacket(returnCMD, 3)
									SessionTable[userUid].outgoing <- *returnPacket
									//ImServerLog("IM Server Error! Because read outlinemessage err")
									sqlerr <- ErrorMysql
									break qfor
								} else {
									//*returnCMD.Sender = 0
									//*returnCMD.Receiver = userUid
									//*returnCMD.Message.Random = random2and4
									//*returnCMD.Message.Data = string(ttype) + ":receiver isn't online!"
									//returnPacket := MakePacket(returnCMD, 3)
									//ClientSession.outgoing <- *returnPacket
									//ImServerLog("Write feedback to user : ", userUid, ". Receiver isn't online!") //////////////////////////////////////////////////
								}
							}
						}
						//写入数据库message表 message
						//err := imDB.AddMessage(userUid, dialogId, pp.GetType(), message, time.Now().Format("2006-01-02 15:04:05"))
						//if err != nil {
							//fmt.Println("AddMessage to mysql err", err)
						//	*returnCMD.Sender = 0
						//	*returnCMD.Receiver = userUid
						//	*returnCMD.Message.Random = 1
						//	*returnCMD.Message.Data = "IM Server Error!"
						//	returnPacket := MakePacket(returnCMD, 3)
						//	ClientSession.outgoing <- *returnPacket
							//ImServerLog("IM Server Error! Because read outlinemessage err")
						//	sqlerr <- ErrorMysql
						//	break qfor
						//}
					} else { //没能在chatMap里面找到，则先扫描activedialog表，若没有则告诉客户端
						_, fll := SessionTable[userUid]
						if fll == true {
							if SessionTable[userUid].chatMap[dialogId] == nil { //初始化chatmap对应的切片
								SessionTable[userUid].chatMap[dialogId] = make([]uint64, 0, MaxMemberInGroup)
							}
							actDialogTable, err := imDB.Db.Query("select * from act_dialog where dialog_id=? order by dialog_id", dialogId)
							if err != nil {
								//fmt.Println("database error in reading act_dialog:" + err.Error())
								*returnCMD.Sender = 0
								*returnCMD.Receiver = userUid
								*returnCMD.Message.Random = 1
								*returnCMD.Message.Data = "IM Server Error!"
								returnPacket := MakePacket(returnCMD, 3)
								SessionTable[userUid].outgoing <- *returnPacket
								//ImServerLog("IM Server Error! Because read outlinemessage err")
								sqlerr <- ErrorMysql
								break qfor
							}
							defer actDialogTable.Close()
							if actDialogTable.Next() { //这里由于数据库定好了act_dialog 的唯一性，所以存在也只会存在一次，用if足够

								//查询dialog表//查询条件：dialog，时间戳为0，返回没有被踢群的用户
								activeUserTable, err := imDB.Db.Query("select uid from dialog where dialog_Id=? and timestamp=?", dialogId, "0000-00-00 00:00:00")

								if err != nil {
									//fmt.Println("Database error in selecting dialog:" + err.Error())
									*returnCMD.Sender = 0
									*returnCMD.Receiver = userUid
									*returnCMD.Message.Random = 1
									*returnCMD.Message.Data = "IM Server Error!"
									returnPacket := MakePacket(returnCMD, 3)
									SessionTable[userUid].outgoing <- *returnPacket
									//ImServerLog("IM Server Error! Because read outlinemessage err")
									sqlerr <- ErrorMysql
									break qfor
								}
								defer activeUserTable.Close()
								var uid uint64
								var flaguidin bool //用户在群内
								for activeUserTable.Next() {
									err = activeUserTable.Scan(&uid) //没有被踢群的用户
									if err != nil {
										//fmt.Println("Database error in selecting dialog:" + err.Error())
										*returnCMD.Sender = 0
										*returnCMD.Receiver = userUid
										*returnCMD.Message.Random = 1
										*returnCMD.Message.Data = "IM Server Error!"
										returnPacket := MakePacket(returnCMD, 3)
										SessionTable[userUid].outgoing <- *returnPacket
										DatabaseLog("Database error in  scaning  dialog:" + err.Error())
										sqlerr <- ErrorMysql
										break qfor
									}
									//****************************成员加入到chatMap
									if uid != userUid {
										_, ffl := SessionTable[userUid]
										if ffl == false {
											break qfor
										}
										_, ffll := SessionTable[userUid].chatMap[dialogId]
										if ffll == false {
											SessionTable[userUid].chatMap[dialogId] = make([]uint64, 0, MaxMemberInGroup)
										}
										SessionTable[userUid].chatMap[dialogId] = append(SessionTable[userUid].chatMap[dialogId], uid)
									} else {
										flaguidin = true //自己在群内，则flaguid为true
									}
								}
								if flaguidin != true { //自己已被踢出群
									delete(SessionTable[userUid].chatMap, dialogId) //删除chatMap
									//这里发给用户一个命令包 type6  用户那边显示并且不能往该dialog 发送内容
									*returnCMD.Sender = 0
									*returnCMD.Receiver = userUid
									*returnCMD.Message.Random = random2and4
									*returnCMD.Message.Data = string(ttype) + ":you are out of group!"
									returnPacket := MakePacket(returnCMD, 3)
									SessionTable[userUid].outgoing <- *returnPacket
									//ImServerLog("User ", userUid, "  is not in dialog: ", dialogId)
								} else { //自己在群内，向chatMap里的uid转发消息
									*returnCMD.Sender = 0
									*returnCMD.Receiver = userUid
									*returnCMD.Message.Random = random2and4
									*returnCMD.Message.Data = string(ttype) + ":Server received successfully."
									returnPacket := MakePacket(returnCMD, 3)
									SessionTable[userUid].outgoing <- *returnPacket
									//ImServerLog("Write feedback to user : ", userUid, ". Server received successfully") ////////////////////////////////////

									for _, receiverUid := range SessionTable[userUid].chatMap[dialogId] {
										sess, fl := SessionTable[receiverUid]
										if fl {

											//重新制作包发给另一端
											*messageCMD.Sender = dialogId
											*messageCMD.Receiver = receiverUid
											*messageCMD.Message.Data = message
											*messageCMD.Message.Random = GenRandom()
											sendPacket := MakePacket(messageCMD, ttype)
											sess.outgoing <- *sendPacket
										} else {
											//用户不在线，存离线outlinemessage
											*messageCMD.Sender = dialogId
											*messageCMD.Receiver = receiverUid
											*messageCMD.Message.Data = message
											*messageCMD.Message.Random = GenRandom()
											outLineMsg2 := MakePacket(messageCMD, ttype)

											err := imDB.AddOutlineMsg(receiverUid, uint32(ttype), outLineMsg2.GetData(), time.Now().Format("2006-01-02 15:04:05"))
											if err != nil {
												//fmt.Println("AddOutlinemessage to mysql err", err)
												*returnCMD.Sender = 0
												*returnCMD.Receiver = userUid
												*returnCMD.Message.Random = 1
												*returnCMD.Message.Data = "IM Server Error!"
												returnPacket := MakePacket(returnCMD, 3)
												SessionTable[userUid].outgoing <- *returnPacket
												ImServerLog("IM Server Error! Because read outlinemessage err", err)
												sqlerr <- ErrorMysql
												break qfor
											} else {
											//	*returnCMD.Sender = 0
												//*returnCMD.Receiver = userUid
												//*returnCMD.Message.Random = random2and4
												//*returnCMD.Message.Data = string(ttype) + ":receiver isn't online!"
												//returnPacket := MakePacket(returnCMD, 3)
												//ClientSession.outgoing <- *returnPacket
												//ImServerLog("Write feedback to user : ", userUid, ". Receiver isn't online!")
											}
										}
									}
									//err := imDB.AddMessage(userUid, dialogId, pp.GetType(), message, time.Now().Format("2006-01-02 15:04:05"))
									//写入数据库message表 message
									//if err != nil {
										//fmt.Println("AddMessage to mysql err", err)
										//*returnCMD.Sender = 0
									//	*returnCMD.Receiver = userUid
										//*returnCMD.Message.Random = 1
									//	*returnCMD.Message.Data = "IM Server Error!"
									//	returnPacket := MakePacket(returnCMD, 3)
									//	ClientSession.outgoing <- *returnPacket
										//ImServerLog("IM Server Error! Because read outlinemessage err")
										//sqlerr <- ErrorMysql
									//	break qfor
									//}
								}
							} else { //act_dialog找不到
								*returnCMD.Sender = 0
								*returnCMD.Receiver = userUid
								*returnCMD.Message.Random = random2and4
								*returnCMD.Message.Data = string(ttype) + ":dialog isn't active!"
								returnPacket3 := MakePacket(returnCMD, 3)
								SessionTable[userUid].outgoing <- *returnPacket3
								//ImServerLog("dialog : ", dialogId, " is not active.")
							}
						}
					}
					//}
				}
			}
		case <-SessionTable[userUid].quitfor:
			{
				break qfor
			}
		}
	}
}

//********创建一个Session********
func CreateSession(conns net.Conn, imDB imdatabase.ImDb) (*Session, error) {
	sqlerr := make(chan (error)) //当数据库读写出错，则返回nil，ErrorMysql
	//ImServerLog("Create a new session by conns as: ", conns)
	ClientSession := setSession(conns)       //初始化session
	publicPacket := MakePacket(PublicCMD, 0) //type=0的包 也就是发公钥的包
	ClientSession.outgoing <- *publicPacket  //首先把RSA的Public Key 发过去

	for { //等待一定时间 回馈包，里面要求 username，appkey，aeskey。时间超过，或者里面内容不符合直接关掉conn
		select {
		case requestPacket := <-ClientSession.incoming:
			{ //对进来的包进行检验
				isloged, userUid := receiveLoginPacket(ClientSession, requestPacket)
				if isloged == false {
					break //进入下次读1号包
				} else {
					//isGetOutLineMessage := goThroughOutLineMessage(userUid, imDB)
					goThroughOutLineMessage(userUid, imDB)
					//if isGetOutLineMessage == false {
					//	SessionTable[userUid].quit()
					//	return nil, ErrorMysql
					//} else {
					go receiveMessagePacket(userUid, imDB, sqlerr)
					return SessionTable[userUid], nil
					//}
				}
			}
		//case <-time.After(time.Second * MaxDelayTime):
		//{
		//	_, flag := SessionTable[userUid]
		//	if flag {
		//		SessionTable[userUid].quit()
		//		ImServerLog("Client didn't login after receiving rsa key in MaxDelayTime.") ///////////////////////////////////////////////////
		//	}
	//		return nil, nil
		//}
		case caseerr := <-sqlerr:
			{
				if caseerr == ErrorMysql {
					return nil, ErrorMysql
				}
			}
		}
	}
	return nil, nil
}*/

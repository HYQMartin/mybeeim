package s2c

//!!!!!!!注意： 停牌的有效时间被我屏蔽了
import (
	"beeim/imdatabase"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"goconfig"
	_ "mysql"
	"net"
	"runtime"
	"strconv"
	"sync"
	"time"
)

var MaxSession int
var FeedBack FeedBackInServer

var TotalConn TotalStruct

type TotalStruct struct {
	Total int
	mutex *sync.Mutex
}

func (t *TotalStruct) GetTotal() int {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.Total
}
func (t *TotalStruct) Add() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.Total = t.Total + 1
}
func (t *TotalStruct) Abstract() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	t.Total = t.Total - 1
}

type FeedBackInServer struct {
	FeedBackInServerTable map[uint32]PacketAndTime
	mutex                 *sync.RWMutex
}

func (f *FeedBackInServer) DeleteOne(random uint32) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	delete(f.FeedBackInServerTable, random)
}

func (f *FeedBackInServer) GetAll() (random []uint32, packetAndTime []PacketAndTime) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	random = make([]uint32, 0, 2048)
	packetAndTime = make([]PacketAndTime, 0, 2048)
	for ran, packet := range f.FeedBackInServerTable {
		random = append(random, ran)
		packetAndTime = append(packetAndTime, packet)
	}
	return random, packetAndTime
}
func (f *FeedBackInServer) GetOne(random uint32) (PacketAndTime, error) {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	packetnil := Packet{0, nil}
	_, flag := f.FeedBackInServerTable[random]
	if flag == false {
		return PacketAndTime{packetnil, 0, 0, 0}, ErrorGetOneError
	}
	return f.FeedBackInServerTable[random], nil
}
func (f *FeedBackInServer) AddOne(random uint32, packetAndTime PacketAndTime) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.FeedBackInServerTable[random] = packetAndTime
}

//********Tables in RAM
//var FeedBackInServerTable map[uint32]PacketAndTime //等待回馈包的表
var UserTable map[uint64]Token // 存在线用户，存token，以及token有效状态
type PacketAndTime struct {
	Packet      Packet
	Receiver    uint64
	StartTime   int64 //time.unix()
	ReSendTimes int   //用户重发次数
}
type Token struct {
	AToken    string
	Situation int // 1:= 有效，2：=180s有效
}

var SessionTable map[uint64]*Session //存在线用户的session

func setTable() {
	//初始化
	//FeedBackInServerTable = make(map[uint32]PacketAndTime) //定期重发,重发一定次数    回馈sender 并删
	FeedBack = FeedBackInServer{
		FeedBackInServerTable: make(map[uint32]PacketAndTime),
		mutex: &sync.RWMutex{},
	}
	TotalConn = TotalStruct{
		Total: int(0),
		mutex: &sync.Mutex{},
	}
	UserTable = make(map[uint64]Token)
	SessionTable = make(map[uint64]*Session)
	//测试用例
	//这里应：当业务服务器调用登录接口，使用MakeToken产生令牌并记录usertable中

	UserTable[111] = Token{"tokenismartin", 1}
	UserTable[222] = Token{"tokenismartin", 1}
	UserTable[333] = Token{"tokenismartin", 1}
	UserTable[444] = Token{"tokenismartin", 1}
	UserTable[555] = Token{"tokenismartin", 1}

	for i := uint64(1000); i < 10000; i++ {
		UserTable[i] = Token{"tokenismartin", 1}
	}

	for i := uint64(10); i < 41; i++ {
		UserTable[i] = Token{"tokenismartin", 1}
	}
	for i := uint64(10000); i < 100000; i++ {
		UserTable[i] = Token{"tokenismartin", 1}
	}
}
func setConfig() {
	var err error
	cfg, err := goconfig.LoadConfigFile("./cfg/config.ini")
	if err != nil {
		ImServerLog("open config.ini err", err)
	}
	//MaxSession
	MaxSession, err = cfg.Int("IMServer", "MAX_SESSIONS")
	if err != nil {
		ImServerLog("Get Maxsessions from config.ini err", err)
	}
	//MaxMemberInGroup
	MaxMemberInGroup, err = cfg.Int("IMServer", "MaxMemberInGroup")
	if err != nil {
		ImServerLog("Get MaxMemberInGroup from config.ini err", err)
	}
	//TokenTime
	TokenTime, err = cfg.Int("IMServer", "TokenTime")
	if err != nil {
		ImServerLog("Get TokenTime from config.ini err", err)
	}
	//MaxRecendTimes
	MaxRecendTimes, err = cfg.Int("IMServer", "MaxRecendTimes")
	if err != nil {
		ImServerLog("Get MaxRecendTimes from config.ini err", err)
	}
	//PacketBufSize
	PacketBufSize, err = cfg.Int("IMServer", "PacketBufSize")
	if err != nil {
		ImServerLog("Get PacketBufSize from config.ini err", err)
	}
}

//产生随机的字符串，做令牌用
func MakeToken() string {
	currentTIme := time.Now().Unix()
	md := md5.New()
	md.Write([]byte(strconv.Itoa(int(currentTIme))))
	return hex.EncodeToString(md.Sum(nil))
}

//********Server 属性
type Server struct {
	imDB             imdatabase.ImDb //数据库
	listener         net.Listener    //监听器
	businesslistener net.Listener    //业务层监听器
	business         chan net.Conn
	pending          chan net.Conn //等待通道
	quiting          chan net.Conn //退出
	maxClient        chan byte     //控制一个服务器下只能有20000个客户端
}

//********Server 方法
//********关闭server
func (self *Server) Stop() { //停止1个服务器
	self.listener.Close()
	ImServerLog("IM Server is stopped.")
}
func (self *Server) listen() { //监听是否有客户端链接或者离开
	go func() {
		for {
			select {
			case conn := <-self.pending:
				go self.join(conn)
			case conn := <-self.quiting:
				self.leave(conn)
			case conn := <-self.business:
				go acceptInterface(conn)
			}
		}
	}()
}
func (self *Server) join(conn net.Conn) { //监听到有客户端连接则创建一个session并且监听session传来的数据包
	self.clientIn()
	session := CreateSession(conn, self.imDB) // 一旦有新的conn  就创建一个会话,这里暂时只考虑数据庫出错。
	//if err == ErrorMysql {
	//	fmt.Println("由于数据库有问题，关闭IM服务器", err)
	//	self.Stop()
	//	return
	//}
	if session == nil {
		self.quiting <- conn
		return
	}
	TotalConn.Add()
	//go func() {
	<-session.quiting

	self.quiting <- session.GetConn()
	if session.conn != nil {
		session.conn.Close()
	}
	delete(SessionTable, session.GetUid())
	TotalConn.Abstract()
	//	}()
}
func (self *Server) leave(conn net.Conn) {
	if conn != nil {
		conn.Close()
	}
	self.clientOut()
}
func (self *Server) clientIn() {
	<-self.maxClient
}
func (self *Server) clientOut() {
	self.maxClient <- 0
}
func (self *Server) ReSend() {
	for {
		time.Sleep(time.Second * 20)
		rand, packetandtime := FeedBack.GetAll()
		for index, ran := range rand {
			packetAndTime := packetandtime[index]
			if (time.Now().Unix() - packetAndTime.StartTime) >= 20 {
				_, flag := SessionTable[packetAndTime.Receiver]
				if flag == true { //在线
					if packetAndTime.ReSendTimes >= 1 {
						packetAndTime.ReSendTimes = packetAndTime.ReSendTimes - 1
						SessionTable[packetAndTime.Receiver].outgoing <- packetAndTime.Packet
						FeedBack.AddOne(ran, packetAndTime)
					} else {
						self.imDB.AddOutlineMsg(packetAndTime.Receiver, uint32(packetAndTime.Packet.GetType()), packetAndTime.Packet.GetData(), time.Now().Format("2006-01-02 15:04:05"))
						FeedBack.DeleteOne(ran)
					}
				} else { //不在线
					self.imDB.AddOutlineMsg(packetAndTime.Receiver, uint32(packetAndTime.Packet.GetType()), packetAndTime.Packet.GetData(), time.Now().Format("2006-01-02 15:04:05"))
					FeedBack.DeleteOne(ran)
				}
			}
		}
	}
}
func (self *Server) DelOutLineMessage() { //如果超过一个月用户没登陆，则删除相关离线信息数据  2592000
	for {
		time.Sleep(time.Second * 2592000)
		deletime := time.Now().AddDate(0, -1, 0)
		err := self.imDB.DeleteDbTable("delete from outlinemessage where timestamp<?", deletime)
		if err != nil {
			fmt.Println("定期删除离线包数据出错：", err)
			DatabaseLog("Periodically delete outlinemessage error：", err)
		}
	}
}

//*******************************************************************************
func (self *Server) Sort() { //多次没收到心跳包则断开conn  删除session
	/*timer := time.NewTimer(time.Second * 60)
	for {
		select {
		case <-timer.C:
			{
				for uid, session := range SessionTable {
					if session.beatCount == 0 {
						SessionTable[uid].quit()
					} else {
						SessionTable[uid].beatCount = 0
					}
				}
			}
		}
		timer.Reset(time.Second * 60)
	}*/
	for {
		time.Sleep(time.Second * 60)
		for uid, session := range SessionTable {
			//ImCountbeatLog("uid:", uid, " count beat ", session.beatCount, " Conn", session.conn)
			if session.beatCount == 0 {
				SessionTable[uid].quit() //关闭失败？？？
			} else {
				session.beatCount = 0
				SessionTable[uid] = session
			}

		}
	}

}

//********创建一个server 注意：这里设置了最多20000个客户
func CreateServer(connString string) { //初始化server并开始监听  调用listen()
	var tempDelay time.Duration
	runtime.GOMAXPROCS(runtime.NumCPU()) //启动多核
	//初始化3个内存map
	setTable()
	setConfig()
	server := &Server{ //初始化 通道 会话等，数值型变量不需要初始化
		pending:   make(chan net.Conn), //传输的是 IP等方面的信息，具体是什么？？
		business:  make(chan net.Conn),
		quiting:   make(chan net.Conn, 10),
		maxClient: make(chan byte, MaxSession),
	}
	err := server.imDB.ConnectDb()
	if err != nil {
		fmt.Println("open imdatabase err", err)
		DatabaseLog("Open imdatabase error :", err) ///////////////////////////////////////////////////////////////////
		return
	}
	//下方两个函数是初始化数据库用
	//server.imDB.CreateTables()
	//server.imDB.Addinit()

	ImServerLog("BeeIM server  is running on. The Server is Listening the port :1212 and waiting client for connecting.")
	fmt.Println("A server is running on and listening the port :1212.")
	go func() {
		for {
			time.Sleep(time.Second * 30)
			ImMaxSession("...........sessionTable length", len(SessionTable), " ........conn", TotalConn.GetTotal())
			/*for u, _ := range SessionTable {
				ImCountbeatLog(u)
			}*/
		}
	}()
	server.listen()
	listeners, err := net.Listen("tcp", connString)
	if err != nil {
		fmt.Println("Server open listener err", err)
		ImServerLog("Listen(tcp) error : " + err.Error())
		return
	}
	businesslisteners, err := net.Listen("tcp", ":3434")
	if err != nil {
		fmt.Println("Server open businesslistener err", err)
		ImBusinessLog("listen busniess error : " + err.Error())
		return
	}
	server.listener = listeners
	server.businesslistener = businesslisteners
	for i := 0; i < MaxSession; i++ { //先maxclient充满0，一旦有人进来就拿走一个0
		server.clientOut()
	}

	go server.Sort()   //在线用户心跳包检测
	go server.ReSend() //启动重发机制
	go server.DelOutLineMessage()
	go func() { //垃圾主动回收机制
		for {
			time.Sleep(60 * time.Second)
			runtime.GC()
		}
	}()
	go func() {
		for {
			conns, err := server.businesslistener.Accept()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					if tempDelay == 0 {
						tempDelay = 5 * time.Millisecond
					} else {
						tempDelay *= 2
					}
					if max := 1 * time.Second; tempDelay > max {
						tempDelay = max
					}
					fmt.Printf("http: Accept error: %v; retrying in %v", err, tempDelay)
					time.Sleep(tempDelay)
					continue
				}
				return
			}
			server.pending <- conns
		}
	}()
	for {
		conn, err := server.listener.Accept() //一旦有客户端连接上来就让服务器与客户端建立tcp连接
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				fmt.Printf("http: Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return
		}
		server.pending <- conn
	}
}

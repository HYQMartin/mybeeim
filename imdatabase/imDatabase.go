package imdatabase

import (
	"database/sql"
	"log"
	_ "mysql"
	"os"
	"strconv"
	"time"
)

var (
	//dbHostsIp = "192.168.1.110:3306" //主机IP和端口号
	//dbHostsIp = "127.0.0.1:3306" //主机IP和端口号
	//dbHostsIp  = "localhost:3306"
	dbHostsIp = "127.0.0.1:3306"
	//dbHostsIp  = "10.66.115.255:3306"
	dbUserName = "IM"        // 用户名
	dbPassword = "beeim1234" //密码
	dbDatabase = "beeim"     // 使用的数据库
)

type ImDb struct {
	Db *sql.DB
}

//日志文档
func DatabaseLog(messages ...interface{}) { //记录数据库操作，以及问题
	var year, month, day = time.Now().Date()
	var LogPath = "./log/" + strconv.Itoa(year) + "/" + month.String() + "/" + strconv.Itoa(day) + "/"
	err := os.MkdirAll(LogPath, 0777)
	if err != nil {
		return
	}
	databaselog, err := os.OpenFile(LogPath+"imDatabase.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		return
	}
	defer databaselog.Close()
	dblog := log.New(databaselog, "\r\n", log.Ldate|log.Ltime)
	dblog.Println(messages)
}

//**************连接数据库*********
func (b *ImDb) ConnectDb() error {
	db, err := sql.Open("mysql", dbUserName+":"+dbPassword+"@tcp("+dbHostsIp+")/"+dbDatabase)
	//等价于：Odb, err := sql.Open("mysql", "IM:beeim1234@tcp(192.168.1.108:3306)/beeim")
	if err != nil {
		DatabaseLog("Connect database  error:" + err.Error())
		return err
	}
	DatabaseLog("Database " + dbDatabase + "  is connected.")
	err = db.Ping()
	if err != nil {
		DatabaseLog("数据库连接失败 ", err)
		return err
	}
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(100)
	b.Db = db
	return nil
}

//**************关闭数据库*********
func (b *ImDb) CloseDb() {
	b.Db.Close()
	DatabaseLog("Database ", dbDatabase, "is closed.")
}

//******************建表 **************
func (b *ImDb) CreateTables() error {
	//dialog表
	result1, err := b.Db.Query(`create table dialog(dialog_id  bigint not null,
			        uid bigint not null,
			        timestamp datetime not null default 0,
			        primary key(dialog_id,uid))`) //添加复合主键，dialogid和uid
	if err != nil {
		DatabaseLog("Create dialogTable fail:" + err.Error())
		return err
	}
	defer result1.Close()
	DatabaseLog("Create dialogTable success.")
	//act_dialog表
	result2, err := b.Db.Query("create table act_dialog(dialog_id  bigint not null unique)")
	if err != nil {
		DatabaseLog("Create act_dialogTable fail:" + err.Error())
		return err
	}
	defer result2.Close()
	//DatabaseLog("Create act_dialogTable success.")
	//message表
	result3, err := b.Db.Query(`create table message(sender_id  bigint not null,
			        dialog_id bigint not null,
			        type int not null,
			        information varchar(2048),
			        timestamp datetime not null default 0)
	`)
	if err != nil {
		DatabaseLog("Create messageTable fail:" + err.Error())
		return err
	}
	defer result3.Close()
	DatabaseLog("Create messageTable success:")
	//outlinemessage表
	result4, err := b.Db.Query(`create table outlinemessage(receiver_id  bigint not null,
			        type int not null,
			        packet varbinary(255),
			        timestamp datetime not null default 0)
			        `)
	if err != nil {
		DatabaseLog("Create oueLineMessageTable fail:" + err.Error())
		return err
	}
	defer result4.Close()
	//DatabaseLog("Create oueLineMessageTable success.")
	//user表
	result5, err := b.Db.Query(`create table user(uid  bigint not null unique)`)
	if err != nil {
		DatabaseLog("Create userTable fail:" + err.Error())
		return err
	}
	defer result5.Close()
	//DatabaseLog("Create userTable success.")
	return nil
}

//********************插入dialog*****************
func (b *ImDb) AddDialog(dialogid uint64, uids []uint64) error {
	stmt, err := b.Db.Prepare("insert into dialog(dialog_id,uid) values(?,?) ")
	if err != nil {
		DatabaseLog("Prepare to insert ", dialogid, uids, " into dialogTable fail:", err.Error())
		return err
	}
	defer stmt.Close()
	//插入数据
	for _, id := range uids {
		_, err = stmt.Exec(dialogid, id)
		if err != nil {
			DatabaseLog("Insert ", dialogid, uids, " into dialogTable fail:", err.Error())
			return err
		}
	}

	return nil
}

//********************插入act_dialog*****************
func (b *ImDb) AddActDialog(dialogid uint64) error {
	stmt, err := b.Db.Prepare("insert into act_dialog(dialog_id) values(?) ")
	if err != nil {
		DatabaseLog("Prepare to insert ", dialogid, " into act_dialogTable fail:", err.Error())
		return err
	}
	defer stmt.Close()
	//插入数据
	_, err = stmt.Exec(dialogid)
	if err != nil {
		DatabaseLog("Insert ", dialogid, " into act_dialogTable fail:", err)
		return err
	}
	return nil
}

//********************插入message*****************
func (b *ImDb) AddMessage(uid uint64, dialogid uint64, msgType uint32, information string, messageTimestamp string) error {
	stmt, err := b.Db.Prepare("insert into message(sender_id,dialog_id,type,information,timestamp) values(?,?,?,?,?) ")
	if err != nil {
		DatabaseLog("Prepare to insert ", uid, dialogid, messageTimestamp, " into messageTable fail:", err.Error())
		return err
	}
	defer stmt.Close()
	//插入数据
	_, err = stmt.Exec(uid, dialogid, msgType, information, messageTimestamp)
	if err != nil {
		DatabaseLog("Insert ", dialogid, uid, msgType, messageTimestamp, " into messageTable fail:", err.Error())
		return err
	}
	//显示关闭
	//stmt.Close()
	return nil
}

//********************插入OutLineMessage*****************
func (b *ImDb) AddOutlineMsg(uid uint64, types uint32, packet []byte, messageTimestamp string) error {
	stmt, err := b.Db.Prepare("insert into outlinemessage(receiver_id,type,packet,timestamp) values(?,?,?,?) ")
	if err != nil {
		DatabaseLog("Prepare to insert ", uid, " into outlinemessageTable fail:", err.Error())
		return err
	}
	defer stmt.Close()
	//插入数据
	_, err = stmt.Exec(uid, types, packet, messageTimestamp)
	if err != nil {
		DatabaseLog("Insert ", uid, " into dialogTable fail:"+err.Error())
		return err
	}
	return nil
}

//********************插入user*****************
func (b *ImDb) AddUser(uid uint64) error {

	//查找用户是否已存在
	rows, err := b.Db.Query("select uid from user where uid=?", uid)
	if err != nil {
		DatabaseLog("Prepare to insert  into userTable fail:" + err.Error())
		//	DbErrLog("Prepare to insert " + uid + " into userTable fail:" + err.Error())
		return err
	}
	defer rows.Close()
	if rows.Next() != true { //不存在，则插入
		stmt, err := b.Db.Prepare("insert into user(uid) values(?) ")
		if err != nil {
			return err
		}
		defer stmt.Close()
		//插入数据
		_, err = stmt.Exec(uid)
		if err != nil {
			DatabaseLog("Insert ", uid, " into userTable fail:"+err.Error())
			return err
		}
		DatabaseLog("Add user ", uid, " to usertable.")
	}
	return nil
}

//删除数据表信息，deleteString为prepare语句，参数在parameter
//例子：
//DeleteDbTable("delete from act_dialog where dialog_id=?",1234)
func (b *ImDb) DeleteDbTable(deletePrepareString string, parameter ...interface{}) error {
	//****************删除*********************
	stmt, err := b.Db.Prepare(deletePrepareString)
	if err != nil {
		DatabaseLog("Prepare to delete Table fail:", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(parameter...)
	if err != nil {
		DatabaseLog("Delete Table fail:", err)
		return err
	}
	DatabaseLog(deletePrepareString, parameter)
	return nil
}

//更新数据，输入update的prepare语句，parameter为参数
//例子：
//UpdateTable("update dialog set timestamp=? where dialog_id=? and uid=?","0000-00-00 00:00:00",111222,333)
func (b *ImDb) UpdateTable(updatePrepareString string, parameter ...interface{}) error {
	stmt, err := b.Db.Prepare(updatePrepareString)
	if err != nil {
		DatabaseLog("Prepare to update table fail:", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(parameter...)
	if err != nil {
		DatabaseLog("Update Table fail:", err)
		return err
	}
	DatabaseLog(updatePrepareString, parameter)
	return nil
}

//******************************************
//******************************************
//**************其他操作********************
/*func others() {
//*******************查询*********************
rows, err := Db.Query("select age,id, timestamp from test where id=?",111) //test是测试用的，以后删除
if err != nil {
	fmt.Println("操作失败：", err)
	return
}
defer rows.Close()
var id uint64
var age int
var timestamp []byte
for rows.Next() {
	if err := rows.Scan(&id, &age, &timestamp); err == nil {
		fmt.Print(id)
		fmt.Print("\t")
		fmt.Println(age)
		fmt.Print("\t")
		fmt.Println(tstamp)
		//	if timestamp == "0000-00-00 00:00:00" {
		//		fmt.Println("id=", id)
		//	}
		fmt.Print("\t\r\n")
	} else {
		fmt.Println("*****************", err)
	}
}

*/
/*
func main() {
	var im ImDb
	im.ConnectDb()
	uids := make([]uint64, 0, 100)
	for i := uint64(10); i <= 40; i++ {
		uids = append(uids, i)
		im.AddUser(i)
	}
	im.AddDialog(110, uids)
	im.AddActDialog(110)
}
*/

package fsdatabase

import (
	"database/sql"
	"log"
	_ "mysql"
	"os"
	"strconv"
	"time"
)

var (
	dbHostsIp = "192.168.1.107:3306" //主机IP和端口号
	//dbHostsIp = "127.0.0.1:3306" //主机IP和端口号
	//dbHostsIp  = "localhost:3306"
	//dbHostsIp  = "182.254.140.237:3306"
	dbUserName = "IM"        // 用户名
	dbPassword = "beeim1234" //密码
	dbDatabase = "beeim"     // 使用的数据库
)

type FsDb struct {
	Db *sql.DB
}

func fsDatabaseLog(messages ...interface{}) { //记录数据库操作，以及问题
	var year, month, day = time.Now().Date()
	var LogPath = "./log/" + strconv.Itoa(year) + "/" + month.String() + "/" + strconv.Itoa(day) + "/"
	err := os.MkdirAll(LogPath, 0777)
	if err != nil {
		return
	}
	fsDatabaseLog, err := os.OpenFile(LogPath+"FileServer.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0777)
	if err != nil {
		return
	}
	defer fsDatabaseLog.Close()
	dblog := log.New(fsDatabaseLog, "\r\n", log.Ldate|log.Ltime)
	dblog.Println(messages)
}

//**************连接数据库*********
func (b *FsDb) ConnectDb() error {
	db, err := sql.Open("mysql", dbUserName+":"+dbPassword+"@tcp("+dbHostsIp+")/"+dbDatabase)
	//等价于：Odb, err := sql.Open("mysql", "IM:beeim1234@tcp(192.168.1.108:3306)/beeim")
	if err != nil {
		fsDatabaseLog("Connect database  error:" + err.Error())
		return err
	}
	fsDatabaseLog("Database " + dbDatabase + "  is connected.")
	err = db.Ping()
	if err != nil {
		fsDatabaseLog("数据库连接失败 ", err)
		return err
	}
	b.Db = db
	db.SetMaxIdleConns(20000)
	return nil
}

//**************关闭数据库*********
func (b *FsDb) CloseDb() {
	b.Db.Close()
	fsDatabaseLog("Database ", dbDatabase, "is closed.")
}

//******************建表 **************
func (b *FsDb) CreateTable() error {
	//dialog表
	result1, err := b.Db.Query(`create table fileserver(sha1  char(20) not null primary key,
		            url varchar(40) not null ,
			        timestamp bigint not null`)
	if err != nil {
		fsDatabaseLog("Create dialogTable fail:" + err.Error())
		return err
	}
	defer result1.Close()
	return nil
}

//********************插入dialog*****************
func (b *FsDb) AddFileServer(sha1 string, url string) error {
	stmt, err := b.Db.Prepare("insert into fileserver(sha1,url,timestamp) values(?,?,?) ")
	if err != nil {
		fsDatabaseLog("Prepare to insert fail:", err.Error())
		return err
	}
	defer stmt.Close()
	//插入数据
	_, err = stmt.Exec(sha1, url, time.Now().Unix())
	if err != nil {
		fsDatabaseLog("Insert  fail:", err.Error())
		return err
	}

	return nil
}

//删除数据表信息
func (b *FsDb) DeleteDbTable(sha1 []byte) error {
	//****************删除*********************
	stmt, err := b.Db.Prepare("delete from fileserver where sha1=?")
	if err != nil {
		fsDatabaseLog("Prepare to delete fail:", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(sha1)
	if err != nil {
		fsDatabaseLog("Delete fail:", err)
		return err
	}
	//fsDatabaseLog(deletePrepareString, parameter)
	return nil
}

//更新数据
func (b *FsDb) UpdateTable(sha1 []byte) error {
	stmt, err := b.Db.Prepare("update dialog set timestamp=? where sha1=?")
	if err != nil {
		fsDatabaseLog("Prepare to update table fail:", err)
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec(time.Now().Unix(), sha1)
	if err != nil {
		fsDatabaseLog("Update fail:", err)
		return err
	}
	return nil
}

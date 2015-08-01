//客户端POST的HTTP请求必须包括以下表单
//（1）、FormFiel：
//“uid”=用户的uid
//“sha1”=上传文件的sha1
//（2）、FormFile
//“uploadfile”=上传文件
//
//
//
package main

import (
	//"./fsdatabase"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"
)

//var fsdb fsdatabase
type Myhandler struct{} //server的默认Handler

var filemap map[string]filedata //保存的文件，map[sha1]文件数据

type filedata struct { //文件数据：文件名和最后一次上传的时间戳
	saveFilename string
	timestamp    int64
}

const (
	//文件服务器的IP和端口号

	Server_IP = "http://192.168.1.107:9090"
	//Server_IP = "http://182.254.140.237:9090"
	//Server_IP  = "http://localhost:9090"
	Upload_Dir = "./upload/"
	deletetime = 2592000 //30天删一次
)

func main() {
	filemap = make(map[string]filedata)
	server := http.Server{
		Addr:        ":9090",
		Handler:     &Myhandler{},
		ReadTimeout: 20 * time.Second,
	}
	go func() { //定期删除文件
		for {
			time.Sleep(30 * time.Second)
			deleteFile()
		}
	}()
	go func() { //垃圾主动回收机制
		for {
			runtime.GC()
			time.Sleep(30 * time.Second)
		}
	}()
	fmt.Println("File server is running and listen port:9090")
	err := server.ListenAndServe() //监听端口
	if err != nil {
		log.Fatal("ListenAndServser:", err)
	}

}

//*****************处理函数*************************
func (*Myhandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	//fmt.Println(r.Method)
	if r.Method == "POST" {
		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			fmt.Fprintf(w, "%v", "上传错误")
			return
		}
		//读取文件内容
		temp, err := ioutil.ReadAll(file)
		if err != nil {
			return
		}
		//文件拓展名
		fileext := filepath.Ext(handler.Filename)
		if check(fileext) == false {
			fmt.Fprintf(w, "%v", "不允许的上传类型")
			return
		}
		//获取客户端上传的sha1
		userSha1 := r.FormValue("sha1")
		//计算上传文件的sha1
		sha1file := sha1.New()
		_, err = io.WriteString(sha1file, string(temp))
		if err != nil {
			return
		}
		uploadSha1 := sha1file.Sum(nil) //文件的sha1
		sh := string(uploadSha1)
		hexSha1 := hex.EncodeToString(uploadSha1)
		//上传文件的sha1和用户端上传的sha1不一致，则返回错误
		if userSha1 != hexSha1 {
			fmt.Fprintf(w, "%v", "上传的文件错误")
			return
		}
		//fmt.Println(userSha1, "\n", hex.EncodeToString(uploadSha1))
		checkfile := filemap[sh].saveFilename
		//服务器已有保存该文件，则不上传，则返回文件的URL
		if checkfile != "" {
			temps := filedata{checkfile, time.Now().Unix()}
			filemap[sh] = temps
			io.WriteString(w, Server_IP+checkfile)
		} else {
			nowString := strconv.FormatUint(uint64(time.Now().UnixNano()), 16)
			filename := nowString + fileext
			uid := r.FormValue("uid")
			//用户目录
			err = os.MkdirAll(Upload_Dir+uid+"/", 0777)
			if err != nil {
				return
			}
			f, err := os.OpenFile(Upload_Dir+uid+"/"+filename, os.O_CREATE|os.O_RDWR, 0777)
			if err != nil {
				return
			}
			defer f.Close()
			_, err = f.Write(temp)
			if err != nil {
				fmt.Fprintf(w, "%v", "上传失败")
				return
			}
			fileUrl := "/" + uid + "/" + filename
			fmt.Println("/////////length", len(fileUrl))
			filemap[sh] = filedata{fileUrl, time.Now().Unix()}
			//返回文件的URL
			io.WriteString(w, Server_IP+fileUrl)
		}

	} else if r.Method == "GET" {
		//fmt.Println(r.URL.Path)
		fileDir := Upload_Dir + r.URL.Path
		_, err := os.Stat(fileDir)
		if err != nil {
			http.Error(w, "File Not Found.", 404)
			return
		}
		//读取文件内容
		buf, err := ioutil.ReadFile(fileDir)
		if err != nil {
			http.Error(w, "Server Error.", 500)
			return
		}
		//写文件到response
		_, err = w.Write(buf)
		if err != nil {
			http.Error(w, "Server Error.", 500)
			return
		}
	}
}

//***************************检测上传文件后缀*************************
func check(name string) bool {
	ext := []string{".exe", ".js"}
	for _, v := range ext {
		if v == name {
			return false
		}
	}
	return true
}

//*****************************定期删除文件**************************
func deleteFile() {
	for key, t := range filemap {
		dura := time.Now().Unix() - t.timestamp
		if dura >= deletetime {
			err := os.Remove(Upload_Dir + t.saveFilename)
			if err != nil {
				fmt.Println("delete File error:")
				return
			}
			delete(filemap, key)
		}
	}
}

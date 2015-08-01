package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strconv"
)

const (
	n         = 1
	uid       = "987654321"
	Server_IP = "http://192.168.1.107:9090"
	//Server_IP = "http://localhost:9090"
)

var filename string

//**********************下载文件************************
func Downlaodfile(targetUrl string, savePath string) error {
	resp, err := http.Get(targetUrl)
	if err != nil {
		fmt.Println("GET err.", err)
		return err
	}
	if resp.StatusCode != 200 {
		fmt.Println("Response Status:", resp.Status)
	} else {
		f, err := os.OpenFile(savePath+"/"+path.Base(targetUrl), os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		defer f.Close()
		_, err = f.Stat() //获取文件状态
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		_, err = io.Copy(f, resp.Body)
		if err != nil {
			return err
		}
		return err
	}
	return nil
}

//*********************上传文件***************************
func PostFile(filename string) (file_URL string, err error) {
	upload_URL := Server_IP
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("uploadfile", filename)
	if err != nil {
		return
	}
	fh, err := os.Open(filename)
	if err != nil {
		return
	}
	defer fh.Close()
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return
	}
	//计算sha1
	fh, err = os.Open(filename)
	if err != nil {
		return
	}
	defer fh.Close()
	fileSha1 := sha1.New()
	_, err = io.Copy(fileSha1, fh)
	if err != nil {
		return
	}
	sha := fileSha1.Sum(nil)
	//将sha1发送到服务器
	p, err := bodyWriter.CreateFormField("sha1")
	if err != nil {
		return
	}
	hexSha1 := hex.EncodeToString(sha)
	_, err = p.Write([]byte(hexSha1))
	if err != nil {
		return
	}
	//写UID
	p, err = bodyWriter.CreateFormField("uid")
	if err != nil {
		return
	}
	_, err = p.Write([]byte(uid))
	if err != nil {
		return
	}
	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()
	resp, err := http.Post(upload_URL, contentType, bodyBuf) // Post 方法把缓存传到服务器。
	if err != nil {
		return
	}
	defer resp.Body.Close()
	re, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	//上传文件的URL
	file_URL = string(re)
	return file_URL, nil
}

//*********************客户端协程**********************************
func load(ch chan int, i int) {
	filename = "./test/" + strconv.Itoa(i) + ".jepg"
	f, _ := PostFile(filename)
	fmt.Println("file  ", i, "is:", f)
	savePath := "./test/"
	err := Downlaodfile(f, savePath)
	if err != nil {
		fmt.Println("downerr :", err)
		return
	}
	fmt.Println("downerr succ:")
	ch <- 1
}

//example
func main() {
	os.MkdirAll("./test/", 0660)
	chs := make([]chan int, n)
	for i := 0; i < n; i++ {
		chs[i] = make(chan int)
		fmt.Println(i)
		go load(chs[i], i)
	}
	for _, ch := range chs {
		<-ch
	}
}

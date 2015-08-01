s2c 
===
放置服务端与客户端通信源程序

packet.go
===========
定义packet io相关的工具，包括包加密，解密，解包等操作

typedef.go
==========
定义packet, session,server 相关结构

session.go
==========
管理会话：对上层输入/输出message，对下层输出tcpstream

server.go
=========
服务容器，启动服务器，监听一个端口的tcp连接


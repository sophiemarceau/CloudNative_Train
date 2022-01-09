package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang/glog"

	"github.com/thinkeridea/go-extend/exnet"
)

//1.接收客户端 request，并将 request 中带的 header 写入 response header
//2.读取当前系统的环境变量中的 VERSION 配置，并写入 response header
//3.Server 端记录访问日志包括客户端 IP，HTTP 返回码，输出到 server 端的标准输出
//4.当访问 localhost/healthz 时，应返回 200
func main() {

	//4.当访问 localhost/healthz 时，应返回 200
	http.HandleFunc("/healthz", healthz)
	http.HandleFunc("/", full)
	http.HandleFunc("/logs", logs)

	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func full(writer http.ResponseWriter, request *http.Request) {
	//1.接收客户端 request，并将 request 中带的 header 写入 response header
	for name, values := range request.Header {
		writer.Header().Set(name, values[0])
	}
	//2.读取当前系统的环境变量中的 VERSION 配置，并写入 response header
	version := os.Getenv("VERSION")
	writer.Header().Set("VERSION", version)
	//3.Server 端记录访问日志包括客户端 IP，HTTP 返回码，输出到 server 端的标准输出
	resCode := 204
	fmt.Println("IP", exnet.ClientIP(request), "code:", resCode, version)
	writer.WriteHeader(resCode)
}

// 4 当访问 localhost/healthz 时，应返回 200
func healthz(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "ok")
	w.WriteHeader(http.StatusOK)
}

/**
启动时通过 -log_dir=log 指定目录，但是log目录必须存在
*/
func logs(w http.ResponseWriter, r *http.Request) {
	defer glog.Flush()
	//flag.Lookup("logtostderr").Value.Set("true")
	//currentPath, _ := os.Getwd()
	//flag.Lookup("log_dir").Value.Set(currentPath + "/log")
	flag.Parse()
	fmt.Print("log_dir:", flag.Lookup("log_dir").Value)

	now := time.Now().Format("yyyy-MM-dd hh:mm:ss")
	ip := exnet.ClientIP(r)
	statusCode := http.StatusOK
	w.Header().Add("statusCode", "200")
	path := r.RequestURI
	glog.V(2).Infof("%s\t%s\t%s\t%d", now, ip, path, statusCode)
}

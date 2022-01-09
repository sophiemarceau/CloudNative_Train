package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"
)

func index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<h1>Welcome to cloud native</n1>"))

	//1将 request 中带的 header 写入 response header
	for k, v := range r.Header {
		for _, vv := range v {
			fmt.Printf("Header key: %s, Header value: %s \n", k, v)
			w.Header().Set(k, vv)
		}
	}
	//2读取当前系统的环境变量中的 VERSION 配置，并写入 response header
	os.Setenv("VERSION", "0.0.4")
	version := os.Getenv("VERSION")
	fmt.Println(version)
	w.Header().Set("VERSION", version)
	fmt.Printf("os version: %s \n", version)

	//取clientIP
	clientIP := r.RemoteAddr
	fmt.Println(clientIP)

	//3Server端记录访问日志包括客户端 IP，HTTP 返回码，输出到 server 端的标准输出
	//如果经过负载均衡器， proxy，RemoteAddr 取clientIP是负载均衡器， proxy的地址
	//不是真实的地址
	// X-REAL_IP
	// X-FORWORD_FOR
	clientIP = getCurrentIP(r)
	log.Printf("Success Response code %d", 200)
	log.Printf("success client Ip: %s", clientIP)
}

func getCurrentIP(r *http.Request) string {
	// 这里也可以通过X-Forwarded-For请求头的第一个值作为用户的ip   // 但是要注意的是这两个请求头代表的ip都有可能是伪造的
	ip := r.Header.Get("X-REAL-IP")
	if ip == "" {
		//RemoteAd IP:PORT
		// 当请求头不存在即不存在代理时直接获取ip
		ip = strings.Split(r.RemoteAddr, ":")[0]
	}
	return ip
}

// ClientIP 尽最大努力实现获取客户端 IP 的算法。
//解析 X-Real-IP 和 X-Forwarded-For 以便于反向代理（nginx 或 haproxy）可以正常工作。
func ClientIP(r *http.Request) string {
	xForwardedFor := r.Header.Get("X-Forwarded-For")
	ip := strings.TrimSpace(strings.Split(xForwardedFor, ",")[0])
	if ip != "" {
		return ip
	}
	ip = strings.TrimSpace(r.Header.Get("X-Real-Ip"))
	if ip != "" {
		return ip
	}
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

//4当访问 localhost/healthz 时，应返回200
func heathz(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "200")
}

/**
编写一个 HTTP
1服务器接收客户端 request，并将 request 中带的 header 写入 response header
2读取当前系统的环境变量中的 VERSION 配置，并写入 response header
3Server 端记录访问日志包括客户端 IP，HTTP 返回码，输出到 server 端的标准输出
4当访问 localhost/healthz 时，应返回200
*/
func main() {
	mux := http.NewServeMux()

	//debug
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	mux.HandleFunc("/", index)
	mux.HandleFunc("/healthz", heathz)
	if error := http.ListenAndServe("localhost:8080", mux); error != nil {
		log.Fatal("start http server failed, %s\n", error.Error())
	}
}

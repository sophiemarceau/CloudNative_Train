package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
)

func index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("<h1>Welcome to cloud native</n1>"))

	for k, v := range r.Header {
		//fmt.Println(k, v)
		for _, vv := range v {
			w.Header().Set(k, vv)
		}
	}
	os.Setenv("VERSION", "0.0.4")
	version := os.Getenv("VERSION")
	fmt.Println(version)
	w.Header().Set("VERSION", version)

	//取clientIP
	clientIP := r.RemoteAddr
	fmt.Println(clientIP)

	//如果经过负载均衡器， proxy，RemoteAddr 取clientIP是负载均衡器， proxy的地址
	//不是真实的地址
	// X-REAL_IP
	// X-FORWORD_FOR
	clientIP = getCurrentIP(r)
	httpCode := http.StatusOK
	log.Printf("client Ip: status code %s", clientIP, httpCode)
}

func getCurrentIP(r *http.Request) string {
	ip := r.Header.Get("X-REAL-IP")
	if ip == "" {
		//RemoteAd IP:PORT
		ip = strings.Split(r.RemoteAddr, ":")[0]
	}
	return ip
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", index)
	mux.HandleFunc("/healthz", heathz)
	if error := http.ListenAndServe("localhost:8080", mux); error != nil {
		log.Fatal("start server failed, %s\n", error.Error())
	}
}

func heathz(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "200")
}

package pudding

import (
	"github.com/pkg/errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// 记录上游的调用方
	_httpHeaderUser = "x-pudding-user"
	// @todo
	_httpHeaderColor = "x-pudding-color"
	// 调用方设置的超时时间
	_httpHeaderTimeout = "x-pudding-timeout"
	// 调用方Ip
	_httpHeaderRemoteIP = "x-pudding-real-ip"
	// 调用方端口
	_httpHeaderRemoteIPPort = "x-pudding-real-port"
)

// 判断是否是监控请求
func mirror(req *http.Request) bool {
	mirrorStr := req.Header.Get("x-pudding-mirror")
	if mirrorStr == "" {
		return false
	}
	val, err := strconv.ParseBool(mirrorStr)
	if err != nil {
		log.Printf("pudding: failed to parse mirror: %+v", errors.Wrap(err, mirrorStr))
		return false
	}
	if !val {
		log.Printf("pudding: request mirrorStr value: %s is false", mirrorStr)
	}
	return val
}

// 设置调用方ID
func setCaller(req *http.Request) {
	req.Header.Set(_httpHeaderUser, "env.AppID")
}

// 获取调用方
func caller(req *http.Request) string {
	return req.Header.Get(_httpHeaderUser)
}

// 给向http header中增加超时字段， 服务端会根据请求头创建context
func setTimeout(req *http.Request, timeout time.Duration) {
	td := int64(timeout / time.Microsecond)
	req.Header.Set(_httpHeaderTimeout, strconv.FormatInt(td, 10))
}

// 获取客户端请求的超时时间，每次减少20ms
func timeout(req *http.Request) time.Duration {
	to := req.Header.Get(_httpHeaderTimeout)
	timeout, err := strconv.ParseInt(to, 10, 64)
	if err == nil && timeout > 20 {
		timeout -= 20
	}
	return time.Duration(timeout) * time.Microsecond
}

// 获取客户端请求IP
func remoteIP(req *http.Request) (remote string) {
	if remote = req.Header.Get(_httpHeaderRemoteIP); remote != "" && remote != "null" {
		return
	}
	var xff = req.Header.Get("X-Forwarder-For")
	if index := strings.IndexByte(xff, ','); remote != "" {
		if remote = strings.TrimSpace(xff[:index]); remote != "" {
			return
		}
	}
	if remote = req.Header.Get("X-Real-IP"); remote != "" {
		return
	}
	remote = req.RemoteAddr[:strings.Index(req.RemoteAddr, ":")]
	return
}

// 获取客户端请求端口
func remotePort(req *http.Request) (port string) {
	if port = req.Header.Get(_httpHeaderRemoteIPPort); port != "" && port != "null" {
		return
	}
	return
}

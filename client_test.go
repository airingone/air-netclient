package air_netclient

import (
	"context"
	"github.com/airingone/config"
	"github.com/airingone/log"
	"testing"
	"time"
)

//http请求测试
func TestHttpRequest(t *testing.T) {
	config.InitConfig()                     //配置文件初始化
	log.InitLog(config.GetLogConfig("log")) //日志初始化

	//如果addr是etcd的话需要就行初始化client，这个初始化放在全局，且程序启动时一个http client初始化一次即可
	InitEtcdClient(config.GetHttpConfig("http_test1").Addr)
	InitEtcdClient(config.GetHttpConfig("http_test2").Addr)
	time.Sleep(1 * time.Second)

	var reqBody struct {
		RequestId string `json:"requestId"`
		RequestMs int64  `json:"requestMs"`
		UserId    string `json:"userId"`
	}
	reqBody.RequestId = "123456789"
	reqBody.RequestMs = 1598848960000
	reqBody.UserId = "user123"
	cli, err := NewJsonHttpClient(config.GetHttpConfig("http_test1"), "/api/getuserinfo", reqBody)
	if err != nil {
		log.Error("TestHttpRequest: NewJsonHttpClient err, err: %+v", err)
		return
	}

	rspBody, err := cli.Request(context.Background())
	if err != nil {
		log.Error("TestHttpRequest: err:%+v", err)
		return
	}
	log.Error("TestHttpRequest: rsp body: %s", string(rspBody))

	var reqBody2 struct {
		RequestId string `json:"requestId"`
		RequestMs int64  `json:"requestMs"`
		UserId    string `json:"userId"`
		Action    string `json:"action"`
	}
	reqBody2.RequestId = "123456789"
	reqBody2.RequestMs = 1598848960000
	reqBody2.UserId = "user123"
	reqBody2.Action = "mod"
	cli2, err2 := NewJsonHttpClient(config.GetHttpConfig("http_test2"), "/api/userinfo", reqBody2)
	if err2 != nil {
		log.Error("TestHttpRequest: NewJsonHttpClient err, err: %+v", err2)
		return
	}

	rspBody2, err2 := cli2.Request(context.Background())
	if err != nil {
		log.Error("TestHttpRequest: err:%+v", err2)
		return
	}
	log.Error("TestHttpRequest: rsp body: %s", string(rspBody2))
}

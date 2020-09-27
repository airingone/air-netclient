package air_netclient

import (
	"context"
	"encoding/json"
	air_etcd "github.com/airingone/air-etcd"
	air_netserver "github.com/airingone/air-netserver"
	"github.com/airingone/config"
	"github.com/airingone/log"
	"testing"
	"time"
)

//http请求测试
func TestHttpRequest(t *testing.T) {
	config.InitConfig()                     //配置文件初始化
	log.InitLog(config.GetLogConfig("log")) //日志初始化

	//如果addr是etcd的话需要就行初始化client，这个初始化放在全局init()，且程序启动时一个http client初始化一次即可
	air_etcd.InitEtcdClient(config.GetEtcdConfig("etcd"),
		config.GetHttpConfig("http_test1").Addr, config.GetHttpConfig("http_test2").Addr)
	defer air_etcd.CloseEtcdClient()
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
		log.Error("[NETCLIENT]: TestHttpRequest NewJsonHttpClient err, err: %+v", err)
		return
	}

	rspBody, err := cli.Request(context.Background())
	if err != nil {
		log.Error("[NETCLIENT]: TestHttpRequest err:%+v", err)
		return
	}
	log.Error("[NETCLIENT]: TestHttpRequest rsp body: %s", string(rspBody))

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
		log.Error("[NETCLIENT]: TestHttpRequest NewJsonHttpClient err, err: %+v", err2)
		return
	}

	rspBody2, err2 := cli2.Request(context.Background())
	if err != nil {
		log.Error("[NETCLIENT]: TestHttpRequest err:%+v", err2)
		return
	}
	log.Error("[NETCLIENT]: TestHttpRequest rsp body: %s", string(rspBody2))
}

//http并发请求测试
func TestHttpRequests(t *testing.T) {
	config.InitConfig()                     //配置文件初始化
	log.InitLog(config.GetLogConfig("log")) //日志初始化

	//如果addr是etcd的话需要就行初始化client，这个初始化放在全局init()，且程序启动时一个http client初始化一次即可
	air_etcd.InitEtcdClient(config.GetEtcdConfig("etcd"),
		config.GetHttpConfig("http_test1").Addr, config.GetHttpConfig("http_test2").Addr)
	defer air_etcd.CloseEtcdClient()
	time.Sleep(2 * time.Second)

	var reqBody1 struct {
		RequestId string `json:"requestId"`
		RequestMs int64  `json:"requestMs"`
		UserId    string `json:"userId"`
	}
	reqBody1.RequestId = "123456789"
	reqBody1.RequestMs = 1598848960000
	reqBody1.UserId = "user123"
	cli1, err := NewJsonHttpClient(config.GetHttpConfig("http_test1"), "/api/getuserinfo", reqBody1)
	if err != nil {
		log.Error("[NETCLIENT]: TestHttpRequest NewJsonHttpClient err, err: %+v", err)
		return
	}

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
		log.Error("[NETCLIENT]: TestHttpRequest NewJsonHttpClient err, err: %+v", err2)
		return
	}

	err = HttpRequests(context.Background(), cli1, cli2)
	if cli1.Status == HttpRequestStatusDone {
		log.Info("[NETCLIENT]: Succ: rspBody1: %s", string(cli1.RspBody))
	} else {
		log.Info("[NETCLIENT]: Failed: status: %s, err: %+v", cli1.Status, cli1.Err)
	}
	if cli2.Status == HttpRequestStatusDone {
		log.Info("[NETCLIENT]: Succ: rspBody2: %s", string(cli2.RspBody))
	} else {
		log.Info("[NETCLIENT]: Failed: status: %s, err: %+v", cli2.Status, cli2.Err)
	}

	time.Sleep(1 * time.Second)
}

type GetUserInfoReq struct {
	RequestId string `json:"request_id"`
	UserId    string `json:"user_id"`
}

type GetUserInfoRsp struct {
	RequestId string `json:"request_id"`
	ErrCode   int32  `json:"err_code"`
	ErrMsg    string `json:"err_msg"`
	UserId    string `json:"user_id"`
	UserName  string `json:"user_name"`
}

//udp并发请求测试
func TestUdpRequests(t *testing.T) {
	config.InitConfig()                     //配置文件初始化
	log.InitLog(config.GetLogConfig("log")) //日志初始化

	clientConfig := config.GetNetConfig("udp_test1")
	//如果addr是etcd的话需要就行初始化client，这个初始化放在全局init()，且程序启动时一个http client初始化一次即可
	air_etcd.InitEtcdClient(config.GetEtcdConfig("etcd"),
		clientConfig.Addr)
	defer air_etcd.CloseEtcdClient()
	time.Sleep(2 * time.Second)

	//请求包
	var req GetUserInfoReq
	req.UserId = "user01"
	req.RequestId = "123456"
	dataReq, _ := json.Marshal(req)
	dataBytes, err := air_netserver.CommonProtocolPacket([]byte(clientConfig.ServerName),
		[]byte("getuserinfo"), dataReq)
	if err != nil {
		log.Error("packet data err, %+v", err)
		return
	}
	cli, err := NewBytesUdpClient(clientConfig, dataBytes)
	if err != nil {
		log.Error("NewBytesUdpClient err, %+v", err)
		return
	}

	var req2 GetUserInfoReq
	req2.UserId = "user02"
	req2.RequestId = "12345678"
	dataReq2, _ := json.Marshal(req2)
	dataBytes2, err := air_netserver.CommonProtocolPacket([]byte(clientConfig.ServerName),
		[]byte("getuserinfo"), dataReq2)
	if err != nil {
		log.Error("packet data err, %+v", err)
		return
	}
	cli2, err := NewBytesUdpClient(clientConfig, dataBytes2)
	if err != nil {
		log.Error("NewBytesUdpClient err, %+v", err)
		return
	}

	//发起请求
	err = UdpRequests(context.Background(), cli, cli2)
	if cli.Err != nil {
		log.Error("request err, %+v", err)
	} else {
		//解析回报
		serverName, cmd, dataRsp, _ := air_netserver.CommonProtocolUnPacket(cli.RspData)
		log.Info("rsp: serverName=%s, cmd=%s, data: %s", serverName, cmd, string(dataRsp))
		var rsp GetUserInfoRsp
		err = json.Unmarshal(dataRsp, &rsp)
		if err != nil {
			log.Error("Unmarshal err, %+v", err)
			return
		}
		log.Info("succ: rsp=%+v", rsp)

		serverName2, cmd2, dataRsp2, _ := air_netserver.CommonProtocolUnPacket(cli2.RspData)
		log.Info("rsp2: serverName=%s, cmd=%s, data: %s", serverName2, cmd2, string(dataRsp2))
		var rsp2 GetUserInfoRsp
		err = json.Unmarshal(dataRsp2, &rsp2)
		if err != nil {
			log.Error("Unmarshal err, %+v", err)
			return
		}
		log.Info("succ: rsp2=%+v", rsp2)
	}
}

//tcp并发请求测试
func TestTcpRequests(t *testing.T) {
	config.InitConfig()                     //配置文件初始化
	log.InitLog(config.GetLogConfig("log")) //日志初始化

	clientConfig := config.GetNetConfig("tcp_test1")
	//如果addr是etcd的话需要就行初始化client，这个初始化放在全局init()，且程序启动时一个http client初始化一次即可
	air_etcd.InitEtcdClient(config.GetEtcdConfig("etcd"),
		clientConfig.Addr)
	defer air_etcd.CloseEtcdClient()
	time.Sleep(2 * time.Second)

	//请求包
	var req GetUserInfoReq
	req.UserId = "user01"
	req.RequestId = "123456"
	dataReq, _ := json.Marshal(req)
	dataBytes, err := air_netserver.CommonProtocolPacket([]byte(clientConfig.ServerName),
		[]byte("getuserinfo"), dataReq)
	if err != nil {
		log.Error("packet data err, %+v", err)
		return
	}

	cli, err := NewTcpClient(clientConfig)
	if err != nil {
		log.Error("NewBytesUdpClient err, %+v", err)
		return
	}
	_ = cli.SetBytesReq(dataBytes)

	var req2 GetUserInfoReq
	req2.UserId = "user02"
	req2.RequestId = "12345678"
	dataReq2, _ := json.Marshal(req2)
	dataBytes2, err := air_netserver.CommonProtocolPacket([]byte(clientConfig.ServerName),
		[]byte("getuserinfo"), dataReq2)
	if err != nil {
		log.Error("packet data err, %+v", err)
		return
	}
	cli2, err := NewTcpClient(clientConfig)
	if err != nil {
		log.Error("NewBytesUdpClient err, %+v", err)
		return
	}
	_ = cli2.SetBytesReq(dataBytes2)

	for {
		//发起请求
		err = TcpRequests(context.Background(), cli, cli2)
		if cli.Err != nil {
			log.Error("request err, %+v", err)
		} else {
			//解析回报
			serverName, cmd, dataRsp, _ := air_netserver.CommonProtocolUnPacket(cli.RspData)
			log.Info("rsp: serverName=%s, cmd=%s, data: %s", serverName, cmd, string(dataRsp))
			var rsp GetUserInfoRsp
			err = json.Unmarshal(dataRsp, &rsp)
			if err != nil {
				log.Error("Unmarshal err, %+v", err)
				return
			}
			log.Info("succ: rsp=%+v", rsp)

			serverName2, cmd2, dataRsp2, _ := air_netserver.CommonProtocolUnPacket(cli2.RspData)
			log.Info("rsp2: serverName=%s, cmd=%s, data: %s", serverName2, cmd2, string(dataRsp2))
			var rsp2 GetUserInfoRsp
			err = json.Unmarshal(dataRsp2, &rsp2)
			if err != nil {
				log.Error("Unmarshal err, %+v", err)
				return
			}
			log.Info("succ: rsp2=%+v", rsp2)
		}

		time.Sleep(300 * time.Second)
	}

	defer cli.Close()
	defer cli2.Close()
}

//tcp并发请求测试
func TestTcpKeepAlive(t *testing.T) {
	config.InitConfig()                     //配置文件初始化
	log.InitLog(config.GetLogConfig("log")) //日志初始化

	clientConfig := config.GetNetConfig("tcp_test1")
	//如果addr是etcd的话需要就行初始化client，这个初始化放在全局init()，且程序启动时一个http client初始化一次即可
	air_etcd.InitEtcdClient(config.GetEtcdConfig("etcd"),
		clientConfig.Addr)
	defer air_etcd.CloseEtcdClient()
	time.Sleep(2 * time.Second)

	var req GetUserInfoReq
	req.UserId = "user01"
	req.RequestId = "123456"
	dataReq, _ := json.Marshal(req)
	dataBytes, err := air_netserver.CommonProtocolPacket([]byte(clientConfig.ServerName),
		[]byte("getuserinfo"), dataReq)
	if err != nil {
		log.Error("packet data err, %+v, %s", err, string(dataBytes))
		return
	}

	cli, err := NewTcpClient(clientConfig)
	if err != nil {
		log.Error("NewBytesUdpClient err, %+v", err)
		return
	}

	//启动keepalive
	cli.KeepAlive(context.Background())

	//send
	go func() {
		for {
			err = cli.Write(context.Background(), dataBytes,
				time.Duration(clientConfig.TimeOutMs)*time.Millisecond)
			time.Sleep(100 * time.Second)
		}
	}()

	//reqd
	go func() {
		for {
			rspBuf, err := cli.Read(context.Background(),
				time.Duration(clientConfig.TimeOutMs)*time.Millisecond)
			if err != nil {
				log.Info("failed: %+v", err)
				time.Sleep(time.Duration(clientConfig.TimeOutMs) * time.Millisecond)
				continue
			}
			log.Info("succ: rsp=%s", string(rspBuf))
			/*var rsp GetUserInfoRsp
			err = json.Unmarshal(rspBuf, &rsp)
			if err != nil {
				log.Error("Unmarshal err, %+v", err)

			}*/
			time.Sleep(time.Duration(clientConfig.TimeOutMs) * time.Millisecond)
		}
	}()

	select {}
}

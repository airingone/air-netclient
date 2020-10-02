# 网络client
## 1.组件描述
- 1.提供http client接口；
- 2.提供udp client接口；
- 3.提供tcp client接口；
- 4.grpc接口直接参考[grpc_test.go](https://github.com/airingone/air-grpc/blob/master/grpc_test.go)。
## 2.如何使用
### 2.1 http client
```
import (
    "github.com/airingone/config"
    "github.com/airingone/log"
    air_etcd "github.com/airingone/air-etcd"
    netclient "github.com/airingone/air-netclient"
)

func main() {
    config.InitConfig()                        //进程启动时调用一次初始化配置文件，配置文件名为config.yml，目录路径为../conf/或./
    log.InitLog(config.GetLogConfig("log"))    //进程启动时调用一次初始化日志
    //如果addr是etcd的话需要就行初始化client，这个初始化放在全局init()，且程序启动时一个http client初始化一次即可
	air_etcd.InitEtcdClient(config.GetEtcdConfig("etcd"),
		config.GetHttpConfig("http_test1").Addr, config.GetHttpConfig("http_test2").Addr)
	defer air_etcd.CloseEtcdClient()
	time.Sleep(1 * time.Second)

	var reqBody1 struct {
		RequestId string `json:"requestId"`
		RequestMs int64  `json:"requestMs"`
		UserId    string `json:"userId"`
	}
	reqBody1.RequestId = "123456789"
	reqBody1.RequestMs = 1598848960000
	reqBody1.UserId = "user123"
	cli1, err := netclient.NewJsonHttpClient(config.GetHttpConfig("http_test1"), "/api/getuserinfo", reqBody1)
	if err != nil {
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
	cli2, err2 := netclient.NewJsonHttpClient(config.GetHttpConfig("http_test2"), "/api/userinfo", reqBody2)
	if err2 != nil {
		return
	}

	err = netclient.HttpRequests(context.Background(), cli1, cli2)
	if cli1.Status == HttpRequestStatusDone {
		log.Info("Succ: rspBody1: %s", string(cli1.RspBody))
	} else {
		log.Info("Failed: status: %s, err: %+v", cli1.Status, cli1.Err)
	}
	if cli2.Status == HttpRequestStatusDone {
		log.Info("Succ: rspBody2: %s", string(cli2.RspBody))
	} else {
		log.Info("Failed: status: %s, err: %+v", cli2.Status, cli2.Err)
	}    
}
```
- [server服务参考](https://github.com/airingone/air-gin/blob/master/gin_test.go)
### 2.2 udp client
```
import (
    "github.com/airingone/config"
    "github.com/airingone/log"
    air_etcd "github.com/airingone/air-etcd"
    netclient "github.com/airingone/air-netclient"
    netserver "github.com/airingone/air-netserver"
)

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

func main() {
    config.InitConfig()                        //进程启动时调用一次初始化配置文件，配置文件名为config.yml，目录路径为../conf/或./
    log.InitLog(config.GetLogConfig("log"))    //进程启动时调用一次初始化日志
    clientConfig := config.GetNetConfig("udp_test1")
	//如果addr是etcd的话需要就行初始化client，这个初始化放在全局init()，且程序启动时一个http client初始化一次即可
	air_etcd.InitEtcdClient(config.GetEtcdConfig("etcd"),
		clientConfig.Addr)
	defer air_etcd.CloseEtcdClient()
	time.Sleep(1 * time.Second)

	//请求包
	var req GetUserInfoReq
	req.UserId = "user01"
	req.RequestId = "123456"
	dataReq, _ := json.Marshal(req)
	dataBytes, err := netserver.CommonProtocolPacket([]byte(clientConfig.ServerName),
		[]byte("getuserinfo"), dataReq)
	if err != nil {
		log.Error("packet data err, %+v", err)
		return
	}
	cli, err := netclient.NewBytesUdpClient(clientConfig, dataBytes)
	if err != nil {
		log.Error("NewBytesUdpClient err, %+v", err)
		return
	}

	var req2 GetUserInfoReq
	req2.UserId = "user02"
	req2.RequestId = "12345678"
	dataReq2, _ := json.Marshal(req2)
	dataBytes2, err := netserver.CommonProtocolPacket([]byte(clientConfig.ServerName),
		[]byte("getuserinfo"), dataReq2)
	if err != nil {
		log.Error("packet data err, %+v", err)
		return
	}
	cli2, err := netclient.NewBytesUdpClient(clientConfig, dataBytes2)
	if err != nil {
		log.Error("NewBytesUdpClient err, %+v", err)
		return
	}

	//发起请求
	err = netclient.UdpRequests(context.Background(), cli, cli2)
	if cli.Err != nil {
		log.Error("request err, %+v", err)
	} else {
		//解析回报
		serverName, cmd, dataRsp, _ := netserver.CommonProtocolUnPacket(cli.RspData)
		log.Info("rsp: serverName=%s, cmd=%s, data: %s", serverName, cmd, string(dataRsp))
		var rsp GetUserInfoRsp
		err = json.Unmarshal(dataRsp, &rsp)
		if err != nil {
			log.Error("Unmarshal err, %+v", err)
			return
		}
		log.Info("succ: rsp=%+v", rsp)

		serverName2, cmd2, dataRsp2, _ := netserver.CommonProtocolUnPacket(cli2.RspData)
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
```
- [server服务参考](https://github.com/airingone/air-netserver/blob/master/net_test.go)
### 2.3 tcp client
```
import (
    "github.com/airingone/config"
    "github.com/airingone/log"
    air_etcd "github.com/airingone/air-etcd"
    netclient "github.com/airingone/air-netclient"
    netserver "github.com/airingone/air-netserver"
)

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

func main() {
    config.InitConfig()                        //进程启动时调用一次初始化配置文件，配置文件名为config.yml，目录路径为../conf/或./
    log.InitLog(config.GetLogConfig("log"))    //进程启动时调用一次初始化日志
    
    clientConfig := config.GetNetConfig("tcp_test1")
	//如果addr是etcd的话需要就行初始化client，这个初始化放在全局init()，且程序启动时一个http client初始化一次即可
	air_etcd.InitEtcdClient(config.GetEtcdConfig("etcd"),
		clientConfig.Addr)
	defer air_etcd.CloseEtcdClient()
	time.Sleep(1 * time.Second)

	var req GetUserInfoReq
	req.UserId = "user01"
	req.RequestId = "123456"
	dataReq, _ := json.Marshal(req)
	dataBytes, err := netserver.CommonProtocolPacket([]byte(clientConfig.ServerName),
		[]byte("getuserinfo"), dataReq)
	if err != nil {
		log.Error("packet data err, %+v, %s", err, string(dataBytes))
		return
	}

	cli, err := netclient.NewTcpClient(clientConfig)
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

	//read
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
			time.Sleep(time.Duration(clientConfig.TimeOutMs) * time.Millisecond)
		}
	}()

	select {}  
}
```
这个例子主要是保持tcp长连接的列子，cli.Read读数据的处理逻辑需要根据具体业务来实现。
- [server服务参考](https://github.com/airingone/air-netserver/blob/master/net_test.go)
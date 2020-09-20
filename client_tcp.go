package air_netclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	airetcd "github.com/airingone/air-etcd"
	"github.com/airingone/config"
	"github.com/airingone/log"
	"github.com/gogo/protobuf/proto"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	TcpMaxRecvbuf = 1024 * 128 // 128k

	TcpRequestStatusInit  = "init"
	TcpRequestStatusStart = "start"
	TcpRequestStatusSend  = "send"
	TcpRequestStatusDone  = "done"
)

//tcp client
type TcpClient struct {
	Config   config.ConfigNet
	conn     net.Conn
	AddrType string //addr type,支持ip,etcd
	Addr     string //addr
	ReqData  []byte
	RspData  []byte
	Err      error
	Status   string //"init", "start", "send" "done"
}

func NewTcpClient(config config.ConfigNet) (*TcpClient, error) {
	cli := &TcpClient{
		Config:  config,
		ReqData: nil,
		RspData: nil,
		Err:     nil,
		Status:  TcpRequestStatusInit,
	}

	//init addr
	err := cli.initAddr()
	if err != nil {
		return nil, err
	}

	//get addr
	addr, err := cli.getAddr()
	if err != nil {
		return nil, err
	}

	//conn
	conn, err := net.DialTimeout("tcp", addr, time.Duration(3)*time.Second)
	if err != nil {
		return nil, err
	}
	cli.conn = conn

	return cli, nil
}

//提取配置文件addr，如果是etcd则需要启动etcd client
func (cli *TcpClient) initAddr() error {
	addr := cli.Config.Addr
	index := strings.IndexAny(addr, ":")
	if index == -1 {
		return errors.New("addr format error")
	}
	cli.AddrType = addr[0:index]
	if cli.AddrType != AddrTypeIp &&
		cli.AddrType != AddrTypeEtcd {
		return errors.New("addr not support")
	}
	cli.Addr = addr[index+1:]

	return nil
}

//获取地址
func (cli *TcpClient) getAddr() (string, error) {
	if cli.AddrType == AddrTypeIp {
		return cli.Addr, nil
	} else if cli.AddrType == AddrTypeEtcd {
		etcdCli, err := airetcd.GetEtcdClientByServerName(cli.Addr)
		if err != nil {
			log.Error("[NETCLIENT]: getUrl GetEtcdClientByServerName err, addr: %s, err: %+v", cli.Addr, err)
			return "", err
		}
		addrInfo, err := etcdCli.RandGetServerAddr()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s:%d", addrInfo.Ip, addrInfo.Port), nil
	}

	return "", errors.New("addr type not support")
}

//关闭连接
func (cli *TcpClient) Close() {
	_ = cli.conn.Close()
}

//设置请求数据，请求body数据为byte[]
func (cli *TcpClient) SetBytesReq(body []byte) error {
	cli.ReqData = body

	return nil
}

//设置请求数据, 请求body数据为pb
func (cli *TcpClient) SetPbReq(config config.ConfigNet, body proto.Message) error {
	data, err := proto.Marshal(body)
	if err != nil {
		return err
	}
	cli.ReqData = data

	return nil
}

//设置请求数据, 请求body数据为json
func (cli *TcpClient) SetJsonReq(config config.ConfigNet, body interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	cli.ReqData = data

	return nil
}

//发送一个请求
func (cli *TcpClient) Request(ctx context.Context) ([]byte, error) {
	timeout := time.Duration(cli.Config.TimeOutMs) * time.Millisecond
	cli.Status = TcpRequestStatusStart

	//write
	err := cli.Write(ctx, timeout)
	if err != nil {
		return nil, err
	}
	cli.Status = TcpRequestStatusSend

	//read
	rBuf, err := cli.Read(ctx, timeout)
	if err != nil {
		return nil, err
	}
	cli.Status = TcpRequestStatusDone

	return rBuf, nil
}

//并发发多个请求
func (cli *TcpClient) Requests(ctx context.Context, clis ...*TcpClient) error {
	if len(clis) == 1 {
		cli := clis[0]
		_, err := cli.Request(ctx)
		return err
	}

	pCtx, pCancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	timeOutMs := uint32(3000)
	for _, cli := range clis {
		if cli.Config.TimeOutMs > timeOutMs {
			timeOutMs = cli.Config.TimeOutMs
		}
		wg.Add(1)
		go func(w *sync.WaitGroup, c context.Context, client *TcpClient) {
			defer func() {
				if r := recover(); r != nil {
					log.PanicTrack()
				}
			}()

			_, _ = client.Request(c)
			w.Done()
		}(&wg, pCtx, cli)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		done <- struct{}{}
	}()

	var err error
	select {
	case <-done:
		err = nil
		break
	case <-time.After(time.Duration(timeOutMs) * time.Millisecond):
		err = errors.New("TcpRequests timeout")
	}
	pCancel()

	return err
}

//发一次包
func (cli *TcpClient) Write(ctx context.Context, timeout time.Duration) error {
	_ = cli.conn.SetReadDeadline(time.Now().Add(timeout))
	sNum, err := cli.conn.Write(cli.ReqData)
	if sNum != len(cli.ReqData) || err != nil {
		return errors.New("conn write error")
	}

	return nil
}

//收一次包，可用于用户主动定时收包 //todo 收包需要粘包
func (cli *TcpClient) Read(ctx context.Context, timeout time.Duration) ([]byte, error) {
	_ = cli.conn.SetReadDeadline(time.Now().Add(timeout))

	//read
	rBuf := make([]byte, TcpMaxRecvbuf)
	rNum, err := cli.conn.Read(rBuf)
	if err != nil {
		return nil, err
	}

	cli.RspData = rBuf[:rNum]

	return cli.ReqData, nil
}

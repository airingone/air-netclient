package air_netclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	airetcd "github.com/airingone/air-etcd"
	"github.com/airingone/config"
	"github.com/airingone/log"
	"github.com/golang/protobuf/proto"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	MaxRecvbuf = 1024 * 128 // 128k

	UdpRequestStatusInit  = "init"
	UdpRequestStatusStart = "start"
	UdpRequestStatusSend  = "send"
	UdpRequestStatusDone  = "done"
)

//udp client
type UdpClient struct {
	Config   config.ConfigNet
	AddrType string //addr type,支持ip,etcd
	Addr     string //addr
	ReqData  []byte
	RspData  []byte
	Err      error
	Status   string //"init", "doing", "done"
}

func newUdpClient(config config.ConfigNet) (*UdpClient, error) {
	cli := &UdpClient{
		Config: config,
		Status: UdpRequestStatusInit,
	}

	//初始化地址
	err := cli.initAddr()
	if err != nil {
		return nil, err
	}

	return cli, nil
}

//提取配置文件addr，如果是etcd则需要启动etcd client
func (cli *UdpClient) initAddr() error {
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
func (cli *UdpClient) getAddr() (string, error) {
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

//udp client, 请求body数据为byte[]
func NewBytesUdpClient(config config.ConfigNet, body []byte) (*UdpClient, error) {
	client, err := newUdpClient(config)
	if err != nil {
		return nil, err
	}
	client.ReqData = body

	return client, nil
}

//udp client, 请求body数据为pb
func NewPbUdpClient(config config.ConfigNet, body proto.Message) (*UdpClient, error) {
	client, err := newUdpClient(config)
	if err != nil {
		return nil, err
	}
	data, err := proto.Marshal(body)
	if err != nil {
		return nil, err
	}
	client.ReqData = data

	return client, nil
}

//udp client, 请求body数据为json
func NewJsonUdpClient(config config.ConfigNet, body interface{}) (*UdpClient, error) {
	client, err := newUdpClient(config)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	client.ReqData = data

	return client, nil
}

//发送请求
func (cli *UdpClient) Request(ctx context.Context) ([]byte, error) {
	cli.Status = UdpRequestStatusStart
	//获取地址
	addr, err := cli.getAddr()
	if err != nil {
		cli.Err = err
		return nil, err
	}

	//创建conn
	conn, err := net.DialTimeout("udp", addr, time.Duration(3)*time.Second)
	defer conn.Close()

	timeout := time.Duration((cli.Config.TimeOutMs+1)/2) * time.Millisecond
	_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	sendNum, err := conn.Write(cli.ReqData)
	if err != nil {
		cli.Err = err
		return nil, err
	}
	if sendNum == 0 {
		cli.Err = errors.New("sendNum is 0")
		return nil, err
	}
	cli.Status = UdpRequestStatusSend

	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	var recvBuf [MaxRecvbuf]byte
	recvNum, err := conn.Read(recvBuf[0:])
	if err != nil {
		return nil, err
	}
	cli.RspData = recvBuf[0:recvNum]
	cli.Status = UdpRequestStatusDone

	return cli.RspData, nil
}

func UdpRequests(ctx context.Context, clis ...*UdpClient) error {
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
		go func(w *sync.WaitGroup, c context.Context, client *UdpClient) {
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
		err = errors.New("UdpRequests timeout")
	}
	pCancel()

	return err
}

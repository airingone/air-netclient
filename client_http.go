package air_netclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	airetcd "github.com/airingone/air-etcd"
	"github.com/airingone/config"
	"github.com/airingone/log"
	"github.com/gogo/protobuf/proto"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	HttpAddrTypeIp   = "ip"
	HttpAddrTypeUrl  = "url"
	HttpAddrTypeEtcd = "etcd"

	HttpRequestStatusInit  = "init"
	HttpRequestStatusDoing = "doing"
	HttpRequestStatusDone  = "Done"
)

//http请求client
type HttpClient struct {
	Cli      *http.Client      //http client，这里外层函数可自由访问，获取或设置http请求参数等
	Config   config.ConfigHttp //配置文化
	AddrType string            //addr type,支持ip,url,etcd
	Addr     string            //addr
	Path     string            //path
	Cookie   string            //http cookie，AddCookie函数增加cookie
	Header   map[string]string //http head
	Req      *http.Request     //http request
	Rsp      *http.Response    //http response
	ReqBody  io.Reader         //请求body数据
	RspBody  []byte            //回包body数据
	Err      error
	Status   string //"init", "doing", "done"
}

//创建http client
func newHttpClient(configHttp config.ConfigHttp, path string) (*HttpClient, error) {
	client := &HttpClient{
		Config: configHttp,
		Path:   path,
		Status: HttpRequestStatusInit,
	}

	//初始化地址
	err := client.initAddr(configHttp.Addr)
	if err != nil {
		return nil, err
	}

	//创建http client
	client.Cli = &http.Client{
		Timeout: time.Duration(client.Config.TimeOutMs) * time.Millisecond,
	}

	var transport *http.Transport
	//http证书
	if client.Config.Scheme == "https" {
		cert, err := tls.LoadX509KeyPair(client.Config.CertFilePath, client.Config.KeyFilePath)
		if err != nil {
			return nil, err
		}
		certBytes, err := ioutil.ReadFile(client.Config.RootCaFilePath)
		if err != nil {
			return nil, err
		}
		rootCaPool := x509.NewCertPool()
		ok := rootCaPool.AppendCertsFromPEM(certBytes)
		if !ok {
			return nil, errors.New("AppendCertsFromPEM err")
		}

		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates:       []tls.Certificate{cert},
				RootCAs:            rootCaPool,
				InsecureSkipVerify: true,
			},
		}

		client.Cli.Transport = transport
	}

	//proxy
	if len(client.Config.Proxy) > 3 {
		proxy, err := url.Parse(client.Config.Proxy)
		if err != nil {
			return nil, err
		}
		if transport == nil {
			transport = &http.Transport{
				Proxy: http.ProxyURL(proxy),
			}
		} else {
			transport.Proxy = http.ProxyURL(proxy)
		}
	}

	return client, nil
}

//如果addr是etcd的话需要就行初始化client，这个初始化放在全局，且程序启动时一个http client初始化一次即可
func InitEtcdClient(addr string) {
	index := strings.IndexAny(addr, ":")
	if index == -1 {
		return
	}
	addrType := addr[0:index]
	serverName := addr[index+1:]
	if addrType == HttpAddrTypeEtcd {
		_, err := airetcd.NewEtcdClient(serverName, config.GetEtcdConfig("etcd").Addrs)
		if err != nil {
			log.Error("InitEtcdClient: NewEtcdClient err: %+v", err)
			return
		}
	}

}

//创建http client, 请求body数据为byte[]
func NewBytesHttpClient(configHttp config.ConfigHttp, path string, body []byte) (*HttpClient, error) {
	client, err := newHttpClient(configHttp, path)
	if err != nil {
		return nil, err
	}
	client.ReqBody = bytes.NewBuffer(body)

	return client, nil
}

//创建http client, 请求body数据为pb
func NewPbHttpClient(configHttp config.ConfigHttp, path string, body proto.Message) (*HttpClient, error) {
	client, err := newHttpClient(configHttp, path)
	if err != nil {
		return nil, err
	}
	client.Config.ContentType = "application/pb"
	data, err := proto.Marshal(body)
	if err != nil {
		return nil, err
	}

	client.ReqBody = bytes.NewBuffer(data)

	return client, nil
}

//创建http client, 请求body数据为json
func NewJsonHttpClient(configHttp config.ConfigHttp, path string, body interface{}) (*HttpClient, error) {
	if len(path) < 1 {
		return nil, errors.New("path err")
	}
	if path[0] != '/' {
		path = "/" + path
	}

	client, err := newHttpClient(configHttp, path)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	client.ReqBody = bytes.NewBuffer(data)

	return client, nil
}

//增加cookie
func (cli *HttpClient) AddCookie(key string, value string) {
	if len(cli.Cookie) == 0 {
		cli.Cookie = fmt.Sprintf("%s=%s", key, value)
	} else {
		cli.Cookie = fmt.Sprintf("%s;%s=%s", cli.Cookie, key, value)
	}
}

//增加header
func (cli *HttpClient) AddHeader(key string, value string) {
	if cli.Header == nil {
		cli.Header = make(map[string]string)
	}

	cli.Header[key] = value
}

//http请求
func (cli *HttpClient) Request(ctx context.Context) ([]byte, error) {
	cli.Status = HttpRequestStatusDoing
	url, err := cli.getUrl()
	if err != nil {
		return nil, err
	}
	log.Info("Request: url: %s", url)

	cli.Req, cli.Err = http.NewRequest(cli.Config.Method, url, cli.ReqBody)
	if cli.Err != nil {
		return nil, cli.Err
	}

	if len(cli.Config.Host) > 3 { //host
		cli.Req.Host = cli.Config.Host
	}
	_, _ = cli.Req.Cookie(cli.Cookie) //cookie
	for k, v := range cli.Header {    //header
		cli.Req.Header.Set(k, v)
	}

	cli.Rsp, cli.Err = cli.Cli.Do(cli.Req)
	if cli.Err != nil {
		return nil, cli.Err
	}
	defer cli.Rsp.Body.Close()

	cli.RspBody, cli.Err = ioutil.ReadAll(cli.Rsp.Body)
	if cli.Err != nil {
		return nil, cli.Err
	}
	cli.Status = HttpRequestStatusDone

	return cli.RspBody, nil
}

//提取配置文件addr，如果是etcd则需要启动etcd client
func (cli *HttpClient) initAddr(addr string) error {
	index := strings.IndexAny(addr, ":")
	if index == -1 {
		return errors.New("addr format error")
	}
	cli.AddrType = addr[0:index]
	if cli.AddrType != HttpAddrTypeIp && cli.AddrType != HttpAddrTypeUrl &&
		cli.AddrType != HttpAddrTypeEtcd {
		return errors.New("addr not support")
	}
	cli.Addr = addr[index+1:]

	return nil
}

//获取地址
func (cli *HttpClient) getUrl() (string, error) {
	if cli.AddrType == HttpAddrTypeIp {
		return fmt.Sprintf("%s://%s%s", cli.Config.Scheme, cli.Addr, cli.Path), nil
	} else if cli.AddrType == HttpAddrTypeUrl {
		return fmt.Sprintf("%s://%s%s", cli.Config.Scheme, cli.Addr, cli.Path), nil
	} else if cli.AddrType == HttpAddrTypeEtcd {
		etcdCli, err := airetcd.GetEtcdClientByServerName(cli.Addr)
		if err != nil {
			log.Error("getUrl: GetEtcdClientByServerName err, addr: %s, err: %+v", cli.Addr, err)
			return "", err
		}
		addrInfo, err := etcdCli.RandGetServerAddr()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s://%s:%d%s", cli.Config.Scheme, addrInfo.Ip, addrInfo.Port, cli.Path), nil
	}

	return "", errors.New("addr type not support")
}

//并发多个http请求，如果有超时情况则判断Status来判断那个请求已完成
func HttpRequests(ctx context.Context, clis ...*HttpClient) error {
	pCtx, pCancel := context.WithCancel(ctx)
	var wg sync.WaitGroup
	timeOutMs := uint32(3000)
	for _, cli := range clis {
		if cli.Config.TimeOutMs > timeOutMs {
			timeOutMs = cli.Config.TimeOutMs
		}
		wg.Add(1)
		go func(w *sync.WaitGroup, c context.Context, client *HttpClient) {
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
		err = errors.New("HttpRequests timeout")
	}
	pCancel()

	return err
}

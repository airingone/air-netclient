package air_netclient

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/airingone/config"
	"github.com/gogo/protobuf/proto"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

//http请求client
type HttpClient struct {
	Cli     *http.Client      //http client，这里外层函数可自由访问，获取或设置http请求参数等
	Config  config.ConfigHttp //配置文化
	Path    string            //path
	Cookie  string            //http cookie，AddCookie函数增加cookie
	Header  map[string]string //http head
	Req     *http.Request     //http request
	Rsp     *http.Response    //http response
	ReqBody io.Reader         //请求body数据
	RspBody []byte            //回包body数据
	Err     error
}

//创建http client
func newHttpClient(configHttp config.ConfigHttp, path string) (*HttpClient, error) {
	client := &HttpClient{
		Config: configHttp,
		Path:   path,
	}

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
	url := fmt.Sprintf("%s://%s%s", cli.Config.Scheme, cli.Config.Addr, cli.Path)

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

	return cli.RspBody, nil
}

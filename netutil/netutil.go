package netutil

import (
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"
)

const (
	postMethod = "POST"
	getMethod  = "GET"
)

var client = http.Client{
	Timeout: time.Second * 5,
}

func PostReq(url string, body []byte, header map[string]string) []byte {
	return execReq(postMethod, url, header, body)
}

func GetReq(url string, header map[string]string) []byte {
	return execReq(getMethod, url, header, nil)
}

func execReq(method string, url string, header map[string]string, msg []byte) []byte {
	// 新建请求
	var body io.Reader = nil
	if msg != nil {
		body = bytes.NewReader(msg)
	}
	req, err := http.NewRequest(method, url, body)
	// 添加头信息
	for k, v := range header {
		req.Header.Add(k, v)
	}
	if err != nil {
		log.Fatalf("post new request error. %v", err)
	}
	// 执行请求
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("post request error. %v", err)
	}
	// 解析结果返回
	ret, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("post response read error. %v", err)
	}
	return ret
}

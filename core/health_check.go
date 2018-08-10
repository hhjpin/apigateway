package core

import (
	"github.com/valyala/fasthttp"
	"bytes"
	"api_gateway/utils/errors"
	"log"
	"fmt"
)

type HealthCheck struct {
	path []byte

	// timeout default is 5 sec. When timed out, will retry to check again based on 'retry' switch on or not
	timeout   uint8
	interval  uint8
	retry     bool
	retryTime uint8
}

var (
	GetMethod = []byte("GET")
)

func (h *HealthCheck) Check(host []byte, port uint8) (bool, error) {
	if h.path == nil {
		return false, errors.New(60)
	}
	revReq := fasthttp.AcquireRequest()
	revReqUri := fasthttp.AcquireURI()
	revRes := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(revReq)
	defer fasthttp.ReleaseResponse(revRes)
	defer fasthttp.ReleaseURI(revReqUri)

	tmpHost := bytes.Join([][]byte{host, []byte(string(port))}, []byte(":"))
	revReqUri.SetHostBytes(tmpHost)
	revReqUri.SetPathBytes(h.path)
	revReqUri.SetScheme("http")

	revReq.SetRequestURIBytes(revReqUri.FullURI())
	revReq.Header.SetMethodBytes(GetMethod)

	err := fasthttp.Do(revReq, revRes)
	if err != nil {
		log.Print(err)
		return false, errors.New(61)
	}
	if statusCode := revRes.StatusCode(); statusCode >= 200 && statusCode < 400 {
		return true, nil
	} else {
		return false, errors.New(61, errors.CustomErrMsg{
			ErrMsg: fmt.Sprintf("%s 健康检查失败, 返回状态码 [%d]", string(tmpHost),statusCode),
			ErrMsgEn: fmt.Sprintf("%s health check failed, status code [%d]", string(tmpHost), statusCode),
		})
	}
}

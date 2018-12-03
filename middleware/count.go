package middleware

import (
	"bytes"
	"encoding/json"
	"git.henghajiang.com/backend/golang_utils/errors"
	"os"
	"strconv"
)

type counting struct {
	ID                  []byte            `json:"id"`
	OriginalPath        []byte            `json:"original_path"`
	Method              []byte            `json:"method"`
	RedirectService     []byte            `json:"redirect_service"`
	RedirectHost        []byte            `json:"redirect_host"`
	RedirectPath        []byte            `json:"redirect_path"`
	RequestTime         int64             `json:"request_time"`
	ResponseTime        int64             `json:"response_time"`
	RequestContentType  []byte            `json:"request_content_type"`
	ResponseContentType []byte            `json:"response_content_type"`
	RequestHeader       map[string]string `json:"request_header"`
	UrlParams           map[string]string `json:"request_params"`
	RequestBody         []byte            `json:"request_body"`
	ResponseBody        []byte            `json:"response_body"`
	ResponseStatusCode  int               `json:"response_status_code"`
	ErrorRaised         bool              `json:"response_error"`
	ResponseError       errors.Error      `json:"response_error"`
}

const (
	CountingShardNumber = 4
)

var (
	CountingCh []chan *counting

	hostName []byte
	applicationJsonType = []byte("application/json")
)

func init() {
	tmp, err := os.Hostname()
	if err != nil {
		logger.Exception(err)
		hostName = []byte("unknown-host")
	} else {
		hostName = []byte(tmp)
	}

	for shard := 0; shard < CountingShardNumber; shard++ {
		CountingCh = append(CountingCh, make(chan *counting, 500))
	}

	//for _, c := range CountingCh {
	//	go persistence(c)
	//}

}

//func (c *counting) String() string {
//
//}

func NewCounting(reqTime, respTime int64,
	oriPath, method, svr, host, path, reqContentType, respContentType []byte,
	reqHeader, urlParam map[string]string, status int, reqBody, respBody []byte) *counting {
	var resp struct {
		ErrCode  int    `json:"err_code"`
		ErrMsg   string `json:"err_msg"`
		ErrMsgEn string `json:"err_msg_en"`
	}
	var c counting

	tmp := []byte(strconv.FormatInt(reqTime, 10))
	tmpHost := hostName
	tmpHost = append(tmpHost, tmp...)
	c.ID = tmpHost
	c.OriginalPath = oriPath
	c.Method = method
	c.RedirectService = svr
	c.RedirectHost = host
	c.RedirectPath = path
	c.RequestTime = reqTime
	c.ResponseTime = respTime
	c.RequestContentType = reqContentType
	c.ResponseContentType = respContentType
	c.RequestHeader = reqHeader
	c.UrlParams = urlParam
	c.ResponseStatusCode = status
	if bytes.Equal(c.RequestContentType, applicationJsonType) {
		c.RequestBody = reqBody
	}
	if bytes.Equal(c.ResponseBody, applicationJsonType) {
		c.ResponseBody = respBody
	}

	if err := json.Unmarshal(c.ResponseBody, &resp); err == nil {
		if resp.ErrCode != 0 {
			c.ErrorRaised = true
			c.ResponseError = errors.New(errors.ErrCode(resp.ErrCode))
			c.ResponseError.ErrMsg = resp.ErrMsg
			c.ResponseError.ErrMsgEn = resp.ErrMsgEn
		}
	}
	return &c
}

func persistence(c []*counting) {



}

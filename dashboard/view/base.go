package view

import (
	"git.henghajiang.com/backend/golang_utils/errors"
	"git.henghajiang.com/backend/golang_utils/log"
	"os"
	"strconv"
)

type BaseResponse struct {
	errors.Error
	Data interface{} `json:"data"`
}

var (
	logger = log.New()
)

func init() {
	serverDebug := os.Getenv("SERVER_DEBUG")
	logger.DisableDebug()
	if serverDebug != "" {
		tmp, err := strconv.ParseInt(serverDebug, 10, 64)
		if err != nil {
			logger.Exception(err)
		}
		if tmp > 0 {
			logger.EnableDebug()
		} else {
			logger.DisableDebug()
		}
	} else {
		logger.EnableDebug()
	}
}

func (b *BaseResponse) Init(errCode errors.ErrCode, d ...interface{}) {

	err := errors.New(errCode)
	b.ErrCode = errCode
	b.ErrMsg = err.ErrMsg
	b.ErrMsgEn = err.ErrMsgEn

	if d == nil {
		b.Data = map[string]string{}
	} else {
		if len(d) > 1 {
			b.Data = d
		} else if len(d) == 1 {
			b.Data = d[0]
		} else {
			b.Data = map[string]string{}
		}
	}
}

func (b *BaseResponse) InitError(err errors.Error, d ...interface{}) {

	b.ErrCode = err.ErrCode
	b.ErrMsg = err.ErrMsg
	b.ErrMsgEn = err.ErrMsgEn

	if d == nil {
		b.Data = map[string]string{}
	} else {
		if len(d) > 1 {
			b.Data = d
		} else if len(d) == 1 {
			b.Data = d[0]
		} else {
			b.Data = map[string]string{}
		}
	}
}

package routing

import (
	"bytes"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/core/constant"
	"git.henghajiang.com/backend/api_gateway_v2/core/utils"
	"git.henghajiang.com/backend/golang_utils/errors"
	"github.com/valyala/fasthttp"
	"strconv"
	"time"
)

type HealthCheck struct {
	id   string
	path []byte

	// timeout default is 5 sec. When timed out, will retry to check again based on 'retry' switch on or not
	timeout   uint8
	interval  uint8
	retry     bool
	retryTime uint8
}

func (h *HealthCheck) Check(host []byte, port int) (bool, error) {
	if h.path == nil {
		return false, errors.New(160)
	}
	revReq := fasthttp.AcquireRequest()
	revReqUri := fasthttp.AcquireURI()
	revRes := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(revReq)
	defer fasthttp.ReleaseResponse(revRes)
	defer fasthttp.ReleaseURI(revReqUri)

	tmpHost := bytes.Join([][]byte{host, []byte(strconv.FormatInt(int64(port), 10))}, []byte(":"))
	revReqUri.SetHostBytes(tmpHost)
	revReqUri.SetPathBytes(h.path)
	revReqUri.SetScheme("http")

	revReq.SetRequestURIBytes(revReqUri.FullURI())
	revReq.Header.SetMethodBytes(constant.StrGet)
	logger.Debugf("健康检查请求: %s", string(revReqUri.FullURI()))
	err := fasthttp.Do(revReq, revRes)
	if err != nil {
		return false, errors.NewFormat(162, err.Error())
	}
	if statusCode := revRes.StatusCode(); statusCode >= 200 && statusCode < 400 {
		return true, nil
	} else {
		return false, errors.NewFormat(162, fmt.Sprintf("服务Host %s, 返回状态码 [%d]", string(tmpHost), statusCode))
	}
}

func (r *Table) HealthCheck() {
	defer func() {
		logger.Debugf("health check quit")
		if err := recover(); err != nil {
			stack := utils.Stack(3)
			logger.Errorf("[Recovery] %s panic recovered:\n%s\n%s", utils.TimeFormat(time.Now()), err, stack)
		}
		go r.HealthCheck()
	}()
	for {
		logger.Debugf("endpoint table: %+v", r.endpointTable.internal)
		r.endpointTable.Range(func(key EndpointNameString, value *Endpoint) bool {
			var status Status
			if value.status == Offline {
				logger.Warningf("endpoints [%s] offline, skip health-check", value.nameString)
				return false
			}
			resp, err := utils.GetKV(r.cli, value.key(constant.StatusKeyString))
			if err != nil {
				logger.Exception(err)
				status = BreakDown
			} else {
				for _, kv := range resp.Kvs {
					if bytes.Equal(kv.Key, []byte(value.key(constant.StatusKeyString))) {
						statusInt, err := strconv.ParseInt(string(kv.Value), 10, 64)
						if err != nil {
							logger.Exception(err)
							status = BreakDown
						} else {
							status = Status(statusInt)
						}
					} else {
						status = BreakDown
					}
				}
			}
			logger.Debugf("status: %d", status)
			if check, err := value.healthCheck.Check(value.host, value.port); err == nil {
				if check {
					if status != Online {
						_ = r.SetEndpointStatus(value, Online)
					}
				} else {
					if status != Offline {
						_ = r.SetEndpointStatus(value, BreakDown)
					}
				}
			} else {
				logger.Error(err.(errors.Error).String())
				_ = r.SetEndpointStatus(value, BreakDown)
			}
			return false
		})
		r.routerTable.Range(func(key RouterNameString, value *Router) {
			confirm, rest := value.service.checkEndpointStatus(Online)
			if len(confirm) > 0 {
				if value.status != Online {
					_, _ = r.SetRouterOnline(value)
				}
				if err := value.service.ResetOnlineEndpointRing(confirm); err != nil {
					logger.Error(err.(errors.Error).String())
				}
			} else {
				for _, i := range rest {
					if i.status == BreakDown && value.status != BreakDown {
						_, _ = r.SetRouterStatus(value, BreakDown)
						return
					}
				}
				if value.status != Offline {
					_, _ = r.SetRouterStatus(value, Offline)
				}
			}
		})

		time.Sleep(time.Second * 10)
	}
}

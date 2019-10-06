package hander

import (
	"git.henghajiang.com/backend/api_gateway_v2/core/routing"
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/gin-gonic/gin"
)

var (
	logger = log.Logger
)

func GetRouteTable(c *gin.Context) *routing.Table {
	v, ok := c.Get("table")
	if !ok || v == nil {
		return nil
	}
	return v.(*routing.Table)
}

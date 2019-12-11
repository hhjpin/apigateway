package hander

import (
	"git.henghajiang.com/backend/api_gateway_v2/core/routing"
	"github.com/gin-gonic/gin"
)

func GetRouteTable(c *gin.Context) *routing.Table {
	v, ok := c.Get("table")
	if !ok || v == nil {
		return nil
	}
	return v.(*routing.Table)
}

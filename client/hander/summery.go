package hander

import (
	"git.henghajiang.com/backend/api_gateway_v2/client/model"
	"github.com/gin-gonic/gin"
	"github.com/hhjpin/goutils/response"
	"net/http"
)

func Summery(c *gin.Context) {
	var resp response.BaseResponse

	mdl := model.SummeryModel{}
	mdl.Table = GetRouteTable(c)
	res, err := mdl.GetSummery()
	if err != nil {
		resp.InitError(err)
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Init(0, res)
	c.JSON(http.StatusOK, resp)
	return
}

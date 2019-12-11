package hander

import (
	"git.henghajiang.com/backend/api_gateway_v2/client/model"
	"github.com/gin-gonic/gin"
	"github.com/hhjpin/goutils/errors"
	"github.com/hhjpin/goutils/response"
	"net/http"
)

func RegisterClient(c *gin.Context) {
	var req model.ClientRegisterReq
	var resp response.BaseResponse
	if err := c.ShouldBindJSON(&req); err != nil {
		resp.InitError(errors.NewFormat(15, err))
		c.JSON(http.StatusOK, resp)
		return
	}

	var mdl model.ClientModel
	mdl.Cl = GetRouteTable(c).GetEtcdClient()
	err := mdl.Register(&req)
	if err != nil {
		resp.InitError(err)
		c.JSON(http.StatusOK, resp)
		return
	}

	resp.Init(0)
	c.JSON(http.StatusOK, resp)
	return
}

package client

import (
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/client/hander"
	"git.henghajiang.com/backend/api_gateway_v2/conf"
	"git.henghajiang.com/backend/api_gateway_v2/core/routing"
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

var (
	logger = log.Logger
)

func Run(table *routing.Table) {
	cf := conf.Conf.DashBoard
	if !cf.Enable {
		logger.Info("dashboard is disabled, not running")
		return
	}

	gin.SetMode(cf.RequestModel)
	pre := cf.RoutePrefix

	r := gin.New()
	r.Use(Recovery(), LoggerWithWriter(os.Stdout), CrossDomain(), Auth(cf.Token), Table(table))
	r.OPTIONS(pre+"/api/v1/gw/*any", func(c *gin.Context) {
		c.String(http.StatusOK, "")
	})
	r.GET(pre+"/api/v1/gw/summery", hander.Summery)
	r.POST(pre+"/api/v1/gw/client/register", hander.RegisterClient)

	if err := r.Run(fmt.Sprintf("%s:%d", cf.ListenHost, cf.ListenPort)); err != nil {
		logger.Exception(err)
		os.Exit(-1)
	}
}

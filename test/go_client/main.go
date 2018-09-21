package main

import (
	"api_gateway/sdk/golang"
	"flag"
	"fmt"
	"github.com/coreos/etcd/clientv3"
	"github.com/gin-gonic/gin"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"runtime"
	"time"
)

var (
	flagPort = flag.Int("port", 7788, "server listening port")
)

func init() {

	node := golang.NewNode("localhost", "127.0.0.1", 7788, golang.NewHealthCheck(
		"/check",
		10,
		5,
		3,
		true,
	))
	svr := golang.NewService("test", node)
	gw := golang.NewApiGatewayRegistrant(
		ConnectToEtcd(),
		node,
		svr,
		[]*golang.Router{
			golang.NewRouter("test1", "/front/$1", "/test/$1", svr),
			golang.NewRouter("test2", "/api/v1/test", "/test", svr),
			golang.NewRouter("test3", "/rd", "/redirect", svr),
		},
	)
	gw.Register()
}

func ConnectToEtcd() *clientv3.Client {
	cli, _ := clientv3.New(
		clientv3.Config{
			Endpoints:            []string{"127.0.0.1:2379"},
			AutoSyncInterval:     time.Duration(0) * time.Second,
			DialTimeout:          time.Duration(3) * time.Second,
			DialKeepAliveTime:    time.Duration(30) * time.Second,
			DialKeepAliveTimeout: time.Duration(5) * time.Second,
			Username:             "",
			Password:             "",
		},
	)
	return cli
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	r := gin.New()
	r.Use(gin.Logger())

	r.POST("/test/:test_id", func(c *gin.Context) {
		param := c.Param("test_id")
		c.JSON(200, map[string]string{"result": "success", "test_id": param})
	})
	r.GET("/check", func(c *gin.Context) {
		c.JSON(200, map[string]string{"result": "success"})
	})
	r.POST("/redirect", func(c *gin.Context) {
		c.Redirect(308, "http://127.0.0.1/test/redirect")
	})

	r.Run(fmt.Sprintf(":%d", *flagPort))
}

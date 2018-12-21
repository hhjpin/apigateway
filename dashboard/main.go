package main

import (
	"flag"
	"fmt"
	"git.henghajiang.com/backend/golang_utils/log"
	"github.com/coreos/etcd/clientv3"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"runtime"
	"strconv"
)

var (
	etcdClient *clientv3.Client
	logger = log.New()
	flagPort    = flag.Int("port", 80, "server listening port")
)

func init() {
	etcdClient = ConnectToEtcd()
}

func main() {
	var port int
	var envPort string
	var serverDebug string
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	serverDebug = os.Getenv("SERVER_DEBUG")
	if serverDebug != "" {
		tmp, err := strconv.ParseInt(serverDebug, 10, 64)
		if err != nil {
			logger.Exception(err)
			os.Exit(-1)
		}
		if tmp > 0 {
			gin.SetMode(gin.DebugMode)
		} else {
			gin.SetMode(gin.ReleaseMode)
		}
	} else {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.New()
	r.Use(Recovery(), LoggerWithWriter(os.Stdout), CrossDomain(), DB())

	r.OPTIONS("/*any", func(c *gin.Context) {
		c.String(http.StatusOK, "")
	})

	envPort = os.Getenv("PORT")
	if envPort == "" {
		port = *flagPort
	} else {
		tmp, err := strconv.ParseInt(envPort, 10, 64)
		if err != nil {
			logger.Exception(err)
			os.Exit(-1)
		}
		port = int(tmp)
	}
	if err := r.Run(fmt.Sprintf(":%d", port)); err != nil {
		logger.Exception(err)
		os.Exit(-1)
	}
}


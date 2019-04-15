package routing

import (
	"git.henghajiang.com/backend/golang_utils/log"
	"os"
	"strconv"
)

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

package middleware

import (
	"fmt"
	"github.com/hhjpin/goutils/logger"
	"net/http"
	"os"
	"time"
)

var (
	green     = "\033[0;32m"
	white     = "\033[1;37m"
	yellow    = "\033[1;33m"
	red       = "\033[0;31m"
	blue      = "\033[0;34m"
	magenta   = "\033[1;31m"
	cyan      = "\033[0;36m"
	lightCyan = "\033[1;36m"

	reset = "\033[0m"
)

func colorForStatus(code int) string {
	switch {
	case code >= http.StatusOK && code < http.StatusMultipleChoices:
		return green
	case code >= http.StatusMultipleChoices && code < http.StatusBadRequest:
		return white
	case code >= http.StatusBadRequest && code < http.StatusInternalServerError:
		return yellow
	default:
		return red
	}
}

func colorForMethod(method string) string {
	switch method {
	case "GET":
		return blue
	case "POST":
		return cyan
	case "PUT":
		return yellow
	case "DELETE":
		return red
	case "PATCH":
		return green
	case "HEAD":
		return magenta
	case "OPTIONS":
		return white
	default:
		return reset
	}
}

func Logger(code int, path, method, clientIP string, start time.Time) {

	end := time.Now()
	latency := end.Sub(start)

	var statusColor, methodColor string
	statusColor = colorForStatus(code)
	methodColor = colorForMethod(method)

	_, err := fmt.Fprintf(os.Stdout, "\033[0;32m[GW]\033[0m    %v |%s %3d \033[0m| \033[1;32m%13v\033[0m | %15s |%s %-7s \033[0m %s %s \033[0m \n",
		end.Format("2006/01/02 15:04:05"),
		statusColor, code,
		latency,
		clientIP,
		methodColor, method,
		lightCyan, path,
	)
	if err != nil {
		logger.Error(err)
	}
}

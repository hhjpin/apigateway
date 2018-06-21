package utils

import (
	"github.com/valyala/fasthttp"
	"fmt"
	"io"
	"time"
	"os"
)

type CtxLogger struct {
	startTime time.Time
}

var (
	green   = string([]byte{27, 91, 57, 55, 59, 52, 50, 109})
	white   = string([]byte{27, 91, 57, 48, 59, 52, 55, 109})
	yellow  = string([]byte{27, 91, 57, 55, 59, 52, 51, 109})
	red     = string([]byte{27, 91, 57, 55, 59, 52, 49, 109})
	blue    = string([]byte{27, 91, 57, 55, 59, 52, 52, 109})
	magenta = string([]byte{27, 91, 57, 55, 59, 52, 53, 109})
	cyan    = string([]byte{27, 91, 57, 55, 59, 52, 54, 109})
	reset   = string([]byte{27, 91, 48, 109})
)

func colorForStatus(code int) string {
	switch {
	case code >= 200 && code < 300:
		return green
	case code >= 300 && code < 400:
		return white
	case code >= 400 && code < 500:
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

func (c *CtxLogger) Log() {
	c.startTime = time.Now()
}

func (c *CtxLogger) End(ctx *fasthttp.RequestCtx, out ...io.Writer) {

	statusCode := ctx.Response.StatusCode()
	statusColor := colorForStatus(statusCode)
	latency := time.Now().Sub(c.startTime)
	clientIP := ctx.RemoteIP()
	method := string(ctx.Method())
	methodColor := colorForMethod(method)
	path := string(ctx.Path())

	if len(out) == 0 {
		out = append(out, os.Stdout)
	}

	for _, i := range out {
		fmt.Fprintf(i, "[API-GATEWAY] %v |%s %3d %s| %13v | %15s |%s %-7s %s %s",
			ctx.Time().Format("2006/01/02 15:04:05"),
			statusColor, statusCode, reset,
			latency,
			clientIP,
			methodColor, method, reset,
			path,
		)
	}
}

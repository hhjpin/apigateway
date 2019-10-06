package client

import (
	"bytes"
	"fmt"
	"git.henghajiang.com/backend/api_gateway_v2/core/routing"
	"git.henghajiang.com/backend/golang_utils"
	"git.henghajiang.com/backend/golang_utils/response"
	"github.com/gin-gonic/gin"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"runtime"
	"strings"
	"time"
)

var (
	dunno     = []byte("???")
	centerDot = []byte("·")
	dot       = []byte(".")
	slash     = []byte("/")

	green     = "\033[0;32m"
	white     = "\033[1;37m"
	yellow    = "\033[1;33m"
	red       = "\033[0;31m"
	blue      = "\033[0;34m"
	magenta   = "\033[1;31m"
	cyan      = "\033[0;36m"
	lightCyan = "\033[1;36m"
)

func timeFormat(t time.Time) string {
	var timeString = t.Format("2006/01/02 - 15:04:05")
	return timeString
}

func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	if lastSlash := bytes.LastIndex(name, slash); lastSlash >= 0 {
		name = name[lastSlash+1:]
	}
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}

func source(lines [][]byte, n int) []byte {
	n--
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}

func stack(skip int) []byte {
	buf := new(bytes.Buffer)

	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		if _, err := fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc); err != nil {
			logger.Exception(err)
		}
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		if _, err := fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line)); err != nil {
			logger.Exception(err)
		}
	}
	return buf.Bytes()
}

func CrossDomain() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "*")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
	}
}

func Auth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token == "" {
			c.Next()
			return
		}
		auth := c.GetHeader("Authorization")
		authArr := strings.SplitN(auth, " ", 2)
		if len(authArr) < 2 || authArr[1] != token {
			var resp response.BaseResponse
			resp.Init(3)
			c.JSON(http.StatusOK, resp)
			c.Abort()
		}
		c.Next()
	}
}

func Recovery() gin.HandlerFunc {
	return RecoveryWithWriter()
}

func RecoveryWithWriter() gin.HandlerFunc {

	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				if logger != nil {
					stack := stack(3)
					httpRequest, _ := httputil.DumpRequest(c.Request, false)
					go golang_utils.ErrMail("api gateway err mail", fmt.Sprintf("[Recovery] %s panic recovered:\n%s\n%s\n%s", timeFormat(time.Now()), string(httpRequest), err, stack))
					logger.Errorf("[Recovery] %s panic recovered:\n%s\n%s\n%s", timeFormat(time.Now()), string(httpRequest), err, stack)
				}
				c.String(200, `{"err_code": 5,"err_msg":"当前访问量过大, 请稍后再试","err_msg_en":"service busy, please try again","data": {}}`)
			}
		}()
		c.Next()
	}
}

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
		return "\033[0m"
	}
}

func Table(table *routing.Table) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("table", table)
		c.Next()
	}
}

func LoggerWithWriter(out io.Writer, notLogged ...string) gin.HandlerFunc {

	var skip map[string]struct{}

	if length := len(notLogged); length > 0 {
		skip = make(map[string]struct{}, length)

		for _, path := range notLogged {
			skip[path] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Log only when path is not being skipped
		if _, ok := skip[path]; !ok {
			// Stop timer
			end := time.Now()
			latency := end.Sub(start)

			clientIP := c.ClientIP()
			method := c.Request.Method
			statusCode := c.Writer.Status()
			var statusColor, methodColor, userId string
			statusColor = colorForStatus(statusCode)
			methodColor = colorForMethod(method)
			userId = c.GetHeader("userid")
			if userId == "" {
				userId = "0"
			}
			userAgent := c.GetHeader("User-Agent")
			osType := c.GetHeader("Os")
			if osType == "" {
				osType = userAgent
			}

			if raw != "" {
				path = path + "?" + raw
			}
			_, err := fmt.Fprintf(out, "\033[0;32m[GIN]\033[0m    %v |%s %3d \033[0m| \033[1;32m%13v\033[0m | %15s |%s %-7s \033[0m %s %s \033[0m \n",
				end.Format("2006/01/02 15:04:05"),
				statusColor, statusCode,
				latency,
				clientIP,
				methodColor, method,
				lightCyan, path,
			)
			if err != nil {
				logger.Exception(err)
			}
		}
	}
}

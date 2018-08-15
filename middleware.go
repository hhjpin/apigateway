package main

import (
	"os"
	"io"
	"fmt"
	"log"
	"time"
	"bytes"
	"strings"
	"runtime"
	"io/ioutil"
	"net/http/httputil"
	"github.com/gin-gonic/gin"
	"github.com/mattn/go-isatty"
	"github.com/dgrijalva/jwt-go"
	"dropshipping_product_detail/db/hhj"
)

type CustomClaims struct {
	UserID int `json:"user_id"`
	jwt.StandardClaims
}

func timeFormat(t time.Time) string {
	var timeString = t.Format("2006/01/02 - 15:04:05")
	return timeString
}

// function returns, if possible, the name of the function containing the PC.
func function(pc uintptr) []byte {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return dunno
	}
	name := []byte(fn.Name())
	// The name includes the path name to the package, which is unnecessary
	// since the file name is already included.  Plus, it has center dots.
	// That is, we see
	//	runtime/debug.*T·ptrmethod
	// and want
	//	*T.ptrmethod
	// Also the package path might contains dot (e.g. code.google.com/...),
	// so first eliminate the path prefix
	if lastSlash := bytes.LastIndex(name, slash); lastSlash >= 0 {
		name = name[lastSlash+1:]
	}
	if period := bytes.Index(name, dot); period >= 0 {
		name = name[period+1:]
	}
	name = bytes.Replace(name, centerDot, dot, -1)
	return name
}

// source returns a space-trimmed slice of the n'th line.
func source(lines [][]byte, n int) []byte {
	n-- // in stack trace, lines are 1-indexed but our array is 0-indexed
	if n < 0 || n >= len(lines) {
		return dunno
	}
	return bytes.TrimSpace(lines[n])
}

// stack returns a nicely formatted stack frame, skipping skip frames.
func stack(skip int) []byte {
	buf := new(bytes.Buffer) // the returned data
	// As we loop, we open files and read them. These variables record the currently
	// loaded file.
	var lines [][]byte
	var lastFile string
	for i := skip; ; i++ { // Skip the expected number of frames
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// Print this much at least.  If we can't find the source, it won't show.
		fmt.Fprintf(buf, "%s:%d (0x%x)\n", file, line, pc)
		if file != lastFile {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				continue
			}
			lines = bytes.Split(data, []byte{'\n'})
			lastFile = file
		}
		fmt.Fprintf(buf, "\t%s: %s\n", function(pc), source(lines, line))
	}
	return buf.Bytes()
}

func Recovery() gin.HandlerFunc {
	return RecoveryWithWriter(gin.DefaultErrorWriter)
}

// RecoveryWithWriter returns a middleware for a given writer that recovers from any panics and writes a 500 if there was one.
func RecoveryWithWriter(out io.Writer) gin.HandlerFunc {
	var logger *log.Logger
	if out != nil {
		logger = log.New(out, "\n\n\x1b[31m", log.LstdFlags)
	}
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				if logger != nil {
					stack := stack(3)
					httpRequest, _ := httputil.DumpRequest(c.Request, false)
					go CollectPanic(fmt.Sprintf("[Recovery] %s panic recovered:\n%s\n%s\n%s%s", timeFormat(time.Now()), string(httpRequest), err, stack, reset))
					logger.Printf("[Recovery] %s panic recovered:\n%s\n%s\n%s%s", timeFormat(time.Now()), string(httpRequest), err, stack, reset)
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
		return reset
	}
}



func LoggerWithWriter(out io.Writer, notLogged ...string) gin.HandlerFunc {
	isTerm := true

	if w, ok := out.(*os.File); !ok ||
		(os.Getenv("TERM") == "dumb" || (!isatty.IsTerminal(w.Fd()) && !isatty.IsCygwinTerminal(w.Fd()))) {
		isTerm = false
	}

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
			var statusColor, methodColor, resetColor, userId string
			if isTerm {
				statusColor = colorForStatus(statusCode)
				methodColor = colorForMethod(method)
				resetColor = reset
			}
			comment := c.Errors.ByType(gin.ErrorTypePrivate).String()
			userId = c.GetHeader("userid")
			if userId == "" {
				userId = "0"
			}
			userAgent := c.GetHeader("User-Agent")
			osType := c.GetHeader("Os")
			if osType == "" {
				osType = userAgent
			}
			clientVersion := c.GetHeader("clientversion")
			deviceType := c.GetHeader("Device")

			if raw != "" {
				path = path + "?" + raw
			}
			if clientVersion != "" && deviceType != "" {
				fmt.Fprintf(out, "[GIN] %v |%s %3d %s| %13v | %15s | %5s | %s | %s | %s |%s %-7s %s %s\n%s",
					end.Format("2006/01/02 15:04:05"),
					statusColor, statusCode, resetColor,
					latency,
					clientIP,
					userId,
					osType,
					clientVersion,
					deviceType,
					methodColor, method, resetColor,
					path,
					comment,
				)
			} else {
				fmt.Fprintf(out, "[GIN] %v |%s %3d %s| %13v | %15s | %5s | %s | %s %-7s %s %s\n%s",
					end.Format("2006/01/02 15:04:05"),
					statusColor, statusCode, resetColor,
					latency,
					clientIP,
					userId,
					osType,
					methodColor, method, resetColor,
					path,
					comment,
				)
			}
		}
	}
}

func Trace() gin.HandlerFunc {
	return func(c *gin.Context) {
		var logTemplate = `
Method: %s
Host: %s
Path: %s
RemoteIP: %s
Connection: %s
User-Agent: %s
Authorization: %s
UserId: %s
ClientVersion: %s
Os: %s
Device: %s

`

		traceLogger.Info(fmt.Sprintf(logTemplate,
			c.Request.Method, c.Request.Host, c.Request.URL.Path, c.Request.RemoteAddr,
			c.GetHeader("Connection"), c.GetHeader("User-Agent"), c.GetHeader("Authorization"),
			c.GetHeader("Userid"), c.GetHeader("Clientversion"), c.GetHeader("Os"), c.GetHeader("Device")))
	}
}

func Verify() gin.HandlerFunc {
	return func(c *gin.Context) {
		var token string
		var tokens hhj.Tokens

		c.Set("hhj_conn", hhjConn)
		c.Set("drop_shipping_conn", dsConn)
		c.Set("is_logined", false)
		c.Set("user_id", 0)
		c.Set("logger", &logger)

		authorization := c.GetHeader("Authorization")
		if authorization != "" {
			tokenList := strings.Split(authorization, " ")
			if len(tokenList) == 2 {
				token = tokenList[1]
			}
		}

		if token != "" {
			t, err := jwt.ParseWithClaims(token, CustomClaims{}, func(token *jwt.Token) (interface{}, error) {
				return []byte(tokenSignature), nil
			})
			if err != nil {
				logger.Error(err)
				logger.Info("invalid token: %s", token)
			} else {
				hhjConn.Where(&hhj.Tokens{Token: token}).First(&tokens)
				if claims, ok := t.Claims.(CustomClaims); ok && t.Valid {
					if claims.UserID == tokens.UserID {
						// authorization passed
						c.Set("is_logined", true)
						c.Set("user_id", claims.UserID)
					}
				}
			}
		}
	}
}

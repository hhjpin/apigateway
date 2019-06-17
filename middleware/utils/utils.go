package utils

import (
	"bytes"
	"fmt"
	"git.henghajiang.com/backend/golang_utils/log"
	"io/ioutil"
	"os"
	"runtime"
	"strconv"
	"time"
)

var (
	logger = log.Logger

	dunno     = []byte("???")
	centerDot = []byte("·")
	dot       = []byte(".")
	slash     = []byte("/")
)

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

func TimeFormat(t time.Time) string {
	var timeString = t.Format("2006/01/02 - 15:04:05")
	return timeString
}

func Stack(skip int) []byte {
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

package contrib

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"qiniupkg.com/x/xlog.v7"
)

var pid = uint32(time.Now().UnixNano() % 4294967291)

func GenReqId() string {
	var b [12]byte
	binary.LittleEndian.PutUint32(b[:], pid)
	binary.LittleEndian.PutUint64(b[4:], uint64(time.Now().UnixNano()))
	return base64.URLEncoding.EncodeToString(b[:])
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

func ErrorLogger() gin.HandlerFunc {
	return ErrorLoggerT(gin.ErrorTypeAny)
}

func ErrorLoggerT(typ gin.ErrorType) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		// avoid writting if we already wrote into the response body
		if !c.Writer.Written() {
			errors := c.Errors.ByType(typ)
			if len(errors) > 0 {
				c.JSON(-1, errors)
			}
		}
	}
}

// Instances a Logger middleware that will write the logs to gin.DefaultWriter
// By default gin.DefaultWriter = os.Stdout
func Logger() gin.HandlerFunc {
	return LoggerWithWriter(gin.DefaultWriter)
}

// Instance a Logger middleware with the specified writter buffer.
// Example: os.Stdout, a file opened in write mode, a socket...
func LoggerWithWriter(out io.Writer, notlogged ...string) gin.HandlerFunc {
	var skip map[string]struct{}

	if length := len(notlogged); length > 0 {
		skip = make(map[string]struct{}, length)

		for _, path := range notlogged {
			skip[path] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		xReqid := c.Request.Header.Get("X-Reqid")

		if xReqid == "" {
			xReqid = GenReqId()
		}

		logger := xlog.New(xReqid)

		c.Header("X-Reqid", logger.ReqId)

		path := c.Request.URL.RequestURI()

		clientIP := c.ClientIP()
		method := c.Request.Method
		methodColor := colorForMethod(method)

		// Log only when path is not being skipped
		if _, ok := skip[path]; !ok {
			fmt.Fprintf(out, "[GIN] [%s] [Route Start]\t%v |%s  %s %-7s %s\n",
				logger.ReqId,
				start.Format("2006/01/02 - 15:04:05"),
				methodColor, reset, method,
				path,
			)
			logger.Infof("%+v", c.Request)
		}

		// Process request
		c.Next()

		// Log only when path is not being skipped
		if _, ok := skip[path]; !ok {
			// Stop timer
			end := time.Now()
			latency := end.Sub(start)

			comment := c.Errors.ByType(gin.ErrorTypePrivate).String()

			statusCode := c.Writer.Status()
			statusColor := colorForStatus(statusCode)
			methodColor = colorForMethod(method)

			fmt.Fprintf(out, "[GIN] [%s] [Route End]\t%v |%s %3d %s| %13v | %s |%s  %s %-7s %s\n%s",
				logger.ReqId,
				end.Format("2006/01/02 - 15:04:05"),
				statusColor, statusCode, reset,
				latency,
				clientIP,
				methodColor, reset, method,
				path,
				comment,
			)
		}
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

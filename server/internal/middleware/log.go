package middleware

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"tutoring_server/pkg/logger"
)

func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/assets") ||
			strings.HasPrefix(c.Request.URL.Path, "/uploads") {
			c.Next()
			return
		}

		t := time.Now()
		c.Next()

		logger.Info("get a request, api=%s, status=%d, latency=%dus, ip=%s",
			c.Request.URL.Path, c.Writer.Status(), time.Since(t).Microseconds(), c.ClientIP())
	}
}

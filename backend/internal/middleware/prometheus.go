package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"ancient-wood-monitor/internal/metrics"
)

func PrometheusMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()

		if path == "" {
			path = c.Request.URL.Path
		}

		c.Next()

		duration := time.Since(start)
		status := strconv.Itoa(c.Writer.Status())
		method := c.Request.Method

		metrics.ObserveHTTPRequest(method, path, status, duration)
	}
}

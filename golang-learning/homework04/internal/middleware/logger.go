package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// RequestLogger 记录每个请求的 method/path/status/耗时/client_ip（见 CLAUDE.md"编码规范"第 7 条）。
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if raw := c.Request.URL.RawQuery; raw != "" {
			path += "?" + raw
		}

		c.Next()

		logrus.WithFields(logrus.Fields{
			"method":   c.Request.Method,
			"path":     path,
			"status":   c.Writer.Status(),
			"duration": time.Since(start).String(),
			"clientIp": c.ClientIP(),
		}).Info("请求处理完成")
	}
}

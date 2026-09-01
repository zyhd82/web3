// Package middleware 存放 gin 中间件：鉴权、请求日志、panic 恢复。
package middleware

import (
	"io"

	apperrors "homework04/internal/pkg/errors"
	"homework04/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Recovery 捕获 handler 链路中的 panic，记录日志并转换成统一错误响应，避免进程崩溃。
// 用 io.Discard 关掉 gin 内置的堆栈信息输出，日志统一走 logrus。
func Recovery() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(io.Discard, func(c *gin.Context, recovered any) {
		if err, ok := recovered.(error); ok {
			logrus.WithError(err).Error("请求处理时发生 panic")
		} else {
			logrus.WithField("recovered", recovered).Error("请求处理时发生 panic")
		}
		response.Fail(c, apperrors.ErrInternal)
		c.Abort()
	})
}

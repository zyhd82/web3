// Package handler 是 gin handler 层，只负责绑定/校验请求、调用 service、用 pkg/response 返回结果，
// 不写业务逻辑。
package handler

import (
	"net/http"

	apperrors "homework04/internal/pkg/errors"
	"homework04/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// bindJSON 解析并做基础校验（gin binding tag），失败时直接写统一错误响应。
// 返回 false 时调用方必须立刻 return，不能继续往下执行。
func bindJSON(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		response.Fail(c, apperrors.Wrap(err, apperrors.CodeInvalidParams, err.Error(), http.StatusBadRequest))
		return false
	}
	return true
}

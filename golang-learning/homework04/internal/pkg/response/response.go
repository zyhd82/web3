// Package response 提供统一的 JSON 响应结构，所有 handler 必须通过这里返回响应，
// 不允许在 handler 里手写 c.JSON 拼响应体（见 CLAUDE.md"编码规范"第 4 条）。
package response

import (
	"errors"
	"net/http"

	apperrors "homework04/internal/pkg/errors"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Body 是统一的响应体结构，字段名与 dto 一致用 camelCase。
type Body struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PageData 是分页列表的响应数据。
type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// Success 返回成功响应，业务码固定为 0。
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Body{Code: 0, Message: "success", Data: data})
}

// PageSuccess 返回分页列表的成功响应。
func PageSuccess(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	Success(c, PageData{List: list, Total: total, Page: page, PageSize: pageSize})
}

// Fail 把 error 转换成统一错误响应。err 是 *errors.AppError 时使用其携带的业务码/消息/HTTP 状态；
// 否则视为未预期的服务端错误，记录日志并统一按 ErrInternal 处理，不把内部错误细节泄露给客户端。
func Fail(c *gin.Context, err error) {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus, Body{Code: appErr.Code, Message: appErr.Message})
		return
	}

	logrus.WithError(err).Error("未分类的服务端错误")
	c.JSON(http.StatusInternalServerError, Body{Code: apperrors.ErrInternal.Code, Message: apperrors.ErrInternal.Message})
}

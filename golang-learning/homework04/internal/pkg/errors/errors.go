// Package errors 定义统一的业务错误类型。业务错误码独立于 HTTP 状态码，
// 按模块分段（见 CLAUDE.md"编码规范"第 4 条）：
//
//	10000-10999 用户模块
//	20000-20999 文章模块
//	30000-30999 评论模块
//	40000-40999 通用/参数校验/鉴权错误
package errors

import "net/http"

// AppError 是业务错误类型，携带业务错误码、对外展示的消息，以及应该配套返回的 HTTP 状态码。
type AppError struct {
	Code       int
	Message    string
	HTTPStatus int
	Err        error
}

// Error 实现 error 接口。
func (e *AppError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap 暴露底层 error，配合 errors.Is/errors.As 使用。
func (e *AppError) Unwrap() error {
	return e.Err
}

// New 创建一个业务错误。httpStatus 可选，不传时缺省为 400（Bad Request）。
func New(code int, message string, httpStatus ...int) *AppError {
	status := http.StatusBadRequest
	if len(httpStatus) > 0 {
		status = httpStatus[0]
	}
	return &AppError{Code: code, Message: message, HTTPStatus: status}
}

// Wrap 用业务错误码/描述包裹一个底层 error，保留错误链，httpStatus 可选、缺省 400。
func Wrap(err error, code int, message string, httpStatus ...int) *AppError {
	ae := New(code, message, httpStatus...)
	ae.Err = err
	return ae
}

// 40000-40999：通用/参数校验/鉴权错误，各模块通用，不需要每个 service 重复定义。
const (
	CodeInvalidParams = 40000
	CodeUnauthorized  = 40001
	CodeForbidden     = 40002
	CodeInternal      = 40003
)

// 通用错误的预定义实例，直接返回或用 errors.Is 判断。
var (
	ErrInvalidParams = New(CodeInvalidParams, "请求参数不合法", http.StatusBadRequest)
	ErrUnauthorized  = New(CodeUnauthorized, "未登录或登录已过期", http.StatusUnauthorized)
	ErrForbidden     = New(CodeForbidden, "没有权限执行该操作", http.StatusForbidden)
	ErrInternal      = New(CodeInternal, "服务器内部错误", http.StatusInternalServerError)
)

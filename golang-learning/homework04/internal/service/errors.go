package service

import (
	"net/http"

	apperrors "homework04/internal/pkg/errors"
)

// 用户模块错误码：10000-10999（见 CLAUDE.md"编码规范"第 4 条的分段规则）。
const (
	CodeUsernameTaken      = 10001
	CodeEmailTaken         = 10002
	CodeInvalidCredentials = 10003
	CodeUserNotFound       = 10004
)

// 用户模块的预定义业务错误。
var (
	ErrUsernameTaken      = apperrors.New(CodeUsernameTaken, "用户名已被占用", http.StatusConflict)
	ErrEmailTaken         = apperrors.New(CodeEmailTaken, "邮箱已被占用", http.StatusConflict)
	ErrInvalidCredentials = apperrors.New(CodeInvalidCredentials, "用户名或密码错误", http.StatusUnauthorized)
	ErrUserNotFound       = apperrors.New(CodeUserNotFound, "用户不存在", http.StatusNotFound)
)

// 文章模块错误码：20000-20999。
const (
	CodePostNotFound = 20001
)

// 文章模块的预定义业务错误。
var (
	ErrPostNotFound = apperrors.New(CodePostNotFound, "文章不存在", http.StatusNotFound)
)

// 评论模块错误码：30000-30999。
const (
	CodeCommentNotFound = 30001
)

// 评论模块的预定义业务错误。
var (
	ErrCommentNotFound = apperrors.New(CodeCommentNotFound, "评论不存在", http.StatusNotFound)
)

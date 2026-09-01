package middleware

import (
	"strings"

	apperrors "homework04/internal/pkg/errors"
	"homework04/internal/pkg/jwtutil"
	"homework04/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// gin.Context 存取当前登录用户信息用的 key。
const (
	ContextKeyUserID   = "userId"
	ContextKeyUsername = "username"
	ContextKeyRole     = "role"
)

// JWTAuth 校验请求头 Authorization: Bearer <token>，把解析出的用户信息写入 gin.Context
// 供后续 handler 使用；校验失败直接中断请求，返回 401。
func JWTAuth(tokenMgr *jwtutil.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || tokenString == "" {
			response.Fail(c, apperrors.ErrUnauthorized)
			c.Abort()
			return
		}

		claims, err := tokenMgr.Parse(tokenString)
		if err != nil {
			response.Fail(c, apperrors.ErrUnauthorized)
			c.Abort()
			return
		}

		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyUsername, claims.Username)
		c.Set(ContextKeyRole, claims.Role)
		c.Next()
	}
}

// CurrentUserID 从 gin.Context 取出当前登录用户的 ID，需要在 JWTAuth 之后调用。
func CurrentUserID(c *gin.Context) (uint, bool) {
	v, ok := c.Get(ContextKeyUserID)
	if !ok {
		return 0, false
	}
	id, ok := v.(uint)
	return id, ok
}

// CurrentRole 从 gin.Context 取出当前登录用户的角色，需要在 JWTAuth 之后调用。
func CurrentRole(c *gin.Context) (string, bool) {
	v, ok := c.Get(ContextKeyRole)
	if !ok {
		return "", false
	}
	role, ok := v.(string)
	return role, ok
}

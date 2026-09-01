// Package jwtutil 封装 JWT 的生成与解析。v1 版本不做 refresh token、不做黑名单，
// 靠较短的过期时间兜底安全性（见 CLAUDE.md"业务设计要点"）。
package jwtutil

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken 表示 token 格式错误、签名不合法或已过期，调用方不需要关心具体是哪种情况。
var ErrInvalidToken = errors.New("invalid or expired token")

// Claims 是本项目 JWT 的自定义载荷。
type Claims struct {
	UserID   uint   `json:"userId"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Manager 负责生成和解析 JWT，持有签名密钥和过期时长。
type Manager struct {
	secret []byte
	expire time.Duration
}

// NewManager 创建 Manager。
func NewManager(secret string, expire time.Duration) *Manager {
	return &Manager{secret: []byte(secret), expire: expire}
}

// Generate 为指定用户签发一个新 token。
func (m *Manager) Generate(userID uint, username, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expire)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// Parse 校验签名并解析 token，返回其中的自定义载荷；签名不对、格式错误或已过期都返回 ErrInvalidToken。
func (m *Manager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		return m.secret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

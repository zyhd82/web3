package dto

import (
	"time"

	"homework04/internal/model"
)

// UserResponse 是对外展示的用户信息，不包含密码哈希等敏感字段。
type UserResponse struct {
	ID        uint      `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Nickname  string    `json:"nickname"`
	Avatar    string    `json:"avatar"`
	Bio       string    `json:"bio"`
	CreatedAt time.Time `json:"createdAt"`
}

// FromUser 把 model.User 转换成对外展示的 UserResponse。
func FromUser(u *model.User) UserResponse {
	return UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Role:      u.Role,
		Nickname:  u.Nickname,
		Avatar:    u.Avatar,
		Bio:       u.Bio,
		CreatedAt: u.CreatedAt,
	}
}

// UpdateProfileRequest 是更新资料请求体。字段用指针类型，区分"未提供该字段"和"把它设为空字符串"。
type UpdateProfileRequest struct {
	Nickname *string `json:"nickname" binding:"omitempty,max=50"`
	Avatar   *string `json:"avatar" binding:"omitempty,max=255"`
	Bio      *string `json:"bio" binding:"omitempty,max=255"`
}

// ChangePasswordRequest 是修改密码请求体。
type ChangePasswordRequest struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required,min=8"`
}

package dto

// RegisterRequest 是注册请求体。
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest 是登录请求体。
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 是登录成功后的响应体。
type LoginResponse struct {
	Token string       `json:"token"`
	User  UserResponse `json:"user"`
}

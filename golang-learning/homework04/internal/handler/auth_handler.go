package handler

import (
	"homework04/internal/dto"
	"homework04/internal/pkg/response"
	"homework04/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler 处理注册、登录。
type AuthHandler struct {
	authSvc *service.AuthService
}

// NewAuthHandler 创建 AuthHandler。
func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// Register 处理 POST /api/v1/auth/register。
//
// @Summary      用户注册
// @Description  公开接口，创建一个新用户，用户名/邮箱不能重复
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.RegisterRequest  true  "注册信息"
// @Success      200   {object}  response.Body{data=dto.UserResponse}
// @Failure      400   {object}  response.Body  "参数不合法"
// @Failure      409   {object}  response.Body  "用户名或邮箱已被占用"
// @Router       /auth/register [post]
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if !bindJSON(c, &req) {
		return
	}

	user, err := h.authSvc.Register(c.Request.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, dto.FromUser(user))
}

// Login 处理 POST /api/v1/auth/login。
//
// @Summary      用户登录
// @Description  公开接口，用户名密码登录，成功返回 JWT
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      dto.LoginRequest  true  "登录信息"
// @Success      200   {object}  response.Body{data=dto.LoginResponse}
// @Failure      400   {object}  response.Body  "参数不合法"
// @Failure      401   {object}  response.Body  "用户名或密码错误"
// @Router       /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !bindJSON(c, &req) {
		return
	}

	user, token, err := h.authSvc.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, dto.LoginResponse{Token: token, User: dto.FromUser(user)})
}

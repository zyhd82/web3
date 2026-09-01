package handler

import (
	"homework04/internal/dto"
	apperrors "homework04/internal/pkg/errors"
	"homework04/internal/middleware"
	"homework04/internal/pkg/response"
	"homework04/internal/service"

	"github.com/gin-gonic/gin"
)

// UserHandler 处理登录用户自己的资料相关接口，都要求先过 middleware.JWTAuth。
type UserHandler struct {
	userSvc *service.UserService
}

// NewUserHandler 创建 UserHandler。
func NewUserHandler(userSvc *service.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

// Profile 处理 GET /api/v1/users/me。
//
// @Summary      获取当前用户资料
// @Tags         user
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  response.Body{data=dto.UserResponse}
// @Failure      401  {object}  response.Body  "未登录或登录已过期"
// @Router       /users/me [get]
func (h *UserHandler) Profile(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Fail(c, apperrors.ErrUnauthorized)
		return
	}

	user, err := h.userSvc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, dto.FromUser(user))
}

// UpdateProfile 处理 PUT /api/v1/users/me。
//
// @Summary      更新当前用户资料
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.UpdateProfileRequest  true  "要更新的字段，未提供的字段不修改"
// @Success      200   {object}  response.Body{data=dto.UserResponse}
// @Failure      400   {object}  response.Body  "参数不合法"
// @Failure      401   {object}  response.Body  "未登录或登录已过期"
// @Router       /users/me [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Fail(c, apperrors.ErrUnauthorized)
		return
	}

	var req dto.UpdateProfileRequest
	if !bindJSON(c, &req) {
		return
	}

	user, err := h.userSvc.UpdateProfile(c.Request.Context(), userID, req.Nickname, req.Avatar, req.Bio)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, dto.FromUser(user))
}

// ChangePassword 处理 PUT /api/v1/users/me/password。
//
// @Summary      修改当前用户密码
// @Tags         user
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.ChangePasswordRequest  true  "旧密码/新密码"
// @Success      200   {object}  response.Body
// @Failure      400   {object}  response.Body  "参数不合法"
// @Failure      401   {object}  response.Body  "未登录或旧密码错误"
// @Router       /users/me/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Fail(c, apperrors.ErrUnauthorized)
		return
	}

	var req dto.ChangePasswordRequest
	if !bindJSON(c, &req) {
		return
	}

	if err := h.userSvc.ChangePassword(c.Request.Context(), userID, req.OldPassword, req.NewPassword); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

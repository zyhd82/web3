package handler

import (
	"homework04/internal/dto"
	"homework04/internal/middleware"
	apperrors "homework04/internal/pkg/errors"
	"homework04/internal/pkg/response"
	"homework04/internal/service"

	"github.com/gin-gonic/gin"
)

// CommentHandler 处理评论的创建、删除、按文章分页查询。Create/Delete 需要登录，List 公开。
type CommentHandler struct {
	commentSvc *service.CommentService
}

// NewCommentHandler 创建 CommentHandler。
func NewCommentHandler(commentSvc *service.CommentService) *CommentHandler {
	return &CommentHandler{commentSvc: commentSvc}
}

// Create 处理 POST /api/v1/posts/:postId/comments。
//
// @Summary      创建评论
// @Tags         comment
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        postId  path      int                       true  "文章 ID"
// @Param        body    body      dto.CreateCommentRequest  true  "评论内容"
// @Success      200     {object}  response.Body{data=dto.CommentResponse}
// @Failure      400     {object}  response.Body  "参数不合法"
// @Failure      401     {object}  response.Body  "未登录或登录已过期"
// @Failure      404     {object}  response.Body  "文章不存在"
// @Router       /posts/{postId}/comments [post]
func (h *CommentHandler) Create(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Fail(c, apperrors.ErrUnauthorized)
		return
	}

	postID, ok := parseIDParam(c, "postId")
	if !ok {
		return
	}

	var req dto.CreateCommentRequest
	if !bindJSON(c, &req) {
		return
	}

	comment, err := h.commentSvc.Create(c.Request.Context(), postID, userID, req.Content)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, dto.FromComment(comment))
}

// Delete 处理 DELETE /api/v1/comments/:id。
//
// @Summary      删除评论
// @Description  只有作者本人或管理员可以操作
// @Tags         comment
// @Produce      json
// @Security     BearerAuth
// @Param        id   path      int  true  "评论 ID"
// @Success      200  {object}  response.Body
// @Failure      401  {object}  response.Body  "未登录或登录已过期"
// @Failure      403  {object}  response.Body  "没有权限"
// @Failure      404  {object}  response.Body  "评论不存在"
// @Router       /comments/{id} [delete]
func (h *CommentHandler) Delete(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Fail(c, apperrors.ErrUnauthorized)
		return
	}
	role, _ := middleware.CurrentRole(c)

	commentID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.commentSvc.Delete(c.Request.Context(), commentID, userID, role); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

// List 处理 GET /api/v1/posts/:postId/comments，公开接口。
//
// @Summary      文章评论列表
// @Description  公开接口，分页返回
// @Tags         comment
// @Produce      json
// @Param        postId    path      int  true   "文章 ID"
// @Param        page      query     int  false  "页码，默认 1"
// @Param        pageSize  query     int  false  "每页数量，默认 10，上限 100"
// @Success      200       {object}  response.Body{data=response.PageData{list=[]dto.CommentResponse}}
// @Router       /posts/{postId}/comments [get]
func (h *CommentHandler) List(c *gin.Context) {
	postID, ok := parseIDParam(c, "postId")
	if !ok {
		return
	}
	page, pageSize := parsePagination(c)

	comments, total, err := h.commentSvc.ListByPostID(c.Request.Context(), postID, page, pageSize)
	if err != nil {
		response.Fail(c, err)
		return
	}

	items := make([]dto.CommentResponse, 0, len(comments))
	for _, cm := range comments {
		items = append(items, dto.FromComment(cm))
	}
	response.PageSuccess(c, items, total, page, pageSize)
}

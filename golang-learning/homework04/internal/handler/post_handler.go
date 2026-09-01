package handler

import (
	"strconv"

	"homework04/internal/dto"
	"homework04/internal/middleware"
	apperrors "homework04/internal/pkg/errors"
	"homework04/internal/pkg/response"
	"homework04/internal/service"

	"github.com/gin-gonic/gin"
)

// PostHandler 处理文章的增删改查。Detail/List 是公开接口，不需要登录；Create/Update/Delete
// 需要先过 middleware.JWTAuth。
type PostHandler struct {
	postSvc *service.PostService
}

// NewPostHandler 创建 PostHandler。
func NewPostHandler(postSvc *service.PostService) *PostHandler {
	return &PostHandler{postSvc: postSvc}
}

// Create 处理 POST /api/v1/posts。
//
// @Summary      创建文章
// @Tags         post
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body  body      dto.CreatePostRequest  true  "文章标题/正文"
// @Success      200   {object}  response.Body{data=dto.PostResponse}
// @Failure      400   {object}  response.Body  "参数不合法"
// @Failure      401   {object}  response.Body  "未登录或登录已过期"
// @Router       /posts [post]
func (h *PostHandler) Create(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Fail(c, apperrors.ErrUnauthorized)
		return
	}

	var req dto.CreatePostRequest
	if !bindJSON(c, &req) {
		return
	}

	post, err := h.postSvc.Create(c.Request.Context(), userID, req.Title, req.Content)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, dto.FromPost(post))
}

// Update 处理 PUT /api/v1/posts/:postId。
//
// @Summary      更新文章
// @Description  只有作者本人或管理员可以操作
// @Tags         post
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        postId  path      int                    true  "文章 ID"
// @Param        body    body      dto.UpdatePostRequest  true  "要更新的字段，未提供的字段不修改"
// @Success      200     {object}  response.Body{data=dto.PostResponse}
// @Failure      400     {object}  response.Body  "参数不合法"
// @Failure      401     {object}  response.Body  "未登录或登录已过期"
// @Failure      403     {object}  response.Body  "没有权限"
// @Failure      404     {object}  response.Body  "文章不存在"
// @Router       /posts/{postId} [put]
func (h *PostHandler) Update(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Fail(c, apperrors.ErrUnauthorized)
		return
	}
	role, _ := middleware.CurrentRole(c)

	postID, ok := parseIDParam(c, "postId")
	if !ok {
		return
	}

	var req dto.UpdatePostRequest
	if !bindJSON(c, &req) {
		return
	}

	post, err := h.postSvc.Update(c.Request.Context(), postID, userID, role, req.Title, req.Content)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, dto.FromPost(post))
}

// Delete 处理 DELETE /api/v1/posts/:postId。
//
// @Summary      删除文章
// @Description  只有作者本人或管理员可以操作，会级联删除该文章下的全部评论
// @Tags         post
// @Produce      json
// @Security     BearerAuth
// @Param        postId  path      int  true  "文章 ID"
// @Success      200     {object}  response.Body
// @Failure      401     {object}  response.Body  "未登录或登录已过期"
// @Failure      403     {object}  response.Body  "没有权限"
// @Failure      404     {object}  response.Body  "文章不存在"
// @Router       /posts/{postId} [delete]
func (h *PostHandler) Delete(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.Fail(c, apperrors.ErrUnauthorized)
		return
	}
	role, _ := middleware.CurrentRole(c)

	postID, ok := parseIDParam(c, "postId")
	if !ok {
		return
	}

	if err := h.postSvc.Delete(c.Request.Context(), postID, userID, role); err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, nil)
}

// Detail 处理 GET /api/v1/posts/:postId，公开接口。
//
// @Summary      文章详情
// @Description  公开接口
// @Tags         post
// @Produce      json
// @Param        postId  path      int  true  "文章 ID"
// @Success      200     {object}  response.Body{data=dto.PostResponse}
// @Failure      404     {object}  response.Body  "文章不存在"
// @Router       /posts/{postId} [get]
func (h *PostHandler) Detail(c *gin.Context) {
	postID, ok := parseIDParam(c, "postId")
	if !ok {
		return
	}

	post, err := h.postSvc.FindByID(c.Request.Context(), postID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Success(c, dto.FromPost(post))
}

// List 处理 GET /api/v1/posts，公开接口，支持 ?userId= 按作者筛选、?page=&pageSize= 分页。
//
// @Summary      文章列表
// @Description  公开接口，分页返回，不含正文；?userId= 可按作者筛选
// @Tags         post
// @Produce      json
// @Param        userId    query     int  false  "按作者 ID 筛选"
// @Param        page      query     int  false  "页码，默认 1"
// @Param        pageSize  query     int  false  "每页数量，默认 10，上限 100"
// @Success      200       {object}  response.Body{data=response.PageData{list=[]dto.PostResponse}}
// @Router       /posts [get]
func (h *PostHandler) List(c *gin.Context) {
	page, pageSize := parsePagination(c)

	var userID *uint
	if raw := c.Query("userId"); raw != "" {
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			response.Fail(c, apperrors.ErrInvalidParams)
			return
		}
		uid := uint(id)
		userID = &uid
	}

	posts, total, err := h.postSvc.List(c.Request.Context(), page, pageSize, userID)
	if err != nil {
		response.Fail(c, err)
		return
	}

	items := make([]dto.PostResponse, 0, len(posts))
	for _, p := range posts {
		items = append(items, dto.FromPost(p))
	}
	response.PageSuccess(c, items, total, page, pageSize)
}

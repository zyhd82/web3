// Package router 把各个 handler 组装成完整的路由树，接入全局中间件。
package router

import (
	"homework04/internal/handler"
	"homework04/internal/middleware"
	"homework04/internal/pkg/jwtutil"
	"homework04/internal/pkg/response"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Handlers 汇总所有业务 handler，由 main.go 组装依赖后传入 New。
type Handlers struct {
	Auth    *handler.AuthHandler
	User    *handler.UserHandler
	Post    *handler.PostHandler
	Comment *handler.CommentHandler
}

// New 组装完整的路由树：全局中间件（恢复、请求日志）+ /healthz + /api/v1 下的业务路由。
//
// /posts 被注册了两个路由组（一个公开、一个挂 JWTAuth），这是有意为之：同一路径前缀按 HTTP 方法
// 区分是否需要登录（GET 公开浏览，POST/PUT/DELETE 需要登录），gin 按 method+path 建路由树，
// 不会冲突。文章自身 ID 的路径参数统一叫 :postId（不用 :id），是为了不和 /posts/:postId/comments
// 冲突，见 CLAUDE.md"业务设计要点"。
func New(h *Handlers, tokenMgr *jwtutil.Manager) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recovery(), middleware.RequestLogger())

	r.GET("/healthz", healthz)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")

	auth := v1.Group("/auth")
	auth.POST("/register", h.Auth.Register)
	auth.POST("/login", h.Auth.Login)

	users := v1.Group("/users")
	users.Use(middleware.JWTAuth(tokenMgr))
	users.GET("/me", h.User.Profile)
	users.PUT("/me", h.User.UpdateProfile)
	users.PUT("/me/password", h.User.ChangePassword)

	publicPosts := v1.Group("/posts")
	publicPosts.GET("", h.Post.List)
	publicPosts.GET("/:postId", h.Post.Detail)
	publicPosts.GET("/:postId/comments", h.Comment.List)

	authedPosts := v1.Group("/posts")
	authedPosts.Use(middleware.JWTAuth(tokenMgr))
	authedPosts.POST("", h.Post.Create)
	authedPosts.PUT("/:postId", h.Post.Update)
	authedPosts.DELETE("/:postId", h.Post.Delete)
	authedPosts.POST("/:postId/comments", h.Comment.Create)

	comments := v1.Group("/comments")
	comments.Use(middleware.JWTAuth(tokenMgr))
	comments.DELETE("/:id", h.Comment.Delete)

	return r
}

// healthz 是基础设施探活接口，走统一响应结构。
func healthz(c *gin.Context) {
	response.Success(c, gin.H{"status": "ok"})
}

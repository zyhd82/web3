package main

import (
	"homework04/internal/config"
	"homework04/internal/handler"
	"homework04/internal/model"
	"homework04/internal/pkg/jwtutil"
	"homework04/internal/pkg/logger"
	"homework04/internal/repository"
	"homework04/internal/router"
	"homework04/internal/service"

	_ "homework04/docs"

	"github.com/sirupsen/logrus"
)

// @title                       个人博客后端 API
// @version                     1.0
// @description                 Go + Gin + GORM 实现的个人博客后端微服务，提供用户认证、文章管理、评论管理接口。
// @BasePath                    /api/v1
// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 除标注"公开接口"外均需要登录，在请求头带上 "Bearer <token>"（token 从登录接口获取）
func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel, cfg.LogFile)

	if err := config.InitDatabase(cfg, &model.User{}, &model.Post{}, &model.Comment{}); err != nil {
		logrus.Fatal(err)
	}

	userRepo := repository.NewUserRepository(config.DB)
	postRepo := repository.NewPostRepository(config.DB)
	commentRepo := repository.NewCommentRepository(config.DB)

	tokenMgr := jwtutil.NewManager(cfg.JWTSecret, cfg.TokenExpireDuration())

	authSvc := service.NewAuthService(userRepo, tokenMgr)
	userSvc := service.NewUserService(userRepo)
	postSvc := service.NewPostService(postRepo, config.DB)
	commentSvc := service.NewCommentService(commentRepo, postRepo)

	handlers := &router.Handlers{
		Auth:    handler.NewAuthHandler(authSvc),
		User:    handler.NewUserHandler(userSvc),
		Post:    handler.NewPostHandler(postSvc),
		Comment: handler.NewCommentHandler(commentSvc),
	}

	r := router.New(handlers, tokenMgr)

	logrus.Infof("服务启动，监听端口 :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		logrus.Fatal(err)
	}
}

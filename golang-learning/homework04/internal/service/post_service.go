package service

import (
	"context"
	"errors"
	"fmt"

	apperrors "homework04/internal/pkg/errors"

	"homework04/internal/model"
	"homework04/internal/repository"

	"gorm.io/gorm"
)

// postRepository 是 PostService 依赖的 repository 能力子集。Create/Update/FindByID/List 走这个
// 接口，方便单测 mock；Delete 需要和评论表做跨表事务，不走这个接口，见 Delete 方法注释。
type postRepository interface {
	Create(ctx context.Context, post *model.Post) error
	Update(ctx context.Context, post *model.Post) error
	FindByID(ctx context.Context, id uint) (*model.Post, error)
	List(ctx context.Context, page, pageSize int, userID *uint) ([]*model.Post, int64, error)
}

// PostService 处理文章的增删改查。
type PostService struct {
	repo postRepository
	db   *gorm.DB
}

// NewPostService 创建 PostService。db 只在 Delete 方法里用来开启跨表事务。
func NewPostService(repo postRepository, db *gorm.DB) *PostService {
	return &PostService{repo: repo, db: db}
}

// Create 创建一篇新文章。
func (s *PostService) Create(ctx context.Context, userID uint, title, content string) (*model.Post, error) {
	post := &model.Post{UserID: userID, Title: title, Content: content}
	if err := s.repo.Create(ctx, post); err != nil {
		return nil, fmt.Errorf("create post: %w", err)
	}
	// Create 不会带上作者信息，重新查一次（带 Preload），方便 handler 直接转换成响应。
	return s.findPost(ctx, post.ID)
}

// Update 更新文章标题/正文，title、content 为 nil 表示不修改该字段；只有作者本人或管理员可以操作。
func (s *PostService) Update(ctx context.Context, postID, userID uint, role string, title, content *string) (*model.Post, error) {
	post, err := s.findPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if !canModifyPost(post, userID, role) {
		return nil, apperrors.ErrForbidden
	}

	if title != nil {
		post.Title = *title
	}
	if content != nil {
		post.Content = *content
	}
	if err := s.repo.Update(ctx, post); err != nil {
		return nil, fmt.Errorf("update post: %w", err)
	}
	return post, nil
}

// Delete 删除文章，并在同一事务里级联软删除其下的全部评论；只有作者本人或管理员可以操作。
//
// 没有物理外键（见 docs/DATABASE_CONVENTIONS.md），级联删除评论必须由 service 层显式处理。
// 这里直接用 repository 包的构造函数创建事务范围内的 repository 实例，是"接口由调用方定义"
// 原则（CLAUDE.md"编码规范"第 3 条）在跨表事务场景下的务实例外：事务编排本来就得知道底层是
// gorm，抽象成通用 interface 收益不大，其它方法仍然通过 postRepository 接口访问数据。
func (s *PostService) Delete(ctx context.Context, postID, userID uint, role string) error {
	post, err := s.findPost(ctx, postID)
	if err != nil {
		return err
	}
	if !canModifyPost(post, userID, role) {
		return apperrors.ErrForbidden
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := repository.NewCommentRepository(tx).DeleteByPostID(ctx, postID); err != nil {
			return fmt.Errorf("delete comments: %w", err)
		}
		if err := repository.NewPostRepository(tx).Delete(ctx, postID); err != nil {
			return fmt.Errorf("delete post: %w", err)
		}
		return nil
	})
}

// FindByID 查询文章详情。
func (s *PostService) FindByID(ctx context.Context, postID uint) (*model.Post, error) {
	return s.findPost(ctx, postID)
}

// List 分页查询文章列表，userID 非空时只查该作者的文章。
func (s *PostService) List(ctx context.Context, page, pageSize int, userID *uint) ([]*model.Post, int64, error) {
	posts, total, err := s.repo.List(ctx, page, pageSize, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("list posts: %w", err)
	}
	return posts, total, nil
}

// findPost 是内部共用的查询逻辑，把 gorm.ErrRecordNotFound 翻译成业务错误 ErrPostNotFound。
func (s *PostService) findPost(ctx context.Context, postID uint) (*model.Post, error) {
	post, err := s.repo.FindByID(ctx, postID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("find post by id: %w", err)
	}
	return post, nil
}

// canModifyPost 判断用户是否有权限修改/删除该文章：本人或管理员（见 CLAUDE.md"业务设计要点"）。
func canModifyPost(post *model.Post, userID uint, role string) bool {
	return post.UserID == userID || role == model.RoleAdmin
}

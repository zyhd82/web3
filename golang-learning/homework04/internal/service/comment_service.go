package service

import (
	"context"
	"errors"
	"fmt"

	apperrors "homework04/internal/pkg/errors"

	"homework04/internal/model"

	"gorm.io/gorm"
)

// commentRepository 是 CommentService 依赖的评论表 repository 能力子集。
type commentRepository interface {
	Create(ctx context.Context, comment *model.Comment) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (*model.Comment, error)
	ListByPostID(ctx context.Context, postID uint, page, pageSize int) ([]*model.Comment, int64, error)
}

// commentPostRepository 是 CommentService 依赖的文章表 repository 能力子集，只用来确认
// PostID 是否存在（见 docs/DATABASE_CONVENTIONS.md"创建评论前先确认 PostID 存在"）。
type commentPostRepository interface {
	FindByID(ctx context.Context, id uint) (*model.Post, error)
}

// CommentService 处理评论的创建、删除、按文章分页查询。
type CommentService struct {
	repo     commentRepository
	postRepo commentPostRepository
}

// NewCommentService 创建 CommentService。
func NewCommentService(repo commentRepository, postRepo commentPostRepository) *CommentService {
	return &CommentService{repo: repo, postRepo: postRepo}
}

// Create 在指定文章下创建一条评论，会先确认文章存在。
func (s *CommentService) Create(ctx context.Context, postID, userID uint, content string) (*model.Comment, error) {
	if _, err := s.postRepo.FindByID(ctx, postID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, fmt.Errorf("find post by id: %w", err)
	}

	comment := &model.Comment{PostID: postID, UserID: userID, Content: content}
	if err := s.repo.Create(ctx, comment); err != nil {
		return nil, fmt.Errorf("create comment: %w", err)
	}

	// Create 不会带上评论者信息，重新查一次（带 Preload），方便 handler 直接转换成响应。
	created, err := s.repo.FindByID(ctx, comment.ID)
	if err != nil {
		return nil, fmt.Errorf("find created comment: %w", err)
	}
	return created, nil
}

// Delete 删除一条评论，只有作者本人或管理员可以操作。
func (s *CommentService) Delete(ctx context.Context, commentID, userID uint, role string) error {
	comment, err := s.findComment(ctx, commentID)
	if err != nil {
		return err
	}
	if !canModifyComment(comment, userID, role) {
		return apperrors.ErrForbidden
	}
	if err := s.repo.Delete(ctx, commentID); err != nil {
		return fmt.Errorf("delete comment: %w", err)
	}
	return nil
}

// ListByPostID 分页查询某篇文章下的评论。
func (s *CommentService) ListByPostID(ctx context.Context, postID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	comments, total, err := s.repo.ListByPostID(ctx, postID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list comments: %w", err)
	}
	return comments, total, nil
}

// findComment 是内部共用的查询逻辑，把 gorm.ErrRecordNotFound 翻译成业务错误 ErrCommentNotFound。
func (s *CommentService) findComment(ctx context.Context, commentID uint) (*model.Comment, error) {
	comment, err := s.repo.FindByID(ctx, commentID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCommentNotFound
		}
		return nil, fmt.Errorf("find comment by id: %w", err)
	}
	return comment, nil
}

// canModifyComment 判断用户是否有权限删除该评论：本人或管理员（和 canModifyPost 对称，
// 暂不允许文章作者删除他人在自己文章下的评论——如需要该能力，后续单独加规则）。
func canModifyComment(comment *model.Comment, userID uint, role string) bool {
	return comment.UserID == userID || role == model.RoleAdmin
}

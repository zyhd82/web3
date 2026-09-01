package repository

import (
	"context"

	"homework04/internal/model"

	"gorm.io/gorm"
)

// CommentRepository 是 comments 表的数据访问实现。
type CommentRepository struct {
	db *gorm.DB
}

// NewCommentRepository 创建 CommentRepository。
func NewCommentRepository(db *gorm.DB) *CommentRepository {
	return &CommentRepository{db: db}
}

// Create 插入一条新评论。
func (r *CommentRepository) Create(ctx context.Context, comment *model.Comment) error {
	return r.db.WithContext(ctx).Create(comment).Error
}

// Delete 软删除指定评论。
func (r *CommentRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Comment{}, id).Error
}

// FindByID 按主键查询评论并预加载评论者信息，不存在时返回 gorm.ErrRecordNotFound。
func (r *CommentRepository) FindByID(ctx context.Context, id uint) (*model.Comment, error) {
	var comment model.Comment
	if err := r.db.WithContext(ctx).Preload("User").First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

// ListByPostID 分页查询某篇文章下的评论（按创建时间倒序），并预加载评论者信息。
func (r *CommentRepository) ListByPostID(ctx context.Context, postID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	page, pageSize = normalizePage(page, pageSize)

	base := r.db.WithContext(ctx).Model(&model.Comment{}).Where("post_id = ?", postID)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var comments []*model.Comment
	err := base.Session(&gorm.Session{}).
		Preload("User").
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&comments).Error
	return comments, total, err
}

// DeleteByPostID 软删除某篇文章下的全部评论，供删除文章时在同一事务里级联调用
// （见 docs/DATABASE_CONVENTIONS.md 关联关系一节）。
func (r *CommentRepository) DeleteByPostID(ctx context.Context, postID uint) error {
	return r.db.WithContext(ctx).Where("post_id = ?", postID).Delete(&model.Comment{}).Error
}

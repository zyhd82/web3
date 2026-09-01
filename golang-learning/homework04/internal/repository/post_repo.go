package repository

import (
	"context"

	"homework04/internal/model"

	"gorm.io/gorm"
)

// PostRepository 是 posts 表的数据访问实现。
type PostRepository struct {
	db *gorm.DB
}

// NewPostRepository 创建 PostRepository。
func NewPostRepository(db *gorm.DB) *PostRepository {
	return &PostRepository{db: db}
}

// Create 插入一篇新文章。
func (r *PostRepository) Create(ctx context.Context, post *model.Post) error {
	return r.db.WithContext(ctx).Create(post).Error
}

// Update 保存文章的全部字段，调用前需先查出完整记录再修改，避免把未涉及的字段清空。
func (r *PostRepository) Update(ctx context.Context, post *model.Post) error {
	return r.db.WithContext(ctx).Save(post).Error
}

// Delete 软删除指定文章，不级联处理评论——由 service 层在事务里显式调用
// CommentRepository.DeleteByPostID（见 docs/DATABASE_CONVENTIONS.md 关联关系一节）。
func (r *PostRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.Post{}, id).Error
}

// FindByID 按主键查询文章详情并预加载作者信息，不存在时返回 gorm.ErrRecordNotFound。
func (r *PostRepository) FindByID(ctx context.Context, id uint) (*model.Post, error) {
	var post model.Post
	if err := r.db.WithContext(ctx).Preload("User").First(&post, id).Error; err != nil {
		return nil, err
	}
	return &post, nil
}

// List 分页查询文章列表（按创建时间倒序），userID 非空时只查该作者的文章。
// 列表查询不返回 Content 字段，避免正文内容拖慢分页接口（见数据库规范"禁止 SELECT *"）。
func (r *PostRepository) List(ctx context.Context, page, pageSize int, userID *uint) ([]*model.Post, int64, error) {
	page, pageSize = normalizePage(page, pageSize)

	base := r.db.WithContext(ctx).Model(&model.Post{})
	if userID != nil {
		base = base.Where("user_id = ?", *userID)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var posts []*model.Post
	err := base.Session(&gorm.Session{}).
		Preload("User").
		Select("id", "user_id", "title", "created_at", "updated_at").
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&posts).Error
	return posts, total, err
}

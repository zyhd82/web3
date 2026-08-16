package models

import (
	"time"

	"gorm.io/gorm"
)

type Comment struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Content   string         `json:"content" gorm:"type:text;not null"`
	UserID    uint           `json:"user_id" gorm:"not null"`
	PostID    uint           `json:"post_id" gorm:"not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	User User `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Post Post `json:"post,omitempty" gorm:"foreignKey:PostID"`
}

// updatePostCommentStatus 检查并更新文章的评论状态
func (c *Comment) updatePostCommentStatus(tx *gorm.DB) error {
	// 1. 查询当前文章剩余的评论数量
	var count int64
	if err := tx.Model(&Comment{}).Where("post_id = ?", c.PostID).Count(&count).Error; err != nil {
		return err
	}

	// 2. 根据数量动态设置状态
	status := "有评论"
	if count == 0 {
		status = "无评论"
	}

	// 3. 使用 UpdateColumn 更新状态，避免触发 Post 模型的钩子导致死循环
	return tx.Model(&Post{}).Where("id = ?", c.PostID).
		UpdateColumn("comment_status", status).Error
}

// AfterCreate 钩子：评论创建成功后触发
func (c *Comment) AfterCreate(tx *gorm.DB) error {
	return c.updatePostCommentStatus(tx)
}

// AfterDelete 钩子：评论物理删除成功后触发
func (c *Comment) AfterDelete(tx *gorm.DB) error {
	return c.updatePostCommentStatus(tx)
}

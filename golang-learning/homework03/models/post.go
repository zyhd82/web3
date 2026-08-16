package models

import (
	"time"

	"gorm.io/gorm"
)

type Post struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	Title         string         `json:"title" gorm:"not null"`
	Content       string         `json:"content" gorm:"type:text;not null"`
	UserID        uint           `json:"user_id" gorm:"not null"`
	CommentStatus string         `json:"comment_status" gorm:"default:无评论"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	User     User      `json:"user,omitempty" gorm:"foreignKey:UserID"`
	Comments []Comment `json:"comments,omitempty" gorm:"foreignKey:PostID"`
}

// 文章创建后更新用户的文章数量统计字段
func (p *Post) AfterCreate(tx *gorm.DB) error {
	if p.UserID != 0 {
		return tx.Model(&User{}).Where("id = ?", p.UserID).UpdateColumn("TotalPost", gorm.Expr("TotalPost + ?", 1)).Error
	}
	return nil
}

package model

import (
	"time"

	"gorm.io/gorm"
)

// Comment 对应 comments 表，存储文章评论。按约定不支持嵌套回复，平铺结构。
type Comment struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	PostID    uint           `json:"post_id" gorm:"index;not null"`
	UserID    uint           `json:"user_id" gorm:"index;not null"`
	Content   string         `json:"content" gorm:"type:text;not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// User 评论者信息，按需 Preload，不建物理外键（见 docs/DATABASE_CONVENTIONS.md）
	User User `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID"`
}

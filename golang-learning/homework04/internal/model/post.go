package model

import (
	"time"

	"gorm.io/gorm"
)

// Post 对应 posts 表，存储文章内容。
type Post struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	UserID    uint           `json:"user_id" gorm:"index;not null"`
	Title     string         `json:"title" gorm:"type:varchar(255);not null"`
	Content   string         `json:"content" gorm:"type:text;not null"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// User 作者信息，按需 Preload，不建物理外键（见 docs/DATABASE_CONVENTIONS.md）
	User User `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID"`
}

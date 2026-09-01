package dto

import (
	"time"

	"homework04/internal/model"
)

// PostResponse 是文章详情/列表项的对外展示结构。列表查询不带 Content（repository 层做的裁剪），
// 序列化时会因为 omitempty 自动不出现在 JSON 里，详情查询则正常带上正文。
type PostResponse struct {
	ID        uint      `json:"id"`
	Title     string    `json:"title"`
	Content   string    `json:"content,omitempty"`
	AuthorID  uint      `json:"authorId"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// FromPost 把 model.Post 转换成 PostResponse。
func FromPost(p *model.Post) PostResponse {
	return PostResponse{
		ID:        p.ID,
		Title:     p.Title,
		Content:   p.Content,
		AuthorID:  p.UserID,
		Author:    p.User.Username,
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
}

// CreatePostRequest 是创建文章请求体。
type CreatePostRequest struct {
	Title   string `json:"title" binding:"required,max=255"`
	Content string `json:"content" binding:"required"`
}

// UpdatePostRequest 是更新文章请求体，字段为 nil 表示不修改。
type UpdatePostRequest struct {
	Title   *string `json:"title" binding:"omitempty,max=255"`
	Content *string `json:"content" binding:"omitempty"`
}

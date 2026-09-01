package dto

import (
	"time"

	"homework04/internal/model"
)

// CommentResponse 是评论的对外展示结构。
type CommentResponse struct {
	ID        uint      `json:"id"`
	PostID    uint      `json:"postId"`
	AuthorID  uint      `json:"authorId"`
	Author    string    `json:"author"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

// FromComment 把 model.Comment 转换成 CommentResponse。
func FromComment(c *model.Comment) CommentResponse {
	return CommentResponse{
		ID:        c.ID,
		PostID:    c.PostID,
		AuthorID:  c.UserID,
		Author:    c.User.Username,
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
	}
}

// CreateCommentRequest 是创建评论请求体。
type CreateCommentRequest struct {
	Content string `json:"content" binding:"required,max=2000"`
}

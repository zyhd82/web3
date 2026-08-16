package repository

import (
	"homework03/configs"
	"homework03/models"

	"gorm.io/gorm"
)

type CommentRepo struct {
}

func (cp *CommentRepo) GetMostCommentsPostID() (uint, error) {
	resultPostID := uint(0)

	// 查询评论表，按文章ID分组，按评论数降序，取第一条
	err := configs.DB.Model(&models.Comment{}).
		Select("post_id, COUNT(*) as cnt").
		Group("post_id").
		Order("cnt DESC").
		Limit(1).
		Pluck("post_id", &resultPostID).Error

	if err != nil {
		return -1, err
	}

	// 如果没有评论数据，postID 会是 0，可以直接返回
	if resultPostID == 0 {
		return 0, gorm.ErrRecordNotFound
	}

	return resultPostID, nil
}

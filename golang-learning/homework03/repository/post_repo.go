package repository

import (
	"homework03/configs"
	"homework03/models"
)

type PostRepo struct{}

func (ur *PostRepo) CreatePost(post *models.Post) error {
	return configs.DB.Create(post).Error
}

func (ur *PostRepo) GetPostListByUser(user *models.User) error {
	// 使用 Preload 嵌套预加载
	return configs.DB.Preload("Posts").Preload("Posts.Comments").First(&user, user.Username).Error
}

func (ur *PostRepo) GetPostByID(id int) (*models.Post, error) {
	var resultPost models.Post
	err := configs.DB.First(&resultPost, id).Error
	if err != nil {
		return nil, err
	}

	return &resultPost, nil
}

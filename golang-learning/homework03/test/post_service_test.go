package test

import (
	"homework03/models"
	"homework03/repository"
	"homework03/service"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPostService_GetFullPostInfo(t *testing.T) {

	postService := service.NewPostService(&repository.PostRepo{}, &repository.CommentRepo{})

	user, err := postService.GetFullPostInfo("")

	assert.NoError(t, err)
	assert.Equal(t, "Tom", user.Username)
}

func TestPostService_GetMostCommentsPost(t *testing.T) {

	postService := service.NewPostService(&repository.PostRepo{}, &repository.CommentRepo{})

	post, err := postService.GetMostCommentsPost(&models.User{Username: "Tom"})

	assert.NoError(t, err)
	assert.Equal(t, "2", post.UserID)
}

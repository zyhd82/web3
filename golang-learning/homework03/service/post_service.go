package service

import (
	"homework03/models"
	"homework03/repository"

	"github.com/sirupsen/logrus"
)

type PostService struct {
	postRepo    *repository.PostRepo
	commentRepo *repository.CommentRepo
}

func NewPostService(repo *repository.PostRepo, commentrepo *repository.CommentRepo) *PostService {
	return &PostService{postRepo: repo, commentRepo: commentrepo}
}

func (ps *PostService) GetFullPostInfo(userName string) (*models.User, error) {
	resultUser := models.User{Username: userName}
	err := ps.postRepo.GetPostListByUser(&resultUser)
	if err != nil {
		logrus.Warn(err)
	}
	return &resultUser, err
}

func (ps *PostService) GetMostCommentsPost(user *models.User) (*models.Post, error) {
	postID, err := ps.commentRepo.GetMostCommentsPostID()
	if err != nil {
		logrus.Warn(err)
		return nil, err
	}

	return ps.postRepo.GetPostByID(postID)
}

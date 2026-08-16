package service

import (
	"homework03/models"
	"homework03/repository"

	"github.com/sirupsen/logrus"
)

type UserService struct {
	Repo *repository.UserRepo
}

func NewUserService(repo *repository.UserRepo) *UserService {
	return &UserService{Repo: repo}
}

func (us *UserService) CreateUser(user *models.User) (*models.User, error) {
	if err := us.Repo.CreateUser(user); err != nil {
		logrus.Warn(err)
	}
	return user, nil
}

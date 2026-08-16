package repository

import (
	"homework03/configs"
	"homework03/models"
)

type UserRepo struct{}

func (ur *UserRepo) CreateUser(user *models.User) error {
	return configs.DB.Create(user).Error
}

func (ur *UserRepo) UpdateUser(user *models.User) error {
	return configs.DB.Save(user).Error
}

func (ur *UserRepo) DeleteUser(id uint) error {
	return configs.DB.Delete(&models.User{}, id).Error
}

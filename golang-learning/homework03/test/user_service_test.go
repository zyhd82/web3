package test

import (
	"homework03/models"
	"homework03/repository"
	"homework03/service"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserService_CreateUser(t *testing.T) {

	userService := service.NewUserService(&repository.UserRepo{})

	user, err := userService.CreateUser(&models.User{Username: "Tom", Email: "tom@test.com", Password: "tom@123"})

	assert.NoError(t, err)
	assert.Equal(t, "Tom", user.Username)
	assert.Equal(t, "tom@test.com", user.Email)
}

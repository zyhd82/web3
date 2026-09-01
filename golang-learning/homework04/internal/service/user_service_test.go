package service

import (
	"context"
	"testing"

	"homework04/internal/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserService_GetProfile_Success(t *testing.T) {
	user := &model.User{ID: 1, Username: "alice"}
	repo := new(mockUserRepo)
	repo.On("FindByID", mock.Anything, uint(1)).Return(user, nil)

	svc := NewUserService(repo)
	got, err := svc.GetProfile(context.Background(), 1)

	require.NoError(t, err)
	assert.Equal(t, "alice", got.Username)
}

func TestUserService_GetProfile_NotFound(t *testing.T) {
	repo := new(mockUserRepo)
	repo.On("FindByID", mock.Anything, uint(999)).Return(nil, gorm.ErrRecordNotFound)

	svc := NewUserService(repo)
	_, err := svc.GetProfile(context.Background(), 999)

	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserService_UpdateProfile_PartialFields(t *testing.T) {
	user := &model.User{ID: 1, Username: "alice", Nickname: "old", Bio: "old bio"}
	repo := new(mockUserRepo)
	repo.On("FindByID", mock.Anything, uint(1)).Return(user, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

	newNickname := "new nickname"
	svc := NewUserService(repo)
	got, err := svc.UpdateProfile(context.Background(), 1, &newNickname, nil, nil)

	require.NoError(t, err)
	assert.Equal(t, "new nickname", got.Nickname)
	assert.Equal(t, "old bio", got.Bio, "没传的字段不应该被清空")
}

func TestUserService_UpdateProfile_NotFound(t *testing.T) {
	repo := new(mockUserRepo)
	repo.On("FindByID", mock.Anything, uint(999)).Return(nil, gorm.ErrRecordNotFound)

	svc := NewUserService(repo)
	_, err := svc.UpdateProfile(context.Background(), 999, nil, nil, nil)

	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUserService_ChangePassword_Success(t *testing.T) {
	user := &model.User{ID: 1, Username: "alice"}
	require.NoError(t, user.HashPassword("oldpassword123"))

	repo := new(mockUserRepo)
	repo.On("FindByID", mock.Anything, uint(1)).Return(user, nil)
	repo.On("Update", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

	svc := NewUserService(repo)
	err := svc.ChangePassword(context.Background(), 1, "oldpassword123", "newpassword123")

	require.NoError(t, err)
	assert.True(t, user.CheckPassword("newpassword123"))
	assert.False(t, user.CheckPassword("oldpassword123"))
}

func TestUserService_ChangePassword_WrongOldPassword(t *testing.T) {
	user := &model.User{ID: 1, Username: "alice"}
	require.NoError(t, user.HashPassword("oldpassword123"))

	repo := new(mockUserRepo)
	repo.On("FindByID", mock.Anything, uint(1)).Return(user, nil)

	svc := NewUserService(repo)
	err := svc.ChangePassword(context.Background(), 1, "wrong-old-password", "newpassword123")

	assert.ErrorIs(t, err, ErrInvalidCredentials)
	repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"homework04/internal/model"
	"homework04/internal/pkg/jwtutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newTestTokenMgr() *jwtutil.Manager {
	return jwtutil.NewManager("test-secret", time.Hour)
}

func TestAuthService_Register_Success(t *testing.T) {
	repo := new(mockUserRepo)
	repo.On("ExistsByUsername", mock.Anything, "alice").Return(false, nil)
	repo.On("ExistsByEmail", mock.Anything, "alice@example.com").Return(false, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).Return(nil)

	svc := NewAuthService(repo, newTestTokenMgr())
	user, err := svc.Register(context.Background(), "alice", "alice@example.com", "password123")

	require.NoError(t, err)
	assert.Equal(t, "alice", user.Username)
	assert.Equal(t, model.RoleUser, user.Role)
	assert.NotEmpty(t, user.Password, "密码应该被哈希过")
	assert.NotEqual(t, "password123", user.Password)
	repo.AssertExpectations(t)
}

func TestAuthService_Register_UsernameTaken(t *testing.T) {
	repo := new(mockUserRepo)
	repo.On("ExistsByUsername", mock.Anything, "alice").Return(true, nil)

	svc := NewAuthService(repo, newTestTokenMgr())
	_, err := svc.Register(context.Background(), "alice", "alice@example.com", "password123")

	assert.ErrorIs(t, err, ErrUsernameTaken)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestAuthService_Register_EmailTaken(t *testing.T) {
	repo := new(mockUserRepo)
	repo.On("ExistsByUsername", mock.Anything, "alice").Return(false, nil)
	repo.On("ExistsByEmail", mock.Anything, "alice@example.com").Return(true, nil)

	svc := NewAuthService(repo, newTestTokenMgr())
	_, err := svc.Register(context.Background(), "alice", "alice@example.com", "password123")

	assert.ErrorIs(t, err, ErrEmailTaken)
	repo.AssertNotCalled(t, "Create", mock.Anything, mock.Anything)
}

func TestAuthService_Register_CreateFails(t *testing.T) {
	repo := new(mockUserRepo)
	repo.On("ExistsByUsername", mock.Anything, "alice").Return(false, nil)
	repo.On("ExistsByEmail", mock.Anything, "alice@example.com").Return(false, nil)
	repo.On("Create", mock.Anything, mock.AnythingOfType("*model.User")).Return(errors.New("db down"))

	svc := NewAuthService(repo, newTestTokenMgr())
	_, err := svc.Register(context.Background(), "alice", "alice@example.com", "password123")

	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrUsernameTaken, "非业务错误不该被误判成业务错误")
}

func TestAuthService_Login_Success(t *testing.T) {
	user := &model.User{ID: 1, Username: "alice", Role: model.RoleUser}
	require.NoError(t, user.HashPassword("password123"))

	repo := new(mockUserRepo)
	repo.On("FindByUsername", mock.Anything, "alice").Return(user, nil)

	svc := NewAuthService(repo, newTestTokenMgr())
	loggedIn, token, err := svc.Login(context.Background(), "alice", "password123")

	require.NoError(t, err)
	assert.Equal(t, user.ID, loggedIn.ID)
	assert.NotEmpty(t, token)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	user := &model.User{ID: 1, Username: "alice", Role: model.RoleUser}
	require.NoError(t, user.HashPassword("password123"))

	repo := new(mockUserRepo)
	repo.On("FindByUsername", mock.Anything, "alice").Return(user, nil)

	svc := NewAuthService(repo, newTestTokenMgr())
	_, _, err := svc.Login(context.Background(), "alice", "wrong-password")

	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	repo := new(mockUserRepo)
	repo.On("FindByUsername", mock.Anything, "nobody").Return(nil, gorm.ErrRecordNotFound)

	svc := NewAuthService(repo, newTestTokenMgr())
	_, _, err := svc.Login(context.Background(), "nobody", "password123")

	assert.ErrorIs(t, err, ErrInvalidCredentials, "用户不存在也应该统一报'用户名或密码错误'，不能泄露用户是否存在")
}

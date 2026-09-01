package service

import (
	"context"
	"errors"
	"fmt"

	"homework04/internal/model"
	"homework04/internal/pkg/jwtutil"

	"gorm.io/gorm"
)

// authUserRepository 是 AuthService 依赖的 repository 能力子集，在本包（消费方）声明，
// 不在 repository 包声明接口（见 CLAUDE.md"编码规范"第 3 条）。
type authUserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	ExistsByUsername(ctx context.Context, username string) (bool, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// AuthService 处理注册、登录，签发 JWT。
type AuthService struct {
	repo     authUserRepository
	tokenMgr *jwtutil.Manager
}

// NewAuthService 创建 AuthService。
func NewAuthService(repo authUserRepository, tokenMgr *jwtutil.Manager) *AuthService {
	return &AuthService{repo: repo, tokenMgr: tokenMgr}
}

// Register 注册新用户，用户名/邮箱唯一性在这里校验（需要查库，dto 的 binding tag 做不到）。
func (s *AuthService) Register(ctx context.Context, username, email, password string) (*model.User, error) {
	exists, err := s.repo.ExistsByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("check username existence: %w", err)
	}
	if exists {
		return nil, ErrUsernameTaken
	}

	exists, err = s.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("check email existence: %w", err)
	}
	if exists {
		return nil, ErrEmailTaken
	}

	user := &model.User{
		Username: username,
		Email:    email,
		Role:     model.RoleUser,
	}
	if err := user.HashPassword(password); err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

// Login 校验用户名密码，成功后返回用户信息和签发的 JWT。
func (s *AuthService) Login(ctx context.Context, username, password string) (*model.User, string, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", fmt.Errorf("find user by username: %w", err)
	}
	if !user.CheckPassword(password) {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.tokenMgr.Generate(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}
	return user, token, nil
}

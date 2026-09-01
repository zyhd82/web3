package service

import (
	"context"
	"errors"
	"fmt"

	"homework04/internal/model"

	"gorm.io/gorm"
)

// profileRepository 是 UserService 依赖的 repository 能力子集，只包含它实际用到的方法
// （比 authUserRepository 需要的方法更少，两者分开声明，见 CLAUDE.md"编码规范"第 3 条）。
type profileRepository interface {
	FindByID(ctx context.Context, id uint) (*model.User, error)
	Update(ctx context.Context, user *model.User) error
}

// UserService 处理登录用户自己的资料：查看、更新资料、改密码。
type UserService struct {
	repo profileRepository
}

// NewUserService 创建 UserService。
func NewUserService(repo profileRepository) *UserService {
	return &UserService{repo: repo}
}

// GetProfile 查询用户资料。
func (s *UserService) GetProfile(ctx context.Context, userID uint) (*model.User, error) {
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateProfile 更新用户资料，nickname/avatar/bio 为 nil 表示不修改该字段。
func (s *UserService) UpdateProfile(ctx context.Context, userID uint, nickname, avatar, bio *string) (*model.User, error) {
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	if nickname != nil {
		user.Nickname = *nickname
	}
	if avatar != nil {
		user.Avatar = *avatar
	}
	if bio != nil {
		user.Bio = *bio
	}

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return user, nil
}

// ChangePassword 校验旧密码后更新为新密码。
func (s *UserService) ChangePassword(ctx context.Context, userID uint, oldPassword, newPassword string) error {
	user, err := s.findUser(ctx, userID)
	if err != nil {
		return err
	}

	if !user.CheckPassword(oldPassword) {
		return ErrInvalidCredentials
	}

	if err := user.HashPassword(newPassword); err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.repo.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	return nil
}

// findUser 是内部共用的查询逻辑，把 gorm.ErrRecordNotFound 翻译成业务错误 ErrUserNotFound。
func (s *UserService) findUser(ctx context.Context, userID uint) (*model.User, error) {
	user, err := s.repo.FindByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return user, nil
}

package repository

import (
	"context"

	"homework04/internal/model"

	"gorm.io/gorm"
)

// UserRepository 是 users 表的数据访问实现，方法集由各 service 包按需通过自己声明的接口消费
// （不在本包声明接口，见 CLAUDE.md"编码规范"第 3 条）。
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建 UserRepository。
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create 插入一条新用户记录。
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update 保存用户的全部字段，调用前需先查出完整记录再修改，避免把未涉及的字段清空。
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete 软删除指定用户。
func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, id).Error
}

// FindByID 按主键查询用户，不存在时返回 gorm.ErrRecordNotFound。
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByUsername 按用户名查询用户，不存在时返回 gorm.ErrRecordNotFound。
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail 按邮箱查询用户，不存在时返回 gorm.ErrRecordNotFound。
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// ExistsByUsername 判断用户名是否已被占用（注册时的唯一性校验）。
func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

// ExistsByEmail 判断邮箱是否已被占用（注册时的唯一性校验）。
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

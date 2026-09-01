package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// 用户角色取值。
const (
	RoleUser  = "user"  // 普通用户，只能管理自己的文章/评论
	RoleAdmin = "admin" // 管理员，可管理任意用户的文章/评论
)

// User 对应 users 表，存储用户账号信息。
type User struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Username  string         `json:"username" gorm:"type:varchar(50);uniqueIndex;not null"`
	Email     string         `json:"email" gorm:"type:varchar(255);uniqueIndex;not null"`
	Password  string         `json:"-" gorm:"type:varchar(100);not null"`
	Role      string         `json:"role" gorm:"type:varchar(20);not null;default:user"`
	Nickname  string         `json:"nickname" gorm:"type:varchar(50)"`
	Avatar    string         `json:"avatar" gorm:"type:varchar(255)"`
	Bio       string         `json:"bio" gorm:"type:varchar(255)"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

// IsAdmin 判断用户是否为管理员角色。
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// HashPassword 用 bcrypt 对明文密码加密并覆盖 u.Password。
// 不用 GORM 的 BeforeCreate/BeforeSave 钩子实现，而是由 service 层在注册、改密等
// 明确需要改密码的场景显式调用，避免更新用户其它字段时被钩子意外重新加密。
func (u *User) HashPassword(plain string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hashed)
	return nil
}

// CheckPassword 校验明文密码是否与已保存的哈希匹配。
func (u *User) CheckPassword(plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plain)) == nil
}

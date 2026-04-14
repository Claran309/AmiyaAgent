package repository

import (
	"time"
	"gorm.io/gorm"
)

type User struct {
	IP            string      `gorm:"primaryKey"`
	Username      string    `gorm:"uniqueIndex;size:64"`
	Points        int       `gorm:"default:0"`           // 积分
	LastSignInAt  *time.Time // 上次签到时间
}

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetUser 获取或创建用户
func (r *UserRepository) GetUser(IP string) (*User, error) {
	var user User
	err := r.db.FirstOrCreate(&user, User{IP: IP}).Error
	return &user, err
}

// UpdateSignIn 更新签到信息 
func (r *UserRepository) UpdateSignIn(user *User, addPoints int) error {
	return r.db.Model(user).Updates(map[string]interface{}{
		"points":          user.Points + addPoints,
		"last_sign_in_at": time.Now(),
	}).Error
}
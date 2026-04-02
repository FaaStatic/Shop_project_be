package domain

import (
	"context"
	"shop_project_be/internal/dto/request"
)

type Users struct {
	ID           uint   `gorm:"primaryKey" json:"id"`
	Username     string `gorm:"type:varchar(100);uniqueIndex;not null" json:"username"`
	PasswordHash string `gorm:"type:varchar(255);not null" json:"-"`
	Role         string `gorm:"type:enum('superadmin','admin','staff');default:'staff'" json:"role"`

	Transactions []Transactions `gorm:"foreignKey:UserID" json:"transactions,omitempty"`
}

type UserRepository interface {
	GetUserLogin(ctx *context.Context, user *Users) error
	RegisterUser(ctx *context.Context, user *Users) error
}

type UserUsecase interface {
	UserLogin(ctx *context.Context, userDto *request.UserLoginRequest) error
	RegisterUser(ctx *context.Context, userDto *request.UserRegisterRequest) error
}

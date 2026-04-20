package domain

import (
	"context"
	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/dto/request"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Users struct {
	Id           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Username     string         `gorm:"type:varchar(100);uniqueIndex;not null" json:"username"`
	Password     string         `gorm:"type:varchar(255);not null" json:"-"`
	Role         enum.UserRole  `gorm:"type:smallint;check:role IN (0,1,2);default:2" json:"role"`
	Transactions []Transactions `gorm:"foreignKey:UserID" json:"transactions,omitempty"`
}

func (user *Users) HashPswd() error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashed)
	return nil
}

func (user *Users) ComparedPwd(pwdUser string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(pwdUser))
	return err == nil
}

func (u *Users) TableName() string {
	return "users"
}

type UserRepository interface {
	GetUserLogin(ctx *context.Context, user *Users) error
	RegisterUser(ctx *context.Context, user *Users) error
}

type UserUsecase interface {
	UserLogin(ctx *context.Context, userDto *request.UserLoginRequest) error
	RegisterUser(ctx *context.Context, userDto *request.UserRegisterRequest) error
}

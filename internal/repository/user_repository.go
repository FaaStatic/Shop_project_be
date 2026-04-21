package repository

import (
	"context"
	"shop_project_be/internal/domain"

	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

// GetUserLogin implements [domain.UserRepository].
func (u *userRepository) GetUserLogin(ctx context.Context, user *domain.Users) error {
	panic("unimplemented")
}

// RegisterUser implements [domain.UserRepository].
func (u *userRepository) RegisterUser(ctx context.Context, user *domain.Users) error {
	panic("unimplemented")
}

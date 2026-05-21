package repository

import (
	"context"
	"errors"
	"fmt"
	"shop_project_be/internal/domain"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

// GetUserById implements [domain.UserRepository].
func (u *userRepository) GetUserById(ctx context.Context, id uuid.UUID) (*domain.Users, error) {
	var userData domain.Users
	result := u.db.WithContext(ctx).Where("id = ?", id).First(&userData)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("user not found: %w", result.Error)
		}
		return nil, fmt.Errorf("failed to get user: %w", result.Error)
	}
	return &userData, nil
}

// GetUserByUsername implements [domain.UserRepository].
func (u *userRepository) GetUserByUsername(ctx context.Context, username string) (*domain.Users, error) {
	var userData domain.Users
	result := u.db.WithContext(ctx).Where("username = ?", username).First(&userData)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("user not found")
	}

	return &userData, nil
}

// GetUserLogin implements [domain.UserRepository].
func (u *userRepository) GetUserLogin(ctx context.Context, id uuid.UUID) (*domain.Users, error) {
	var userData domain.Users
	result := u.db.WithContext(ctx).Where("id = ?", id).First(&userData)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("item tidak ditemukan")
		}
		return nil, result.Error
	}

	return &userData, nil
}

// RegisterUser implements [domain.UserRepository].
func (u *userRepository) RegisterUser(ctx context.Context, user *domain.Users) error {
	result := u.db.WithContext(ctx).Create(user)
	if result.Error != nil {
		return fmt.Errorf("Failed create user: %w", result.Error)
	}
	return nil
}

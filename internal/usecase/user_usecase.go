package usecase

import (
	"context"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"

	"go.uber.org/zap"
)

type userUsecase struct {
	userRepo domain.UserRepository
	log      *zap.Logger
}

func NewUserUsecase(userRepo domain.UserRepository, log *zap.Logger) domain.UserUsecase {
	return &userUsecase{
		userRepo: userRepo,
		log:      log,
	}
}

// RegisterUser implements [domain.UserUsecase].
func (u *userUsecase) RegisterUser(ctx context.Context, userDto *requestdto.UserRegisterRequest) (*responsedto.UserRegisterResponse, error) {
	panic("unimplemented")
}

// UserLogin implements [domain.UserUsecase].
func (u *userUsecase) UserLogin(ctx context.Context, userDto *requestdto.UserLoginRequest) (*responsedto.UserLoginResponse, error) {
	panic("unimplemented")
}

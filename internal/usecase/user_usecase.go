package usecase

import (
	"context"
	"fmt"
	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"
	"shop_project_be/pkg/jwt"
	"time"

	"go.uber.org/zap"
)

type userUsecase struct {
	userRepo    domain.UserRepository
	sessionRepo domain.SessionRepository
	log         *zap.Logger
	jwtService  *jwt.JWTService
}

func NewUserUsecase(userRepo domain.UserRepository, sessionRepo domain.SessionRepository, log *zap.Logger, jwtService *jwt.JWTService) domain.UserUsecase {
	return &userUsecase{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		log:         log,
		jwtService:  jwtService,
	}
}

// RegisterUser implements [domain.UserUsecase].
func (u *userUsecase) RegisterUser(ctx context.Context, userDto *requestdto.UserRegisterRequest) (*responsedto.UserRegisterResponse, error) {
	existing, err := u.userRepo.GetUserByUsername(ctx, userDto.Username)
	if err != nil {
		u.log.Error("user already exists", zap.Error(err))
		return &responsedto.UserRegisterResponse{
			Message: "internal server error",
			Status:  500,
		}, fmt.Errorf("internal server error")
	}
	if existing != nil {
		u.log.Error("user already exists", zap.Error(err))

		return &responsedto.UserRegisterResponse{
			Message: "user already exists",
			Status:  409,
		}, fmt.Errorf("user already exists")
	}

	roleEnum, err := enum.ParseUserRole(userDto.Role)
	if err != nil {
		u.log.Error("Error Parsing Role", zap.Error(err))
		return &responsedto.UserRegisterResponse{
			Message: "role invalid",
			Status:  400,
		}, fmt.Errorf("role invalid")
	}

	user := &domain.Users{
		Username: userDto.Username,
		Password: userDto.Password,
		Role:     roleEnum,
	}
	err = user.HashPswd()
	if err != nil {
		u.log.Error("Error Hashing Password", zap.Error(err))
		return &responsedto.UserRegisterResponse{
			Message: "internal server error",
			Status:  500,
		}, fmt.Errorf("internal server error")
	}
	err = u.userRepo.RegisterUser(ctx, user)
	if err != nil {
		return &responsedto.UserRegisterResponse{
			Message: "register Failed",
			Status:  500,
		}, err
	}

	return &responsedto.UserRegisterResponse{
		Message: "register success",
		Status:  201,
	}, nil

}

// UserLogin implements [domain.UserUsecase].
func (u *userUsecase) UserLogin(ctx context.Context, userDto *requestdto.UserLoginRequest) (*responsedto.UserLoginResponse, error) {
	user, err := u.userRepo.GetUserByUsername(ctx, userDto.Username)
	if err != nil {
		u.log.Error("user already exists", zap.Error(err))
		return nil, fmt.Errorf("internal server error")
	}
	if user == nil {
		u.log.Error("user not found", zap.Error(err))
		return nil, fmt.Errorf("user not found")
	}
	if !user.ComparedPwd(userDto.Password) {
		u.log.Error("wrong password", zap.Error(err))
		return nil, fmt.Errorf("wrong password")
	}
	roleUser, err := enum.ParseUserRole(user.Role.String())
	if err != nil {
		u.log.Error("error parsing role", zap.Error(err))
		return nil, fmt.Errorf("internal server error")
	}
	tokenPair, err := u.jwtService.GenerateTokenPair(user.ID.String(), roleUser.String())
	if err != nil {
		u.log.Error("error gen token", zap.Error(err))
		return nil, fmt.Errorf("internal server error")
	}

	sessionKey := "session:" + tokenPair.AccessToken
	session := &domain.Session{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		UserID:       user.ID.String(),
		Role:         roleUser.String(),
		ExpiresAt:    time.Now().Add(time.Duration(tokenPair.ExpiresIn) * time.Second),
	}

	ttl := time.Duration(tokenPair.ExpiresIn) * time.Second
	if err := u.sessionRepo.CreateSession(ctx, session, sessionKey, ttl); err != nil {
		u.log.Error("error save session", zap.Error(err))
		return nil, fmt.Errorf("internal server error")
	}

	// Tandai user online (non-fatal: login tetap sukses bila penanda gagal).
	online := domain.OnlineUser{
		UserID:   user.ID.String(),
		Username: user.Username,
		Role:     roleUser.String(),
	}
	if err := u.sessionRepo.SetUserOnline(ctx, online, ttl); err != nil {
		u.log.Warn("failed to mark user online", zap.Error(err))
	}

	return &responsedto.UserLoginResponse{
		ID:           user.ID.String(),
		Username:     user.Username,
		Role:         roleUser.String(),
		Token:        tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiredTime:  int64(tokenPair.ExpiresIn),
	}, nil

}

// Logout implements [domain.UserUsecase].
// Menghapus session (access token) dari Redis dan penanda online user, sehingga
// token tidak bisa dipakai lagi dan user tidak lagi terhitung online.
func (u *userUsecase) Logout(ctx context.Context, accessToken, userID string) error {
	sessionKey := "session:" + accessToken
	if err := u.sessionRepo.DeleteSessionByAccessToken(ctx, sessionKey); err != nil {
		u.log.Error("failed to delete session", zap.Error(err))
		return fmt.Errorf("failed to logout")
	}
	// Hapus penanda online (non-fatal).
	if err := u.sessionRepo.RemoveUserOnline(ctx, userID); err != nil {
		u.log.Warn("failed to remove online marker", zap.Error(err))
	}
	return nil
}

// ListOnlineUsers implements [domain.UserUsecase].
func (u *userUsecase) ListOnlineUsers(ctx context.Context) ([]domain.OnlineUser, error) {
	users, err := u.sessionRepo.ListOnlineUsers(ctx)
	if err != nil {
		u.log.Error("failed to list online users", zap.Error(err))
		return nil, fmt.Errorf("failed to get online users")
	}
	return users, nil
}

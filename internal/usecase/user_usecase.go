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
	"golang.org/x/crypto/bcrypt"
)

// dummyPasswordHash is used by UserLogin to equalize compute time when the
// username does not exist: bcrypt is still run so a user's existence does not
// leak via timing differences. Generated once when the package loads.
var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("timing-equalizer-not-a-real-password"), bcrypt.DefaultCost)

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

	// Public register is staff-only; admin/superadmin are created directly via the DB.
	roleEnum, _ := enum.ParseUserRole("staff")

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
		// Run a dummy bcrypt so the duration matches the "user exists" path -> username
		// existence does not leak via timing. The message is unified with the
		// wrong-password case -> no enumeration via message content.
		bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(userDto.Password))
		u.log.Error("user not found", zap.Error(err))
		return nil, fmt.Errorf("username atau password salah")
	}
	if !user.ComparedPwd(userDto.Password) {
		u.log.Error("wrong password", zap.Error(err))
		return nil, fmt.Errorf("username atau password salah")
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

	if err := u.sessionRepo.CreateSession(ctx, session, sessionKey, time.Duration(tokenPair.ExpiresIn)*time.Second); err != nil {
		u.log.Error("error save session", zap.Error(err))
		return nil, fmt.Errorf("internal server error")
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

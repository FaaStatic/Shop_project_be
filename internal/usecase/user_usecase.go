package usecase

import (
	"context"
	"errors"
	"fmt"
	"shop_project_be/internal/constant/enum"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	responsedto "shop_project_be/internal/dto/response_dto"
	"shop_project_be/pkg/jwt"
	"time"

	"github.com/google/uuid"
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
		u.log.Error("error get user by username", zap.Error(err))
		return &responsedto.UserRegisterResponse{
			Message: "internal server error",
			Status:  500,
		}, fmt.Errorf("internal server error")
	}
	if existing != nil {
		u.log.Error("user already exists")

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
		u.log.Error("error get user by username", zap.Error(err))
		return nil, fmt.Errorf("internal server error")
	}
	if user == nil {
		// Run a dummy bcrypt so the duration matches the "user exists" path -> username
		// existence does not leak via timing. The message is unified with the
		// wrong-password case -> no enumeration via message content.
		bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(userDto.Password))
		u.log.Error("user not found")
		return nil, fmt.Errorf("username atau password salah")
	}
	if !user.ComparedPwd(userDto.Password) {
		u.log.Error("wrong password")
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
	if err := u.sessionRepo.CreateSession(ctx, session, "refresh:"+tokenPair.RefreshToken, u.jwtService.RefreshTokenTTL()); err != nil {
		u.log.Error("error save refresh session", zap.Error(err))
		_ = u.sessionRepo.DeleteSessionByAccessToken(ctx, sessionKey)
		return nil, fmt.Errorf("internal server error")
	}

	return &responsedto.UserLoginResponse{
		ID:           user.ID,
		Username:     user.Username,
		Role:         roleUser.String(),
		Token:        tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int64(tokenPair.ExpiresIn),
	}, nil

}

// RefreshToken implements [domain.UserUsecase]. It validates the refresh token
// (JWT type "refresh" + an active Redis session), rotates the pair, and returns a
// fresh access/refresh token without re-authentication.
func (u *userUsecase) RefreshToken(ctx context.Context, refreshDto *requestdto.UserRefreshTokenRequest) (*responsedto.UserLoginResponse, error) {
	claims, err := u.jwtService.ValidateToken(refreshDto.RefreshToken)
	if err != nil || claims.Type != "refresh" {
		return nil, fmt.Errorf("invalid refresh token")
	}

	refreshKey := "refresh:" + refreshDto.RefreshToken
	// Pop atomically (GETDEL) so two concurrent refreshes with the same token
	// cannot both succeed — only one caller receives the session.
	session, err := u.sessionRepo.PopSessionByRefreshToken(ctx, refreshKey)
	if err != nil {
		u.log.Error("error pop refresh session", zap.Error(err))
		return nil, fmt.Errorf("internal server error")
	}
	if session == nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	userID, err := uuid.Parse(session.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}
	user, err := u.userRepo.GetUserById(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, fmt.Errorf("invalid refresh token")
		}
		u.log.Error("error get user", zap.Error(err))
		return nil, fmt.Errorf("internal server error")
	}
	if user == nil {
		return nil, fmt.Errorf("invalid refresh token")
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

	// The refresh session was already popped above; drop the old access session.
	if session.AccessToken != "" {
		if err := u.sessionRepo.DeleteSessionByAccessToken(ctx, "session:"+session.AccessToken); err != nil {
			u.log.Error("error delete old session", zap.Error(err))
		}
	}

	newSession := &domain.Session{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		UserID:       user.ID.String(),
		Role:         roleUser.String(),
		ExpiresAt:    time.Now().Add(time.Duration(tokenPair.ExpiresIn) * time.Second),
	}
	if err := u.sessionRepo.CreateSession(ctx, newSession, "session:"+tokenPair.AccessToken, time.Duration(tokenPair.ExpiresIn)*time.Second); err != nil {
		u.log.Error("error save session", zap.Error(err))
		return nil, fmt.Errorf("internal server error")
	}
	if err := u.sessionRepo.CreateSession(ctx, newSession, "refresh:"+tokenPair.RefreshToken, u.jwtService.RefreshTokenTTL()); err != nil {
		u.log.Error("error save refresh session", zap.Error(err))
		_ = u.sessionRepo.DeleteSessionByAccessToken(ctx, "session:"+tokenPair.AccessToken)
		return nil, fmt.Errorf("internal server error")
	}

	return &responsedto.UserLoginResponse{
		ID:           user.ID,
		Username:     user.Username,
		Role:         roleUser.String(),
		Token:        tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    int64(tokenPair.ExpiresIn),
	}, nil
}

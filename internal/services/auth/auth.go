package auth

import (
	"context"
	"time"

	"github.com/Heidric/create-your-fantasy/internal/lib/jwt"
	"github.com/Heidric/create-your-fantasy/internal/logger"
	"github.com/Heidric/create-your-fantasy/internal/model"
	"github.com/Heidric/create-your-fantasy/internal/storage"
	"github.com/Heidric/create-your-fantasy/pkg/security"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

var log zerolog.Logger

const (
	adminAccessTokenTTL     = time.Minute * 10
	moderatorAccessTokenTTL = time.Minute * 30
	playerAccessTokenTTL    = time.Hour * 6
	refreshTokenTTL         = time.Hour * 24 * 30
)

var (
	ErrTokenNotFound      = errors.New("token not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailNotFound      = errors.New("email not found")
	ErrEmailNotUnique     = errors.New("email not unique")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type AuthStorage interface {
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	GetUserByID(ctx context.Context, ID string) (*model.User, error)
	RegisterUser(ctx context.Context, user *model.RegisterUserDTO) (*string, error)
	CreateSession(ctx context.Context, session *model.Session) error
	GetSessionBySID(ctx context.Context, sID string) (*model.Session, error)
	GetSessionByRToken(ctx context.Context, rToken string) (*model.Session, error)
	DeleteSessionByRToken(ctx context.Context, rToken string) error
	DeleteSessionsByUserID(ctx context.Context, userID string) error
	SetNewPassword(ctx context.Context, userID string, password string, temporary bool) error
}

type Auth struct {
	storage AuthStorage
}

func (a *Auth) ValidateSession(ctx context.Context) error {
	claims := ctx.Value(jwt.CtxKeyClaims).(model.UserClaim)
	token := ctx.Value(jwt.CtxKeyToken).(string)

	session, err := a.storage.GetSessionBySID(ctx, claims.SID)
	if err != nil {
		return errors.Wrap(err, "session not found")
	}

	if session.UserID != claims.ID {
		return errors.New("session not valid")
	}

	if token != session.AccessToken {
		return errors.New("token not valid")
	}

	return nil
}

func New(storage AuthStorage) *Auth {
	log = *logger.Log
	log = log.With().Str("name", "auth-service").Logger()

	return &Auth{storage: storage}
}

func (a *Auth) Register(ctx context.Context, email string) error {
	_, err := a.storage.GetUserByEmail(ctx, email)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrEntityNotFound):
			break
		default:
			return errors.Wrap(err, "get user by email")
		}
	} else {
		return ErrEmailNotUnique
	}

	_, err = a.storage.RegisterUser(ctx, &model.RegisterUserDTO{Email: email})
	if err != nil {
		return errors.Wrap(err, "create user")
	}

	return nil
}

func (a *Auth) Login(ctx context.Context, email, password string) (*model.LoginResponse, error) {
	user, err := a.storage.GetUserByEmail(ctx, email)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrEntityNotFound):
			return nil, ErrInvalidCredentials
		default:
			return nil, errors.Wrap(err, "login")
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	sID := uuid.New().String()

	var ttl time.Duration

	switch user.Role {
	case model.Admin:
		ttl = adminAccessTokenTTL
		break
	case model.Moderator:
		ttl = moderatorAccessTokenTTL
		break
	case model.Player:
	default:
		ttl = playerAccessTokenTTL
		break
	}

	accessToken, err := jwt.NewToken(model.JwtDTO{
		ID:                user.ID,
		Role:              user.Role,
		SID:               sID,
		PasswordTemporary: user.PasswordTemporary,
	}, ttl)
	if err != nil {
		log.Error().Msgf("failed to generate access token: %v", err)
		return nil, errors.Wrap(err, "generate access token")
	}
	refreshToken, err := jwt.NewToken(model.JwtDTO{
		ID:  user.ID,
		SID: sID,
	}, refreshTokenTTL)
	if err != nil {
		log.Error().Msgf("failed to generate refresh token: %v", err)
		return nil, errors.Wrap(err, "generate refresh token")
	}
	refreshTokenExpirationDate := time.Now().Add(refreshTokenTTL)

	if err := a.storage.CreateSession(ctx, &model.Session{
		ID:           sID,
		UserID:       user.ID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    refreshTokenExpirationDate,
	}); err != nil {
		return nil, errors.Wrap(err, "create session")
	}

	return &model.LoginResponse{
		TokenType:    "Bearer",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *Auth) RefreshToken(ctx context.Context, refreshToken string) (*model.RefreshTokenResponse, error) {
	token, err := a.storage.GetSessionByRToken(ctx, refreshToken)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrEntityNotFound):
			return nil, ErrTokenNotFound
		default:
			return nil, errors.Wrap(err, "refresh token")
		}
	}

	user, err := a.storage.GetUserByID(ctx, token.UserID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrEntityNotFound):
			return nil, ErrUserNotFound
		default:
			return nil, errors.Wrap(err, "refresh token")
		}
	}

	sID := uuid.New()

	var ttl time.Duration

	switch user.Role {
	case model.Admin:
		ttl = adminAccessTokenTTL
		break
	case model.Moderator:
		ttl = moderatorAccessTokenTTL
		break
	case model.Player:
	default:
		ttl = playerAccessTokenTTL
		break
	}

	accessToken, err := jwt.NewToken(model.JwtDTO{
		ID:   user.ID,
		Role: user.Role,
		SID:  sID.String(),
	}, ttl)
	if err != nil {
		log.Error().Msgf("failed to generate access token: %v", err)
		return nil, errors.Wrap(err, "generate access token")
	}
	refreshToken, err = jwt.NewToken(model.JwtDTO{
		ID:   user.ID,
		Role: user.Role,
		SID:  sID.String(),
	}, refreshTokenTTL)
	if err != nil {
		log.Error().Msgf("failed to generate refresh token: %v", err)
		return nil, errors.Wrap(err, "generate refresh token")
	}
	refreshTokenExpirationDate := time.Now().Add(refreshTokenTTL)

	err = a.storage.DeleteSessionByRToken(ctx, refreshToken)
	if err != nil {
		return nil, errors.Wrap(err, "delete sessions")
	}

	if err = a.storage.CreateSession(ctx, &model.Session{
		ID:           sID.String(),
		UserID:       user.ID,
		ExpiresAt:    refreshTokenExpirationDate,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}); err != nil {
		return nil, errors.Wrap(err, "create session")
	}

	return &model.RefreshTokenResponse{
		TokenType:    "Bearer",
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *Auth) ChangePassword(ctx context.Context, password, newPassword string) error {
	claims := ctx.Value(jwt.CtxKeyClaims).(model.UserClaim)
	err := a.storage.DeleteSessionsByUserID(ctx, claims.ID)
	if err != nil {
		return errors.Wrap(err, "delete sessions")
	}

	err = a.storage.SetNewPassword(ctx, claims.ID, newPassword, false)
	if err != nil {
		return errors.Wrap(err, "set new password")
	}

	return nil
}

func (a *Auth) ResetPassword(ctx context.Context, email string) error {
	user, err := a.storage.GetUserByEmail(ctx, email)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrEntityNotFound):
			return ErrUserNotFound
		default:
			return errors.Wrap(err, "reset password")
		}
	}

	err = a.storage.DeleteSessionsByUserID(ctx, user.ID)
	if err != nil {
		return errors.Wrap(err, "reset password")
	}

	password, err := security.GenerateSecureString(8)

	err = a.storage.SetNewPassword(ctx, user.ID, password, true)
	if err != nil {
		return errors.Wrap(err, "reset password")
	}

	// TODO: send mail

	return nil
}

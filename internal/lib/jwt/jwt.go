package jwt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Heidric/create-your-fantasy/internal/logger"
	"github.com/Heidric/create-your-fantasy/internal/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

type ctxKey string

const (
	CtxKeyClaims ctxKey = "claims"
	CtxKeyToken  ctxKey = "token"
)

var (
	audience string
	issuer   string
	secret   string
	log      zerolog.Logger
)

func Initialize(cfg *Config) {
	audience = cfg.Audience
	issuer = cfg.Issuer
	secret = cfg.Secret
	log = *logger.Log
	log = log.With().Str("name", "jwt").Logger()
}

func NewToken(dto model.JwtDTO, duration time.Duration) (string, error) {
	claims := &model.UserClaim{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			Audience:  jwt.ClaimStrings{audience},
		},
		ID:                dto.ID,
		Role:              dto.Role,
		SID:               dto.SID,
		PasswordTemporary: dto.PasswordTemporary,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func Verify(token string) (*model.UserClaim, error) {
	t, err := jwt.ParseWithClaims(
		token,
		&model.UserClaim{},
		func(token *jwt.Token) (interface{}, error) {
			_, ok := token.Method.(*jwt.SigningMethodHMAC)
			if !ok {
				return nil, fmt.Errorf("unexpected signing method")
			}

			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	claims, ok := t.Claims.(*model.UserClaim)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	if err := verifyAudience(*claims); err != nil {
		return nil, err
	}

	return claims, nil
}

func verifyAudience(claim model.UserClaim) error {
	value, err := claim.GetAudience()
	if err != nil {
		return err
	}

	if len(value) == 0 {
		return errors.New("audience is empty")
	}

	for _, v := range value {
		if v == audience {
			return nil
		}
	}

	return errors.New("audience is wrong")
}

type Verifier interface {
	ValidateSession(ctx context.Context) error
}

func Authenticator(v Verifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			const prefix = "Bearer "
			bearer := r.Header.Get("Authorization")
			if !(len(bearer) > 7 && strings.ToUpper(bearer[0:6]) == "BEARER") {
				unauthorizedError(w, "No token provided")
				return
			}
			if !strings.HasPrefix(strings.ToLower(bearer), strings.ToLower(prefix)) {
				unauthorizedError(w, "No token provided")
				return
			}
			token := strings.TrimSpace(bearer[len(prefix):])
			if token == "" {
				unauthorizedError(w, "No token provided")
				return
			}

			claims, err := Verify(token)
			if err != nil {
				unauthorizedError(w, err.Error())
				return
			}

			ctx := context.WithValue(r.Context(), CtxKeyClaims, *claims)
			ctx = context.WithValue(ctx, CtxKeyToken, token)

			if err := v.ValidateSession(ctx); err != nil {
				unauthorizedError(w, err.Error())
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

func unauthorizedError(w http.ResponseWriter, detail string) {
	res := struct {
		Title  string `json:"title"`
		Status int    `json:"status"`
		Detail string `json:"detail"`
		Code   string `json:"code"`
	}{
		Title:  "Unauthorized",
		Status: http.StatusUnauthorized,
		Detail: detail,
		Code:   "UNAUTHORIZED",
	}

	log.Error().Msg(detail)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(res)
}

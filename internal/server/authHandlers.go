package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Heidric/create-your-fantasy/internal/model"
	"github.com/Heidric/create-your-fantasy/internal/services/auth"
)

func (s *Server) registerHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var dto model.RegisterDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		log.Error().Err(err).Msg("Error parsing request body")
		ParsingError(w)
		return
	}

	if err := dto.Validate(); len(err) > 0 {
		log.Error().Msgf("Error validating request body: %v", err)
		ValidationError(w, err)
		return
	}

	err := s.auth.Register(ctx, dto.Email)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailNotUnique):
			LogicError(w, ErrEmailNotUnique)
		default:
			InternalError(w)
		}
		log.Error().Err(err).Msg("Error login")
		return
	}
}

func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var dto model.LoginDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		log.Error().Err(err).Msg("Error parsing request body")
		ParsingError(w)
		return
	}

	if err := dto.Validate(); len(err) > 0 {
		log.Error().Msgf("Error validating request body: %v", err)
		ValidationError(w, err)
		return
	}

	res, err := s.auth.Login(ctx, dto.Email, dto.Password)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUserNotFound):
			LogicError(w, ErrUserNotFound)
		case errors.Is(err, auth.ErrInvalidCredentials):
			UnauthorizedError(w)
		default:
			InternalError(w)
		}
		log.Error().Err(err).Msg("Error login")
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Error().Err(err).Msg("Error encoding response")
		InternalError(w)
		return
	}
}

func (s *Server) refreshTokenHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var dto model.RefreshTokenDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		ParsingError(w)
		return
	}

	if err := dto.Validate(); len(err) > 0 {
		log.Error().Msgf("Error validating request body: %v", err)
		ValidationError(w, err)
		return
	}

	res, err := s.auth.RefreshToken(ctx, dto.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrTokenNotFound):
			LogicError(w, ErrTokenInvalid)
		case errors.Is(err, auth.ErrInvalidCredentials):
			UnauthorizedError(w)
		default:
			InternalError(w)
		}
		log.Error().Err(err).Msg("Error refreshing token")
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Error().Err(err).Msg("Error encoding response")
		InternalError(w)
		return
	}
}

func (s *Server) changePasswordHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var dto model.ChangePasswordDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		log.Error().Err(err).Msg("Error parsing request body")
		ParsingError(w)
		return
	}

	if err := dto.Validate(); len(err) > 0 {
		log.Error().Msgf("Error validating request body: %v", err)
		ValidationError(w, err)
		return
	}

	err := s.auth.ChangePassword(ctx, dto.Password, dto.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			UnauthorizedError(w)
		default:
			InternalError(w)
		}
		log.Error().Err(err).Msg("Error changing password")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) resetPasswordHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var dto model.ResetPasswordDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		log.Error().Err(err).Msg("Error parsing request body")
		ParsingError(w)
		return
	}

	if err := dto.Validate(); len(err) > 0 {
		log.Error().Msgf("Error validating request body: %v", err)
		ValidationError(w, err)
		return
	}

	err := s.auth.ResetPassword(ctx, dto.Email)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailNotFound):
			LogicError(w, ErrEmailNotFound)
		case errors.Is(err, auth.ErrInvalidCredentials):
			UnauthorizedError(w)
		default:
			InternalError(w)
		}
		log.Error().Err(err).Msg("Error resetting password")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

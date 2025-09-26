package profile

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Heidric/create-your-fantasy/internal/logger"
	"github.com/Heidric/create-your-fantasy/internal/model"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

type Storage interface {
	GetUserByID(ctx context.Context, id string) (*model.User, error)
	CreateModerationRequest(ctx context.Context, req *model.ModerationRequest) error
}

type Service struct {
	storage Storage
}

func New(storage Storage) *Service {
	log = *logger.Log
	log = log.With().Str("name", "profile-service").Logger()
	return &Service{storage: storage}
}

func (s *Service) Get(ctx context.Context, userID string) (*model.ProfileResponse, error) {
	u, err := s.storage.GetUserByID(ctx, userID)
	if err != nil {
		return nil, errors.Wrap(err, "get user by id")
	}
	return &model.ProfileResponse{
		Email:  u.Email,
		Name:   u.Name,
		Role:   u.Role,
		Avatar: u.Avatar,
	}, nil
}

func (s *Service) Update(ctx context.Context, userID string, dto model.UpdateProfileDTO) error {
	payload := map[string]any{
		"name":        dto.Name,
		"contentType": dto.ContentType,
		"avatar":      dto.Avatar,
	}
	raw, _ := json.Marshal(payload)

	req := &model.ModerationRequest{
		ID:        uuid.NewString(),
		UserID:    userID,
		Type:      model.ModerationTypeProfileChange,
		Payload:   string(raw),
		Status:    model.ModerationStatusCreated,
		CreatedAt: time.Now(),
	}

	if err := s.storage.CreateModerationRequest(ctx, req); err != nil {
		return errors.Wrap(err, "create moderation request")
	}
	return nil
}

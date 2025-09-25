package report

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

var (
	ErrSubjectNotFound = errors.New("subject not found")
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
	log = log.With().Str("name", "report-service").Logger()
	return &Service{storage: storage}
}

func (s *Service) Create(ctx context.Context, reporterID string, dto model.CreateReportDTO) error {
	switch dto.Type {
	case "profileReport":
		if _, err := s.storage.GetUserByID(ctx, dto.SubjectID); err != nil {
			return ErrSubjectNotFound
		}
	}

	payload := map[string]any{
		"type":      dto.Type,
		"subjectId": dto.SubjectID,
		"comment":   dto.Comment,
		"reporter":  reporterID,
	}
	raw, _ := json.Marshal(payload)

	req := &model.ModerationRequest{
		ID:        uuid.NewString(),
		UserID:    reporterID,
		Type:      dto.Type,
		Payload:   string(raw),
		Status:    model.ModerationStatusCreated,
		CreatedAt: time.Now(),
	}

	if err := s.storage.CreateModerationRequest(ctx, req); err != nil {
		return errors.Wrap(err, "create moderation request")
	}
	return nil
}

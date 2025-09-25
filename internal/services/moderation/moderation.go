package moderation

import (
	"context"
	"encoding/json"
	"math"

	"github.com/Heidric/create-your-fantasy/internal/model"
	"github.com/pkg/errors"
)

const (
	DecisionChangesRequested = "changesRequested"
	DecisionSubjectApproved  = "subjectApproved"
	DecisionSubjectHidden    = "subjectHidden"
)

var (
	ErrAlreadyFinished = errors.New("already finished")
	ErrInvalidDecision = errors.New("invalid decision")
	ErrInvalidPayload  = errors.New("invalid payload")
	ErrUnsupportedType = errors.New("unsupported moderation type for approval")
)

type Storage interface {
	ListModerationRequests(ctx context.Context, filter ModerationFilter, limit, offset int) ([]DBModerationRow, error)
	CountModerationRequests(ctx context.Context, filter ModerationFilter) (int64, error)
	GetModerationByID(ctx context.Context, id string) (*DBModerationRow, error)
	UpdateModerationStatus(ctx context.Context, id string, status string, assignedTo *string) error
	UpdateUserProfile(ctx context.Context, userID, name, avatar string) error
}

type Service struct {
	storage Storage
}

func New(storage Storage) *Service { return &Service{storage: storage} }

type ModerationFilter struct {
	Status     string
	Type       string
	AssignedTo string
}

type DBModerationRow struct {
	ID         string  `db:"id"`
	Type       string  `db:"type"`
	UserID     string  `db:"user_id"`
	Payload    string  `db:"payload"`
	CreatedBy  string  `db:"user_id"`
	AssignedTo *string `db:"assigned_to"`
	Status     string  `db:"status"`
}

func (s *Service) List(ctx context.Context, q model.ModerationListQuery) (*model.ModerationListResponse, error) {
	page := 1
	if q.Page != nil {
		page = *q.Page
	}
	size := 50
	if q.PageSize != nil {
		size = *q.PageSize
	}

	filter := ModerationFilter{Status: q.Status, Type: q.Type, AssignedTo: q.AssignedTo}

	total, err := s.storage.CountModerationRequests(ctx, filter)
	if err != nil {
		return nil, errors.Wrap(err, "count")
	}

	if total == 0 {
		return &model.ModerationListResponse{Items: []model.ModerationItem{}, Page: page, PageSize: size, TotalPages: 0}, nil
	}

	offset := (page - 1) * size
	rows, err := s.storage.ListModerationRequests(ctx, filter, size, offset)
	if err != nil {
		return nil, errors.Wrap(err, "list")
	}

	items := make([]model.ModerationItem, 0, len(rows))
	for _, r := range rows {
		asg := ""
		if r.AssignedTo != nil {
			asg = *r.AssignedTo
		}
		items = append(items, model.ModerationItem{
			ID: r.ID, Type: r.Type, CreatedBy: r.CreatedBy, AssignedTo: asg, Status: r.Status,
		})
	}

	return &model.ModerationListResponse{
		Items: items,
		Page:  page, PageSize: size,
		TotalPages: int(math.Ceil(float64(total) / float64(size))),
	}, nil
}

func (s *Service) Review(ctx context.Context, moderatorID, id string, dto model.ModerationReviewDTO) error {
	req, err := s.storage.GetModerationByID(ctx, id)
	if err != nil {
		return errors.Wrap(err, "get moderation by id")
	}

	if req.Status == model.ModerationStatusFinished {
		return ErrAlreadyFinished
	}

	if req.AssignedTo == nil || *req.AssignedTo == "" {
		_ = s.storage.UpdateModerationStatus(ctx, req.ID, model.ModerationStatusAssigned, &moderatorID)
	}

	switch dto.Decision {
	case DecisionChangesRequested:
		return s.storage.UpdateModerationStatus(ctx, req.ID, model.ModerationStatusChangesRequested, &moderatorID)

	case DecisionSubjectHidden:
		return s.storage.UpdateModerationStatus(ctx, req.ID, model.ModerationStatusFinished, &moderatorID)

	case DecisionSubjectApproved:
		switch req.Type {
		case model.ModerationTypeProfileChange:
			var p struct {
				Name        string `json:"name"`
				ContentType string `json:"contentType"`
				Avatar      string `json:"avatar"`
			}
			if err := json.Unmarshal([]byte(req.Payload), &p); err != nil {
				return ErrInvalidPayload
			}
			if err := s.storage.UpdateUserProfile(ctx, req.UserID, p.Name, p.Avatar); err != nil {
				return errors.Wrap(err, "apply profile change")
			}
			return s.storage.UpdateModerationStatus(ctx, req.ID, model.ModerationStatusFinished, &moderatorID)

		default:
			return ErrUnsupportedType
		}
	default:
		return ErrInvalidDecision
	}
}

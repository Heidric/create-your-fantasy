package session

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/Heidric/create-your-fantasy/internal/model"
)

var (
	ErrSessionNotFound    = errors.New("session not found")
	ErrSessionArchived    = errors.New("session not active")
	ErrAlreadyMember      = errors.New("already member")
	ErrCapacityExceeded   = errors.New("capacity exceeded")
	ErrForbiddenJoin      = errors.New("forbidden join")
	ErrNotMember          = errors.New("not a member")
	ErrNoActiveSession    = errors.New("no active play session")
	ErrMuted              = errors.New("muted in active play session")
	ErrNotOwner           = errors.New("not owner")
	ErrPlayerNotFound     = errors.New("player not found")
	ErrPlayerNotInSession = errors.New("player not in session")
	ErrCannotRemoveOwner  = errors.New("cannot remove owner")
)

type Storage interface {
	CreatePlaySession(ctx context.Context, ps *model.PlaySessionDB) error
	GetPlaySessionByID(ctx context.Context, id string) (*model.PlaySessionDB, error)

	AddPlaySessionMember(ctx context.Context, m *model.PlaySessionMemberDB) error
	IsPlaySessionMember(ctx context.Context, sessionID, userID string) (bool, error)
	CountPlaySessionMembers(ctx context.Context, sessionID string) (int, error)
	DeletePlaySessionMember(ctx context.Context, sessionID, userID string) error
	IsMemberMuted(ctx context.Context, sessionID, userID string) (bool, error)
	GetUserActiveSessionID(ctx context.Context, userID string) (string, error)
	SetUserActiveSession(ctx context.Context, userID, sessionID string) error
	ClearUserActiveSession(ctx context.Context, userID string) error
	IsBannedFromSession(ctx context.Context, sessionID, userID string) (bool, error)
	GetPlaySessionMember(ctx context.Context, sessionID, userID string) (*model.PlaySessionMemberDB, error)
	GetUserByID(ctx context.Context, ID string) (*model.User, error)
	SetMemberMuted(ctx context.Context, sessionID, userID string, muted bool) error
	ClearUserActiveSessionIfMatch(ctx context.Context, userID, sessionID string) error
	InsertPlaySessionBan(ctx context.Context, sessionID, userID string) error

	CountPlaySessionMessages(ctx context.Context, sessionID string) (int, error)
	ListPlaySessionMessages(ctx context.Context, sessionID string, limit, offset int) ([]model.MessageRow, error)
	NextSessionSeq(ctx context.Context, sessionID string) (int64, error)
	InsertPlaySessionMessage(ctx context.Context, id, sessionID, userID string, seq int64, text string, createdAt time.Time) error
	UpdatePlaySessionStatus(ctx context.Context, sessionID, status string) error
	ClearActiveSessionForAll(ctx context.Context, sessionID string) error
}

type Service struct{ storage Storage }

func New(storage Storage) *Service { return &Service{storage: storage} }

func (s *Service) CanConnect(ctx context.Context, sessionID, userID string) (bool, error) {
	ps, err := s.storage.GetPlaySessionByID(ctx, sessionID)
	if err != nil {
		return false, err
	}
	ok, err := s.storage.IsPlaySessionMember(ctx, sessionID, userID)
	if err != nil {
		return false, err
	}
	if ps.Visibility == model.VisibilityPrivate && !ok {
		return false, nil
	}
	if ps.Visibility == model.VisibilityPublic && !ok {
		return false, nil
	}
	return true, nil
}

func (s *Service) Create(ctx context.Context, ownerID string, dto model.CreatePlaySessionDTO) (string, error) {
	id := uuid.NewString()
	var starts *time.Time
	if !dto.StartsAt.IsZero() {
		t := dto.StartsAt.UTC()
		starts = &t
	}
	rec := &model.PlaySessionDB{
		ID:          id,
		OwnerID:     ownerID,
		Title:       dto.Title,
		Ruleset:     dto.Ruleset,
		Description: dto.Description,
		Capacity:    dto.Capacity,
		Visibility:  dto.Visibility,
		StartsAt:    starts,
		Status:      model.Active,
		CreatedAt:   time.Now().UTC(),
	}
	if err := s.storage.CreatePlaySession(ctx, rec); err != nil {
		return "", errors.Wrap(err, "create play session")
	}
	member := &model.PlaySessionMemberDB{
		SessionID: id,
		UserID:    ownerID,
		Role:      model.Gm,
		Muted:     false,
	}
	if err := s.storage.AddPlaySessionMember(ctx, member); err != nil {
		return "", errors.Wrap(err, "add owner as gm")
	}
	return id, nil
}

func (s *Service) Join(ctx context.Context, sessionID, userID string) error {
	ps, err := s.storage.GetPlaySessionByID(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}
	if ps.Status != model.Active {
		return ErrSessionArchived
	}

	if ps.Visibility == model.VisibilityPrivate && ps.OwnerID != userID {
		return ErrForbiddenJoin
	}

	isMember, err := s.storage.IsPlaySessionMember(ctx, sessionID, userID)
	if err != nil {
		return err
	}
	if isMember {
		return ErrAlreadyMember
	}

	count, err := s.storage.CountPlaySessionMembers(ctx, sessionID)
	if err != nil {
		return err
	}
	if count >= ps.Capacity {
		return ErrCapacityExceeded
	}

	banned, err := s.storage.IsBannedFromSession(ctx, sessionID, userID)
	if err != nil {
		return err
	}
	if banned {
		return ErrForbiddenJoin
	}

	m := &model.PlaySessionMemberDB{
		SessionID: sessionID,
		UserID:    userID,
		Role:      model.Player,
		Muted:     false,
	}
	if err := s.storage.AddPlaySessionMember(ctx, m); err != nil {
		return err
	}

	_ = s.storage.SetUserActiveSession(ctx, userID, sessionID)

	return nil
}

func (s *Service) Leave(ctx context.Context, userID string) (string, error) {
	sID, err := s.storage.GetUserActiveSessionID(ctx, userID)
	if err != nil {
		return "", err
	}
	if sID == "" {
		return "", ErrNoActiveSession
	}

	ps, err := s.storage.GetPlaySessionByID(ctx, sID)
	if err != nil {
		return "", ErrSessionNotFound
	}
	if ps.Status != model.Active {
		return "", ErrSessionArchived
	}

	isMember, err := s.storage.IsPlaySessionMember(ctx, sID, userID)
	if err != nil {
		return "", err
	}
	if !isMember {
		return "", ErrNotMember
	}

	if ps.OwnerID == userID {
		return "", ErrForbiddenJoin
	}

	if err := s.storage.DeletePlaySessionMember(ctx, sID, userID); err != nil {
		return "", err
	}
	_ = s.storage.ClearUserActiveSessionIfMatch(ctx, userID, sID)
	return sID, nil
}

func (s *Service) ListMessages(ctx context.Context, sessionID, userID string, q model.MessagesQuery) (*model.MessagesResponse, error) {
	_, err := s.storage.GetPlaySessionByID(ctx, sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	isMember, err := s.storage.IsPlaySessionMember(ctx, sessionID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotMember
	}

	after := 0
	if q.LastMessage != nil {
		after = *q.LastMessage
	}
	size := 50
	if q.PageSize != nil {
		size = *q.PageSize
	}

	rows, err := s.storage.ListPlaySessionMessages(ctx, sessionID, after, size)
	if err != nil {
		return nil, errors.Wrap(err, "list after")
	}

	items := make([]model.MessageItem, 0, len(rows))
	last := after
	for _, r := range rows {
		items = append(items, model.MessageItem{
			ID: r.ID, UserID: r.UserID, Text: r.Text, CreatedAt: r.CreatedAt,
		})
		if r.SeqID > last {
			last = r.SeqID
		}
	}
	return &model.MessagesResponse{Items: items, LastMessage: last}, nil
}

func (s *Service) SendMessage(ctx context.Context, userID string, text string) (sessionID string, seq int64, sentAt time.Time, err error) {
	sid, err := s.storage.GetUserActiveSessionID(ctx, userID)
	if err != nil {
		return "", 0, time.Time{}, errors.Wrap(err, "get active session")
	}
	if sid == "" {
		return "", 0, time.Time{}, ErrNoActiveSession
	}

	ps, err := s.storage.GetPlaySessionByID(ctx, sid)
	if err != nil {
		return "", 0, time.Time{}, ErrSessionNotFound
	}
	if ps.Status != model.Active {
		return "", 0, time.Time{}, ErrSessionArchived
	}
	ok, err := s.storage.IsPlaySessionMember(ctx, sid, userID)
	if err != nil {
		return "", 0, time.Time{}, err
	}
	if !ok {
		return "", 0, time.Time{}, ErrNotMember
	}

	muted, err := s.storage.IsMemberMuted(ctx, sid, userID)
	if err != nil {
		return "", 0, time.Time{}, err
	}
	if muted {
		return "", 0, time.Time{}, ErrMuted
	}

	seq, err = s.storage.NextSessionSeq(ctx, sid)
	if err != nil {
		return "", 0, time.Time{}, err
	}
	now := time.Now().UTC()
	id := uuid.NewString()
	if err := s.storage.InsertPlaySessionMessage(ctx, id, sid, userID, seq, text, now); err != nil {
		return "", 0, time.Time{}, err
	}
	return sid, seq, now, nil
}

func (s *Service) Mute(ctx context.Context, gmUserID, targetUserID string) (sessionID string, err error) {
	sid, err := s.storage.GetUserActiveSessionID(ctx, gmUserID)
	if err != nil {
		return "", err
	}
	if sid == "" {
		return "", ErrNoActiveSession
	}

	ps, err := s.storage.GetPlaySessionByID(ctx, sid)
	if err != nil {
		return "", ErrSessionNotFound
	}
	if ps.OwnerID != gmUserID {
		return "", ErrNotOwner
	}

	if _, err := s.storage.GetUserByID(ctx, targetUserID); err != nil {
		return "", ErrPlayerNotFound
	}

	m, err := s.storage.GetPlaySessionMember(ctx, sid, targetUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrPlayerNotInSession
		}
		return "", err
	}
	if m.Role != model.Player {
		return "", ErrPlayerNotInSession
	}

	if err := s.storage.SetMemberMuted(ctx, sid, targetUserID, true); err != nil {
		return "", err
	}
	return sid, nil
}

func (s *Service) Unmute(ctx context.Context, gmUserID, targetUserID string) (sessionID string, err error) {
	sid, err := s.storage.GetUserActiveSessionID(ctx, gmUserID)
	if err != nil {
		return "", err
	}
	if sid == "" {
		return "", ErrNoActiveSession
	}

	ps, err := s.storage.GetPlaySessionByID(ctx, sid)
	if err != nil {
		return "", ErrSessionNotFound
	}
	if ps.OwnerID != gmUserID {
		return "", ErrNotOwner
	}

	if _, err := s.storage.GetUserByID(ctx, targetUserID); err != nil {
		return "", ErrPlayerNotFound
	}

	m, err := s.storage.GetPlaySessionMember(ctx, sid, targetUserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrPlayerNotInSession
		}
		return "", err
	}
	if m.Role != model.Player {
		return "", ErrPlayerNotInSession
	}

	if err := s.storage.SetMemberMuted(ctx, sid, targetUserID, false); err != nil {
		return "", err
	}
	return sid, nil
}

func (s *Service) Remove(ctx context.Context, gmUserID, targetUserID string) (sessionID string, err error) {
	sid, err := s.storage.GetUserActiveSessionID(ctx, gmUserID)
	if err != nil {
		return "", err
	}
	if sid == "" {
		return "", ErrNoActiveSession
	}

	ps, err := s.storage.GetPlaySessionByID(ctx, sid)
	if err != nil {
		return "", ErrSessionNotFound
	}
	if ps.OwnerID != gmUserID {
		return "", ErrNotOwner
	}

	if _, err := s.storage.GetUserByID(ctx, targetUserID); err != nil {
		return "", ErrPlayerNotFound
	}

	if targetUserID == ps.OwnerID {
		return "", ErrCannotRemoveOwner
	}

	isMember, err := s.storage.IsPlaySessionMember(ctx, sid, targetUserID)
	if err != nil {
		return "", err
	}
	if !isMember {
		return "", ErrPlayerNotInSession
	}

	if err := s.storage.DeletePlaySessionMember(ctx, sid, targetUserID); err != nil {
		return "", err
	}
	if err := s.storage.InsertPlaySessionBan(ctx, sid, targetUserID); err != nil {
		return "", err
	}
	_ = s.storage.ClearUserActiveSessionIfMatch(ctx, targetUserID, sid)

	return sid, nil
}

func (s *Service) End(ctx context.Context, gmUserID string) (sessionID string, err error) {
	sid, err := s.storage.GetUserActiveSessionID(ctx, gmUserID)
	if err != nil {
		return "", err
	}
	if sid == "" {
		return "", ErrNoActiveSession
	}

	ps, err := s.storage.GetPlaySessionByID(ctx, sid)
	if err != nil {
		return "", ErrSessionNotFound
	}
	if ps.Status != model.Active {
		return "", ErrSessionArchived
	}
	if ps.OwnerID != gmUserID {
		return "", ErrNotOwner
	}

	if err := s.storage.UpdatePlaySessionStatus(ctx, sid, model.Archived); err != nil {
		return "", err
	}
	if err := s.storage.ClearActiveSessionForAll(ctx, sid); err != nil {
		return "", err
	}
	return sid, nil
}

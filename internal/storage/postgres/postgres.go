package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/Heidric/create-your-fantasy/internal/logger"
	"github.com/Heidric/create-your-fantasy/internal/model"
	"github.com/Heidric/create-your-fantasy/internal/services/moderation"
	"github.com/Heidric/create-your-fantasy/internal/storage"
	"github.com/Heidric/create-your-fantasy/pkg/pgx"
	"github.com/Heidric/create-your-fantasy/pkg/security"
	"github.com/huandu/go-sqlbuilder"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"
)

var log zerolog.Logger

type Storage struct {
	db *pgx.Postgres
}

func NewStorage(ctx context.Context, db *pgx.Postgres) *Storage {
	log = *logger.Log
	log = log.With().Str("name", "storage").Logger()

	return &Storage{db: db}
}

func (s *Storage) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("id", "email", "name", "role", "avatar", "password_hash").
		From(`"user"`).
		Where(sb.Equal("email", email))

	query, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)
	var user model.User
	conn := s.db.GetConn()
	if err := conn.GetContext(ctx, &user, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrEntityNotFound
		}
		return nil, errors.Wrap(err, "get user by email")
	}

	return &user, nil
}

func (s *Storage) GetUserByID(ctx context.Context, ID string) (*model.User, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select(`"user".id as ID`, "email", `"user".name as name`, "role", "avatar").
		From(`"user"`).
		Where(sb.Equal(`"user".id`, ID))

	query, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)

	var user model.User
	conn := s.db.GetConn()
	if err := conn.GetContext(ctx, &user, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrEntityNotFound
		}
		return nil, errors.Wrap(err, "get user by ID")
	}

	return &user, nil
}

func (s *Storage) CreateSession(ctx context.Context, session *model.Session) error {
	sb := sqlbuilder.NewInsertBuilder()

	sb.InsertInto("session")
	sb.Cols("id", "user_id", "access_token", "refresh_token", "expires_at")
	sb.Values(session.ID, session.UserID, session.AccessToken, session.RefreshToken, session.ExpiresAt)

	query, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)

	conn := s.db.GetConn()
	if _, err := conn.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "create session")
	}
	return nil
}

func (s *Storage) DeleteSessionByRToken(ctx context.Context, rToken string) error {
	db := sqlbuilder.NewDeleteBuilder()
	db.DeleteFrom("session").Where(db.Equal("refresh_token", rToken))

	query, args := db.BuildWithFlavor(sqlbuilder.PostgreSQL)
	conn := s.db.GetConn()
	if _, err := conn.ExecContext(ctx, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return errors.Wrap(err, "delete session by refresh token")
	}

	return nil
}

func (s *Storage) DeleteSessionsByUserID(ctx context.Context, userID string) error {
	db := sqlbuilder.NewDeleteBuilder()
	db.DeleteFrom("session").Where(db.Equal("user_id", userID))

	query, args := db.BuildWithFlavor(sqlbuilder.PostgreSQL)
	conn := s.db.GetConn()
	if _, err := conn.ExecContext(ctx, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return errors.Wrap(err, "delete session by user id")
	}

	return nil
}

func (s *Storage) GetSessionBySID(ctx context.Context, sID string) (*model.Session, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("id", "user_id", "access_token", "refresh_token", "expires_at").
		From("session").
		Where(sb.Equal("id", sID))

	query, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)

	var session model.Session
	conn := s.db.GetConn()
	if err := conn.GetContext(ctx, &session, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrEntityNotFound
		}
		return nil, errors.Wrap(err, "get session by sID")
	}

	return &session, nil
}

func (s *Storage) GetSessionByRToken(ctx context.Context, rToken string) (*model.Session, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("id", "user_id", "access_token", "refresh_token", "expires_at").
		From("session").
		Where(sb.Equal("refresh_token", rToken))

	query, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)

	var session model.Session
	conn := s.db.GetConn()
	if err := conn.GetContext(ctx, &session, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, storage.ErrEntityNotFound
		}
		return nil, errors.Wrap(err, "get session by refresh token")
	}

	if session.ExpiresAt.Before(time.Now()) {
		db := sqlbuilder.NewDeleteBuilder()
		db.DeleteFrom("session").Where(db.Equal("refresh_token", rToken))

		query, args = db.BuildWithFlavor(sqlbuilder.PostgreSQL)
		if _, err := conn.ExecContext(ctx, query, args...); err != nil {
			return nil, errors.Wrap(err, "delete expired session")
		}

		return nil, storage.ErrEntityNotFound
	}

	return &session, nil
}

func (s *Storage) SetNewPassword(ctx context.Context, userID string, password string, temp bool) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 0)
	if err != nil {
		return errors.Wrap(err, "hash")
	}

	ub := sqlbuilder.NewUpdateBuilder()
	ub.Update(`"user"`).
		Set(ub.Assign("password_hash", string(hash)),
			ub.Assign("password_temporary", temp)).
		Where(ub.Equal("id", userID))

	query, args := ub.BuildWithFlavor(sqlbuilder.PostgreSQL)

	conn := s.db.GetConn()
	if _, err := conn.ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "set new password")
	}
	return nil
}

func (s *Storage) RegisterUser(ctx context.Context, user *model.RegisterUserDTO) (*string, error) {
	password, err := security.GenerateSecureString(8)
	if err != nil {
		return nil, errors.Wrap(err, "password gen")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 0)
	if err != nil {
		return nil, errors.Wrap(err, "hash")
	}

	cb := sqlbuilder.NewInsertBuilder()

	cb.InsertInto(`"user"`).
		Cols("email", "password_hash", "role", "password_temporary").
		Values(user.Email, string(hash), model.Player, true).
		SQL("RETURNING id")

	query, args := cb.BuildWithFlavor(sqlbuilder.PostgreSQL)

	var userID string
	conn := s.db.GetConn()
	err = conn.GetContext(ctx, &userID, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "create user")
	}

	return &userID, nil
}

func (s *Storage) CreateModerationRequest(ctx context.Context, req *model.ModerationRequest) error {
	ib := sqlbuilder.NewInsertBuilder()
	ib.InsertInto("moderation_request").
		Cols("id", "user_id", "type", "payload", "status", "created_at").
		Values(req.ID, req.UserID, req.Type, req.Payload, req.Status, req.CreatedAt)

	query, args := ib.BuildWithFlavor(sqlbuilder.PostgreSQL)
	if _, err := s.db.GetConn().ExecContext(ctx, query, args...); err != nil {
		return errors.Wrap(err, "insert moderation_request")
	}
	return nil
}

func (s *Storage) ListModerationRequests(ctx context.Context, f moderation.ModerationFilter, limit, offset int) ([]moderation.DBModerationRow, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("id", "type", "user_id", "assigned_to", "status").
		From("moderation_request")

	if f.Status != "" {
		sb.Where(sb.Equal("status", f.Status))
	}
	if f.Type != "" {
		sb.Where(sb.Equal("type", f.Type))
	}
	if f.AssignedTo != "" {
		sb.Where(sb.Equal("assigned_to", f.AssignedTo))
	}

	sb.OrderBy("created_at DESC").Limit(limit).Offset(offset)
	q, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := s.db.GetConn().QueryxContext(ctx, q, args...)
	if err != nil {
		return nil, errors.Wrap(err, "query list moderation")
	}
	defer rows.Close()

	var out []moderation.DBModerationRow
	for rows.Next() {
		var r moderation.DBModerationRow
		if err := rows.StructScan(&r); err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Storage) CountModerationRequests(ctx context.Context, f moderation.ModerationFilter) (int64, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("COUNT(1)").From("moderation_request")

	if f.Status != "" {
		sb.Where(sb.Equal("status", f.Status))
	}
	if f.Type != "" {
		sb.Where(sb.Equal("type", f.Type))
	}
	if f.AssignedTo != "" {
		sb.Where(sb.Equal("assigned_to", f.AssignedTo))
	}

	q, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)
	var total int64
	if err := s.db.GetConn().QueryRowContext(ctx, q, args...).Scan(&total); err != nil {
		return 0, errors.Wrap(err, "count moderation")
	}
	return total, nil
}

func (s *Storage) GetModerationByID(ctx context.Context, id string) (*moderation.DBModerationRow, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("id", "user_id", "type", "status", "assigned_to", "payload").
		From("moderation_request").
		Where(sb.Equal("id", id)).
		Limit(1)
	q, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)

	var out moderation.DBModerationRow
	if err := s.db.GetConn().QueryRowxContext(ctx, q, args...).StructScan(&out); err != nil {
		return nil, errors.Wrap(err, "select moderation by id")
	}
	return &out, nil
}

func (s *Storage) UpdateModerationStatus(ctx context.Context, id string, status string, assignedTo *string) error {
	ub := sqlbuilder.NewUpdateBuilder()
	ub.Update("moderation_request").Set(ub.Assign("status", status))
	if assignedTo != nil {
		ub.SetMore(ub.Assign("assigned_to", *assignedTo))
	}
	ub.Where(ub.Equal("id", id))
	q, args := ub.BuildWithFlavor(sqlbuilder.PostgreSQL)
	_, err := s.db.GetConn().ExecContext(ctx, q, args...)
	return errors.Wrap(err, "update moderation status")
}

func (s *Storage) UpdateUserProfile(ctx context.Context, userID, name, avatar string) error {
	ub := sqlbuilder.NewUpdateBuilder()
	ub.Update(`"user"`).
		Set(
			ub.Assign("name", name),
			ub.Assign("avatar", avatar),
		).
		Where(ub.Equal("id", userID))
	q, args := ub.BuildWithFlavor(sqlbuilder.PostgreSQL)
	_, err := s.db.GetConn().ExecContext(ctx, q, args...)
	return errors.Wrap(err, "update user profile")
}

func (s *Storage) CreatePlaySession(ctx context.Context, ps *model.PlaySessionDB) error {
	ib := sqlbuilder.NewInsertBuilder()
	ib.InsertInto("play_session").
		Cols("id", "owner_id", "title", "ruleset", "description", "capacity", "visibility", "starts_at", "status", "created_at").
		Values(ps.ID, ps.OwnerID, ps.Title, ps.Ruleset, ps.Description, ps.Capacity, ps.Visibility, ps.StartsAt, ps.Status, ps.CreatedAt)
	q, args := ib.BuildWithFlavor(sqlbuilder.PostgreSQL)
	_, err := s.db.GetConn().ExecContext(ctx, q, args...)
	return errors.Wrap(err, "insert play_session")
}

func (s *Storage) AddPlaySessionMember(ctx context.Context, m *model.PlaySessionMemberDB) error {
	ib := sqlbuilder.NewInsertBuilder()
	ib.InsertInto("play_session_member").
		Cols("session_id", "user_id", "role", "muted").
		Values(m.SessionID, m.UserID, m.Role, m.Muted)
	q, args := ib.BuildWithFlavor(sqlbuilder.PostgreSQL)
	_, err := s.db.GetConn().ExecContext(ctx, q, args...)
	return errors.Wrap(err, "insert play_session_member")
}

func (s *Storage) GetPlaySessionByID(ctx context.Context, id string) (*model.PlaySessionDB, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("id", "owner_id", "title", "ruleset", "description", "capacity", "visibility", "starts_at", "status", "created_at").
		From("play_session").Where(sb.Equal("id", id)).Limit(1)
	q, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)
	var ps model.PlaySessionDB
	if err := s.db.GetConn().QueryRowxContext(ctx, q, args...).StructScan(&ps); err != nil {
		return nil, errors.Wrap(err, "get play_session by id")
	}
	return &ps, nil
}

func (s *Storage) IsPlaySessionMember(ctx context.Context, sessionID, userID string) (bool, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("1").From("play_session_member").
		Where(sb.And(
			sb.Equal("session_id", sessionID),
			sb.Equal("user_id", userID),
		)).Limit(1)
	q, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)
	var one int
	err := s.db.GetConn().QueryRowContext(ctx, q, args...).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, errors.Wrap(err, "is member")
	}
	return true, nil
}

func (s *Storage) CountPlaySessionMembers(ctx context.Context, sessionID string) (int, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("COUNT(1)").
		From("play_session_member").
		Where(sb.Equal("session_id", sessionID))
	q, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)

	var n int
	if err := s.db.GetConn().QueryRowContext(ctx, q, args...).Scan(&n); err != nil {
		return 0, errors.Wrap(err, "count play_session_member")
	}
	return n, nil
}

func (s *Storage) DeletePlaySessionMember(ctx context.Context, sessionID, userID string) error {
	db := sqlbuilder.NewDeleteBuilder()
	db.DeleteFrom("play_session_member").
		Where(db.And(
			db.Equal("session_id", sessionID),
			db.Equal("user_id", userID),
		))
	q, args := db.BuildWithFlavor(sqlbuilder.PostgreSQL)
	_, err := s.db.GetConn().ExecContext(ctx, q, args...)
	return errors.Wrap(err, "delete play_session_member")
}

func (s *Storage) CountPlaySessionMessages(ctx context.Context, sessionID string) (int, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("COUNT(1)").From("play_session_message").Where(sb.Equal("session_id", sessionID))
	q, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)
	var n int
	if err := s.db.GetConn().QueryRowContext(ctx, q, args...).Scan(&n); err != nil {
		return 0, errors.Wrap(err, "count play_session_message")
	}
	return n, nil
}

func (s *Storage) ListPlaySessionMessages(ctx context.Context, sessionID string, afterSeq, limit int) ([]model.MessageRow, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("id", "user_id", "text", "created_at", "seq_id").
		From("play_session_message").
		Where(sb.And(
			sb.Equal("session_id", sessionID),
			sb.GreaterThan("seq_id", afterSeq),
		)).
		OrderBy("seq_id ASC").
		Limit(limit)
	q, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)

	rows, err := s.db.GetConn().QueryxContext(ctx, q, args...)
	if err != nil {
		return nil, errors.Wrap(err, "list after seq")
	}
	defer rows.Close()

	var out []model.MessageRow
	for rows.Next() {
		var r model.MessageRow
		if err := rows.StructScan(&r); err != nil {
			return nil, errors.Wrap(err, "scan")
		}
		out = append(out, r)
	}
	return out, nil
}

func (s *Storage) GetUserActiveSessionID(ctx context.Context, userID string) (string, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("active_session_id").
		From(`"user"`).
		Where(sb.Equal("id", userID)).
		Limit(1)
	q, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)
	var sid *string
	if err := s.db.GetConn().QueryRowContext(ctx, q, args...).Scan(&sid); err != nil {
		return "", errors.Wrap(err, "get user active_session_id")
	}
	if sid == nil || *sid == "" {
		return "", nil
	}
	return *sid, nil
}

func (s *Storage) IsMemberMuted(ctx context.Context, sessionID, userID string) (bool, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("muted").
		From("play_session_member").
		Where(sb.And(
			sb.Equal("session_id", sessionID),
			sb.Equal("user_id", userID),
		)).Limit(1)
	q, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)
	var muted bool
	if err := s.db.GetConn().QueryRowContext(ctx, q, args...).Scan(&muted); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, errors.Wrap(err, "is member muted")
	}
	return muted, nil
}

func (s *Storage) NextSessionSeq(ctx context.Context, sessionID string) (int64, error) {
	ub := sqlbuilder.NewUpdateBuilder()
	ub.Update("play_session").
		Set(
			ub.Assign("last_seq", sqlbuilder.Raw("last_seq + 1")),
		).
		Where(ub.Equal("id", sessionID)).
		SQL("RETURNING last_seq")
	q, args := ub.BuildWithFlavor(sqlbuilder.PostgreSQL)

	var seq int64
	if err := s.db.GetConn().QueryRowContext(ctx, q, args...).Scan(&seq); err != nil {
		return 0, errors.Wrap(err, "next session seq")
	}
	return seq, nil
}

func (s *Storage) InsertPlaySessionMessage(ctx context.Context, id, sessionID, userID string, seq int64, text string, createdAt time.Time) error {
	ib := sqlbuilder.NewInsertBuilder()
	ib.InsertInto("play_session_message").
		Cols("id", "session_id", "user_id", "seq_id", "text", "created_at").
		Values(id, sessionID, userID, seq, text, createdAt)
	q, args := ib.BuildWithFlavor(sqlbuilder.PostgreSQL)
	_, err := s.db.GetConn().ExecContext(ctx, q, args...)
	return errors.Wrap(err, "insert play_session_message")
}

func (s *Storage) SetUserActiveSession(ctx context.Context, userID, sessionID string) error {
	ub := sqlbuilder.NewUpdateBuilder()
	ub.Update(`"user"`).Set(ub.Assign("active_session_id", sessionID)).Where(ub.Equal("id", userID))
	q, args := ub.BuildWithFlavor(sqlbuilder.PostgreSQL)
	_, err := s.db.GetConn().ExecContext(ctx, q, args...)
	return errors.Wrap(err, "set active_session_id")
}

func (s *Storage) ClearUserActiveSession(ctx context.Context, userID string) error {
	ub := sqlbuilder.NewUpdateBuilder()
	ub.Update(`"user"`).Set(ub.Assign("active_session_id", nil)).Where(ub.Equal("id", userID))
	q, args := ub.BuildWithFlavor(sqlbuilder.PostgreSQL)
	_, err := s.db.GetConn().ExecContext(ctx, q, args...)
	return errors.Wrap(err, "clear active_session_id")
}

func (s *Storage) GetPlaySessionMember(ctx context.Context, sessionID, userID string) (*model.PlaySessionMemberDB, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("session_id", "user_id", "role", "muted").
		From("play_session_member").
		Where(sb.And(sb.Equal("session_id", sessionID), sb.Equal("user_id", userID))).
		Limit(1)
	q, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)

	var m model.PlaySessionMemberDB
	if err := s.db.GetConn().QueryRowxContext(ctx, q, args...).StructScan(&m); err != nil {
		return nil, errors.Wrap(err, "get play_session_member")
	}
	return &m, nil
}

func (s *Storage) SetMemberMuted(ctx context.Context, sessionID, userID string, muted bool) error {
	ub := sqlbuilder.NewUpdateBuilder()
	ub.Update("play_session_member").
		Set(ub.Assign("muted", muted)).
		Where(ub.And(ub.Equal("session_id", sessionID), ub.Equal("user_id", userID)))
	q, args := ub.BuildWithFlavor(sqlbuilder.PostgreSQL)
	_, err := s.db.GetConn().ExecContext(ctx, q, args...)
	return errors.Wrap(err, "set member muted")
}

func (s *Storage) IsBannedFromSession(ctx context.Context, sessionID, userID string) (bool, error) {
	sb := sqlbuilder.NewSelectBuilder()
	sb.Select("1").From("play_session_ban").
		Where(sb.And(sb.Equal("session_id", sessionID), sb.Equal("user_id", userID))).Limit(1)
	q, args := sb.BuildWithFlavor(sqlbuilder.PostgreSQL)

	var one int
	if err := s.db.GetConn().QueryRowContext(ctx, q, args...).Scan(&one); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, errors.Wrap(err, "is banned")
	}
	return true, nil
}

func (s *Storage) InsertPlaySessionBan(ctx context.Context, sessionID, userID string) error {
	ib := sqlbuilder.NewInsertBuilder()
	ib.InsertInto("play_session_ban").
		Cols("session_id", "user_id").
		Values(sessionID, userID)
	q, args := ib.BuildWithFlavor(sqlbuilder.PostgreSQL)
	_, err := s.db.GetConn().ExecContext(ctx, q, args...)
	return errors.Wrap(err, "insert play_session_ban")
}

func (s *Storage) ClearUserActiveSessionIfMatch(ctx context.Context, userID, sessionID string) error {
	ub := sqlbuilder.NewUpdateBuilder()
	ub.Update(`"user"`).
		Set(ub.Assign("active_session_id", nil)).
		Where(ub.And(
			ub.Equal("id", userID),
			ub.Equal("active_session_id", sessionID),
		))
	q, args := ub.BuildWithFlavor(sqlbuilder.PostgreSQL)
	_, err := s.db.GetConn().ExecContext(ctx, q, args...)
	return errors.Wrap(err, "clear active_session_id if match")
}

func (s *Storage) UpdatePlaySessionStatus(ctx context.Context, sessionID, status string) error {
	ub := sqlbuilder.NewUpdateBuilder()
	ub.Update("play_session").
		Set(ub.Assign("status", status)).
		Where(ub.Equal("id", sessionID))
	q, args := ub.BuildWithFlavor(sqlbuilder.PostgreSQL)
	_, err := s.db.GetConn().ExecContext(ctx, q, args...)
	return errors.Wrap(err, "update play_session status")
}

func (s *Storage) ClearActiveSessionForAll(ctx context.Context, sessionID string) error {
	ub := sqlbuilder.NewUpdateBuilder()
	ub.Update(`"user"`).
		Set(ub.Assign("active_session_id", nil)).
		Where(ub.Equal("active_session_id", sessionID))
	q, args := ub.BuildWithFlavor(sqlbuilder.PostgreSQL)
	_, err := s.db.GetConn().ExecContext(ctx, q, args...)
	return errors.Wrap(err, "clear active_session_id for all")
}

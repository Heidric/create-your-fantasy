package server

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Heidric/create-your-fantasy/internal/lib/jwt"
	"github.com/Heidric/create-your-fantasy/internal/model"
	"github.com/Heidric/create-your-fantasy/internal/services/session"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

func (s *Server) createPlaySessionHandler(w http.ResponseWriter, r *http.Request) {
	var dto model.CreatePlaySessionDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		ParsingError(w)
		return
	}
	if errs := dto.Validate(); len(errs) > 0 {
		ValidationError(w, errs)
		return
	}

	claims, ok := r.Context().Value(jwt.CtxKeyClaims).(model.UserClaim)
	if !ok {
		UnauthorizedError(w)
		return
	}

	id, err := s.playSession.Create(r.Context(), claims.ID, dto)
	if err != nil {
		InternalError(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]string{"id": id}); err != nil {
		log.Error().Err(err).Msg("Error encoding response")
		InternalError(w)
		return
	}
}

func (s *Server) joinPlaySessionHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		BadRequestError(w)
		return
	}
	claims, ok := r.Context().Value(jwt.CtxKeyClaims).(model.UserClaim)
	if !ok {
		UnauthorizedError(w)
		return
	}

	if err := s.playSession.Join(r.Context(), id, claims.ID); err != nil {
		switch err {
		case session.ErrSessionNotFound:
			NotFoundError(w)
			return
		case session.ErrForbiddenJoin:
			ForbiddenError(w)
			return
		case session.ErrAlreadyMember:
			ConflictError(w)
			return
		case session.ErrCapacityExceeded:
			ConflictError(w)
			return
		case session.ErrSessionArchived:
			LogicError(w, "SESSION_ARCHIVED")
			return
		default:
			InternalError(w)
			return
		}
	}

	if s.wsHub != nil {
		env := map[string]any{
			"type": "system.join",
			"user": claims.ID,
		}
		b, _ := json.Marshal(env)
		s.wsHub.Broadcast("ps:"+id, b)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) leavePlaySessionHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(jwt.CtxKeyClaims).(model.UserClaim)
	if !ok {
		UnauthorizedError(w)
		return
	}

	sID, err := s.playSession.Leave(r.Context(), claims.ID)
	if err != nil {
		switch err {
		case session.ErrSessionNotFound:
			NotFoundError(w)
			return
		case session.ErrSessionArchived:
			LogicError(w, "SESSION_ARCHIVED")
			return
		case session.ErrNotMember:
			ConflictError(w)
			return
		case session.ErrForbiddenJoin:
			ForbiddenError(w)
			return
		default:
			InternalError(w)
			return
		}
	}

	if s.wsHub != nil {
		ev := map[string]any{"type": "system.leave", "user": claims.ID}
		b, _ := json.Marshal(ev)
		s.wsHub.Broadcast("ps:"+sID, b)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listPlaySessionMessagesHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		BadRequestError(w)
		return
	}

	qp := r.URL.Query()
	var q model.MessagesQuery
	if v := qp.Get("lastMessage"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			ValidationError(w, map[string]string{"lastMessage": model.ErrInvalidField})
			return
		}
		q.LastMessage = &n
	}
	if v := qp.Get("pageSize"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			ValidationError(w, map[string]string{"pageSize": model.ErrInvalidField})
			return
		}
		q.PageSize = &n
	}
	if errs := q.Validate(); len(errs) > 0 {
		ValidationError(w, errs)
		return
	}

	claims, ok := r.Context().Value(jwt.CtxKeyClaims).(model.UserClaim)
	if !ok {
		UnauthorizedError(w)
		return
	}

	res, err := s.playSession.ListMessages(r.Context(), id, claims.ID, q)
	if err != nil {
		switch err {
		case session.ErrSessionNotFound:
			NotFoundError(w)
			return
		case session.ErrNotMember:
			ForbiddenError(w)
			return
		default:
			InternalError(w)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Error().Err(err).Msg("Error encoding response")
		InternalError(w)
		return
	}
}

func (s *Server) sendPlaySessionMessageHandler(w http.ResponseWriter, r *http.Request) {
	var dto model.SendMessageDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		BadRequestError(w)
		return
	}
	if errs := dto.Validate(); len(errs) > 0 {
		ValidationError(w, errs)
		return
	}

	claims, ok := r.Context().Value(jwt.CtxKeyClaims).(model.UserClaim)
	if !ok {
		UnauthorizedError(w)
		return
	}

	sessionID, seq, sentAt, err := s.playSession.SendMessage(r.Context(), claims.ID, dto.Text)
	if err != nil {
		switch err {
		case session.ErrNoActiveSession:
			LogicError(w, "NO_ACTIVE_PLAY_SESSION")
			return
		case session.ErrMuted:
			LogicError(w, "MUTED_IN_ACTIVE_PLAY_SESSION")
			return
		case session.ErrSessionArchived:
			LogicError(w, "SESSION_ARCHIVED")
			return
		case session.ErrNotMember:
			ForbiddenError(w)
			return
		default:
			InternalError(w)
			return
		}
	}

	if s.wsHub != nil {
		env := map[string]any{
			"type":     "chat.message",
			"id":       uuid.NewString(),
			"seqId":    seq,
			"senderId": claims.ID,
			"sentAt":   sentAt,
			"text":     dto.Text,
		}
		b, _ := json.Marshal(env)
		s.wsHub.Broadcast("ps:"+sessionID, b)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) mutePlaySessionMemberHandler(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")
	if targetID == "" {
		BadRequestError(w)
		return
	}

	claims, ok := r.Context().Value(jwt.CtxKeyClaims).(model.UserClaim)
	if !ok {
		UnauthorizedError(w)
		return
	}

	sessionID, err := s.playSession.Mute(r.Context(), claims.ID, targetID)
	if err != nil {
		switch err {
		case session.ErrNoActiveSession:
			LogicError(w, "NO_ACTIVE_PLAY_SESSION")
			return
		case session.ErrNotOwner:
			ForbiddenError(w)
			return
		case session.ErrPlayerNotFound:
			NotFoundError(w)
			return
		case session.ErrPlayerNotInSession:
			LogicError(w, "PLAYER_NOT_IN_PLAY_SESSION")
			return
		default:
			InternalError(w)
			return
		}
	}

	if s.wsHub != nil {
		ev := map[string]any{"type": "system.mute", "user": targetID, "by": claims.ID}
		b, _ := json.Marshal(ev)
		s.wsHub.Broadcast("ps:"+sessionID, b)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) unmutePlaySessionMemberHandler(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")
	if targetID == "" {
		BadRequestError(w)
		return
	}

	claims, ok := r.Context().Value(jwt.CtxKeyClaims).(model.UserClaim)
	if !ok {
		UnauthorizedError(w)
		return
	}

	sessionID, err := s.playSession.Unmute(r.Context(), claims.ID, targetID)
	if err != nil {
		switch err {
		case session.ErrNoActiveSession:
			LogicError(w, "NO_ACTIVE_PLAY_SESSION")
			return
		case session.ErrNotOwner:
			ForbiddenError(w)
			return
		case session.ErrPlayerNotFound:
			NotFoundError(w)
			return
		case session.ErrPlayerNotInSession:
			LogicError(w, "PLAYER_NOT_IN_PLAY_SESSION")
			return
		default:
			InternalError(w)
			return
		}
	}

	if s.wsHub != nil {
		ev := map[string]any{"type": "system.unmute", "user": targetID, "by": claims.ID}
		b, _ := json.Marshal(ev)
		s.wsHub.Broadcast("ps:"+sessionID, b)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) removePlaySessionMemberHandler(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")
	if targetID == "" {
		BadRequestError(w)
		return
	}

	claims, ok := r.Context().Value(jwt.CtxKeyClaims).(model.UserClaim)
	if !ok {
		UnauthorizedError(w)
		return
	}

	sessionID, err := s.playSession.Remove(r.Context(), claims.ID, targetID)
	if err != nil {
		switch err {
		case session.ErrNoActiveSession:
			LogicError(w, "NO_ACTIVE_PLAY_SESSION")
			return
		case session.ErrNotOwner:
			ForbiddenError(w)
			return
		case session.ErrPlayerNotFound:
			NotFoundError(w)
			return
		case session.ErrPlayerNotInSession:
			LogicError(w, "PLAYER_NOT_IN_PLAY_SESSION")
			return
		case session.ErrCannotRemoveOwner:
			LogicError(w, "CANNOT_REMOVE_OWNER")
			return
		default:
			InternalError(w)
			return
		}
	}

	if s.wsHub != nil {
		ev := map[string]any{"type": "system.remove", "user": targetID, "by": claims.ID}
		b, _ := json.Marshal(ev)
		s.wsHub.Broadcast("ps:"+sessionID, b)
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) endPlaySessionHandler(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(jwt.CtxKeyClaims).(model.UserClaim)
	if !ok {
		UnauthorizedError(w)
		return
	}

	sid, err := s.playSession.End(r.Context(), claims.ID)
	if err != nil {
		switch err {
		case session.ErrNoActiveSession:
			LogicError(w, "NO_ACTIVE_PLAY_SESSION")
			return
		case session.ErrSessionArchived:
			LogicError(w, "SESSION_ARCHIVED")
			return
		case session.ErrNotOwner:
			ForbiddenError(w)
			return
		case session.ErrSessionNotFound:
			NotFoundError(w)
			return
		default:
			InternalError(w)
			return
		}
	}

	// Сигналим всем и закрываем комнату
	if s.wsHub != nil {
		ev := map[string]any{"type": "system.end", "by": claims.ID}
		b, _ := json.Marshal(ev)
		s.wsHub.Broadcast("ps:"+sid, b)
		s.wsHub.CloseRoom("ps:" + sid)
	}

	w.WriteHeader(http.StatusNoContent)
}

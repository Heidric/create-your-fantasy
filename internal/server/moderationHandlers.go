package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/Heidric/create-your-fantasy/internal/lib/jwt"
	"github.com/Heidric/create-your-fantasy/internal/model"
	"github.com/Heidric/create-your-fantasy/internal/services/moderation"
	"github.com/go-chi/chi"
)

func (s *Server) moderationListHandler(w http.ResponseWriter, r *http.Request) {
	qp := r.URL.Query()

	var q model.ModerationListQuery
	if v := qp.Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			ValidationError(w, map[string]string{"page": model.ErrInvalidField})
			return
		}
		q.Page = &n
	}
	if v := qp.Get("pageSize"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			ValidationError(w, map[string]string{"pageSize": model.ErrInvalidField})
			return
		}
		q.PageSize = &n
	}
	q.Status = qp.Get("status")
	q.Type = qp.Get("type")
	q.AssignedTo = qp.Get("assignedTo")

	if errs := q.Validate(); len(errs) > 0 {
		ValidationError(w, errs)
		return
	}

	res, err := s.moderation.List(r.Context(), q)
	if err != nil {
		InternalError(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(res); err != nil {
		log.Error().Err(err).Msg("Error encoding response")
		InternalError(w)
		return
	}
}

func (s *Server) moderationReviewHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		BadRequestError(w)
		return
	}

	var dto model.ModerationReviewDTO
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

	if err := s.moderation.Review(r.Context(), claims.ID, id, dto); err != nil {
		switch {
		case errors.Is(err, moderation.ErrAlreadyFinished):
			ConflictError(w)
			return
		case errors.Is(err, moderation.ErrInvalidDecision):
		case errors.Is(err, moderation.ErrUnsupportedType):
		case errors.Is(err, moderation.ErrInvalidPayload):
			LogicError(w, model.ErrInvalidField)
			return
		default:
			InternalError(w)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

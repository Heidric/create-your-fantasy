package server

import (
	"encoding/json"
	"net/http"

	"github.com/Heidric/create-your-fantasy/internal/lib/jwt"
	"github.com/Heidric/create-your-fantasy/internal/model"
)

func (s *Server) profileGetHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	claims, ok := ctx.Value(jwt.CtxKeyClaims).(model.UserClaim)
	if !ok {
		UnauthorizedError(w)
		return
	}
	res, err := s.profile.Get(ctx, claims.ID)
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

func (s *Server) profileUpdateHandler(w http.ResponseWriter, r *http.Request) {
	var dto model.UpdateProfileDTO
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

	if err := s.profile.Update(r.Context(), claims.ID, dto); err != nil {
		InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Heidric/create-your-fantasy/internal/lib/jwt"
	"github.com/Heidric/create-your-fantasy/internal/model"
	"github.com/Heidric/create-your-fantasy/internal/services/report"
)

func (s *Server) createReportHandler(w http.ResponseWriter, r *http.Request) {
	var dto model.CreateReportDTO
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

	if err := s.report.Create(r.Context(), claims.ID, dto); err != nil {
		if errors.Is(err, report.ErrSubjectNotFound) {
			NotFoundError(w)
			return
		}
		InternalError(w)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

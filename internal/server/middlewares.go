package server

import (
	"net/http"
	"strings"

	"github.com/Heidric/create-your-fantasy/internal/lib/jwt"
	"github.com/Heidric/create-your-fantasy/internal/model"
)

func (s *Server) requirePermanentPassword(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v := r.Context().Value(jwt.CtxKeyClaims)
		claims, ok := v.(model.UserClaim)
		if !ok {
			UnauthorizedError(w)
			return
		}
		if claims.PasswordTemporary {
			ForbiddenError(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireModerator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(jwt.CtxKeyClaims).(model.UserClaim)
		if !ok {
			UnauthorizedError(w)
			return
		}
		role := strings.ToLower(claims.Role)
		if role != model.Moderator && role != model.Admin {
			ForbiddenError(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

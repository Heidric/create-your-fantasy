package server

import (
	"encoding/json"
	"net/http"

	"github.com/Heidric/create-your-fantasy/internal/lib/jwt"
	"github.com/Heidric/create-your-fantasy/internal/model"
	"github.com/go-chi/chi"
)

func (s *Server) playSessionWSHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := chi.URLParam(r, "id")
	if sessionID == "" {
		BadRequestError(w)
		return
	}
	claims, ok := r.Context().Value(jwt.CtxKeyClaims).(model.UserClaim)
	if !ok {
		UnauthorizedError(w)
		return
	}
	allowed, err := s.playSession.CanConnect(r.Context(), sessionID, claims.ID)
	if err != nil {
		InternalError(w)
		return
	}
	if !allowed {
		ForbiddenError(w)
		return
	}

	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	room := s.wsHub.EnsureRoom("ps:" + sessionID)
	client := s.wsNewClient(conn, room, claims.ID)

	room.Add(client)

	go client.ReadPump(func(from string, msg []byte) {
		env := map[string]any{
			"type": "chat.message",
			"from": from,
			"data": json.RawMessage(msg),
		}
		b, _ := json.Marshal(env)
		room.Broadcast(b)
	})

	go client.WritePump()
}

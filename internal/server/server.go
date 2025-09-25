package server

import (
	"context"
	"net/http"
	"time"

	"github.com/Heidric/create-your-fantasy/internal/lib/jwt"
	"github.com/Heidric/create-your-fantasy/internal/logger"
	"github.com/Heidric/create-your-fantasy/internal/model"
	"github.com/Heidric/create-your-fantasy/internal/ws"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

var log zerolog.Logger

type Auth interface {
	Register(ctx context.Context, email string) error
	Login(ctx context.Context, email, password string) (*model.LoginResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*model.RefreshTokenResponse, error)
	ResetPassword(ctx context.Context, email string) error
	ChangePassword(ctx context.Context, password, newPassword string) error
	ValidateSession(ctx context.Context) error
}

type Profile interface {
	Get(ctx context.Context, id string) (*model.ProfileResponse, error)
	Update(ctx context.Context, userID string, dto model.UpdateProfileDTO) error
}

type Moderation interface {
	List(ctx context.Context, q model.ModerationListQuery) (*model.ModerationListResponse, error)
	Review(ctx context.Context, moderatorID, id string, dto model.ModerationReviewDTO) error
}

type Report interface {
	Create(ctx context.Context, reporterID string, dto model.CreateReportDTO) error
}

type PlaySession interface {
	Create(ctx context.Context, ownerID string, dto model.CreatePlaySessionDTO) (string, error)
	CanConnect(ctx context.Context, sessionID, userID string) (bool, error)
	Join(ctx context.Context, sessionID, userID string) error
	Leave(ctx context.Context, userID string) (string, error)
	ListMessages(ctx context.Context, sessionID, userID string, q model.MessagesQuery) (*model.MessagesResponse, error)
	SendMessage(ctx context.Context, userID string, text string) (sessionID string, seq int64, sentAt time.Time, err error)
	Mute(ctx context.Context, gmUserID, targetUserID string) (sessionID string, err error)
	Unmute(ctx context.Context, gmUserID, targetUserID string) (sessionID string, err error)
	Remove(ctx context.Context, gmUserID, targetUserID string) (sessionID string, err error)
	End(ctx context.Context, gmUserID string) (sessionID string, err error)
}

type websocketUpgrader interface {
	Upgrade(w http.ResponseWriter, r *http.Request, responseHeader http.Header) (*websocket.Conn, error)
}

type Server struct {
	srv         *http.Server
	auth        Auth
	profile     Profile
	moderation  Moderation
	report      Report
	playSession PlaySession
	wsHub       *ws.Hub
	wsUpgrader  websocketUpgrader
	wsNewClient func(conn *websocket.Conn, room *ws.Room, userID string) *ws.Client
}

func NewServer(addr string, auth Auth, profile Profile, moderation Moderation, report Report, playSession PlaySession) *Server {
	log = *logger.Log
	log = log.With().Str("name", "http").Logger()

	r := chi.NewRouter()

	hub := ws.NewHub()

	s := &Server{
		srv:         &http.Server{Addr: addr, Handler: r},
		auth:        auth,
		profile:     profile,
		moderation:  moderation,
		report:      report,
		playSession: playSession,
		wsHub:       hub,
		wsUpgrader:  &ws.Upgrader,
		wsNewClient: ws.NewClient,
	}

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(logger.Middleware)
	r.Use(middleware.Recoverer)

	r.Group(func(r chi.Router) {
		r.Post(`/api/v1/auth/register`, s.registerHandler)
		r.Post(`/api/v1/auth/login`, s.loginHandler)
		r.Post(`/api/v1/auth/resetPassword`, s.resetPasswordHandler)
		r.Post(`/api/v1/auth/refreshToken`, s.refreshTokenHandler)
	})

	r.Group(func(r chi.Router) {
		r.Use(jwt.Authenticator(s.auth))

		r.Post(`/api/v1/auth/changePassword`, s.changePasswordHandler)
	})

	r.Group(func(r chi.Router) {
		r.Use(jwt.Authenticator(s.auth))
		r.Use(s.requirePermanentPassword)

		r.Get(`/api/v1/profile`, s.profileGetHandler)
		r.Put(`/api/v1/profile`, s.profileUpdateHandler)

		r.With(s.requireModerator).Get(`/api/v1/moderation/requests`, s.moderationListHandler)
		r.With(s.requireModerator).Put(`/api/v1/moderation/review/{id}`, s.moderationReviewHandler)

		r.Post(`/api/v1/createReport`, s.createReportHandler)

		r.Post(`/api/v1/playSession`, s.createPlaySessionHandler)
		r.Post(`/api/v1/playSession/join/{id}`, s.joinPlaySessionHandler)
		r.Post(`/api/v1/playSession/leave/{id}`, s.leavePlaySessionHandler)
		r.Get(`/api/v1/playSession/messages/{id}`, s.listPlaySessionMessagesHandler)
		r.Post(`/api/v1/playSession/sendMessage`, s.sendPlaySessionMessageHandler)
		r.Post(`/api/v1/playSession/mute/{id}`, s.mutePlaySessionMemberHandler)
		r.Post(`/api/v1/playSession/unmute/{id}`, s.unmutePlaySessionMemberHandler)
		r.Post(`/api/v1/playSession/remove/{id}`, s.removePlaySessionMemberHandler)
		r.Post(`/api/v1/playSession/end`, s.endPlaySessionHandler)

		r.Get(`/api/v1/playSession/ws/{id}`, s.playSessionWSHandler)
	})

	r.HandleFunc(`/*`, notFoundHandler)

	return s
}

func (s *Server) Run(ctx context.Context, runner *errgroup.Group) {
	logger.Log.Info().Msg("Http server started.")

	runner.Go(func() error {
		if err := s.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})
}

func (s *Server) Shutdown(ctx context.Context) error {
	logger.Log.Info().Msg("Http server stopped.")

	nctx, stop := context.WithTimeout(ctx, time.Second*10)
	defer stop()

	return s.srv.Shutdown(nctx)
}

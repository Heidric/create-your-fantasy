package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Heidric/create-your-fantasy/internal/model"
	"github.com/Heidric/create-your-fantasy/internal/server"
	"github.com/go-chi/chi"
)

type fakePlaySessionSvc struct {
	createRespID string
	createErr    error

	sendSess string
	sendSeq  int64
	sendAt   time.Time
	sendErr  error

	listResp *model.MessagesResponse
	listErr  error
}

func (f *fakePlaySessionSvc) Create(ctx context.Context, ownerID string, dto model.CreatePlaySessionDTO) (string, error) {
	return f.createRespID, f.createErr
}
func (f *fakePlaySessionSvc) ListMessages(ctx context.Context, sid, uid string, q model.MessagesQuery) (*model.MessagesResponse, error) {
	return f.listResp, f.listErr
}
func (f *fakePlaySessionSvc) SendMessage(ctx context.Context, uid string, text string) (string, int64, time.Time, error) {
	return f.sendSess, f.sendSeq, f.sendAt, f.sendErr
}

func (f *fakePlaySessionSvc) Join(ctx context.Context, sid, uid string) error { return nil }
func (f *fakePlaySessionSvc) Leave(ctx context.Context, uid string) error     { return nil }
func (f *fakePlaySessionSvc) Mute(ctx context.Context, gm, target string) (string, error) {
	return "", nil
}
func (f *fakePlaySessionSvc) Unmute(ctx context.Context, gm, target string) (string, error) {
	return "", nil
}
func (f *fakePlaySessionSvc) Remove(ctx context.Context, gm, target string) (string, error) {
	return "", nil
}
func (f *fakePlaySessionSvc) CanConnect(ctx context.Context, sid, uid string) (bool, error) {
	return true, nil
}
func (f *fakePlaySessionSvc) End(ctx context.Context, gm string) (string, error) { return "", nil }

func withUser(next http.Handler, claim model.UserClaim) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), modelCtxKeyClaims(), claim)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type ctxKey string

func modelCtxKeyClaims() ctxKey { return ctxKey("claims") }

func TestSendMessageHandler_Success(t *testing.T) {
	ps := &fakePlaySessionSvc{
		sendSess: "ps1", sendSeq: 7, sendAt: time.Now(),
	}
	s := makeServerForTest(ps)
	body := []byte(`{"text":"hello"}`)

	req := httptest.NewRequest("POST", "/api/v1/playSession/sendMessage", bytes.NewReader(body))
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", withUser(http.HandlerFunc(s.SendPlaySessionMessageHandlerForTest), model.UserClaim{ID: "u1"}))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("want 204, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestSendMessageHandler_Validation(t *testing.T) {
	ps := &fakePlaySessionSvc{}
	s := makeServerForTest(ps)

	req := httptest.NewRequest("POST", "/api/v1/playSession/sendMessage", bytes.NewReader([]byte(`{"text":""}`)))
	w := httptest.NewRecorder()

	r := chi.NewRouter()
	r.Mount("/", withUser(http.HandlerFunc(s.SendPlaySessionMessageHandlerForTest), model.UserClaim{ID: "u1"}))
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("want 422, got %d", w.Code)
	}
}

type testServer struct {
	s  *server.Server
	ps *fakePlaySessionSvc
}

func makeServerForTest(ps *fakePlaySessionSvc) *testServer {
	return &testServer{ps: ps}
}

func (ts *testServer) SendPlaySessionMessageHandlerForTest(w http.ResponseWriter, r *http.Request) {
	var dto model.SendMessageDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		server.BadRequestError(w)
		return
	}
	if errs := dto.Validate(); len(errs) > 0 {
		server.ValidationError(w, errs)
		return
	}
	claims, _ := r.Context().Value(modelCtxKeyClaims()).(model.UserClaim)
	if _, _, _, err := ts.ps.SendMessage(r.Context(), claims.ID, dto.Text); err != nil {
		server.InternalError(w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

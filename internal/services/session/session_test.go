package session_test

import (
	"context"
	"database/sql"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Heidric/create-your-fantasy/internal/model"
	"github.com/Heidric/create-your-fantasy/internal/services/session"
)

type memStore struct {
	mu sync.Mutex

	users        map[string]*model.User
	activeByUser map[string]string

	sessions map[string]*model.PlaySessionDB
	members  map[string]map[string]*model.PlaySessionMemberDB
	bans     map[string]map[string]bool
	seq      map[string]int64
	msgs     map[string][]model.MessageRow
}

func newMem() *memStore {
	return &memStore{
		users:        map[string]*model.User{},
		activeByUser: map[string]string{},
		sessions:     map[string]*model.PlaySessionDB{},
		members:      map[string]map[string]*model.PlaySessionMemberDB{},
		bans:         map[string]map[string]bool{},
		seq:          map[string]int64{},
		msgs:         map[string][]model.MessageRow{},
	}
}

func (m *memStore) addUser(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[id] = &model.User{ID: id}
}

func (m *memStore) setActive(u, sid string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if sid == "" {
		delete(m.activeByUser, u)
		return
	}
	m.activeByUser[u] = sid
}

func (m *memStore) ban(sid, uid string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.bans[sid] == nil {
		m.bans[sid] = map[string]bool{}
	}
	m.bans[sid][uid] = true
}

func (m *memStore) CreatePlaySession(ctx context.Context, ps *model.PlaySessionDB) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[ps.ID]; ok {
		return errors.New("dup")
	}
	m.sessions[ps.ID] = ps
	return nil
}

func (m *memStore) AddPlaySessionMember(ctx context.Context, mb *model.PlaySessionMemberDB) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members[mb.SessionID] == nil {
		m.members[mb.SessionID] = map[string]*model.PlaySessionMemberDB{}
	}
	if _, ok := m.members[mb.SessionID][mb.UserID]; ok {
		return errors.New("dup member")
	}
	m.members[mb.SessionID][mb.UserID] = mb
	return nil
}

func (m *memStore) GetPlaySessionByID(ctx context.Context, id string) (*model.PlaySessionDB, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ps, ok := m.sessions[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return ps, nil
}

func (m *memStore) IsPlaySessionMember(ctx context.Context, sid, uid string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members[sid] == nil {
		return false, nil
	}
	_, ok := m.members[sid][uid]
	return ok, nil
}

func (m *memStore) CountPlaySessionMembers(ctx context.Context, sid string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.members[sid]), nil
}

func (m *memStore) GetUserActiveSessionID(ctx context.Context, uid string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.activeByUser[uid], nil
}

func (m *memStore) SetUserActiveSession(ctx context.Context, uid, sid string) error {
	m.setActive(uid, sid)
	return nil
}

func (m *memStore) ClearUserActiveSession(ctx context.Context, uid string) error {
	m.setActive(uid, "")
	return nil
}

func (m *memStore) ClearUserActiveSessionIfMatch(ctx context.Context, uid, sid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeByUser[uid] == sid {
		delete(m.activeByUser, uid)
	}
	return nil
}

func (m *memStore) GetUserByID(ctx context.Context, id string) (*model.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}

func (m *memStore) NextSessionSeq(ctx context.Context, sid string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq[sid]++
	return m.seq[sid], nil
}

func (m *memStore) InsertPlaySessionMessage(ctx context.Context, id, sid, uid string, seq int64, text string, createdAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs[sid] = append(m.msgs[sid], model.MessageRow{
		ID: id, UserID: uid, Text: text, CreatedAt: createdAt, SeqID: int(seq),
	})
	return nil
}

func (m *memStore) CountPlaySessionMessages(ctx context.Context, sid string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.msgs[sid]), nil
}

func (m *memStore) ListPlaySessionMessages(ctx context.Context, sid string, afterSeq, limit int) ([]model.MessageRow, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	arr := m.msgs[sid]
	out := make([]model.MessageRow, 0, limit)
	for _, r := range arr {
		if r.SeqID > afterSeq {
			out = append(out, r)
		}
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *memStore) DeletePlaySessionMember(ctx context.Context, sid, uid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members[sid] == nil {
		return nil
	}
	delete(m.members[sid], uid)
	return nil
}

func (m *memStore) SetMemberMuted(ctx context.Context, sid, uid string, muted bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members[sid] == nil || m.members[sid][uid] == nil {
		return errors.New("no member")
	}
	m.members[sid][uid].Muted = muted
	return nil
}

func (m *memStore) GetPlaySessionMember(ctx context.Context, sid, uid string) (*model.PlaySessionMemberDB, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members[sid] == nil || m.members[sid][uid] == nil {
		return nil, sql.ErrNoRows
	}
	mb := m.members[sid][uid]
	return &model.PlaySessionMemberDB{SessionID: sid, UserID: uid, Role: mb.Role, Muted: mb.Muted}, nil
}

func (m *memStore) IsMemberMuted(ctx context.Context, sid, uid string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members[sid] == nil || m.members[sid][uid] == nil {
		return false, nil
	}
	return m.members[sid][uid].Muted, nil
}

func (m *memStore) IsBannedFromSession(ctx context.Context, sid, uid string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bans[sid] != nil && m.bans[sid][uid], nil
}

func (m *memStore) InsertPlaySessionBan(ctx context.Context, sid, uid string) error {
	m.ban(sid, uid)
	return nil
}

func (m *memStore) UpdatePlaySessionStatus(ctx context.Context, sid, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sessions[sid] == nil {
		return errors.New("no session")
	}
	m.sessions[sid].Status = status
	return nil
}

func (m *memStore) ClearActiveSessionForAll(ctx context.Context, sid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for u, s := range m.activeByUser {
		if s == sid {
			delete(m.activeByUser, u)
		}
	}
	return nil
}

func makeSvc() (*session.Service, *memStore, string, string) {
	st := newMem()
	svc := session.New(st)
	gm := "u-gm"
	p1 := "u-p1"
	st.addUser(gm)
	st.addUser(p1)
	return svc, st, gm, p1
}

func createSession(t *testing.T, svc *session.Service, st *memStore, gm string) string {
	t.Helper()
	dto := model.CreatePlaySessionDTO{
		Title: "Run", Ruleset: model.DnD5eRS, Capacity: 3, Visibility: model.VisibilityPublic,
	}
	id, err := svc.Create(context.Background(), gm, dto)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return id
}

func TestCreate_AddsGMAndActive(t *testing.T) {
	svc, st, gm, _ := makeSvc()
	id := createSession(t, svc, st, gm)

	is, _ := st.IsPlaySessionMember(context.Background(), id, gm)
	if !is {
		t.Fatalf("gm must be member")
	}
}

func TestJoin_SuccessAndCapacity(t *testing.T) {
	svc, st, gm, p1 := makeSvc()
	id := createSession(t, svc, st, gm)

	if err := svc.Join(context.Background(), id, p1); err != nil {
		t.Fatalf("join: %v", err)
	}
	if sid, _ := st.GetUserActiveSessionID(context.Background(), p1); sid != id {
		t.Fatalf("active session not set")
	}

	p2 := "u-p2"
	st.addUser(p2)
	if err := svc.Join(context.Background(), id, p2); err != nil {
		t.Fatalf("join p2: %v", err)
	}
	p3 := "u-p3"
	st.addUser(p3)
	if err := svc.Join(context.Background(), id, p3); !errors.Is(err, session.ErrCapacityExceeded) {
		t.Fatalf("want capacity exceeded, got %v", err)
	}
}

func TestJoin_Banned(t *testing.T) {
	svc, st, gm, p1 := makeSvc()
	id := createSession(t, svc, st, gm)
	st.ban(id, p1)

	if err := svc.Join(context.Background(), id, p1); !errors.Is(err, session.ErrForbiddenJoin) {
		t.Fatalf("want forbidden for banned, got %v", err)
	}
}

func TestLeave_ActiveOnly_And_NotOwner(t *testing.T) {
	svc, st, gm, p1 := makeSvc()
	id := createSession(t, svc, st, gm)
	_ = svc.Join(context.Background(), id, p1)

	if _, err := svc.Leave(context.Background(), p1); err != nil {
		t.Fatalf("leave: %v", err)
	}
	if sid, _ := st.GetUserActiveSessionID(context.Background(), p1); sid != "" {
		t.Fatalf("active not cleared")
	}
	st.setActive(gm, id)
	if _, err := svc.Leave(context.Background(), gm); !errors.Is(err, session.ErrForbiddenJoin) {
		t.Fatalf("gm leave must be forbidden, got %v", err)
	}
}

func TestSendMessage_NoActive_And_Muted(t *testing.T) {
	svc, st, gm, p1 := makeSvc()
	id := createSession(t, svc, st, gm)
	_ = svc.Join(context.Background(), id, p1)

	st.setActive(gm, id)

	_, _ = svc.Mute(context.Background(), gm, p1)
	_, _, _, err := svc.SendMessage(context.Background(), p1, "hi")
	if !errors.Is(err, session.ErrMuted) {
		t.Fatalf("want muted error, got %v", err)
	}

	_ = st.ClearUserActiveSession(context.Background(), p1)
	_, _, _, err = svc.SendMessage(context.Background(), p1, "hi")
	if !errors.Is(err, session.ErrNoActiveSession) {
		t.Fatalf("want no active, got %v", err)
	}
}

func TestMessages_AfterSeq(t *testing.T) {
	svc, st, gm, p1 := makeSvc()
	id := createSession(t, svc, st, gm)
	_ = svc.Join(context.Background(), id, p1)

	for i := 0; i < 3; i++ {
		if _, _, _, err := svc.SendMessage(context.Background(), p1, "m"); err != nil {
			t.Fatalf("send %d: %v", i, err)
		}
	}
	resp, err := svc.ListMessages(context.Background(), id, p1, model.MessagesQuery{
		LastMessage: ptrInt(1), PageSize: ptrInt(50),
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(resp.Items) != 2 || resp.LastMessage != 3 {
		t.Fatalf("want 2 items and last=3, got %d last=%d", len(resp.Items), resp.LastMessage)
	}
}

func TestMute_Unmute_Remove_End(t *testing.T) {
	svc, st, gm, p1 := makeSvc()
	id := createSession(t, svc, st, gm)
	_ = svc.Join(context.Background(), id, p1)

	st.setActive(gm, id)

	if _, err := svc.Mute(context.Background(), gm, p1); err != nil {
		t.Fatalf("mute: %v", err)
	}
	if _, err := svc.Unmute(context.Background(), gm, p1); err != nil {
		t.Fatalf("unmute: %v", err)
	}
	if _, err := svc.Remove(context.Background(), gm, p1); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if err := svc.Join(context.Background(), id, p1); !errors.Is(err, session.ErrForbiddenJoin) {
		t.Fatalf("rejoin after ban must be forbidden, got %v", err)
	}
	st.setActive(gm, id)
	if _, err := svc.End(context.Background(), gm); err != nil {
		t.Fatalf("end: %v", err)
	}
	ps, _ := st.GetPlaySessionByID(context.Background(), id)
	if ps.Status != model.Archived {
		t.Fatalf("must be archived")
	}
}

func ptrInt(n int) *int { return &n }

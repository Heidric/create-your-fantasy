package ws_test

import (
	"sync"
	"testing"

	"github.com/Heidric/create-your-fantasy/internal/ws"
)

func TestHub_RoomBroadcast(t *testing.T) {
	h := ws.NewHub()
	r := h.EnsureRoom("ps:1")
	var got [][]byte
	var mu sync.Mutex

	c1 := &ws.Client{}
	c1send := make(chan []byte, 1)
	c1SendField := getSendFieldPtr(c1)
	*c1SendField = c1send

	r.Add(c1)
	h.Broadcast("ps:1", []byte("hi"))

	select {
	case b := <-c1send:
		mu.Lock()
		got = append(got, b)
		mu.Unlock()
	default:
		t.Fatalf("no broadcast received")
	}
}

func getSendFieldPtr(c *ws.Client) *chan []byte {
	return &c.Send
}

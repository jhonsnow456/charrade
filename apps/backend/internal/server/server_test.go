package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hey-amanthakur/charrade/apps/backend/internal/game"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(New())
	t.Cleanup(ts.Close)
	return ts
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatalf("post %s: %v", url, err)
	}
	return resp
}

func decodeJSON(t *testing.T, resp *http.Response, out any) {
	t.Helper()
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
}

func TestCreateRoom(t *testing.T) {
	ts := newTestServer(t)
	resp := postJSON(t, ts.URL+"/api/rooms", createRoomRequest{Name: "Alice", Avatar: "avatar-1"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}

	var out createRoomResponse
	decodeJSON(t, resp, &out)
	if out.RoomID == "" || out.PlayerID == "" || out.HostID == "" {
		t.Fatalf("response has empty fields: %+v", out)
	}
	if out.PlayerID != out.HostID {
		t.Errorf("player %q should be host %q", out.PlayerID, out.HostID)
	}
}

func TestCreateRoomMissingName(t *testing.T) {
	ts := newTestServer(t)
	resp := postJSON(t, ts.URL+"/api/rooms", createRoomRequest{Avatar: "avatar-1"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", resp.StatusCode)
	}
}

func TestRoomIDIsShareableCode(t *testing.T) {
	ts := newTestServer(t)
	created := createRoomViaAPI(t, ts, "Alice")
	if len(created.RoomID) != 6 {
		t.Fatalf("roomID = %q, want 6 chars", created.RoomID)
	}
	for _, r := range created.RoomID {
		if !strings.ContainsRune("0123456789abcdef", r) {
			t.Fatalf("roomID = %q, want lowercase hex", created.RoomID)
		}
	}
	resp := postJSON(t, ts.URL+"/api/rooms/"+created.RoomID+"/players", createRoomRequest{
		Name: "Bob", Avatar: "avatar-2",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("join with shared code: status = %d, want 201", resp.StatusCode)
	}
}

func TestGetRoom(t *testing.T) {
	ts := newTestServer(t)
	created := createRoomViaAPI(t, ts, "Alice")

	resp, err := http.Get(ts.URL + "/api/rooms/" + created.RoomID)
	if err != nil {
		t.Fatalf("get room: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var room stateMessage
	decodeJSON(t, resp, &room)
	if len(room.Room.Players) != 1 {
		t.Errorf("len(players) = %d, want 1", len(room.Room.Players))
	}
	if room.Room.HostID != created.PlayerID {
		t.Errorf("host = %q, want %q", room.Room.HostID, created.PlayerID)
	}
}

func TestGetRoomNotFound(t *testing.T) {
	ts := newTestServer(t)
	resp, err := http.Get(ts.URL + "/api/rooms/nope")
	if err != nil {
		t.Fatalf("get room: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestAddPlayer(t *testing.T) {
	ts := newTestServer(t)
	created := createRoomViaAPI(t, ts, "Alice")

	resp := postJSON(t, ts.URL+"/api/rooms/"+created.RoomID+"/players", createRoomRequest{
		Name: "Bob", Avatar: "avatar-2",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("status = %d, want 201", resp.StatusCode)
	}
	var out createRoomResponse
	decodeJSON(t, resp, &out)
	if out.PlayerID == "" {
		t.Error("response playerID is empty")
	}

	getResp, _ := http.Get(ts.URL + "/api/rooms/" + created.RoomID)
	var room stateMessage
	decodeJSON(t, getResp, &room)
	if len(room.Room.Players) != 2 {
		t.Errorf("len(players) = %d, want 2", len(room.Room.Players))
	}
}

func TestAddPlayerToMissingRoom(t *testing.T) {
	ts := newTestServer(t)
	resp := postJSON(t, ts.URL+"/api/rooms/nope/players", createRoomRequest{Name: "Bob"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", resp.StatusCode)
	}
}

func TestAddPlayerAfterGameStarted(t *testing.T) {
	ts := newTestServer(t)
	host := createRoomViaAPI(t, ts, "Alice")
	createRoomViaAPI(t, ts, "Bob", "/api/rooms/"+host.RoomID+"/players")

	hostConn := dialWS(t, ts, host.RoomID, host.PlayerID)
	sendWS(t, hostConn, clientMessage{Type: "start"})
	waitFor(t, hostConn, func(s stateMessage) bool { return s.Room.Phase == "playing" })

	resp := postJSON(t, ts.URL+"/api/rooms/"+host.RoomID+"/players", createRoomRequest{
		Name: "Carol", Avatar: "avatar-3",
	})
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409", resp.StatusCode)
	}
}

func TestWebSocketFullRound(t *testing.T) {
	ts := newTestServer(t)
	host := createRoomViaAPI(t, ts, "Alice")
	guesser := createRoomViaAPI(t, ts, "Bob", "/api/rooms/"+host.RoomID+"/players")

	hostConn := dialWS(t, ts, host.RoomID, host.PlayerID)
	guesserConn := dialWS(t, ts, host.RoomID, guesser.PlayerID)

	sendWS(t, hostConn, clientMessage{Type: "start"})
	waitFor(t, hostConn, func(s stateMessage) bool { return s.Room.Phase == "playing" })
	waitFor(t, guesserConn, func(s stateMessage) bool { return s.Room.Phase == "playing" })

	sendWS(t, hostConn, clientMessage{Type: "startRound"})
	hostView := waitFor(t, hostConn, func(s stateMessage) bool {
		return s.Room.Round != nil && !s.Room.Round.Completed
	})
	if hostView.Room.Round.Word == "" {
		t.Fatal("host (actor) should see the round word")
	}
	guesserView := waitFor(t, guesserConn, func(s stateMessage) bool {
		return s.Room.Round != nil && !s.Room.Round.Completed
	})
	if guesserView.Room.Round.Word != "" {
		t.Error("guesser should not see the round word")
	}

	sendWS(t, guesserConn, clientMessage{Type: "guess", Text: hostView.Room.Round.Word})

	final := waitFor(t, guesserConn, func(s stateMessage) bool {
		if s.Room.Round == nil || !s.Room.Round.Completed {
			return false
		}
		guesserPlayer, ok := findPlayer(s.Room.Players, guesser.PlayerID)
		return ok && guesserPlayer.Score == 1
	})
	hostPlayer, _ := findPlayer(final.Room.Players, host.PlayerID)
	if hostPlayer.Score != 1 {
		t.Errorf("host score = %d, want 1", hostPlayer.Score)
	}
}

func createRoomViaAPI(t *testing.T, ts *httptest.Server, name string, path ...string) createRoomResponse {
	t.Helper()
	p := "/api/rooms"
	if len(path) > 0 {
		p = path[0]
	}
	resp := postJSON(t, ts.URL+p, createRoomRequest{Name: name, Avatar: "avatar-1"})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create room status = %d, want 201", resp.StatusCode)
	}
	var out createRoomResponse
	decodeJSON(t, resp, &out)
	return out
}

func dialWS(t *testing.T, ts *httptest.Server, roomID, playerID string) *websocket.Conn {
	t.Helper()
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/api/rooms/" + roomID + "/ws?playerId=" + playerID
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func sendWS(t *testing.T, conn *websocket.Conn, msg clientMessage) {
	t.Helper()
	if err := conn.WriteJSON(msg); err != nil {
		t.Fatalf("write ws: %v", err)
	}
}

func waitFor(t *testing.T, conn *websocket.Conn, pred func(stateMessage) bool) stateMessage {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn.SetReadDeadline(deadline)
		var msg stateMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read ws: %v", err)
		}
		if msg.Type != "state" {
			continue
		}
		if pred(msg) {
			return msg
		}
	}
	t.Fatal("timed out waiting for matching state message")
	return stateMessage{}
}

func TestWebSocketSignalRelay(t *testing.T) {
	ts := newTestServer(t)
	actor := createRoomViaAPI(t, ts, "Alice")
	viewer := createRoomViaAPI(t, ts, "Bob", "/api/rooms/"+actor.RoomID+"/players")

	actorConn := dialWS(t, ts, actor.RoomID, actor.PlayerID)
	viewerConn := dialWS(t, ts, actor.RoomID, viewer.PlayerID)

	sendWS(t, actorConn, clientMessage{
		Type:    "signal",
		To:      viewer.PlayerID,
		Payload: json.RawMessage(`{"type":"offer","sdp":"offer-1"}`),
	})

	got := waitForSignal(t, viewerConn, func(s signalMessage) bool { return true })
	if got.From != actor.PlayerID {
		t.Errorf("signal from = %q, want %q", got.From, actor.PlayerID)
	}
	var payload map[string]any
	if err := json.Unmarshal(got.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["type"] != "offer" || payload["sdp"] != "offer-1" {
		t.Errorf("payload = %v, want offer/offer-1", payload)
	}
}

func TestWebSocketAcceptsLargeSignal(t *testing.T) {
	ts := newTestServer(t)
	actor := createRoomViaAPI(t, ts, "Alice")
	viewer := createRoomViaAPI(t, ts, "Bob", "/api/rooms/"+actor.RoomID+"/players")

	actorConn := dialWS(t, ts, actor.RoomID, actor.PlayerID)
	viewerConn := dialWS(t, ts, actor.RoomID, viewer.PlayerID)

	bigSDP := strings.Repeat("a=line\r\n", 500)
	sendWS(t, actorConn, clientMessage{
		Type:    "signal",
		To:      viewer.PlayerID,
		Payload: json.RawMessage(`{"type":"offer","sdp":` + strconv.Quote(bigSDP) + `}`),
	})

	got := waitForSignal(t, viewerConn, func(s signalMessage) bool { return true })
	if got.From != actor.PlayerID {
		t.Errorf("signal from = %q, want %q", got.From, actor.PlayerID)
	}
	var payload map[string]any
	if err := json.Unmarshal(got.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["sdp"] != bigSDP {
		t.Error("relayed sdp does not match the sent payload")
	}
}

func TestWebSocketSignalToMissingPlayer(t *testing.T) {
	ts := newTestServer(t)
	actor := createRoomViaAPI(t, ts, "Alice")
	actorConn := dialWS(t, ts, actor.RoomID, actor.PlayerID)

	sendWS(t, actorConn, clientMessage{
		Type:    "signal",
		To:      "ghost",
		Payload: json.RawMessage(`{"candidate":{}}`),
	})

	err := waitForError(t, actorConn)
	if !strings.Contains(err, "not in room") {
		t.Errorf("error = %q, want a not-in-room message", err)
	}
}

func TestWebSocketSignalToDisconnectedPlayer(t *testing.T) {
	ts := newTestServer(t)
	actor := createRoomViaAPI(t, ts, "Alice")
	offline := createRoomViaAPI(t, ts, "Bob", "/api/rooms/"+actor.RoomID+"/players")

	actorConn := dialWS(t, ts, actor.RoomID, actor.PlayerID)

	sendWS(t, actorConn, clientMessage{
		Type:    "signal",
		To:      offline.PlayerID,
		Payload: json.RawMessage(`{"candidate":{}}`),
	})

	err := waitForError(t, actorConn)
	if !strings.Contains(err, "not connected") {
		t.Errorf("error = %q, want a not-connected message", err)
	}
}

func TestWebSocketSignalToSelf(t *testing.T) {
	ts := newTestServer(t)
	actor := createRoomViaAPI(t, ts, "Alice")
	actorConn := dialWS(t, ts, actor.RoomID, actor.PlayerID)

	sendWS(t, actorConn, clientMessage{
		Type:    "signal",
		To:      actor.PlayerID,
		Payload: json.RawMessage(`{"candidate":{}}`),
	})

	got := waitForSignal(t, actorConn, func(s signalMessage) bool { return true })
	if got.From != actor.PlayerID {
		t.Errorf("signal from = %q, want %q", got.From, actor.PlayerID)
	}
}

func waitForSignal(t *testing.T, conn *websocket.Conn, pred func(signalMessage) bool) signalMessage {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn.SetReadDeadline(deadline)
		var msg signalMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read ws: %v", err)
		}
		if msg.Type != "signal" {
			continue
		}
		if pred(msg) {
			return msg
		}
	}
	t.Fatal("timed out waiting for matching signal message")
	return signalMessage{}
}

func waitForError(t *testing.T, conn *websocket.Conn) string {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		conn.SetReadDeadline(deadline)
		var msg errorMessage
		if err := conn.ReadJSON(&msg); err != nil {
			t.Fatalf("read ws: %v", err)
		}
		if msg.Type == "error" {
			return msg.Message
		}
	}
	t.Fatal("timed out waiting for error message")
	return ""
}

func findPlayer(players []*game.Player, id string) (*game.Player, bool) {
	for _, p := range players {
		if p.ID == id {
			return p, true
		}
	}
	return nil, false
}

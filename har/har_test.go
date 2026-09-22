package har

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/websocket"
)

// fakeDevtools replays CDP frames, then holds the socket open like Chrome does.
func fakeDevtools(t *testing.T, frames []string) (string, chan struct{}) {
	t.Helper()
	enabled := make(chan struct{}, 1)
	handler := websocket.Handler(func(conn *websocket.Conn) {
		var first string
		if websocket.Message.Receive(conn, &first) == nil && strings.Contains(first, "Target.setAutoAttach") {
			enabled <- struct{}{}
		}
		for _, f := range frames {
			if websocket.Message.Send(conn, f) != nil {
				return
			}
		}
		<-time.After(5 * time.Second)
	})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return strings.TrimPrefix(server.URL, "http://"), enabled
}

func readHAR(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no HAR written: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("HAR is not valid JSON: %v", err)
	}
	return doc["log"].(map[string]any)
}

const requestFrame = `{"method":"Network.requestWillBeSent","params":{"requestId":"1",
 "timestamp":1000.0,"wallTime":1700000000.0,
 "request":{"url":"https://example.com/a","method":"GET","headers":{"Accept":"*/*"}}}}`

const responseFrame = `{"method":"Network.responseReceived","params":{"requestId":"1",
 "response":{"status":200,"statusText":"OK","mimeType":"text/html","protocol":"http/1.1",
 "headers":{"Content-Type":"text/html"}}}}`

const finishedFrame = `{"method":"Network.loadingFinished","params":{"requestId":"1",
 "timestamp":1000.25,"encodedDataLength":2048}}`

func TestUnfinishedRequestHasEmptyResponse(t *testing.T) {
	addr, enabled := fakeDevtools(t, []string{requestFrame})
	path := filepath.Join(t.TempDir(), "session.har")

	rec, err := Start(addr, "s1", path, "test")
	if err != nil {
		t.Fatal(err)
	}
	<-enabled
	time.Sleep(200 * time.Millisecond)
	if err := rec.Close(); err != nil {
		t.Fatal(err)
	}

	entries := readHAR(t, path)["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	response := entries[0].(map[string]any)["response"].(map[string]any)
	for _, key := range []string{"headers", "cookies"} {
		if _, ok := response[key].([]any); !ok {
			t.Errorf("response.%s should be an array, got %v", key, response[key])
		}
	}
}

func TestRecordsOneEntry(t *testing.T) {
	addr, enabled := fakeDevtools(t, []string{requestFrame, responseFrame, finishedFrame})
	path := filepath.Join(t.TempDir(), "session.har")

	rec, err := Start(addr, "s1", path, "test")
	if err != nil {
		t.Fatal(err)
	}
	<-enabled
	time.Sleep(200 * time.Millisecond)
	if err := rec.Close(); err != nil {
		t.Fatal(err)
	}

	log := readHAR(t, path)
	if log["version"] != "1.2" {
		t.Errorf("want HAR 1.2, got %v", log["version"])
	}
	entries := log["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0].(map[string]any)
	req := e["request"].(map[string]any)
	res := e["response"].(map[string]any)
	if req["url"] != "https://example.com/a" || req["method"] != "GET" {
		t.Errorf("request not recorded: %v", req)
	}
	if res["status"].(float64) != 200 || res["httpVersion"] != "http/1.1" {
		t.Errorf("response not recorded: %v", res)
	}
	if res["bodySize"].(float64) != 2048 {
		t.Errorf("encodedDataLength should become bodySize, got %v", res["bodySize"])
	}
	// 1000.25 - 1000.0 seconds, in milliseconds.
	if got := e["time"].(float64); got < 249 || got > 251 {
		t.Errorf("timing not derived from CDP timestamps: %v", got)
	}
	if !strings.HasPrefix(e["startedDateTime"].(string), "2023-11-14T") {
		t.Errorf("startedDateTime not derived from wallTime: %v", e["startedDateTime"])
	}
}

func TestIgnoresResponseWithoutRequest(t *testing.T) {
	addr, enabled := fakeDevtools(t, []string{responseFrame, finishedFrame})
	path := filepath.Join(t.TempDir(), "session.har")

	rec, err := Start(addr, "s1", path, "test")
	if err != nil {
		t.Fatal(err)
	}
	<-enabled
	time.Sleep(200 * time.Millisecond)
	_ = rec.Close()

	if entries := readHAR(t, path)["entries"].([]any); len(entries) != 0 {
		t.Fatalf("an unpaired response must not become an entry: %v", entries)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	addr, _ := fakeDevtools(t, nil)
	path := filepath.Join(t.TempDir(), "session.har")
	rec, err := Start(addr, "s1", path, "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := rec.Close(); err != nil {
		t.Fatal(err)
	}
	if err := rec.Close(); err != nil {
		t.Fatalf("second Close must be a no-op, got %v", err)
	}
}

func TestStartFailsOnUnreachableDevtools(t *testing.T) {
	if _, err := Start("127.0.0.1:1", "s1", filepath.Join(t.TempDir(), "x.har"), "test"); err == nil {
		t.Fatal("dialling a closed port must fail rather than record nothing silently")
	}
}

func TestEmptySessionStillWritesValidHAR(t *testing.T) {
	addr, _ := fakeDevtools(t, nil)
	path := filepath.Join(t.TempDir(), "session.har")
	rec, _ := Start(addr, "s1", path, "test")
	_ = rec.Close()

	log := readHAR(t, path)
	if log["entries"] == nil {
		t.Fatal("a session with no traffic must still produce an entries array")
	}
}

func TestEnablesNetworkOnAttachedTarget(t *testing.T) {
	attach := `{"method":"Target.attachedToTarget","params":{"sessionId":"page-1",
	 "targetInfo":{"type":"page"}}}`
	commands := make(chan string, 4)
	server := httptest.NewServer(websocket.Handler(func(conn *websocket.Conn) {
		var msg string
		for websocket.Message.Receive(conn, &msg) == nil {
			commands <- msg
			if strings.Contains(msg, "Target.setAutoAttach") {
				_ = websocket.Message.Send(conn, attach)
			}
		}
	}))
	defer server.Close()

	rec, err := Start(strings.TrimPrefix(server.URL, "http://"), "s1", filepath.Join(t.TempDir(), "x.har"), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer rec.Close()

	if first := <-commands; !strings.Contains(first, "Target.setAutoAttach") {
		t.Fatalf("expected auto-attach first, got %s", first)
	}
	select {
	case second := <-commands:
		if !strings.Contains(second, `"Network.enable"`) || !strings.Contains(second, `"sessionId":"page-1"`) {
			t.Fatalf("Network.enable must target the attached page session, got %s", second)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Network was never enabled on the attached target")
	}
}

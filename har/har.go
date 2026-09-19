// Package har records a session's network traffic as a HAR 1.2 log over CDP.
package har

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type Log struct {
	Version string  `json:"version"`
	Creator Creator `json:"creator"`
	Entries []Entry `json:"entries"`
}

type Creator struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type NameValue struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Request struct {
	Method      string      `json:"method"`
	URL         string      `json:"url"`
	HTTPVersion string      `json:"httpVersion"`
	Headers     []NameValue `json:"headers"`
	QueryString []NameValue `json:"queryString"`
	Cookies     []NameValue `json:"cookies"`
	HeadersSize int         `json:"headersSize"`
	BodySize    int         `json:"bodySize"`
}

type Content struct {
	Size     int64  `json:"size"`
	MimeType string `json:"mimeType"`
}

type Response struct {
	Status      int         `json:"status"`
	StatusText  string      `json:"statusText"`
	HTTPVersion string      `json:"httpVersion"`
	Headers     []NameValue `json:"headers"`
	Cookies     []NameValue `json:"cookies"`
	Content     Content     `json:"content"`
	RedirectURL string      `json:"redirectURL"`
	HeadersSize int         `json:"headersSize"`
	BodySize    int64       `json:"bodySize"`
}

type Timings struct {
	Send    float64 `json:"send"`
	Wait    float64 `json:"wait"`
	Receive float64 `json:"receive"`
}

type Entry struct {
	StartedDateTime string   `json:"startedDateTime"`
	Time            float64  `json:"time"`
	Request         Request  `json:"request"`
	Response        Response `json:"response"`
	Cache           struct{} `json:"cache"`
	Timings         Timings  `json:"timings"`

	started  float64
	finished float64
	seq      int
}

type cdpMessage struct {
	Method string `json:"method"`
	Params struct {
		SessionID string  `json:"sessionId"`
		RequestID string  `json:"requestId"`
		Timestamp float64 `json:"timestamp"`
		WallTime  float64 `json:"wallTime"`
		Request   struct {
			URL     string            `json:"url"`
			Method  string            `json:"method"`
			Headers map[string]string `json:"headers"`
		} `json:"request"`
		Response struct {
			Status     int               `json:"status"`
			StatusText string            `json:"statusText"`
			Headers    map[string]string `json:"headers"`
			MimeType   string            `json:"mimeType"`
			Protocol   string            `json:"protocol"`
		} `json:"response"`
		EncodedDataLength float64 `json:"encodedDataLength"`
	} `json:"params"`
}

// Recorder consumes CDP Network events until Close writes the file.
type Recorder struct {
	conn    *websocket.Conn
	path    string
	version string

	mu      sync.Mutex
	entries map[string]*Entry
	seq     int
	cmdID   int

	done   chan struct{}
	closed sync.Once
}

// Start goes through the hub's devtools proxy, which strips the Origin that CDP
// refuses and x/net/websocket always sends.
func Start(hubHostPort, sessionId, path, version string) (*Recorder, error) {
	url := fmt.Sprintf("ws://%s/devtools/%s/", hubHostPort, sessionId)
	conn, err := websocket.Dial(url, "", "http://"+hubHostPort)
	if err != nil {
		return nil, fmt.Errorf("dial devtools: %w", err)
	}
	r := &Recorder{
		conn:    conn,
		path:    path,
		version: version,
		entries: map[string]*Entry{},
		done:    make(chan struct{}),
	}
	// Browser-level endpoint: no page events until targets are attached.
	autoAttach := `{"id":1,"method":"Target.setAutoAttach","params":{"autoAttach":true,"waitForDebuggerOnStart":false,"flatten":true}}`
	if err := websocket.Message.Send(conn, autoAttach); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("auto-attach to targets: %w", err)
	}
	go r.read()
	return r, nil
}

func (r *Recorder) read() {
	defer close(r.done)
	for {
		var raw string
		if err := websocket.Message.Receive(r.conn, &raw); err != nil {
			return
		}
		var msg cdpMessage
		if json.Unmarshal([]byte(raw), &msg) != nil {
			continue
		}
		if msg.Method == "Target.attachedToTarget" && msg.Params.SessionID != "" {
			r.enableNetwork(msg.Params.SessionID)
			continue
		}
		r.apply(msg)
	}
}

func (r *Recorder) enableNetwork(sessionID string) {
	r.mu.Lock()
	r.cmdID++
	id := r.cmdID + 1
	r.mu.Unlock()
	cmd := fmt.Sprintf(`{"id":%d,"method":"Network.enable","sessionId":%q}`, id, sessionID)
	if err := websocket.Message.Send(r.conn, cmd); err != nil {
		return
	}
}

func (r *Recorder) apply(msg cdpMessage) {
	id := msg.Params.RequestID
	if id == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	switch msg.Method {
	case "Network.requestWillBeSent":
		e := &Entry{seq: r.seq, started: msg.Params.Timestamp}
		r.seq++
		e.StartedDateTime = time.Unix(0, int64(msg.Params.WallTime*float64(time.Second))).UTC().Format(time.RFC3339Nano)
		e.Request = Request{
			Method:      msg.Params.Request.Method,
			URL:         msg.Params.Request.URL,
			HTTPVersion: "HTTP/1.1",
			Headers:     pairs(msg.Params.Request.Headers),
			QueryString: []NameValue{},
			Cookies:     []NameValue{},
			HeadersSize: -1,
			BodySize:    -1,
		}
		r.entries[id] = e
	case "Network.responseReceived":
		if e, ok := r.entries[id]; ok {
			e.Response = Response{
				Status:      msg.Params.Response.Status,
				StatusText:  msg.Params.Response.StatusText,
				HTTPVersion: protocol(msg.Params.Response.Protocol),
				Headers:     pairs(msg.Params.Response.Headers),
				Cookies:     []NameValue{},
				Content:     Content{MimeType: msg.Params.Response.MimeType},
				RedirectURL: "",
				HeadersSize: -1,
				BodySize:    -1,
			}
		}
	case "Network.loadingFinished":
		if e, ok := r.entries[id]; ok {
			e.finished = msg.Params.Timestamp
			e.Response.BodySize = int64(msg.Params.EncodedDataLength)
			e.Response.Content.Size = e.Response.BodySize
		}
	}
}

// Close stops recording and writes the file; safe to call twice.
func (r *Recorder) Close() error {
	var err error
	r.closed.Do(func() {
		_ = r.conn.Close()
		<-r.done
		err = r.write()
	})
	return err
}

func (r *Recorder) write() error {
	r.mu.Lock()
	entries := make([]Entry, 0, len(r.entries))
	for _, e := range r.entries {
		entry := *e
		if entry.finished > entry.started {
			entry.Time = (entry.finished - entry.started) * 1000
		}
		entry.Timings = Timings{Send: 0, Wait: entry.Time, Receive: 0}
		entries = append(entries, entry)
	}
	r.mu.Unlock()

	sort.Slice(entries, func(i, j int) bool { return entries[i].seq < entries[j].seq })
	doc := struct {
		Log Log `json:"log"`
	}{Log{Version: "1.2", Creator: Creator{Name: "WebSummoner", Version: r.version}, Entries: entries}}

	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	return os.WriteFile(r.path, data, 0o644)
}

func pairs(headers map[string]string) []NameValue {
	out := make([]NameValue, 0, len(headers))
	for name, value := range headers {
		out = append(out, NameValue{Name: name, Value: value})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func protocol(p string) string {
	if p == "" {
		return "HTTP/1.1"
	}
	return p
}

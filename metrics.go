package main

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync/atomic"

	"github.com/websummoner/websummoner/session"
)

// Prometheus /metrics endpoint (upstream issue #889). Counters are atomic so
// they can be incremented from any request goroutine without locks. The
// output is plain Prometheus text format — no client library needed.

var (
	metricsSessionsCreated  atomic.Uint64
	metricsSessionsFailed   atomic.Uint64
	metricsSessionsTimedOut atomic.Uint64
	metricsSessionsDeleted  atomic.Uint64
	metricsVideoSessions    atomic.Uint64
	metricsVncSessions      atomic.Uint64
	metricsAudioSessions    atomic.Uint64
)

func init() {
	// session metrics are incremented from the proxy code path via these hooks
}

// MetricsSnapshot captures the current gauge values alongside the counters.
type MetricsSnapshot struct {
	SessionsActive   int
	SessionsLimit    int
	QueueDepth       int
	QueuePending     int
	BrowsersInUse    map[string]int // "chrome:152.0" -> count
	SessionsCreated  uint64
	SessionsFailed   uint64
	SessionsTimedOut uint64
	SessionsDeleted  uint64
	VideoSessions    uint64
	VncSessions      uint64
	AudioSessions    uint64
}

// collectMetrics assembles a snapshot from the running hub state.
func collectMetrics(limit int, queued int, pending int) MetricsSnapshot {
	m := MetricsSnapshot{
		SessionsLimit:    limit,
		QueueDepth:       queued,
		QueuePending:     pending,
		SessionsCreated:  metricsSessionsCreated.Load(),
		SessionsFailed:   metricsSessionsFailed.Load(),
		SessionsTimedOut: metricsSessionsTimedOut.Load(),
		SessionsDeleted:  metricsSessionsDeleted.Load(),
		VideoSessions:    metricsVideoSessions.Load(),
		VncSessions:      metricsVncSessions.Load(),
		AudioSessions:    metricsAudioSessions.Load(),
		BrowsersInUse:    make(map[string]int),
	}
	if sessions != nil {
		m.SessionsActive = sessions.Len()
		sessions.Each(func(_ string, s *session.Session) {
			key := s.Caps.BrowserName()
			if s.Caps.Version != "" {
				key = fmt.Sprintf("%s:%s", key, s.Caps.Version)
			}
			m.BrowsersInUse[key]++
		})
	}
	return m
}

// renderPrometheus produces the Prometheus text exposition format.
func renderPrometheus(m MetricsSnapshot) string {
	var sb strings.Builder

	writeGauge := func(name, help string, value int) {
		fmt.Fprintf(&sb, "# HELP %s %s\n# TYPE %s gauge\n%s %d\n", name, help, name, name, value)
	}
	writeCounter := func(name, help string, value uint64) {
		fmt.Fprintf(&sb, "# HELP %s %s\n# TYPE %s counter\n%s %d\n", name, help, name, name, value)
	}

	writeGauge("websummoner_sessions_active", "Currently running browser sessions", m.SessionsActive)
	writeGauge("websummoner_sessions_limit", "Maximum simultaneous sessions (-limit flag)", m.SessionsLimit)
	writeGauge("websummoner_queue_depth", "Requests waiting in the queue", m.QueueDepth)
	writeGauge("websummoner_queue_pending", "Requests being processed", m.QueuePending)

	writeCounter("websummoner_sessions_created_total", "Sessions successfully created", m.SessionsCreated)
	writeCounter("websummoner_sessions_failed_total", "Sessions that failed to start", m.SessionsFailed)
	writeCounter("websummoner_sessions_timed_out_total", "Sessions closed by idle timeout", m.SessionsTimedOut)
	writeCounter("websummoner_sessions_deleted_total", "Sessions deleted by client request", m.SessionsDeleted)
	writeCounter("websummoner_video_sessions_total", "Sessions with video recording enabled", m.VideoSessions)
	writeCounter("websummoner_vnc_sessions_total", "Sessions with VNC enabled", m.VncSessions)
	writeCounter("websummoner_audio_sessions_total", "Sessions with audio recording enabled", m.AudioSessions)

	// Per-browser gauges, sorted for deterministic output.
	if len(m.BrowsersInUse) > 0 {
		sb.WriteString("# HELP websummoner_browser_sessions Browser sessions by browser and version\n")
		sb.WriteString("# TYPE websummoner_browser_sessions gauge\n")
		keys := make([]string, 0, len(m.BrowsersInUse))
		for k := range m.BrowsersInUse {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&sb, "websummoner_browser_sessions{browser=%q} %d\n", k, m.BrowsersInUse[k])
		}
	}

	return sb.String()
}

// metricsHandler serves the Prometheus /metrics endpoint.
func metricsHandler(q queueMetrics) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		m := collectMetrics(limit, q.Queued(), q.Pending())
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		_, _ = w.Write([]byte(renderPrometheus(m)))
	}
}

// queueMetrics is a narrow interface so the handler can be tested without
// the full protect.Queue.
type queueMetrics interface {
	Queued() int
	Pending() int
}

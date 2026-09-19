package main

import (
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// resetMetricsCounters zeroes the package-level counters. The session tests
// drive the production paths in websummoner.go that increment them, so a test
// making absolute assertions has to start from a known zero — otherwise it
// passes on the first pass of the suite and fails on the second.
func resetMetricsCounters() {
	for _, c := range []*atomic.Uint64{
		&metricsSessionsCreated, &metricsSessionsFailed, &metricsSessionsTimedOut,
		&metricsSessionsDeleted, &metricsVideoSessions, &metricsVncSessions,
		&metricsAudioSessions,
	} {
		c.Store(0)
	}
}

type fakeQueueMetrics struct {
	queued, pending        int
	congested              bool
	noWait, timedOut, shed uint64
}

func (f fakeQueueMetrics) Queued() int     { return f.queued }
func (f fakeQueueMetrics) Pending() int    { return f.pending }
func (f fakeQueueMetrics) Congested() bool { return f.congested }
func (f fakeQueueMetrics) Rejected() (uint64, uint64, uint64) {
	return f.noWait, f.timedOut, f.shed
}

func TestMetricsEndpointFormat(t *testing.T) {
	handler := metricsHandler(fakeQueueMetrics{queued: 2, pending: 1})
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest("GET", "/metrics", nil))

	if w.Code != 200 {
		t.Fatalf("status = %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "# TYPE websummoner_sessions_active gauge") {
		t.Error("missing sessions_active gauge")
	}
	if !strings.Contains(body, "# TYPE websummoner_sessions_created_total counter") {
		t.Error("missing sessions_created counter")
	}
	if !strings.Contains(body, "websummoner_queue_depth 2") {
		t.Error("missing queue_depth value")
	}
	if !strings.Contains(w.Header().Get("Content-Type"), "version=0.0.4") {
		t.Error("wrong content type for Prometheus format")
	}
}

func TestMetricsCounterIncrement(t *testing.T) {
	resetMetricsCounters()
	t.Cleanup(resetMetricsCounters)

	metricsSessionsCreated.Add(5)
	metricsSessionsFailed.Add(1)
	snap := collectMetrics(10, fakeQueueMetrics{})
	if snap.SessionsCreated != 5 {
		t.Errorf("created = %d, want 5", snap.SessionsCreated)
	}
	if snap.SessionsFailed != 1 {
		t.Errorf("failed = %d, want 1", snap.SessionsFailed)
	}
}

func TestRemoveVendorOptionsStripsVersion(t *testing.T) {
	input := []byte(`{
		"capabilities": {
			"alwaysMatch": {
				"browserName": "chrome",
				"version": "152.0",
				"websummoner:options": {"enableVNC": true}
			}
		}
	}`)
	output := removeVendorOptions(input)
	s := string(output)
	if strings.Contains(s, `"version"`) {
		t.Error("legacy version capability must be stripped")
	}
	if strings.Contains(s, "websummoner:options") {
		t.Error("vendor options must be stripped")
	}
	if !strings.Contains(s, "browserName") {
		t.Error("browserName must be preserved")
	}
}

func TestRemoveVendorOptionsStripsFromLegacyToo(t *testing.T) {
	input := []byte(`{
		"desiredCapabilities": {
			"browserName": "firefox",
			"version": "154.0",
			"selenoid:options": {"enableVideo": true}
		}
	}`)
	output := removeVendorOptions(input)
	s := string(output)
	if strings.Contains(s, `"version"`) {
		t.Error("legacy version must be stripped from desiredCapabilities")
	}
	if strings.Contains(s, "selenoid:options") {
		t.Error("legacy vendor options must be stripped")
	}
	if !strings.Contains(s, "firefox") {
		t.Error("browserName must be preserved")
	}
}

func TestMetricsExposeAdmissionSignals(t *testing.T) {
	handler := metricsHandler(fakeQueueMetrics{
		queued: 7, congested: true, noWait: 3, timedOut: 11, shed: 5,
	})
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest("GET", "/metrics", nil))
	body := w.Body.String()

	for _, want := range []string{
		"websummoner_queue_congested 1",
		`websummoner_queue_rejected_total{reason="no_wait"} 3`,
		`websummoner_queue_rejected_total{reason="timeout"} 11`,
		`websummoner_queue_rejected_total{reason="shed"} 5`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in:\n%s", want, body)
		}
	}
}

func TestMetricsCongestionGaugeIsZeroWhenHealthy(t *testing.T) {
	handler := metricsHandler(fakeQueueMetrics{queued: 2})
	w := httptest.NewRecorder()
	handler(w, httptest.NewRequest("GET", "/metrics", nil))
	if !strings.Contains(w.Body.String(), "websummoner_queue_congested 0") {
		t.Error("a queue with waiters but no sustained backlog must not report congestion")
	}
}

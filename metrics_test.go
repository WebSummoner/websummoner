package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeQueueMetrics struct{ queued, pending int }

func (f fakeQueueMetrics) Queued() int  { return f.queued }
func (f fakeQueueMetrics) Pending() int { return f.pending }

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
	metricsSessionsCreated.Add(5)
	metricsSessionsFailed.Add(1)
	snap := collectMetrics(10, 0, 0)
	if snap.SessionsCreated != 5 {
		t.Errorf("created = %d, want 5", snap.SessionsCreated)
	}
	if snap.SessionsFailed != 1 {
		t.Errorf("failed = %d, want 1", snap.SessionsFailed)
	}
	// reset for other tests
	metricsSessionsCreated.Store(0)
	metricsSessionsFailed.Store(0)
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

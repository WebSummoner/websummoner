package protect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestQueueProtectChecksNewAndLegacyHeaders(t *testing.T) {
	for _, header := range []string{"X-WebSummoner-No-Wait", "X-Selenoid-No-Wait"} {
		queue := New(0, false)
		called := 0
		handler := queue.Try(queue.Protect(func(w http.ResponseWriter, _ *http.Request) { called++ }))
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set(header, "1")
		handler.ServeHTTP(rec, req)
		if called != 0 {
			t.Fatalf("handler must not run when queue is full (%s)", header)
		}
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("unexpected status via %s: %d", header, rec.Code)
		}
	}
}

func TestQueueProtectRecordsClientDisconnect(t *testing.T) {
	queue := New(0, false)
	called := 0
	handler := queue.Protect(func(w http.ResponseWriter, _ *http.Request) { called++ })
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), req)
		close(done)
	}()
	cancel()
	<-done
	if called != 0 {
		t.Fatal("handler must not run for a disconnected client")
	}
}

func TestQueueProtectRejectsWhenFull(t *testing.T) {
	// Without the no-wait header the request stays queued (blocked) until a
	// slot appears — emulate the client giving up.
	queue := New(0, false)
	called := 0
	handler := queue.Try(queue.Protect(func(w http.ResponseWriter, _ *http.Request) { called++ }))
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), req)
		close(done)
	}()
	cancel()
	<-done
	if called != 0 {
		t.Fatal("handler must not run when queue is full")
	}
}

func TestQueueTryNoWaitRepliesImmediatelyWhenFull(t *testing.T) {
	queue := New(0, false)
	called := 0
	handler := queue.Try(queue.Protect(func(w http.ResponseWriter, _ *http.Request) { called++ }))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-WebSummoner-No-Wait", "1")
	handler.ServeHTTP(rec, req)
	if called != 0 {
		t.Fatal("handler must not run when queue is full")
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestQueueTryNoWaitHonorsEmptyHeaderValue(t *testing.T) {
	for _, header := range []string{"X-WebSummoner-No-Wait", "X-Selenoid-No-Wait"} {
		queue := New(0, false)
		called := 0
		handler := queue.Try(queue.Protect(func(w http.ResponseWriter, _ *http.Request) { called++ }))
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		req.Header.Set(header, "")
		handler.ServeHTTP(rec, req)
		if called != 0 {
			t.Fatalf("handler must not run when queue is full (%s)", header)
		}
		if rec.Code != http.StatusTooManyRequests {
			t.Fatalf("empty-valued %s header ignored: status %d", header, rec.Code)
		}
	}
}

func TestQueueCheckRejectsWhenFullAndDisabled(t *testing.T) {
	queue := New(0, true)
	called := 0
	handler := queue.Check(func(w http.ResponseWriter, _ *http.Request) { called++ })
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if called != 0 {
		t.Fatal("handler must not be called when queue is full")
	}
	// Check-disabled path returns UnknownError (500), not 429.
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
}

func TestQueueCheckPassesWhenSlotFree(t *testing.T) {
	queue := New(1, false)
	called := 0
	handler := queue.Check(func(w http.ResponseWriter, _ *http.Request) { called++ })
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if called != 1 {
		t.Fatal("handler must be called when a slot is free")
	}
}

func TestQueueTryPassesWhenSlotFree(t *testing.T) {
	queue := New(1, false)
	called := 0
	handler := queue.Try(func(w http.ResponseWriter, _ *http.Request) { called++ })
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
	if called != 1 {
		t.Fatal("handler must be called when a slot is free")
	}
}

func TestQueueLifecycleCounters(t *testing.T) {
	queue := New(1, false)
	// Simulate an accepted request occupying the only slot with a pending session.
	queue.limit <- struct{}{}
	queue.pending <- struct{}{}
	queue.Create()
	if queue.Used() != 1 {
		t.Fatalf("used = %d, want 1", queue.Used())
	}
	if queue.Pending() != 0 {
		t.Fatalf("pending after create = %d, want 0", queue.Pending())
	}
	queue.Release()
	if queue.Used() != 0 {
		t.Fatalf("used after release = %d, want 0", queue.Used())
	}

	// A dropped request frees the slot without creating a session.
	queue.limit <- struct{}{}
	queue.pending <- struct{}{}
	queue.Drop()
	if queue.Pending() != 0 {
		t.Fatalf("pending after drop = %d, want 0", queue.Pending())
	}
}

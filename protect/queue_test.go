package protect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

func TestQueueProtectChecksNewAndLegacyHeaders(t *testing.T) {
	for _, header := range []string{"X-WebSummoner-No-Wait", "X-Selenoid-No-Wait"} {
		queue := New(0, false, 0, 0)
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
	queue := New(0, false, 0, 0)
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
	queue := New(0, false, 0, 0)
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
	queue := New(0, false, 0, 0)
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
		queue := New(0, false, 0, 0)
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
	queue := New(0, true, 0, 0)
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
	queue := New(1, false, 0, 0)
	called := 0
	handler := queue.Check(func(w http.ResponseWriter, _ *http.Request) { called++ })
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	if called != 1 {
		t.Fatal("handler must be called when a slot is free")
	}
}

func TestQueueTryPassesWhenSlotFree(t *testing.T) {
	queue := New(1, false, 0, 0)
	called := 0
	handler := queue.Try(func(w http.ResponseWriter, _ *http.Request) { called++ })
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/", nil))
	if called != 1 {
		t.Fatal("handler must be called when a slot is free")
	}
}

func TestQueueLifecycleCounters(t *testing.T) {
	queue := New(1, false, 0, 0)
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

func TestQueueTimeoutRepliesWhenSlotNeverFrees(t *testing.T) {
	queue := New(0, false, 50*time.Millisecond, 0)
	called := 0
	handler := queue.Protect(func(w http.ResponseWriter, _ *http.Request) { called++ })
	rec := httptest.NewRecorder()

	start := time.Now()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	elapsed := time.Since(start)

	if called != 0 {
		t.Fatal("handler must not run when the wait timed out")
	}
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("want 429, got %d", rec.Code)
	}
	if elapsed < 50*time.Millisecond {
		t.Fatalf("returned before the timeout elapsed: %s", elapsed)
	}
	if got := rec.Header().Get("Retry-After"); got != "1" {
		t.Fatalf("Retry-After must be at least one second, got %q", got)
	}
	// Or /status reports phantom waiters forever after the first timeout.
	if q := queue.Queued(); q != 0 {
		t.Fatalf("queued gauge leaked: %d", q)
	}
}

func TestQueueTimeoutDoesNotBlockAnAvailableSlot(t *testing.T) {
	queue := New(1, false, time.Minute, 0)
	called := 0
	handler := queue.Protect(func(w http.ResponseWriter, _ *http.Request) { called++ })
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))

	if called != 1 {
		t.Fatalf("handler must run when a slot is free, ran %d times", called)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", rec.Code)
	}
}

func TestQueueZeroTimeoutKeepsWaiting(t *testing.T) {
	// Zero must keep the historical wait-until-the-client-gives-up behaviour.
	queue := New(0, false, 0, 0)
	handler := queue.Protect(func(w http.ResponseWriter, _ *http.Request) {})
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	ctx, cancel := context.WithCancel(req.Context())
	req = req.WithContext(ctx)

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(httptest.NewRecorder(), req)
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("returned without waiting; zero must not time out")
	case <-time.After(100 * time.Millisecond):
	}
	cancel()
	<-done
}

func TestQueueNoWaitCarriesRetryAfter(t *testing.T) {
	queue := New(0, false, 30*time.Second, 0)
	handler := queue.Try(queue.Protect(func(w http.ResponseWriter, _ *http.Request) {}))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("X-WebSummoner-No-Wait", "1")

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("want 429, got %d", rec.Code)
	}
	got := rec.Header().Get("Retry-After")
	n, err := strconv.Atoi(got)
	if err != nil {
		t.Fatalf("Retry-After must be an integer number of seconds, got %q", got)
	}
	if n < 1 || n > 30 {
		t.Fatalf("Retry-After %d outside the jitter window [1, 30]", n)
	}
}

func TestRetryAfterIsJittered(t *testing.T) {
	// A constant value would align every rejected client onto one second.
	seen := map[int]bool{}
	for i := 0; i < 300; i++ {
		rec := httptest.NewRecorder()
		retryAfter(rec, time.Minute)
		n, err := strconv.Atoi(rec.Header().Get("Retry-After"))
		if err != nil {
			t.Fatalf("Retry-After not an integer: %v", err)
		}
		if n < 1 || n > 60 {
			t.Fatalf("Retry-After %d outside [1, 60]", n)
		}
		seen[n] = true
	}
	if len(seen) < 20 {
		t.Fatalf("only %d distinct Retry-After values in 300 responses; clients would resynchronise", len(seen))
	}
}

func TestBudgetFollowsControlledDelay(t *testing.T) {
	// if lastEmptyTime < now - N -> timeout = M, else timeout = N
	const tolerate, congested = 100 * time.Millisecond, 20 * time.Millisecond
	q := New(0, false, tolerate, congested)

	if d, shed := q.budget(); d != tolerate || shed {
		t.Fatalf("a queue that just served somebody is healthy, got %s shed=%v", d, shed)
	}

	// No prompt admission for longer than the tolerated interval.
	q.lastEmpty.Store(time.Now().Add(-time.Second).UnixNano())
	if d, shed := q.budget(); d != congested || !shed {
		t.Fatalf("a stalled queue must shed, got %s shed=%v", d, shed)
	}

	// A prompt admission clears it again.
	q.admitted(0)
	if d, shed := q.budget(); d != tolerate || shed {
		t.Fatalf("a prompt admission must clear the congested state, got %s shed=%v", d, shed)
	}
}

func TestSlowAdmissionDoesNotClearCongestion(t *testing.T) {
	// Otherwise a grid freeing the odd slot flaps out of shedding.
	q := New(0, false, time.Minute, time.Second)
	q.lastEmpty.Store(time.Now().Add(-time.Hour).UnixNano())

	q.admitted(30 * time.Second) // served, but only after a long wait
	if _, shed := q.budget(); !shed {
		t.Fatal("an admission slower than target must not count as keeping up")
	}

	q.admitted(time.Second) // within target
	if _, shed := q.budget(); shed {
		t.Fatal("an admission within target must clear the congested state")
	}
}

func TestCongestionIgnoresInstantaneousDepth(t *testing.T) {
	// A saturated grid empties constantly as waiters expire, so depth reads
	// healthy when it is worst.
	q := New(0, false, 50*time.Millisecond, 10*time.Millisecond)
	q.lastEmpty.Store(time.Now().Add(-time.Second).UnixNano())

	if n := q.Queued(); n != 0 {
		t.Fatalf("precondition: queue should look empty, got %d", n)
	}
	if _, shed := q.budget(); !shed {
		t.Fatal("an empty-looking but stalled queue must still shed")
	}
	// The gauge is the exception: it needs real waiters before firing.
	if q.Congested() {
		t.Fatal("the gauge must stay clear while nobody is waiting")
	}
	q.queued <- struct{}{}
	if !q.Congested() {
		t.Fatal("the gauge must fire once demand is actually queued")
	}
}

func TestQueueShedsAfterSustainedCongestion(t *testing.T) {
	const tolerate, congested = 2 * time.Second, 20 * time.Millisecond
	q := New(0, false, tolerate, congested)
	q.lastEmpty.Store(time.Now().Add(-time.Minute).UnixNano())

	handler := q.Protect(func(w http.ResponseWriter, _ *http.Request) {
		t.Error("handler must not run while shedding")
	})
	rec := httptest.NewRecorder()
	start := time.Now()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))
	elapsed := time.Since(start)

	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("want 429, got %d", rec.Code)
	}
	// A saturated grid must reject in M, not N.
	if elapsed > tolerate/2 {
		t.Fatalf("shed took %s; it should use the aggressive budget of %s", elapsed, congested)
	}
	noWait, timedOut, shed := q.Rejected()
	if shed != 1 || timedOut != 0 || noWait != 0 {
		t.Fatalf("wrong rejection reason recorded: noWait=%d timeout=%d shed=%d", noWait, timedOut, shed)
	}
	if q.Queued() != 0 {
		t.Fatalf("queued gauge leaked: %d", q.Queued())
	}
}

func TestCongestedBudgetDerivedFromTolerate(t *testing.T) {
	q := New(0, false, 5*time.Minute, 0)
	if q.congested != 30*time.Second {
		t.Fatalf("congested budget should default to a tenth of tolerate, got %s", q.congested)
	}
	if q.target != 15*time.Second {
		t.Fatalf("target should default to a twentieth of tolerate, got %s", q.target)
	}
	// Never below a second, or a burst sheds before a browser can start.
	q = New(0, false, 2*time.Second, 0)
	if q.congested != time.Second {
		t.Fatalf("congested budget should floor at 1s, got %s", q.congested)
	}
}

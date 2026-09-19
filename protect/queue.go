// Package protect implements admission control using CoDel, the controlled
// delay algorithm (Nichols & Jacobson, ACM Queue 2012, queue.acm.org/detail.cfm?id=2209336)
// applied to session slots rather than packets: tolerate is its interval,
// target is its target. Retry-After uses full jitter.
package protect

import (
	"errors"
	"fmt"
	"log"
	"math"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/websummoner/websummoner/info"
	"github.com/websummoner/websummoner/jsonerror"
)

// Queue - struct to hold a number of sessions
type Queue struct {
	disabled bool

	tolerate  time.Duration // CoDel interval, and the healthy wait budget; zero disables
	congested time.Duration // wait budget while shedding
	target    time.Duration // CoDel target: a wait this short means we are keeping up

	// Last admission within target, in unix nanos. Not queue depth: a saturated
	// grid empties constantly as waiters expire, so depth looks healthy when it
	// is worst.
	lastEmpty atomic.Int64

	rejectedNoWait  atomic.Uint64
	rejectedTimeout atomic.Uint64
	rejectedShed    atomic.Uint64

	limit   chan struct{}
	queued  chan struct{}
	pending chan struct{}
	used    chan struct{}
}

// Try - when X-WebSummoner-No-Wait (or legacy X-Selenoid-No-Wait) header is set
// reply to client immediately if queue is full
func (q *Queue) Try(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Ggr sends these with an empty value, so presence is what matters.
		noWait := len(r.Header.Values("X-WebSummoner-No-Wait")) > 0 ||
			len(r.Header.Values("X-Selenoid-No-Wait")) > 0
		select {
		case q.limit <- struct{}{}:
			<-q.limit
		default:
			if noWait {
				q.rejectedNoWait.Add(1)
				budget, _ := q.budget()
				retryAfter(w, budget)
				err := errors.New(http.StatusText(http.StatusTooManyRequests))
				jsonerror.TooManyRequests(err).Encode(w)
				return
			}
		}
		next.ServeHTTP(w, r)
	}
}

// Check - if queue disabled
func (q *Queue) Check(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case q.limit <- struct{}{}:
			<-q.limit
		default:
			if q.disabled {
				user, remote := info.RequestInfo(r)
				log.Printf("[-] [QUEUE_IS_FULL] [%s] [%s]", user, remote)
				err := errors.New("queue is full")
				jsonerror.UnknownError(err).Encode(w)
				return
			}
		}
		next.ServeHTTP(w, r)
	}
}

// budget returns this request's wait allowance and whether we are shedding:
//
//	if lastEmptyTime < now - N { timeout = M } else { timeout = N }
func (q *Queue) budget() (time.Duration, bool) {
	if q.tolerate <= 0 {
		return 0, false
	}
	if q.stalled() {
		return q.congested, true
	}
	return q.tolerate, false
}

// stalled - nobody served promptly for longer than tolerate
func (q *Queue) stalled() bool {
	return time.Since(time.Unix(0, q.lastEmpty.Load())) > q.tolerate
}

// admitted - only a wait within target clears the congested state, so a grid
// freeing the odd slot does not flap back to lenient while everyone else waits.
func (q *Queue) admitted(waited time.Duration) {
	if waited <= q.target {
		q.lastEmpty.Store(time.Now().UnixNano())
	}
}

// Protect - handler to control limit of sessions
func (q *Queue) Protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, remote := info.RequestInfo(r)
		log.Printf("[-] [NEW_REQUEST] [%s] [%s]", user, remote)
		s := time.Now()

		budget, congested := q.budget()

		go func() {
			q.queued <- struct{}{}
		}()

		// A nil channel blocks forever, which is the no-budget behaviour.
		var expired <-chan time.Time
		if budget > 0 {
			t := time.NewTimer(budget)
			defer t.Stop()
			expired = t.C
		}

		select {
		case <-r.Context().Done():
			<-q.queued
			log.Printf("[-] [CLIENT_DISCONNECTED] [%s] [%s] [%s]", user, remote, time.Since(s))
			return
		case <-expired:
			<-q.queued
			status, reason := "QUEUE_TIMEOUT", "waiting for a free slot"
			if congested {
				q.rejectedShed.Add(1)
				status, reason = "QUEUE_SHED", "grid has been saturated for longer than the tolerated interval"
			} else {
				q.rejectedTimeout.Add(1)
			}
			log.Printf("[-] [%s] [%s] [%s] [%s]", status, user, remote, time.Since(s))
			retryAfter(w, q.tolerate)
			jsonerror.TooManyRequests(
				fmt.Errorf("timed out after %s: %s", budget, reason)).Encode(w)
			return
		case q.limit <- struct{}{}:
			q.pending <- struct{}{}
		}
		<-q.queued
		q.admitted(time.Since(s))
		log.Printf("[-] [NEW_REQUEST_ACCEPTED] [%s] [%s]", user, remote)
		next.ServeHTTP(w, r)
	}
}

// Used - get created sessions
func (q *Queue) Used() int {
	return len(q.used)
}

// Pending - get pending sessions
func (q *Queue) Pending() int {
	return len(q.pending)
}

// Queued - get queued sessions
func (q *Queue) Queued() int {
	return len(q.queued)
}

// Rejected - admission refusals by reason, for /metrics
func (q *Queue) Rejected() (noWait, timeout, shed uint64) {
	return q.rejectedNoWait.Load(), q.rejectedTimeout.Load(), q.rejectedShed.Load()
}

// Congested - shedding state for /metrics. Needs real waiters too, or an idle
// hub would page somebody overnight.
func (q *Queue) Congested() bool {
	if q.tolerate <= 0 || len(q.queued) == 0 {
		return false
	}
	return q.stalled()
}

// Drop - session is not created
func (q *Queue) Drop() {
	<-q.limit
	<-q.pending
}

// Create - session is created
func (q *Queue) Create() {
	q.used <- <-q.pending
}

// Release - session is closed
func (q *Queue) Release() {
	<-q.limit
	<-q.used
}

// retryAfter - full jitter over [1, d]. A constant value is worse than none:
// it aligns independent clients onto the same second.
func retryAfter(w http.ResponseWriter, d time.Duration) {
	if d <= 0 {
		d = time.Minute
	}
	upper := int(math.Ceil(d.Seconds()))
	if upper < 1 {
		upper = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(1+rand.IntN(upper)))
}

// Derived from tolerate so operators tune one number. The 5% target:interval
// ratio is CoDel's own (5ms against 100ms).
const (
	DefaultCongestedFraction = 10
	DefaultTargetFraction    = 20
)

// New - create and initialize queue. Zero tolerate disables admission
// timeouts; zero congested derives one from tolerate.
func New(size int, disabled bool, tolerate, congested time.Duration) *Queue {
	if congested <= 0 {
		// Only the derived default gets a floor; an explicit 20ms means 20ms.
		congested = tolerate / DefaultCongestedFraction
		if tolerate > 0 && congested < time.Second {
			congested = time.Second
		}
	}
	target := tolerate / DefaultTargetFraction
	if tolerate > 0 && target < time.Second {
		target = time.Second
	}
	q := &Queue{
		disabled:  disabled,
		tolerate:  tolerate,
		congested: congested,
		target:    target,
		limit:     make(chan struct{}, size),
		queued:    make(chan struct{}, math.MaxInt32),
		pending:   make(chan struct{}, math.MaxInt32),
		used:      make(chan struct{}, math.MaxInt32),
	}
	q.lastEmpty.Store(time.Now().UnixNano())
	return q
}

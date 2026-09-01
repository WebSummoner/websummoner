package protect

import (
	"errors"
	"github.com/websummoner/websummoner/info"
	"log"
	"math"
	"net/http"
	"time"

	"github.com/websummoner/websummoner/jsonerror"
)

// Queue - struct to hold a number of sessions
type Queue struct {
	disabled bool
	limit    chan struct{}
	queued   chan struct{}
	pending  chan struct{}
	used     chan struct{}
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

// Protect - handler to control limit of sessions
func (q *Queue) Protect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, remote := info.RequestInfo(r)
		log.Printf("[-] [NEW_REQUEST] [%s] [%s]", user, remote)
		s := time.Now()
		go func() {
			q.queued <- struct{}{}
		}()
		select {
		case <-r.Context().Done():
			<-q.queued
			log.Printf("[-] [CLIENT_DISCONNECTED] [%s] [%s] [%s]", user, remote, time.Since(s))
			return
		case q.limit <- struct{}{}:
			q.pending <- struct{}{}
		}
		<-q.queued
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

// New - create and initialize queue
func New(size int, disabled bool) *Queue {
	return &Queue{
		disabled,
		make(chan struct{}, size),
		make(chan struct{}, math.MaxInt32),
		make(chan struct{}, math.MaxInt32),
		make(chan struct{}, math.MaxInt32),
	}
}

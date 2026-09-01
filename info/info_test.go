package info

import (
	"net/http/httptest"
	"testing"
)

func TestRequestInfoUnknownUserAndAddress(t *testing.T) {
	r := httptest.NewRequest("GET", "http://localhost/", nil)
	user, remote := RequestInfo(r)
	if user != "unknown" {
		t.Fatalf("unexpected user: %s", user)
	}
	if remote != "192.0.2.1" {
		t.Fatalf("unexpected remote: %s", remote)
	}
}

func TestRequestInfoBasicAuth(t *testing.T) {
	r := httptest.NewRequest("GET", "http://localhost/", nil)
	r.SetBasicAuth("alice", "secret")
	user, _ := RequestInfo(r)
	if user != "alice" {
		t.Fatalf("unexpected user: %s", user)
	}
}

func TestRequestInfoForwardedFor(t *testing.T) {
	r := httptest.NewRequest("GET", "http://localhost/", nil)
	r.Header.Set("X-Forwarded-For", "10.0.0.1")
	_, remote := RequestInfo(r)
	if remote != "10.0.0.1" {
		t.Fatalf("unexpected remote: %s", remote)
	}
}

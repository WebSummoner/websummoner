package main

import (
	"net/http/httptest"
	"testing"
)

func TestPingHeadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("HEAD", "/ping", nil)
	ping(w, r)
	if w.Code != 200 {
		t.Fatalf("HEAD /ping status = %d, want 200", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("HEAD /ping must have empty body, got %d bytes", w.Body.Len())
	}
}

func TestPingGetRequest(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/ping", nil)
	ping(w, r)
	if w.Code != 200 {
		t.Fatalf("GET /ping status = %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Fatal("GET /ping must return a JSON body")
	}
}

func TestStatusHeadRequest(t *testing.T) {
	handler := handler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("HEAD", "/status", nil)
	handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("HEAD /status status = %d, want 200 (ready when no sessions)", w.Code)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("HEAD /status must have empty body, got %d bytes", w.Body.Len())
	}
}

func TestStatusGetRequest(t *testing.T) {
	handler := handler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/status", nil)
	handler.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("GET /status status = %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Fatal("GET /status must return a JSON body")
	}
}

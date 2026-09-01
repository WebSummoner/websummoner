package jsonerror

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSeleniumErrorMessage(t *testing.T) {
	wrapped := errors.New("boom")
	se := InvalidArgument(wrapped)
	if se.Error() != "invalid argument: boom" {
		t.Fatalf("unexpected message: %s", se.Error())
	}
}

func TestSeleniumErrorEncode(t *testing.T) {
	w := httptest.NewRecorder()
	InvalidSessionID(errors.New("session timed out or not found")).Encode(w)
	if w.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: %d", w.Code)
	}
	var parsed struct {
		Value struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		} `json:"value"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("not valid json: %v", err)
	}
	if parsed.Value.Error != "invalid session id" {
		t.Fatalf("unexpected error name: %s", parsed.Value.Error)
	}
}

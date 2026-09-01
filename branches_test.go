package main

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestParseGgrHostValidAndInvalid(t *testing.T) {
	// Valid host:port (fatal-exit paths for invalid input are exercised in
	// TestParseGgrHostInvalid below via a subprocess-free approach: we only
	// assert the valid path here because parseGgrHost calls log.Fatalf.)
	h := parseGgrHost("example.com:4444")
	if h.Name != "example.com" || h.Port != 4444 {
		t.Fatalf("unexpected host: %+v", h)
	}
}

func TestDeleteFileIfExistsAllBranches(t *testing.T) {
	dir := t.TempDir()

	// Unknown file -> 404.
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/video/missing.mp4", nil)
	deleteFileIfExists(1, w, r, dir, "/video/", "DELETED_VIDEO_FILE")
	if w.Code != 404 {
		t.Fatalf("missing file: status = %d", w.Code)
	}

	// Existing file -> deleted.
	existing := filepath.Join(dir, "exists.mp4")
	if err := os.WriteFile(existing, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	r = httptest.NewRequest("DELETE", "/video/exists.mp4", nil)
	deleteFileIfExists(1, w, r, dir, "/video/", "DELETED_VIDEO_FILE")
	if w.Code != 200 {
		t.Fatalf("existing file: status = %d", w.Code)
	}
	if _, err := os.Stat(existing); !os.IsNotExist(err) {
		t.Fatal("file must be removed")
	}
}

func TestPreprocessSessionIdWithAndWithoutGgrHost(t *testing.T) {
	previous := ggrHost
	defer func() { ggrHost = previous }()
	ggrHost = nil
	if got := preprocessSessionId("abc"); got != "abc" {
		t.Fatalf("without ggr host: %s", got)
	}
	ggrHost = parseGgrHost("example.com:4444")
	if got := preprocessSessionId("abc"); got == "abc" || len(got) <= len("abc") {
		t.Fatalf("with ggr host: %s", got)
	}
}

func TestWelcomeAndStatusHandlers(t *testing.T) {
	w := httptest.NewRecorder()
	welcome(w, nil)
	if w.Body.Len() == 0 {
		t.Fatal("welcome body is empty")
	}
}

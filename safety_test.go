package main

import (
	"fmt"
	"strings"
	"testing"

	assert "github.com/stretchr/testify/require"
)

func TestProcessBodyNoCdpForSafariAndFirefox(t *testing.T) {
	for _, bn := range []string{"safari", "firefox"} {
		body := fmt.Sprintf(`{"value": {"capabilities": {"browserName": "%s"}, "sessionId": "abc"}}`, bn)
		out, _, err := processBody([]byte(body), "example.com:4444")
		assert.NoError(t, err)
		assert.False(t, strings.Contains(string(out), "se:cdp"), bn)
	}
}

func TestProcessBodyCdpForChromium(t *testing.T) {
	body := `{"value": {"capabilities": {"browserName": "chrome"}, "sessionId": "abc"}}`
	out, _, err := processBody([]byte(body), "example.com:4444")
	assert.NoError(t, err)
	assert.True(t, strings.Contains(string(out), "se:cdp"))
}

// operadriver answers in legacy JSONWP unless asked for W3C explicitly.
func TestAdaptDriverCapabilitiesOpera(t *testing.T) {
	out := adaptDriverCapabilities([]byte(`{"capabilities":{"alwaysMatch":{"browserName":"opera"}}}`), "opera", 7)
	assert.Contains(t, string(out), `"capabilities"`)
	assert.NotContains(t, string(out), `"desiredCapabilities"`)
	assert.Contains(t, string(out), `"browserName":"chrome"`)
	assert.Contains(t, string(out), `"/usr/bin/opera"`)
	assert.Contains(t, string(out), `"w3c":true`)
}

func TestAdaptDriverCapabilitiesYandex(t *testing.T) {
	in := `{"capabilities":{"firstMatch":[{"browserName":"yandex"}]}}`
	out := adaptDriverCapabilities([]byte(in), "yandex", 9)
	assert.Contains(t, string(out), `"browserName":"chrome"`)
	assert.Contains(t, string(out), `"/usr/bin/yandex-browser"`)
}

func TestAdaptDriverCapabilitiesPassthrough(t *testing.T) {
	in := `{"capabilities":{"alwaysMatch":{"browserName":"firefox"}}}`
	assert.Equal(t, in, string(adaptDriverCapabilities([]byte(in), "firefox", 1)))
}

func TestIsSafeFileName(t *testing.T) {
	safe := []string{"video.mp4", "session-1.log", "websummoner00ff.mp4", "..hidden.mp4"}
	for _, name := range safe {
		assert.True(t, isSafeFileName(name), name)
	}
	unsafe := []string{"", ".", "..", "../x.mp4", "a/../../b.mp4", "/etc/passwd", `..\x.mp4`, "dir/x.mp4"}
	for _, name := range unsafe {
		assert.False(t, isSafeFileName(name), name)
	}
}

func TestProcessBodyMissingSessionIdDoesNotPanic(t *testing.T) {
	for _, body := range []string{
		`{"value": {"capabilities": {}}}`,
		`{"value": {"capabilities": {}, "sessionId": 42}}`,
	} {
		_, _, err := processBody([]byte(body), "localhost:4444")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "sessionId")
	}
}

func TestProcessBodyValidW3CResponse(t *testing.T) {
	out, sessionId, err := processBody([]byte(`{"value": {"capabilities": {"browserVersion": "152.0"}, "sessionId": "abc"}}`), "example.com:4444")
	assert.NoError(t, err)
	assert.Equal(t, "abc", sessionId)
	assert.True(t, strings.Contains(string(out), "se:cdp"))
}

func TestProcessBodyJsonwpErrorIsNotASession(t *testing.T) {
	// operadriver replies with HTTP 200 even on failure
	body := `{"sessionId":"4ad6392e","status":33,"value":{"message":"session not created"}}`
	_, sessionId, err := processBody([]byte(body), "localhost:4444")
	assert.NoError(t, err)
	assert.Equal(t, "", sessionId)
}

func TestProcessBodyJsonwpSuccess(t *testing.T) {
	body := `{"sessionId":"abc123","status":0,"value":{"browserName":"opera"}}`
	out, sessionId, err := processBody([]byte(body), "localhost:4444")
	assert.NoError(t, err)
	assert.Equal(t, "abc123", sessionId)
	// JSONWP replies are wrapped in the W3C envelope modern clients expect
	assert.Contains(t, string(out), `"sessionId":"abc123"`)
	assert.Contains(t, string(out), `"capabilities":{"browserName":"opera"}`)
}

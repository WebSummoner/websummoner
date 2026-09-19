package main

import (
	"fmt"
	"log"
	"os"
	"testing"

	assert "github.com/stretchr/testify/require"
	"github.com/websummoner/websummoner/config"
	"github.com/websummoner/websummoner/session"
)

const testLogConf = "config/container-logs.json"

func configfile(s string) string {
	tmp, err := os.CreateTemp("", "config")
	if err != nil {
		log.Fatal(err)
	}
	_, err = tmp.Write([]byte(s))
	if err != nil {
		log.Fatal(err)
	}
	err = tmp.Close()
	if err != nil {
		log.Fatal(err)
	}
	return tmp.Name()
}

func TestConfig(t *testing.T) {
	confFile := configfile(`{}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	err := conf.Load(confFile, testLogConf)
	assert.NoError(t, err)
}

func TestConfigError(t *testing.T) {
	confFile := configfile(`{}`)
	_ = os.Remove(confFile)
	conf := config.NewConfig()
	err := conf.Load(confFile, testLogConf)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), fmt.Sprintf("browsers config: read error: open %s: no such file or directory", confFile))
}

func TestLogConfigError(t *testing.T) {
	confFile := configfile(`{}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	err := conf.Load(confFile, "some-missing-file")
	assert.Error(t, err)
}

func TestConfigParseError(t *testing.T) {
	confFile := configfile(`{`)
	defer os.Remove(confFile)
	var conf config.Config
	err := conf.Load(confFile, testLogConf)
	assert.Error(t, err)
	assert.Equal(t, err.Error(), "browsers config: parse error: unexpected end of JSON input")
}

func TestConfigEmptyState(t *testing.T) {
	confFile := configfile(`{}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	_ = conf.Load(confFile, testLogConf)

	state := conf.State(session.NewMap(), 0, 0, 0, false)
	assert.Equal(t, state.Total, 0)
	assert.Equal(t, state.Queued, 0)
	assert.Equal(t, state.Pending, 0)
	assert.Equal(t, state.Used, 0)
}

func TestConfigNonEmptyState(t *testing.T) {
	confFile := configfile(`{}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	_ = conf.Load(confFile, testLogConf)

	sessions := session.NewMap()
	sessions.Put("0", &session.Session{Caps: session.Caps{Name: "firefox", Version: "49.0"}, Quota: "unknown"})
	state := conf.State(sessions, 1, 0, 0, false)
	assert.Equal(t, state.Total, 1)
	assert.Equal(t, state.Queued, 0)
	assert.Equal(t, state.Pending, 0)
	assert.Equal(t, state.Used, 1)
	assert.Equal(t, state.Browsers["firefox"]["49.0"]["unknown"].Count, 1)
}

func TestConfigEmptyVersions(t *testing.T) {
	confFile := configfile(`{"firefox":{}}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	_ = conf.Load(confFile, testLogConf)

	sessions := session.NewMap()
	sessions.Put("0", &session.Session{Caps: session.Caps{Name: "firefox", Version: "49.0"}, Quota: "unknown"})
	state := conf.State(sessions, 1, 0, 0, false)
	assert.Equal(t, state.Total, 1)
	assert.Equal(t, state.Queued, 0)
	assert.Equal(t, state.Pending, 0)
	assert.Equal(t, state.Used, 1)
	assert.Equal(t, state.Browsers["firefox"]["49.0"]["unknown"].Count, 1)
}

func TestConfigNonEmptyVersions(t *testing.T) {
	confFile := configfile(`{"firefox":{"default":"49.0","versions":{"49.0":{}}}}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	_ = conf.Load(confFile, testLogConf)

	sessions := session.NewMap()
	sessions.Put("0", &session.Session{Caps: session.Caps{Name: "firefox", Version: "49.0"}, Quota: "unknown"})
	state := conf.State(sessions, 1, 0, 0, false)
	assert.Equal(t, state.Total, 1)
	assert.Equal(t, state.Queued, 0)
	assert.Equal(t, state.Pending, 0)
	assert.Equal(t, state.Used, 1)
	assert.Equal(t, state.Browsers["firefox"]["49.0"]["unknown"].Count, 1)
}

func TestConfigFindMissingBrowser(t *testing.T) {
	confFile := configfile(`{}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	_ = conf.Load(confFile, testLogConf)

	_, _, ok := conf.Find("firefox", "")
	assert.False(t, ok)
}

func TestConfigFindDefaultVersionError(t *testing.T) {
	confFile := configfile(`{"firefox":{"default":""}}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	_ = conf.Load(confFile, testLogConf)

	_, _, ok := conf.Find("firefox", "")
	assert.False(t, ok)
}

func TestConfigFindDefaultVersion(t *testing.T) {
	confFile := configfile(`{"firefox":{"default":"49.0"}}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	err := conf.Load(confFile, testLogConf)
	assert.NoError(t, err)

	_, v, ok := conf.Find("firefox", "")
	assert.False(t, ok)
	assert.Equal(t, v, "49.0")
}

func TestConfigFindFoundByEmptyPrefix(t *testing.T) {
	confFile := configfile(`{"firefox":{"default":"49.0","versions":{"49.0":{}}}}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	err := conf.Load(confFile, testLogConf)
	assert.NoError(t, err)

	_, v, ok := conf.Find("firefox", "")
	assert.True(t, ok)
	assert.Equal(t, v, "49.0")
}

func TestConfigFindFoundByPrefix(t *testing.T) {
	confFile := configfile(`{"firefox":{"default":"49.0","versions":{"49.0":{}}}}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	err := conf.Load(confFile, testLogConf)
	assert.NoError(t, err)

	_, v, ok := conf.Find("firefox", "49")
	assert.True(t, ok)
	assert.Equal(t, v, "49.0")
}

func TestConfigFindFoundByMatch(t *testing.T) {
	confFile := configfile(`{"firefox":{"default":"49.0","versions":{"49.0":{}}}}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	err := conf.Load(confFile, testLogConf)
	assert.NoError(t, err)

	_, v, ok := conf.Find("firefox", "49.0")
	assert.True(t, ok)
	assert.Equal(t, v, "49.0")
}

func TestConfigFindPrefersLongestPrefix(t *testing.T) {
	confFile := configfile(`{"firefox":{"default":"154.0","versions":{"154.0":{},"154.0.1":{}}}}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	err := conf.Load(confFile, testLogConf)
	assert.NoError(t, err)

	_, v, ok := conf.Find("firefox", "154")
	assert.True(t, ok)
	assert.Equal(t, v, "154.0.1")
}

func TestConfigFindExactKeyWins(t *testing.T) {
	confFile := configfile(`{"firefox":{"default":"154.0","versions":{"154.0":{},"154.0.1":{}}}}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	err := conf.Load(confFile, testLogConf)
	assert.NoError(t, err)

	_, v, ok := conf.Find("firefox", "154.0")
	assert.True(t, ok)
	assert.Equal(t, v, "154.0")
}

func TestConfigFindImage(t *testing.T) {
	confFile := configfile(`{"firefox":{"default":"49.0","versions":{"49.0":{"image":"image","port":"5555", "path":"/"}}}}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	err := conf.Load(confFile, testLogConf)
	assert.NoError(t, err)

	b, v, ok := conf.Find("firefox", "49.0")
	assert.True(t, ok)
	assert.Equal(t, v, "49.0")
	assert.Equal(t, b.Image, "image")
	assert.Equal(t, b.Port, "5555")
	assert.Equal(t, b.Path, "/")
}

func TestConfigConcurrentLoad(t *testing.T) {
	confFile := configfile(`{"firefox":{"default":""}}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()

	done := make(chan struct{})
	go func() {
		_ = conf.Load(confFile, testLogConf)
		done <- struct{}{}
	}()
	_ = conf.Load(confFile, testLogConf)
	<-done
}

func TestConfigConcurrentLoadAndRead(t *testing.T) {
	confFile := configfile(`{"firefox":{"default":"49.0","versions":{"49.0":{"image":"image","port":"5555", "path":"/", "tmpfs": {"/tmp":"size=64k"}}}}}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	err := conf.Load(confFile, testLogConf)
	assert.NoError(t, err)
	done := make(chan string)
	go func() {
		browser, _, _ := conf.Find("firefox", "")
		done <- browser.Tmpfs["/tmp"]
	}()
	err = conf.Load(confFile, testLogConf)
	assert.NoError(t, err)
	<-done
}

func TestConfigConcurrentRead(t *testing.T) {
	confFile := configfile(`{"firefox":{"default":"49.0","versions":{"49.0":{"image":"image","port":"5555", "path":"/", "tmpfs": {"/tmp":"size=64k"}}}}}`)
	defer os.Remove(confFile)
	var conf config.Config
	err := conf.Load(confFile, testLogConf)
	assert.NoError(t, err)
	done := make(chan string)
	go func() {
		browser, _, _ := conf.Find("firefox", "")
		done <- browser.Tmpfs["/tmp"]
	}()
	go func() {
		browser, _, _ := conf.Find("firefox", "")
		done <- browser.Tmpfs["/tmp"]
	}()
	<-done
	<-done
}

func TestStateReportsShedding(t *testing.T) {
	confFile := configfile(`{}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	_ = conf.Load(confFile, testLogConf)

	assert.False(t, conf.State(session.NewMap(), 1, 0, 0, false).Shedding)
	assert.True(t, conf.State(session.NewMap(), 1, 0, 0, true).Shedding)
}

func TestStateReportsSessionProtocols(t *testing.T) {
	confFile := configfile(`{}`)
	defer os.Remove(confFile)
	conf := config.NewConfig()
	_ = conf.Load(confFile, testLogConf)

	sessions := session.NewMap()
	sessions.Put("0", &session.Session{
		Quota:    "unknown",
		Caps:     session.Caps{Name: "chrome", Version: "153.0", HAR: true},
		HostPort: session.HostPort{Bidi: "172.17.0.2:9222", Devtools: "127.0.0.1:7070", VNC: "127.0.0.1:5900"},
	})
	sessions.Put("1", &session.Session{
		Quota: "unknown",
		Caps:  session.Caps{Name: "chrome", Version: "153.0"},
	})

	state := conf.State(sessions, 2, 0, 0, false)
	byId := map[string]config.Session{}
	for _, s := range state.Browsers["chrome"]["153.0"]["unknown"].Sessions {
		byId[s.ID] = s
	}

	assert.True(t, byId["0"].Bidi)
	assert.True(t, byId["0"].Cdp)
	assert.True(t, byId["0"].HAR)
	assert.True(t, byId["0"].VNC)

	// A session that asked for none of them must not advertise any.
	assert.False(t, byId["1"].Bidi)
	assert.False(t, byId["1"].Cdp)
	assert.False(t, byId["1"].HAR)
}

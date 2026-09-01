package main

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/websummoner/websummoner/config"
	"github.com/websummoner/websummoner/service"
	"github.com/websummoner/websummoner/session"
)

func testPythonAvailable(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not available")
	}
}

// httpServerCommand returns a command speaking WebSummoner's driver
// convention (the --port=NNN argument appended by the service).
func httpServerCommand(t *testing.T) []interface{} {
	t.Helper()
	script := filepath.Join(t.TempDir(), "srv.py")
	body := "import sys, http.server as h\nh.test(h.SimpleHTTPRequestHandler, port=int(sys.argv[1].split('=')[1]))\n"
	if err := os.WriteFile(script, []byte(body), 0644); err != nil {
		t.Fatalf("write helper: %v", err)
	}
	return []interface{}{"python3", script}
}

func driverEnv() service.Environment {
	return service.Environment{
		StartupTimeout: 10 * time.Second,
	}
}

func TestDriverStartsAndStopsProcess(t *testing.T) {
	testPythonAvailable(t)
	d := &service.Driver{
		ServiceBase: service.ServiceBase{
			Service: &config.Browser{
				Image: httpServerCommand(t),
			},
		},
		Environment: driverEnv(),
		Caps:        session.Caps{},
	}
	s, err := d.StartWithCancel()
	if err != nil {
		t.Fatalf("failed to start driver: %v", err)
	}
	resp, err := http.Head(s.Url.String())
	if err != nil {
		t.Fatalf("driver service not reachable: %v", err)
	}
	_ = resp.Body.Close()
	if s.Origin == "" {
		t.Fatal("origin must be set")
	}
	s.Cancel()
	// After cancel the port must no longer answer.
	time.Sleep(300 * time.Millisecond)
	if _, err := http.Head(s.Url.String()); err == nil {
		t.Fatal("driver process must be stopped after cancel")
	}
}

func TestDriverVNCHostPort(t *testing.T) {
	testPythonAvailable(t)
	d := &service.Driver{
		ServiceBase: service.ServiceBase{
			Service: &config.Browser{Image: httpServerCommand(t)},
		},
		Environment: driverEnv(),
		Caps:        session.Caps{VNC: true},
	}
	s, err := d.StartWithCancel()
	if err != nil {
		t.Fatalf("failed to start driver: %v", err)
	}
	defer s.Cancel()
	if s.HostPort.VNC == "" {
		t.Fatal("VNC host:port must be advertised for VNC sessions")
	}
}

func TestDriverInvalidImageConfigurations(t *testing.T) {
	cases := []struct {
		name  string
		image interface{}
	}{
		{"not an array", "python3"},
		{"non-string element", []interface{}{"python3", 42}},
		{"empty array", []interface{}{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := &service.Driver{
				ServiceBase: service.ServiceBase{Service: &config.Browser{Image: tc.image}},
				Environment: driverEnv(),
			}
			if _, err := d.StartWithCancel(); err == nil {
				t.Fatal("expected configuration error")
			}
		})
	}
}

func TestDriverFailingBinary(t *testing.T) {
	d := &service.Driver{
		ServiceBase: service.ServiceBase{
			Service: &config.Browser{Image: []interface{}{"/nonexistent-binary-xyz"}},
		},
		Environment: driverEnv(),
	}
	if _, err := d.StartWithCancel(); err == nil {
		t.Fatal("expected start failure for missing binary")
	}
}

func TestDriverStartupTimeoutKillsProcess(t *testing.T) {
	testPythonAvailable(t)
	// sleep never opens a port, so wait() must time out and stop the process.
	d := &service.Driver{
		ServiceBase: service.ServiceBase{
			Service: &config.Browser{Image: []interface{}{"python3", "-c", "import time; time.sleep(60)"}},
		},
		Environment: service.Environment{StartupTimeout: 1 * time.Second},
	}
	if _, err := d.StartWithCancel(); err == nil {
		t.Fatal("expected startup timeout")
	}
}

func TestDriverSavesLogFile(t *testing.T) {
	testPythonAvailable(t)
	dir := t.TempDir()
	d := &service.Driver{
		ServiceBase: service.ServiceBase{
			Service: &config.Browser{Image: httpServerCommand(t)},
		},
		Environment: service.Environment{
			StartupTimeout: 10 * time.Second,
			LogOutputDir:   dir,
			SaveAllLogs:    true,
		},
		Caps: session.Caps{LogName: "driver.log"},
	}
	s, err := d.StartWithCancel()
	if err != nil {
		t.Fatalf("failed to start driver: %v", err)
	}
	s.Cancel()
	if _, err := os.Stat(filepath.Join(dir, "driver.log")); err != nil {
		t.Fatalf("driver log file not created: %v", err)
	}
}

package main

import (
	"net"
	"syscall"
	"testing"
	"time"
)

func TestRunShutsDownGracefullyOnSignal(t *testing.T) {
	// Pick a free port and remember the previous global listener address.
	previous := listen
	defer func() { listen = previous }()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate port: %v", err)
	}
	port := l.Addr().String()
	_ = l.Close()
	listen = port

	go func() {
		time.Sleep(300 * time.Millisecond)
		_ = syscall.Kill(syscall.Getpid(), syscall.SIGTERM)
	}()
	previousGraceful := gracefulPeriod
	gracefulPeriod = 5 * time.Second
	defer func() { gracefulPeriod = previousGraceful }()

	if err := run(); err != nil {
		t.Fatalf("run returned error: %v", err)
	}
}

func TestRunFailsWhenPortIsTaken(t *testing.T) {
	previous := listen
	defer func() { listen = previous }()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate port: %v", err)
	}
	defer l.Close()
	listen = l.Addr().String()

	if err := run(); err == nil {
		t.Fatal("run must fail when the listen port is already taken")
	}
}

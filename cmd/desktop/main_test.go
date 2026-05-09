// Package main — smoke tests for the desktop entry point.
//
// These tests verify that the Ebitengine wiring compiles and the default
// configuration values are correct. Build with -tags=requires_display and
// DISPLAY=:99 (or equivalent) in headless CI environments; without the tag
// these files are excluded so that `go test ./...` succeeds without triggering
// the GLFW init() panic that Ebitengine raises at package initialisation.

//go:build requires_display

package main

import (
	"testing"

	ebapp "github.com/opd-ai/bostonfear/client/ebiten/app"
	rootcmd "github.com/opd-ai/bostonfear/cmd"
)

// TestDefaultServerURL verifies that the -server flag defaults to the documented
// localhost WebSocket address.
func TestDefaultServerURL(t *testing.T) {
	const want = "ws://localhost:8080/ws"
	cmd := rootcmd.NewDesktopCommand()
	flag := cmd.Flags().Lookup("server")
	if flag == nil {
		t.Fatal("expected -server flag to be registered")
	}
	if flag.DefValue != want {
		t.Errorf("default server URL = %q, want %q", flag.DefValue, want)
	}
}

// TestNewGame_DoesNotPanic verifies that creating an Ebitengine Game object
// with a (non-reachable) server URL does not panic.
// The NetClient's dial goroutine will fail to connect — that is expected and safe.
func TestNewGame_DoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewGame panicked: %v", r)
		}
	}()
	g := ebapp.NewGame("ws://127.0.0.1:0/ws") // port 0 — connection will be refused
	if g == nil {
		t.Error("NewGame returned nil")
	}
}

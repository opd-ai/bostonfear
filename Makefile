# Makefile — build, test, and lint targets for the bostonfear project.

.PHONY: all build test test-browser test-display vet clean

## all: build the server and all clients.
all: build

## build: compile all packages.
build:
	go build ./...

## test: run the standard test suite (no display required; CI-safe).
test:
	go test -race ./...

## test-browser: smoke-check browser client JavaScript syntax.
test-browser:
	node --check client/game.js

## test-display: run Ebitengine tests that require a local or virtual display.
## Set DISPLAY before calling if no physical display is available:
##   Xvfb :99 -screen 0 1280x720x24 & DISPLAY=:99 make test-display
test-display:
	go test -race -tags=requires_display ./client/ebiten/app/... ./client/ebiten/render/...

## vet: run go vet across all packages.
vet:
	go vet ./...

## clean: remove build artifacts.
clean:
	go clean ./...

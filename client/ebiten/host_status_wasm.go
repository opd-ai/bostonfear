//go:build js && wasm

package ebiten

import (
	"strings"
	"syscall/js"
)

func updateHostStatus(status string) {
	status = strings.TrimSpace(status)
	if status == "" {
		return
	}
	document := js.Global().Get("document")
	if !document.Truthy() {
		return
	}
	element := document.Call("getElementById", "status")
	if !element.Truthy() {
		return
	}
	element.Set("textContent", status)
}

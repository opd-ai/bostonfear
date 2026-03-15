package app

import "testing"

func TestLayout(t *testing.T) {
    g := &Game{}
    w, h := g.Layout(0, 0)
    if w != 800 || h != 600 {
        t.Errorf("expected 800x600, got %dx%d", w, h)
    }
}

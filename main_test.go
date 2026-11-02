package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// The view runs without a window in tests, which click as a user would.
func TestView(t *testing.T) {
	a := &pomodoro{workMin: 25, breakMin: 5}
	a.remaining = a.total()
	tt := ui.NewTester(a.view, 440, 620)
	if !tt.HasText("25:00") {
		t.Errorf("the clock should show 25:00, texts %q", tt.Texts())
	}
	if err := tt.Click("开始"); err != nil {
		t.Fatal(err)
	}
	if !a.running {
		t.Error("clicking 开始 should start the clock")
	}
	if err := tt.Click("暂停"); err != nil {
		t.Fatal(err)
	}
	if a.running {
		t.Error("clicking 暂停 should stop the clock")
	}
	if err := tt.Click("跳过"); err != nil {
		t.Fatal(err)
	}
	if a.mode != modeBreak || !tt.HasText("05:00") {
		t.Errorf("skipping focus should show the break, mode %d texts %q", a.mode, tt.Texts())
	}
}

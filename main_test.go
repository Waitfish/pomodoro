package main

import (
	"testing"
	"time"

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

// The clock runs on its own goroutine, so sessions turn over and count
// even when the window draws no frame at all.
func TestBackgroundClock(t *testing.T) {
	a := &pomodoro{workMin: 25, breakMin: 5}
	a.remaining = a.total()
	a.start()
	if a.tickerStop == nil {
		t.Error("start should run the background clock")
	}
	// A session that ended turns over, counts, and keeps running.
	a.endsAt = time.Now().Add(-time.Second)
	a.tick(time.Now())
	if a.done != 1 || a.mode != modeBreak || !a.running {
		t.Errorf("an ended focus session should count and roll into the break: done %d mode %d running %v", a.done, a.mode, a.running)
	}
	a.pause()
	if a.tickerStop != nil {
		t.Error("pause should stop the background clock")
	}
}

// The count is per day: crossing local midnight starts it over, and
// within a day it keeps accumulating.
func TestDailyRollover(t *testing.T) {
	a := &pomodoro{workMin: 25, breakMin: 5, done: 7, date: "2000-01-01"}
	a.tick(time.Now())
	if a.done != 0 {
		t.Errorf("crossing midnight should reset the count: got %d", a.done)
	}
	if a.date != time.Now().Format("2006-01-02") {
		t.Errorf("the count should be stamped with today: got %q", a.date)
	}
	a.endsAt = time.Now().Add(-time.Second)
	a.running = true
	a.mode = modeWork
	a.tick(time.Now())
	if a.done != 1 {
		t.Errorf("within a day the count should grow: got %d", a.done)
	}
}

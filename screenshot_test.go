package main

import (
	"image/png"
	"os"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// TestScreenshots renders the view to docs/*.png without a window, at
// 2x scale, for the README. Run it with:
//
//	SHOTS=1 go test -run TestScreenshots
func TestScreenshots(t *testing.T) {
	if os.Getenv("SHOTS") == "" {
		t.Skip("set SHOTS=1 to render the README screenshots")
	}
	if err := os.MkdirAll("docs", 0o755); err != nil {
		t.Fatal(err)
	}

	// Light: idle focus session, a full ring at 25:00.
	a := &pomodoro{workMin: 25, breakMin: 5, set: settings{Sound: 1, Volume: 0.8, Notify: true}}
	a.remaining = a.total()
	shot(t, a, "docs/screenshot.png", false)

	// Dark: mid-break with two sessions done and the settings open.
	b := &pomodoro{
		workMin: 25, breakMin: 5,
		mode: modeBreak, remaining: 4*time.Minute + 12*time.Second,
		done: 2, showSettings: true,
		set: settings{Sound: 2, Volume: 0.6, Notify: true},
	}
	shot(t, b, "docs/screenshot-dark.png", true)
}

func shot(t *testing.T, a *pomodoro, path string, dark bool) {
	tt := ui.NewTester(a.view, 440, 700)
	tt.SetScale(2)
	tt.SetDark(dark)
	tt.Frame()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, tt.Image()); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s", path)
}

// 番茄钟 (Pomodoro) — a demo app in MyGo native UI.
//
// The window is drawn by MyGo itself on the GPU: no HTML, no webview.
// The state struct holds the clock; the view builds the interface from
// it every frame, and c.After keeps the clock ticking once a second.
//
// The app sounds an alarm of its own synthesis (package audio), sends
// desktop notifications, lives in the system tray, and keeps its
// settings in the app's data directory.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"os"
	"path/filepath"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/audio"
	"github.com/egoist/mygo/ui"
)

// The two session modes.
const (
	modeWork = iota
	modeBreak
)

// tomato is the color of a ripe tomato, worn by focus sessions.
var tomato = ui.RGB(224, 82, 66)

// settings are what the user tunes, kept in settings.json.
type settings struct {
	Sound  int     `json:"sound"`  // index into soundNames
	Volume float64 `json:"volume"` // 0..1
	Notify bool    `json:"notify"` // desktop notifications
}

// soundNames are the built-in alarms, in the order buildSounds makes them.
var soundNames = []string{"经典提示", "清脆铃声", "三连钟", "木鱼"}

// pomodoro is the state the window shows. Its view builds the interface
// from it whenever the window needs a frame: after input, and every
// second while the clock runs (asked for with Context.After).
type pomodoro struct {
	mode      int           // the current session: focus or break
	running   bool          // whether the clock is moving
	endsAt    time.Time     // when the session ends, while running
	remaining time.Duration // time left, while paused
	workMin   float64       // focus length, in minutes
	breakMin  float64       // break length, in minutes
	done      int           // focus sessions completed
	note      string        // one line of status under the controls

	set          settings       // the user's settings
	showSettings bool           // the settings panel, open or closed
	settingsPath string         // where settings.json lives
	sounds       []*audio.Sound // the built-in alarms, made once

	win       *mygo.Window
	tray      *mygo.Tray
	lastTitle string
}

// total is the length of the current session.
func (a *pomodoro) total() time.Duration {
	min := a.breakMin
	if a.mode == modeWork {
		min = a.workMin
	}
	return time.Duration(min*60) * time.Second
}

// setMode moves to a session and stops its clock at the start.
func (a *pomodoro) setMode(m int) {
	a.mode = m
	a.running = false
	a.remaining = a.total()
}

func (a *pomodoro) start() {
	a.endsAt = time.Now().Add(a.remaining)
	a.running = true
}

func (a *pomodoro) pause() {
	if r := a.endsAt.Sub(time.Now()); r > 0 {
		a.remaining = r
	}
	a.running = false
}

func (a *pomodoro) reset() {
	a.running = false
	a.remaining = a.total()
	a.note = ""
}

// toggle starts or stops the clock, for the tray menu.
func (a *pomodoro) toggle() {
	if a.running {
		a.pause()
	} else {
		a.start()
	}
}

// skip ends the session that is showing without counting it.
func (a *pomodoro) skip() {
	if a.mode == modeWork {
		a.note = "已跳过专注，进入休息"
		a.setMode(modeBreak)
	} else {
		a.note = "已跳过休息，回到专注"
		a.setMode(modeWork)
	}
}

// tick moves the countdown and turns over the session when one ends.
// A session runs on the wall clock (endsAt), so frames that come late,
// or not at all while the window is hidden, lose no time.
func (a *pomodoro) tick(now time.Time) {
	if !a.running {
		return
	}
	a.remaining = a.endsAt.Sub(now)
	if a.remaining > 0 {
		return
	}
	if a.mode == modeWork {
		a.done++
		a.note = fmt.Sprintf("完成第 %d 个番茄，开始休息", a.done)
		a.mode = modeBreak
		a.announce("专注完成", fmt.Sprintf("第 %d 个番茄完成，休息 %d 分钟", a.done, int(a.breakMin)))
	} else {
		a.note = "休息结束，回到专注"
		a.mode = modeWork
		a.announce("休息结束", "回到专注，开始下一个番茄")
	}
	a.remaining = a.total()
	a.endsAt = now.Add(a.remaining)
}

// announce sounds the alarm and, when asked, sends a desktop
// notification. The notification is silent: the alarm is the sound.
func (a *pomodoro) announce(title, body string) {
	a.playAlarm()
	if a.set.Notify && mygo.NotificationsSupported() {
		_ = mygo.NewNotification(mygo.NotificationOptions{
			Title:  title,
			Body:   body,
			Group:  "pomodoro",
			Silent: true,
		}).Show()
	}
}

// playAlarm plays the chosen sound at the chosen volume.
func (a *pomodoro) playAlarm() {
	if a.set.Sound >= 0 && a.set.Sound < len(a.sounds) {
		a.sounds[a.set.Sound].NewPlayer().SetVolume(a.set.Volume).Play()
	}
}

// clock is the time left, as mm:ss.
func (a *pomodoro) clock() string {
	d := a.remaining.Round(time.Second)
	if d < 0 {
		d = 0
	}
	return fmt.Sprintf("%02d:%02d", int(d.Minutes()), int(d.Seconds())%60)
}

// modeName is what the current session is called.
func (a *pomodoro) modeName() string {
	if a.mode == modeWork {
		return "专注中"
	}
	return "休息中"
}

// title shows the clock in the window title and the tray while running.
func (a *pomodoro) title() {
	if a.win != nil && !a.win.IsDestroyed() {
		title := "番茄钟"
		if a.running {
			title = fmt.Sprintf("%s · %s — 番茄钟", a.clock(), a.modeName())
		}
		if title != a.lastTitle {
			a.lastTitle = title
			a.win.SetTitle(title)
		}
	}
	if a.tray != nil {
		if a.running {
			a.tray.SetTitle(a.clock())
		} else {
			a.tray.SetTitle("")
		}
	}
}

// loadSettings reads settings.json from the app's data directory, if it
// is there, and clamps what it finds.
func (a *pomodoro) loadSettings() {
	dir, err := mygo.App.Path(mygo.PathUserData)
	if err != nil {
		return
	}
	a.settingsPath = filepath.Join(dir, "settings.json")
	if b, err := os.ReadFile(a.settingsPath); err == nil {
		_ = json.Unmarshal(b, &a.set)
	}
	a.set.Sound = max(0, min(a.set.Sound, len(soundNames)-1))
	a.set.Volume = max(0, min(1, a.set.Volume))
}

// saveSettings writes them back.
func (a *pomodoro) saveSettings() {
	if a.settingsPath == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(a.settingsPath), 0o755)
	if b, err := json.MarshalIndent(a.set, "", "  "); err == nil {
		_ = os.WriteFile(a.settingsPath, b, 0o644)
	}
}

// showWindow opens the window again after it was closed to the tray.
func (a *pomodoro) showWindow() {
	if a.win == nil || a.win.IsDestroyed() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "番茄钟",
			Width:     440,
			Height:    700,
			MinWidth:  380,
			MinHeight: 560,
			StateKey:  "main",
			Content:   ui.View(a.view),
		})
	}
	a.win.Show()
}

// view builds the window from the state, on the main thread, one frame
// at a time.
func (a *pomodoro) view(c *ui.Context) {
	t := c.Theme()
	a.tick(c.Now())
	if a.running {
		c.After(time.Second) // ask for a frame at the clock's next beat
	}
	a.title()

	accent, modeName := tomato, "专注"
	if a.mode == modeBreak {
		accent, modeName = t.Success, "休息"
	}

	ui.Column(c).Fill().Gap(14).Padding(24).AlignItems(ui.Center).Children(func() {
		// Header: the name, and a tomato for each session done.
		ui.Row(c).Fill().Children(func() {
			ui.Text(c, "番茄钟").FontSize(22).Bold()
			ui.Spacer(c)
			if a.done > 0 {
				ui.Row(c).Gap(3).AlignItems(ui.Center).Children(func() {
					shown := min(a.done, 5)
					for range shown {
						ui.Box(c).Size(18, 18).Draw(drawTomato)
					}
					if a.done > 5 {
						ui.Textf(c, "+%d", a.done-5).FontSize(12).TextColor(t.TextMuted)
					}
				})
			}
		})

		// The mode, which the user can also switch by hand.
		sel := a.mode
		ui.Segmented(c, &sel, "专注", "休息")
		if sel != a.mode {
			a.setMode(sel)
			a.note = ""
		}

		// The ring: a track, the part of the session left drawn over
		// it, and the clock in the middle. Path.Trim keeps the part of
		// a path's length, so the arc unwinds as the session runs.
		// Draw paints under the children, so the clock can be real
		// text: selectable, and read by assistive technology.
		ui.Box(c).Size(240, 240).
			Label(fmt.Sprintf("%s %s", modeName, a.clock())).
			Draw(func(p *ui.Painter, r ui.Rect) {
				cx, cy := r.X+r.W/2, r.Y+r.H/2
				rad := r.W/2 - 13
				frac := float32(1)
				if total := a.total(); total > 0 {
					frac = float32(a.remaining) / float32(total)
					frac = max(0, min(1, frac))
				}
				p.StrokePath((&ui.Path{}).Circle(cx, cy, rad), 10, t.Border)
				p.StrokePath(circleFromTop(cx, cy, rad).Trim(0, frac), 10, accent)
			}).
			Children(func() {
				ui.Column(c).Fill().Center().Gap(4).Children(func() {
					ui.Text(c, a.clock()).FontSize(42).Bold().TextAlign(ui.Center)
					ui.Text(c, modeName).FontSize(13).TextColor(t.TextMuted).TextAlign(ui.Center)
				})
			})

		// The controls.
		ui.Row(c).Gap(10).Children(func() {
			if a.running {
				if ui.Button(c.Key("pause"), "暂停").Width(72).Clicked() {
					a.pause()
				}
			} else {
				if ui.PrimaryButton(c.Key("start"), "开始").Width(72).Clicked() {
					a.start()
				}
			}
			if ui.Button(c.Key("reset"), "重置").Width(72).Clicked() {
				a.reset()
			}
			if ui.Button(c.Key("skip"), "跳过").Width(72).Clicked() {
				a.skip()
			}
		})

		// The lengths, in minutes.
		ui.Column(c).Fill().Gap(8).Children(func() {
			a.sliderRow(c, "专注", &a.workMin, 5, 60, modeWork)
			a.sliderRow(c, "休息", &a.breakMin, 1, 30, modeBreak)
		})

		// The settings: alarm sound, volume, notifications.
		ui.Collapsible(c, "设置", &a.showSettings, func() {
			ui.Column(c).Fill().Gap(8).Padding(4, 2).Children(func() {
				ui.Row(c).Fill().Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "铃声").Width(40).FontSize(13).TextColor(t.TextMuted)
					name := soundNames[a.set.Sound]
					if ui.Select(c, &name, soundNames).Width(140).Changed() {
						a.set.Sound = soundIndex(name)
						a.saveSettings()
					}
					if ui.Button(c, "试听").Clicked() {
						a.playAlarm()
					}
				})
				ui.Row(c).Fill().Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "音量").Width(40).FontSize(13).TextColor(t.TextMuted)
					if ui.Slider(c, &a.set.Volume, 0, 1).Grow(1).Changed() {
						a.saveSettings()
					}
					ui.Textf(c, "%d%%", int(a.set.Volume*100)).Width(56).TextAlign(ui.End).FontSize(13)
				})
				ui.Row(c).Fill().Gap(10).AlignItems(ui.Center).Children(func() {
					ui.Text(c, "通知").Width(40).FontSize(13).TextColor(t.TextMuted)
					if ui.Switch(c, &a.set.Notify).Changed() {
						a.saveSettings()
					}
					ui.Text(c, "会话切换时发送桌面通知").FontSize(12).TextColor(t.TextMuted).Grow(1)
				})
			})
		})

		// One line of status, so finishing a session is visible.
		if a.note != "" {
			ui.Text(c, a.note).FontSize(12).TextColor(t.TextMuted)
		} else {
			ui.Text(c, "专注结束会自动进入休息；关闭窗口后程序留在托盘继续计时").
				FontSize(12).TextColor(t.TextMuted)
		}
	})
}

// sliderRow is one length: a name, a slider, and its value.
func (a *pomodoro) sliderRow(c *ui.Context, name string, v *float64, lo, hi float64, mode int) {
	t := c.Theme()
	ui.Row(c).Fill().Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Text(c, name).Width(40).FontSize(13).TextColor(t.TextMuted)
		sl := ui.Slider(c, v, lo, hi).Step(1).Grow(1)
		if sl.Changed() && a.mode == mode && !a.running {
			a.remaining = a.total() // a stopped clock shows its new length
		}
		ui.Textf(c, "%d 分钟", int(*v)).Width(56).TextAlign(ui.End).FontSize(13)
	})
}

// soundIndex is where a sound name sits, 0 for an unknown one.
func soundIndex(name string) int {
	for i, n := range soundNames {
		if n == name {
			return i
		}
	}
	return 0
}

// circleFromTop traces a circle clockwise from the top, so Trim(0, frac)
// reads as the part of a session left, starting where clocks start.
func circleFromTop(cx, cy, r float32) *ui.Path {
	const k = 0.5522847498 // one quarter arc as a cubic Bézier
	return (&ui.Path{}).
		MoveTo(cx, cy-r).
		CubeTo(cx+r*k, cy-r, cx+r, cy-r*k, cx+r, cy).
		CubeTo(cx+r, cy+r*k, cx+r*k, cy+r, cx, cy+r).
		CubeTo(cx-r*k, cy+r, cx-r, cy+r*k, cx-r, cy).
		CubeTo(cx-r, cy-r*k, cx-r*k, cy-r, cx, cy-r)
}

// drawTomato paints a small tomato: a red body, a stem and a leaf.
func drawTomato(p *ui.Painter, r ui.Rect) {
	cx, cy := r.X+r.W/2, r.Y+r.H/2+r.H*0.06
	rad := r.W * 0.36
	p.FillPath((&ui.Path{}).Circle(cx, cy, rad), tomato)
	stem := (&ui.Path{}).MoveTo(cx, cy-rad).LineTo(cx, cy-rad-r.H*0.14)
	p.StrokePath(stem, 1.5, ui.RGB(76, 145, 70))
	leaf := (&ui.Path{}).MoveTo(cx, cy-rad).
		QuadTo(cx-rad*0.7, cy-rad*1.15, cx-rad*0.95, cy-rad*0.55)
	p.StrokePath(leaf, 1.5, ui.RGB(76, 145, 70))
}

// The alarms are synthesized once at startup: no sound files, just
// math. Each is a few decaying sine partials, with a soft attack so
// nothing clicks.

const sampleRate = 48000

// synth makes a Sound from a function of time in seconds.
func synth(dur time.Duration, fn func(t float64) float64) *audio.Sound {
	n := int(dur.Seconds() * sampleRate)
	s := make([]float32, n)
	for i := range s {
		v := fn(float64(i) / sampleRate)
		s[i] = float32(max(-1, min(1, v)))
	}
	return audio.NewSound(sampleRate, 1, s)
}

// tone is one decaying partial, silent before it starts.
func tone(t, at, freq, decay float64) float64 {
	x := t - at
	if x < 0 {
		return 0
	}
	env := math.Exp(-x * decay)
	if x < 0.004 { // a 4 ms attack, so the start does not click
		env *= x / 0.004
	}
	return math.Sin(2*math.Pi*freq*x) * env
}

// buildSounds makes the alarms, in the order of soundNames.
func buildSounds() []*audio.Sound {
	return []*audio.Sound{
		// 经典提示: two short blips, up a third.
		synth(500*time.Millisecond, func(t float64) float64 {
			return 0.55*tone(t, 0, 660, 16) + 0.55*tone(t, 0.16, 880, 16)
		}),
		// 清脆铃声: one strike, with a bell's inharmonic partials.
		synth(2*time.Second, func(t float64) float64 {
			return 0.5 * (tone(t, 0, 880, 2) +
				0.35*tone(t, 0, 880*2.76, 4) +
				0.15*tone(t, 0, 880*5.4, 7))
		}),
		// 三连钟: C E G, one after another.
		synth(2400*time.Millisecond, func(t float64) float64 {
			return 0.45 * (tone(t, 0, 523.25, 2) +
				tone(t, 0.22, 659.25, 2) +
				tone(t, 0.44, 783.99, 1.6))
		}),
		// 木鱼: a short wooden pluck.
		synth(600*time.Millisecond, func(t float64) float64 {
			return 0.7*tone(t, 0, 392, 13) + 0.3*tone(t, 0, 1046, 22)
		}),
	}
}

// trayIconPNG draws the tray icon: a tomato silhouette, black on
// transparent, which macOS tints to match the menu bar.
func trayIconPNG() ([]byte, error) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	black := color.NRGBA{0, 0, 0, 255}
	for y := range 32 {
		for x := range 32 {
			dx, dy := float64(x-16), float64(y-19)
			inBody := dx*dx+dy*dy <= 10*10
			inStem := x >= 15 && x <= 17 && y >= 6 && y <= 10
			inLeaf := y >= 8 && y <= 10 && ((x >= 9 && x <= 13) || (x >= 19 && x <= 23))
			if inBody || inStem || inLeaf {
				img.Set(x, y, black)
			}
		}
	}
	var b bytes.Buffer
	err := png.Encode(&b, img)
	return b.Bytes(), err
}

// makeTray puts the tomato in the menu bar. Its menu shows the window,
// starts and stops the clock, and quits the app.
func (a *pomodoro) makeTray() {
	icon, err := trayIconPNG()
	if err != nil {
		log.Println("tray icon:", err)
		return
	}
	tray, err := mygo.NewTray(mygo.TrayOptions{
		ID:             "main",
		Icon:           icon,
		IconIsTemplate: true,
		ToolTip:        "番茄钟",
		Menu: mygo.NewMenu([]*mygo.MenuItem{
			{Label: "显示番茄钟", Click: func(*mygo.MenuItem, *mygo.Window) { a.showWindow() }},
			{Label: "开始 / 暂停", Click: func(*mygo.MenuItem, *mygo.Window) { a.toggle() }},
			{Label: "重置", Click: func(*mygo.MenuItem, *mygo.Window) { a.reset() }},
			mygo.Separator(),
			{Role: mygo.RoleQuit},
		}),
	})
	if err != nil {
		log.Println("tray:", err)
		return
	}
	a.tray = tray
	// Closing the window keeps the app in the tray; the menu quits it.
	mygo.App.OnWindowAllClosed(func() {})
}

func main() {
	a := &pomodoro{
		workMin: 25,
		breakMin: 5,
		set:     settings{Sound: 1, Volume: 0.8, Notify: true},
	}
	a.remaining = a.total()
	mygo.App.WhenReady(func() {
		a.loadSettings()
		a.sounds = buildSounds()
		a.makeTray()
		a.showWindow()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

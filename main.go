/*
   Copyright 2025 Google LLC

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       https://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"context"
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"go.sbk.wtf/ambient-glance/apps"
	"go.sbk.wtf/ambient-glance/display"
	"go.sbk.wtf/ambient-glance/display/drivers/ld220"
	"go.sbk.wtf/ambient-glance/display/drivers/tui"
	"go.sbk.wtf/ambient-glance/scheduler"
)

func main() {
	config, err := LoadConfig("config.json")
	if err != nil {
		log.Fatal(err)
	}

	app := tview.NewApplication()

	logView := tview.NewTextView()
	logView.SetTextAlign(tview.AlignLeft)
	logView.SetBorder(false)
	logView.SetWordWrap(true)
	logView.ScrollToEnd()
	logView.SetChangedFunc(func() {
		app.Draw()
	})

	l := log.New(logView, "", log.Ltime)

	tuiView := tview.NewTextView()
	tuiView.SetText("12345678901234567890\n12345678901234567890")
	tuiView.SetTextAlign(tview.AlignLeft)
	tuiView.SetTitle(" Display ")
	tuiView.SetBorder(true)
	tuiView.SetBackgroundColor(tcell.ColorBlack)
	tuiView.SetTextColor(tcell.ColorBlue)
	tuiView.SetChangedFunc(func() {
		app.Draw()
	})

	hpld220, err := ld220.Open("/dev/ttyUSB0")
	if err != nil {
		l.Println("Error opening serial hpld220:", err)
	} else {
		l.Println("Opened serial hpld220:", hpld220)
		defer hpld220.Close()
	}

	tui220 := tui.NewTUI220(tuiView, l)
	d := tui220
	if hpld220 != nil {
		l.Println("chain")
		d = display.NewChain(hpld220, tui220)
	}

	header := tview.NewTextView()
	header.SetText("Ambient Glance - HP LD 220 Debugger")
	header.SetTextAlign(tview.AlignCenter)
	header.SetBorder(false)

	cur := tview.NewTextView()
	cur.SetText("TBD")
	cur.SetBorder(false)
	cur.SetTextAlign(tview.AlignLeft)
	cur.SetBorderPadding(0, 0, 2, 2)

	schedulerDisp := display.NewLockable(d, l, "SCHEDULER")
	playDisp := display.NewLockable(d, l, "PLAY")
	derekDisp := display.NewLockable(d, l, "DEREK")
	fortuneDisp := display.NewLockable(d, l, "FORTUNE")
	boxDisp := display.NewLockable(d, l, "BOX")

	active := 0
	displays := []struct {
		Name    string
		Display display.LockableDisplay
	}{
		{"scheduler", schedulerDisp},
		{"play", playDisp},
		{"derek", derekDisp},
		{"fortune", fortuneDisp},
		{"box", boxDisp},
	}
	cur.SetText("Active: " + displays[active].Name)
	displays[active].Display.Enable()

	clockApp := apps.NewClock()
	derekApp := apps.NewDerek()
	playApp := apps.NewPlay()
	fortuneApp := apps.NewFortune()

	sch, _, intents := scheduler.NewScheduler(schedulerDisp, l, clockApp, fortuneApp)
	var (
		runningScheduler = false
		schedCtx         context.Context
		schedCancel      context.CancelFunc
	)

	sharkApp := apps.NewBabyShark(intents)

	adsb := apps.NewADSB(config.ADSBTar1090Endpoint, config.ADSBLat, config.ADSBLon, config.ADSBRadius, l, intents)
	var (
		runningADSB = false
		adsbCtx     context.Context
		adsbCancel  context.CancelFunc
	)

	list := tview.NewList()
	list.ShowSecondaryText(false)
	list.AddItem("Scheduler", "", 's', func() {
		if runningScheduler {
			l.Println("Stop scheduler")
			schedCancel()
			runningScheduler = false
			return
		}
		l.Println("Run scheduler")
		runningScheduler = true
		schedCtx, schedCancel = context.WithCancel(context.Background())
		go func() {
			if err := sch.Run(schedCtx); err != nil {
				l.Println("scheduler err", err)
			}
		}()
	})
	list.AddItem("Manual cycle", "", 'c', func() {
		l.Println("Enable next, current display:", displays[active].Name)

		displays[active].Display.Disable()
		active = (active + 1) % len(displays)
		displays[active].Display.Enable()
		l.Println("Now active:", displays[active].Name)
		cur.SetText("Active: " + displays[active].Name)
	})

	list.AddItem("ADSB", "", 'a', func() {
		if runningADSB {
			l.Println("Stop adsb")
			adsbCancel()
			runningADSB = false
			return
		}
		l.Println("Run adsb")
		runningADSB = true
		adsbCtx, adsbCancel = context.WithCancel(context.Background())
		go func() {
			if err := adsb.Run(adsbCtx); err != nil {
				l.Println("adsb err", err)
			}
		}()
	})

	list.AddItem("Play", "", 'p', func() {
		go func() {
			activity, err := playApp.Activate("manual")
			if err != nil {
				l.Println("play err", err)
			}
			if err := activity.Run(context.Background(), playDisp); err != nil {
				l.Println("play err", err)
			}
		}()
	})
	list.AddItem("Derek", "", 'd', func() {
		go func() {
			activity, err := derekApp.Activate("manual")
			if err != nil {
				l.Println("derek err", err)
				return
			}
			if err := activity.Run(context.Background(), derekDisp); err != nil {
				l.Println("derek err", err)
			}
		}()
	})
	list.AddItem("Fortune", "", 'f', func() {
		go func() {
			activity, err := fortuneApp.Activate("manual")
			if err != nil {
				l.Println("fortune err", err)
				return
			}
			if err := activity.Run(context.Background(), fortuneDisp); err != nil {
				l.Println("fortune err", err)
			}
		}()
	})
	list.AddItem("Shark", "", 'h', func() {
		go func() {
			if err := sharkApp.SignalIntent(); err != nil {
				l.Println("shark err", err)
				return
			}
			l.Println("Shark activated")
		}()
	})
	list.AddItem("Box", "", 'b', func() {
		go func() {
			d := boxDisp
			d.Reset()
			d.Write([]byte{0xB0, 0xB1, 0xB2, 0xDB, 0xDC, 0xDD, 0xDE, 0xDF, 0xFE})
			d.MoveCursor(display.CursorBottomLeft)
			d.Write([]byte{0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB, 0xDB})
		}()
	})
	list.AddItem("Reset", "", 'r', func() {
		go func() {
			d.Reset()
		}()
	})
	list.AddItem("Quit", "", 'q', func() {
		app.Stop()
	})

	grid := tview.NewGrid().
		SetRows(2, 4, 0).
		SetColumns(22, 0).
		SetBorders(true).
		AddItem(header, 0, 0, 1, 2, 0, 0, false).
		AddItem(tuiView, 1, 0, 1, 1, 0, 0, false).
		AddItem(cur, 1, 1, 1, 1, 0, 0, false).
		AddItem(list, 2, 0, 1, 1, 0, 0, false).
		AddItem(logView, 2, 1, 1, 1, 0, 0, false)

	if err := app.SetRoot(grid, true).SetFocus(list).Run(); err != nil {
		panic(err)
	}
}

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

package apps

import (
	"context"
	"time"

	"go.sbk.wtf/ambient-glance/display"
	"go.sbk.wtf/ambient-glance/scheduler"
)

type eightclap struct {
	intents chan<- scheduler.Intent
}

func New8Clap() scheduler.App {
	return &eightclap{}
}
func New8ClapIntent(intents chan<- scheduler.Intent) IntentApp {
	return &eightclap{
		intents: intents,
	}
}

func (e *eightclap) Name() string {
	return "Eight Clap"
}

func (e *eightclap) Activate(_ string) (scheduler.Activity, error) {
	return &eightclapActivity{}, nil
}

func (e *eightclap) Stop(_ string) error {
	return nil
}

func (e *eightclap) SignalIntent() error {
	e.intents <- scheduler.Intent{
		Name:     e.Name(),
		Activity: &eightclapActivity{},
	}
	return nil
}

type eightclapActivity struct{}

func (a eightclapActivity) Run(_ context.Context, d display.Display) error {
	if err := d.Reset(); err != nil {
		return err
	}
	if _, err := d.Write([]byte("one ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("two ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("three ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("four ")); err != nil {
		return err
	}
	if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("five ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("six ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("seven ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("eight")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if err := d.Clear(); err != nil {
		return err
	}
	if err := d.MoveCursor(display.CursorTopLeft); err != nil {
		return err
	}
	clap := func() error {
		if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
			return err
		}
		if err := d.ClearLine(); err != nil {
			return err
		}
		if _, err := d.Write([]byte("clap ")); err != nil {
			return err
		}
		time.Sleep(250 * time.Millisecond)
		if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
			return err
		}
		if _, err := d.Write([]byte("     clap")); err != nil {
			return err
		}
		time.Sleep(250 * time.Millisecond)
		if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
			return err
		}
		if _, err := d.Write([]byte("          clap")); err != nil {
			return err
		}
		time.Sleep(250 * time.Millisecond)
		if err := d.ClearLine(); err != nil {
			return err
		}
		return nil
	}
	if _, err := d.Write([]byte("U")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if err := clap(); err != nil {
		return err
	}
	if err := d.MoveCursorCR(6, 1); err != nil {

	}
	if _, err := d.Write([]byte("C")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if err := clap(); err != nil {
		return err
	}
	if err := d.MoveCursorCR(11, 1); err != nil {

	}
	if _, err := d.Write([]byte("L")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if err := clap(); err != nil {
		return err
	}
	if err := d.MoveCursorCR(16, 1); err != nil {

	}
	if _, err := d.Write([]byte("A   !")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if err := clap(); err != nil {
		return err
	}

	if err := d.MoveCursor(display.CursorTopLeft); err != nil {
		return err
	}
	if err := d.Clear(); err != nil {
		return err
	}

	if _, err := d.Write([]byte("U    ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("C    ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("L    ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)

	if _, err := d.Write([]byte("A   !")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)

	if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
		return err
	}
	if _, err := d.Write([]byte("FIGHT ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("FIGHT ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("FIGHT!")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)

	return d.Reset()
}

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

type derek struct{}

func NewDerek() scheduler.App {
	return derek{}
}

func (d derek) Name() string {
	return "derek"
}

func (d derek) Activate(_ string) (scheduler.Activity, error) {
	return &derekActivity{}, nil
}

func (d derek) Stop(_ string) error {
	return nil
}

type derekActivity struct{}

func (a derekActivity) Run(_ context.Context, d display.Display) error {
	if err := d.Reset(); err != nil {
		return err
	}
	if _, err := d.Write([]byte("Hello")); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
		return err
	}
	if _, err := d.Write([]byte("DEREK")); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	if err := d.Reset(); err != nil {
		return err
	}
	if _, err := d.Write([]byte("D ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("e ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("r ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("e ")); err != nil {
		return err
	}
	time.Sleep(250 * time.Millisecond)
	if _, err := d.Write([]byte("k ")); err != nil {
		return err
	}
	time.Sleep(time.Second)
	if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
		return err
	}
	if _, err := d.Write([]byte("or should I say...")); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	if err := d.Reset(); err != nil {
		return err
	}
	if _, err := d.Write([]byte("D'Erik?!?!?!?!")); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return d.Reset()
}

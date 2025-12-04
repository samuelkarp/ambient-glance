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

type play struct{}

func NewPlay() scheduler.App {
	return &play{}
}

func (p play) Name() string {
	return "play"
}

func (p play) Activate(_ string) (scheduler.Activity, error) {
	return &playActivity{}, nil
}

func (p play) Stop(_ string) error {
	return nil
}

type playActivity struct{}

func (p playActivity) Run(ctx context.Context, d display.Display) error {
	if _, err := d.Write([]byte("Hello, World!")); err != nil {
		return err
	}
	time.Sleep(time.Second)
	if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
		return err
	}
	if _, err := d.Write([]byte("This is bottom")); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	if err := d.MoveCursor(display.CursorTopLeft); err != nil {
		return err
	}
	if err := d.ClearLine(); err != nil {
		return err
	}
	if _, err := d.Write([]byte("This is top")); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	if err := d.MoveCursorCR(19, 1); err != nil {
		return err
	}
	if _, err := d.Write([]byte("20")); err != nil {
		return err
	}
	if err := d.MoveCursorCR(19, 2); err != nil {
		return err
	}
	if _, err := d.Write([]byte("25")); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	if err := d.Reset(); err != nil {
		return err
	}
	if _, err := d.Write([]byte("01234567890123456789")); err != nil {
		return err
	}
	if _, err := d.Write([]byte("98765432109876453120")); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	if _, err := d.Write([]byte("abcdefghijklmnopqrst")); err != nil {
		return err
	}
	time.Sleep(4 * time.Second)
	if err := d.Reset(); err != nil {
		return err
	}
	if _, err := d.Write([]byte("One\nTwo\nThree")); err != nil {
		return err
	}
	return nil
}

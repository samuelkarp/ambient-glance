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
	"fmt"
	"time"

	"go.sbk.wtf/ambient-glance/display"
	"go.sbk.wtf/ambient-glance/scheduler"
)

type clock struct{}

func NewClock() scheduler.App {
	return &clock{}
}

func (c *clock) Name() string {
	return "clock"
}

func (c *clock) Activate(_ string) (scheduler.Activity, error) {
	return &clockActivity{}, nil
}

func (c *clock) Stop(_ string) error {
	return nil
}

type clockActivity struct{}

func (c *clockActivity) Run(ctx context.Context, d display.Display) error {
	for {
		start := time.Now()
		rounded := start.Truncate(time.Second)
		snooze := time.Until(rounded.Add(time.Second))
		wake := time.After(snooze)
		select {
		case <-ctx.Done():
			return nil
		case <-wake:
			if err := c.print(d); err != nil {
				return err
			}
		}
	}
}

func (c *clockActivity) print(d display.Display) error {
	now := time.Now()
	colon := ":"
	if now.Second()%2 != 0 {
		colon = " "
	}
	space := " "
	switch now.Hour() {
	case 0, 10, 11, 12, 22, 23:
		space = "0"
	}
	now.Hour()
	str := now.Format(fmt.Sprintf("    %s3%s04%s05 PM", space, colon, colon))
	if err := d.MoveCursor(display.CursorTopLeft); err != nil {
		return err
	}
	if _, err := d.Write([]byte(str)); err != nil {
		return err
	}
	return nil
}

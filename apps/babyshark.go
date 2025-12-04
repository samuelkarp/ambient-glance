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
	"errors"
	"sync/atomic"
	"time"

	"go.sbk.wtf/ambient-glance/display"
	"go.sbk.wtf/ambient-glance/scheduler"
)

type babyShark struct {
	wantToPlay atomic.Bool
	intents    chan<- scheduler.Intent
}

type IntentApp interface {
	scheduler.App
	SignalIntent() error
}

func NewBabyShark(intents chan<- scheduler.Intent) IntentApp {
	return &babyShark{intents: intents}
}

func (b *babyShark) Name() string {
	return "babyshark"
}

func (b *babyShark) Activate(id string) (scheduler.Activity, error) {
	want := b.wantToPlay.CompareAndSwap(true, false)
	if !want {
		return nil, errors.New("no intent scheduled")
	}
	return &babySharkActivity{}, nil
}

func (b *babyShark) Stop(id string) error {
	return nil
}
func (b *babyShark) SignalIntent() error {
	b.intents <- scheduler.Intent{
		Name:     b.Name(),
		Activity: &babySharkActivity{},
	}
	return nil
}

type babySharkActivity struct{}

func (b babySharkActivity) Run(ctx context.Context, d display.Display) error {
	dodododo := func() error {
		if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
			return err
		}
		if err := d.ClearLine(); err != nil {
		}
		if _, err := d.Write([]byte("do")); err != nil {
			return err
		}
		time.Sleep(250 * time.Millisecond)
		if _, err := d.Write([]byte(" do")); err != nil {
			return err
		}
		time.Sleep(250 * time.Millisecond)
		if err := d.ClearLine(); err != nil {
			return err
		}
		if _, err := d.Write([]byte("      do")); err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
		if _, err := d.Write([]byte(" do")); err != nil {
			return err
		}
		time.Sleep(250 * time.Millisecond)
		if err := d.ClearLine(); err != nil {
			return err
		}
		if _, err := d.Write([]byte("           do")); err != nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
		if _, err := d.Write([]byte(" do")); err != nil {
			return err
		}
		time.Sleep(250 * time.Millisecond)
		return nil
	}

	topline := func(one, two, three string, sleep time.Duration) error {
		if err := d.Reset(); err != nil {
			return err
		}
		if _, err := d.Write([]byte(one)); err != nil {
			return err
		}
		time.Sleep(sleep)
		if _, err := d.Write([]byte(two)); err != nil {
			return err
		}
		time.Sleep(sleep)
		if _, err := d.Write([]byte(three)); err != nil {
			return err
		}
		time.Sleep(sleep)
		return nil
	}

	verse := func(one, two, three string) error {
		topline(one, two, three, 500*time.Millisecond)
		if err := dodododo(); err != nil {
			return err
		}

		topline(one, two, three, 250*time.Millisecond)
		time.Sleep(250 * time.Millisecond)
		if err := dodododo(); err != nil {
			return err
		}

		topline(one, two, three, 250*time.Millisecond)
		time.Sleep(250 * time.Millisecond)
		if err := dodododo(); err != nil {
			return err
		}

		topline(one, two, three, 250*time.Millisecond)
		time.Sleep(time.Second)
		return nil
	}

	if err := verse("Ba", "by ", "SHARK!"); err != nil {
		return err
	}
	if err := verse("Mom", "my ", "SHARK!"); err != nil {
		return err
	}
	if err := verse("Dad", "dy ", "SHARK!"); err != nil {
		return err
	}
	if err := verse("Grand", "ma ", "SHARK!"); err != nil {
		return err
	}
	if err := verse("Grand", "pa ", "SHARK!"); err != nil {
		return err
	}
	if err := verse("Let's ", "go ", "hunt!"); err != nil {
		return err
	}
	if err := verse("Run ", "a", "way!"); err != nil {
		return err
	}
	if err := verse("Safe ", "at ", "last!"); err != nil {
		return err
	}
	if err := verse("It's ", "the ", "end!"); err != nil {
		return err
	}

	time.Sleep(5 * time.Second)
	return d.Reset()
}

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
	"os/exec"
	"strings"
	"time"

	"go.sbk.wtf/ambient-glance/display"
	"go.sbk.wtf/ambient-glance/scheduler"
)

const (
	fortuneCmd = "/usr/games/fortune"
)

type fortune struct{}

func NewFortune() scheduler.App {
	return &fortune{}
}

func (f fortune) Name() string {
	return "fortune"
}

func (f fortune) Activate(id string) (scheduler.Activity, error) {
	if !fortuneAvailable() {
		return nil, errors.New("fortune not available")
	}
	return &fortuneActivity{}, nil
}

func (f fortune) Stop(id string) error {
	return nil
}

type fortuneActivity struct{}

func (f *fortuneActivity) Run(ctx context.Context, d display.Display) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		out, err := execFortune()
		if err != nil {
			return err
		}
		if err := d.Reset(); err != nil {
			return err
		}
		lines := wordwrap(out)
		for i, line := range lines {
			if i%2 == 0 {
				if err := d.Clear(); err != nil {
					return err
				}
			} else {
				if err := d.ClearLine(); err != nil {
					return err
				}
			}
			if _, err := d.Write([]byte(line)); err != nil {
				return err
			}
			if i%2 == 0 {
				if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
					return err
				}
			} else {
				if err := d.MoveCursor(display.CursorTopLeft); err != nil {
					return err
				}
				time.Sleep(3 * time.Second)
			}
		}
		if len(lines)+1%2 == 0 {
			time.Sleep(3 * time.Second)
		}
		time.Sleep(2 * time.Second)
	}
}

func wordwrap(in string) []string {
	var out []string
	splits := strings.Split(in, "\n")
	for _, s := range splits {
		out = append(out, wrapOne(s)...)
	}
	return out
}

func wrapOne(in string) []string {
	if len(in) == 0 {
		return nil
	}
	words := strings.Fields(in)
	line := ""
	var out []string
	const maxLen = 20
	for _, word := range words {
		if len(line) == maxLen {
			out = append(out, line)
			line = ""
		}
		space := " "
		if len(line) == 0 {
			space = ""
		}
		if len(line)+len(space)+len(word) > maxLen {
			out = append(out, line)
			line = ""
			space = ""
		}
		if len(word) > maxLen {
			w := word
			for len(w) > maxLen {
				out = append(out, w[:maxLen])
				w = w[maxLen:]
			}
			word = w
		}
		line += space + word
	}
	out = append(out, line)
	return out
}

// Available checks that the external fortune command is present in the PATH
func fortuneAvailable() bool {
	_, err := exec.LookPath(fortuneCmd)
	if err != nil {
		return false
	}
	return true
}

// Fortune returns a fortune string.
func execFortune() (string, error) {
	cmd := exec.Command(fortuneCmd)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(output), nil
}

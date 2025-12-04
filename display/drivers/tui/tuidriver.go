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

package tui

import (
	"fmt"
	"log"

	"github.com/rivo/tview"
	"go.sbk.wtf/ambient-glance/display"
)

type tui220 struct {
	tv  *tview.TextView
	row int
	col int
	buf [rows][cols]byte
	log *log.Logger
}

const (
	rows = 2
	cols = 20
)

func NewTUI220(tv *tview.TextView, log *log.Logger) display.Display {
	t := &tui220{
		tv:  tv,
		log: log,
	}
	t.Clear()
	return t
}

func (t *tui220) update() {
	// t.log.Println("update", string(t.buf[0][:]), string(t.buf[1][:]))
	t.tv.SetText(string(t.buf[0][:]) + "\n" + string(t.buf[1][:]))
}

func (t *tui220) Write(p []byte) (n int, err error) {
	for _, b := range p {
		if b == '\n' {
			t.row = (t.row + 1) % rows
			continue
		}
		t.buf[t.row][t.col] = b
		t.col = (t.col + 1) % cols
		if t.col == 0 {
			t.row = (t.row + 1) % rows
		}
	}
	t.update()
	return len(p), nil
}

func (t *tui220) Close() error {
	return nil
}

func (t *tui220) Reset() error {
	err := t.Clear()
	if err != nil {
		return err
	}
	err = t.MoveCursor(display.CursorTopLeft)
	return err
}

func (t *tui220) Clear() error {
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			t.buf[r][c] = ' '
		}
	}
	t.update()
	return nil
}

func (t *tui220) MoveCursor(position display.CursorPosition) error {
	switch position {
	case display.CursorLeft:
		t.col = t.col - 1
		if t.col < 0 {
			t.col = 0
		}
	case display.CursorRight:
		t.col++
		if t.col >= cols {
			t.col = cols - 1
		}
	case display.CursorUp:
		t.row--
		if t.row < 0 {
			t.row = 0
		}
	case display.CursorDown:
		t.row++
		if t.row >= rows {
			t.row = rows - 1
		}
	case display.CursorTopLeft:
		t.row = 0
		t.col = 0
	case display.CursorBottomLeft:
		t.row = rows - 1
		t.col = 0
	}
	return nil
}

// MoveCursorCR moves the cursor to the specified position
// The interface functions on 1-indexed positions but internally this implementation uses 0-based indexes
func (t *tui220) MoveCursorCR(c byte, r byte) error {
	if c < 1 || c > cols || r < 1 || r > rows {
		return fmt.Errorf("invalid cursor position %d", c)
	}
	t.col = int(c) - 1
	t.row = int(r) - 1
	return nil
}

func (t *tui220) ClearLine() error {
	for c := 0; c < cols; c++ {
		t.buf[t.row][c] = ' '
	}
	t.MoveCursorCR(1, byte(t.row+1))
	t.update()
	return nil
}

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

package ld220

import (
	"errors"
	"fmt"
	"strings"

	"go.bug.st/serial"
	"go.sbk.wtf/ambient-glance/display"
)

type hpld220 struct {
	port serial.Port
}

func Open(port string) (display.Display, error) {
	if port == "" {
		ports, err := serial.GetPortsList()
		if err != nil {
			return nil, err
		}
		if len(ports) == 0 {
			return nil, fmt.Errorf("no serial ports found")
		}
		if len(ports) >= 2 {
			for _, port := range ports {
				fmt.Printf("Found port: %v\n", port)
			}
			fmt.Println("Must specify port")
			return nil, fmt.Errorf("multiple serial ports found, specify which to open: %s", strings.Join(ports, ", "))
		}
		port = ports[0]
	}

	mode := &serial.Mode{
		BaudRate: 9600,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	p, err := serial.Open(port, mode)
	if err != nil {
		return nil, fmt.Errorf("failed to open serial port: %w", err)
	}

	d := &hpld220{port: p}
	err = d.Reset()
	if err != nil {
		return nil, fmt.Errorf("failed to reset display: %w", err)
	}

	return d, nil
}

func (d *hpld220) Write(p []byte) (n int, err error) {
	return d.port.Write(translate(p))
}

func translate(p []byte) []byte {
	s := string(p)
	out := make([]byte, 0)
	for _, c := range s {
		b := []byte(string([]rune{c}))
		switch c {
		case 'ú':
			b = []byte{0xA3}
		}
		out = append(out, b...)
	}
	return out
}

func (d *hpld220) Close() error {
	return d.port.Close()
}

func (d *hpld220) Reset() error {
	_, err := d.Write([]byte{0x1B, 0x40})
	return err
}

func (d *hpld220) Clear() error {
	_, err := d.Write([]byte{0x0C})
	return err
}

func (d *hpld220) MoveCursor(p display.CursorPosition) error {
	var b []byte
	switch p {
	case display.CursorUnchanged:
		return nil
	case display.CursorTopLeft:
		b = []byte{0x0B}
	case display.CursorBottomLeft:
		b = []byte{0x1F, 0x42, 0x0D}
	case display.CursorRight:
		b = []byte{0x09}
	case display.CursorLeft:
		b = []byte{0x08}
	case display.CursorUp:
		b = []byte{0x1F, 0x0A}
	case display.CursorDown:
		b = []byte{0x0A}
	}
	_, err := d.Write(b)
	return err
}

func (d *hpld220) MoveCursorCR(col, row byte) error {
	if row == 0 || row > 2 {
		return errors.New("invalid row number")
	}
	if col == 0 || col > 20 {
		return errors.New("invalid column number")
	}
	_, err := d.Write([]byte{0x1F, 0x24, col, row})
	return err
}

func (d *hpld220) ClearLine() error {
	_, err := d.Write([]byte{0x18})
	return err
}

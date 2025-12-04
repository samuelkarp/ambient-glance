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

package display

import (
	"errors"
)

type chain struct {
	d []Display
}

func NewChain(d ...Display) Display {
	return &chain{d: d}
}

func (c chain) Write(p []byte) (n int, err error) {
	var errs []error
	for _, d := range c.d {
		_, err := d.Write(p)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return len(p), errors.Join(errs...)
}

func (c chain) Close() error {
	var errs []error
	for _, d := range c.d {
		err := d.Close()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (c chain) Reset() error {
	var errs []error
	for _, d := range c.d {
		err := d.Reset()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (c chain) Clear() error {
	var errs []error
	for _, d := range c.d {
		err := d.Clear()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (c chain) MoveCursor(position CursorPosition) error {
	var errs []error
	for _, d := range c.d {
		err := d.MoveCursor(position)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (c chain) MoveCursorCR(b byte, b2 byte) error {
	var errs []error
	for _, d := range c.d {
		err := d.MoveCursorCR(b, b2)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (c chain) ClearLine() error {
	var errs []error
	for _, d := range c.d {
		err := d.ClearLine()
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

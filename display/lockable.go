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
	"log"
	"sync"
)

type LockableDisplay interface {
	Display
	Enable()
	Disable()
}
type lockable struct {
	d       Display
	l       sync.RWMutex
	enabled bool
	logger  *log.Logger
	name    string
}

type lockedErr struct {
}

func (lockedErr) Error() string {
	return "operation not permitted"
}

var notPermitted error = &lockedErr{}

func NewLockable(d Display, log *log.Logger, name string) LockableDisplay {
	return &lockable{
		d:      d,
		logger: log,
		name:   name,
	}
}

func (l *lockable) Enable() {
	l.l.Lock()
	defer l.l.Unlock()
	l.enabled = true
	l.logger.Println("Enabled", l.name)
}

func (l *lockable) Disable() {
	l.l.Lock()
	defer l.l.Unlock()
	l.enabled = false
	l.logger.Println("Disabled", l.name)
}

func (l *lockable) Write(p []byte) (n int, err error) {
	l.l.RLock()
	defer l.l.RUnlock()
	if !l.enabled {
		return 0, notPermitted
	}
	return l.d.Write(p)
}

func (l *lockable) Close() error {
	l.l.RLock()
	defer l.l.RUnlock()
	if !l.enabled {
		return notPermitted
	}
	return l.d.Close()
}

func (l *lockable) Reset() error {
	l.l.RLock()
	defer l.l.RUnlock()
	if !l.enabled {
		return notPermitted
	}
	return l.d.Reset()
}

func (l *lockable) Clear() error {
	l.l.RLock()
	defer l.l.RUnlock()
	if !l.enabled {
		return notPermitted
	}
	return l.d.Clear()
}

func (l *lockable) MoveCursor(position CursorPosition) error {
	l.l.RLock()
	defer l.l.RUnlock()
	if !l.enabled {
		return notPermitted
	}
	return l.d.MoveCursor(position)
}

func (l *lockable) MoveCursorCR(b byte, b2 byte) error {
	l.l.RLock()
	defer l.l.RUnlock()
	if !l.enabled {
		return notPermitted
	}
	return l.d.MoveCursorCR(b, b2)
}

func (l *lockable) ClearLine() error {
	l.l.RLock()
	defer l.l.RUnlock()
	if !l.enabled {
		return notPermitted
	}
	return l.d.ClearLine()
}

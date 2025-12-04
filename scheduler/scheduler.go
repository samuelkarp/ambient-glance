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

package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.sbk.wtf/ambient-glance/display"
)

type Scheduler interface {
	Run(context.Context) error
}

type scheduler struct {
	display display.Display
	apps    []App
	status  chan Status
	intent  <-chan Intent
	log     *log.Logger
}

type Status struct {
	Name     string
	ID       string
	Deadline time.Time
}

type App interface {
	Name() string
	Activate(id string) (Activity, error)
	Stop(id string) error
}

type Activity interface {
	Run(ctx context.Context, d display.Display) error
}

type Intent struct {
	Name     string
	Activity Activity
}

func NewScheduler(display display.Display, log *log.Logger, apps ...App) (Scheduler, <-chan Status, chan<- Intent) {
	status := make(chan Status)
	intent := make(chan Intent)
	return &scheduler{
		display: display,
		apps:    apps,
		status:  status,
		log:     log,
		intent:  intent,
	}, status, intent
}

func (s scheduler) Run(ctx context.Context) error {
	s.log.Printf("Starting scheduler with %d apps", len(s.apps))
	i := 0
	var priority *Intent
	for {
		for _, app := range s.apps {
			select {
			case <-ctx.Done():
				s.log.Printf("Stopping scheduler with %d apps", len(s.apps))
				return nil
			default:
			}

			if priority != nil {
				i++
				ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
				id := fmt.Sprintf("%s-%d", priority.Name, i)
				s.runActivity(ctx, priority.Name, id, priority.Activity)
				cancel()
				priority = nil
			}
			select {
			case intent := <-s.intent:
				s.log.Printf("Got intent! %q", intent.Name)
				priority = &intent
				continue
			default:
			}

			i++
			id := fmt.Sprintf("%s-%d", app.Name(), i)
			s.log.Printf("Starting app %q", app.Name())
			activity, err := app.Activate(id)
			if err != nil {
				s.log.Printf("Error activating app %q: %v", app.Name(), err)
				continue
			}
			deadline := time.Now().Add(2 * time.Minute)
			ctx, cancel := context.WithDeadline(ctx, deadline)
			done := make(chan error)
			go func() {
				done <- s.runActivity(ctx, app.Name(), id, activity)
				close(done)
			}()
			s.log.Printf("Running app %q", app.Name())
			select {
			case intent := <-s.intent:
				s.log.Printf("Got intent! %q", intent.Name)
				cancel()
				priority = &intent
			case err := <-done:
				s.log.Printf("App %q done: %v", app.Name(), err)
				if err != nil {
					return err
				}
			}
			cancel()
		}
	}
	// return nil
}

func (s scheduler) runActivity(ctx context.Context, name string, id string, a Activity) error {
	s.log.Printf("Starting activity %q", id)
	d := display.NewLockable(s.display, s.log, id)
	d.Enable()
	defer d.Disable()
	if err := d.Reset(); err != nil {
		s.log.Printf("Error resetting display %q: %v", id, err)
		return err
	}
	select {
	case s.status <- Status{
		Name: name,
		ID:   id,
	}:
	default:
	}
	done := make(chan error)
	go func() {
		done <- a.Run(ctx, d)
	}()
	select {
	case err := <-done:
		if err != nil {
			s.log.Printf("Error running activity %q: %v", id, err)
		}
	case <-ctx.Done():
		s.log.Printf("Activity ended by deadline %s", id)
	}
	return nil
}

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
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	onebusaway "github.com/OneBusAway/go-sdk"
	"github.com/OneBusAway/go-sdk/option"
	"go.sbk.wtf/ambient-glance/display"
	"go.sbk.wtf/ambient-glance/scheduler"
)

type oba struct {
	log     *log.Logger
	loc     *time.Location
	intents chan<- scheduler.Intent
	key     string
	stops   []string
	alias   map[string]string
	cache   obaCache
}

type obaCache struct {
	l            sync.RWMutex
	nextArrivals []obaArrival
}

type obaArrival struct {
	shortName string
	time      time.Time
	headsign  string
	agency    string
}

func NewOBA(key string, stops []string, alias map[string]string, log *log.Logger) *oba {
	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		log.Fatal(err)
	}
	return &oba{
		log:   log,
		loc:   loc,
		key:   key,
		stops: stops,
		alias: alias,
	}
}

func (o *oba) WithIntents(intents chan<- scheduler.Intent) IntentApp {
	o.intents = intents
	return o
}

func (o *oba) Name() string {
	return "onebusaway"
}

func (o *oba) Stop(id string) error {
	return nil
}

func (o *oba) Run(ctx context.Context) error {
	client := onebusaway.NewClient(
		option.WithAPIKey(o.key),
	)
	first := true
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if first {
				first = false
				break
			}
			time.Sleep(30 * time.Second)
		}
		arrivals := make([]obaArrival, 0)
		for _, s := range o.stops {
			res, err := client.ArrivalAndDeparture.List(ctx, s, onebusaway.ArrivalAndDepartureListParams{
				MinutesAfter: onebusaway.Int(20),
			})
			if err != nil {
				o.log.Printf("error listing arrivals for %q: %v", s, err)
				continue
			}
			o.log.Printf("oba: got %d arrivals for stop %q\n", len(res.Data.Entry.ArrivalsAndDepartures), s)
			agencies := make(map[string]string)
			for _, a := range res.Data.References.Agencies {
				agencies[a.ID] = a.Name
			}
			routes := make(map[string]string)
			for _, r := range res.Data.References.Routes {
				routes[r.ID] = r.AgencyID
			}
			for _, a := range res.Data.Entry.ArrivalsAndDepartures {
				arr := time.UnixMilli(a.PredictedArrivalTime).In(o.loc)
				now := time.Now()
				if now.After(arr) || now.Add(20*time.Minute).Before(arr) {
					continue
				}
				name := a.RouteShortName
				if n, ok := o.alias[a.RouteID]; ok {
					name = n
				}
				arrivals = append(arrivals, obaArrival{
					shortName: name,
					headsign:  a.TripHeadsign,
					time:      arr,
					agency:    agencies[routes[a.RouteID]],
				})
				o.log.Printf("oba: added arrival for %q at %q arr %s\n", name, s, arr.Format(time.Kitchen))
			}
		}
		sort.Slice(arrivals, func(i, j int) bool {
			return arrivals[i].time.Before(arrivals[j].time)
		})
		o.cache.l.Lock()
		o.cache.nextArrivals = arrivals
		o.cache.l.Unlock()
	}
}

func (o *oba) SignalIntent() error {
	if o.intents == nil {
		return errors.New("no intents set")
	}
	o.intents <- scheduler.Intent{
		Name: o.Name(),
		Activity: &obaActivity{
			id:    "intent",
			log:   o.log,
			cache: &o.cache,
		},
	}
	return nil
}

func (o *oba) Activate(id string) (scheduler.Activity, error) {
	return &obaActivity{
		id:    id,
		log:   o.log,
		cache: &o.cache,
	}, nil
}

type obaActivity struct {
	id    string
	log   *log.Logger
	cache *obaCache
}

func (o *obaActivity) Run(ctx context.Context, d display.Display) error {
	if err := d.Reset(); err != nil {
		return err
	}
	for i := 0; i < 3; i++ {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		var arrivals []obaArrival
		o.cache.l.RLock()
		arrivals = o.cache.nextArrivals
		o.cache.l.RUnlock()
		if len(arrivals) == 0 {
			return nil
		}
		for j := 0; j < len(arrivals) && j < 10; j += 2 {
			for k := 0; k < 30; k++ {
				var one, two string
				one = formatArrival(arrivals[j], k)
				if j+1 < len(arrivals) {
					two = formatArrival(arrivals[j+1], k)
				} else {
					two = strings.Repeat(" ", 20)
				}
				if err := d.MoveCursor(display.CursorTopLeft); err != nil {
					return err
				}
				if _, err := d.Write([]byte(one)); err != nil {
					return err
				}
				if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
					return err
				}
				if _, err := d.Write([]byte(two)); err != nil {
					return err
				}
				if k == 0 {
					time.Sleep(2 * time.Second)
				} else {
					time.Sleep(200 * time.Millisecond)
				}
			}
			var one, two string
			one = formatArrival(arrivals[j], 0)
			if j+1 < len(arrivals) {
				two = formatArrival(arrivals[j+1], 0)
			} else {
				two = strings.Repeat(" ", 20)
			}
			if err := d.MoveCursor(display.CursorTopLeft); err != nil {
				return err
			}
			if _, err := d.Write([]byte(one)); err != nil {
				return err
			}
			if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
				return err
			}
			if _, err := d.Write([]byte(two)); err != nil {
				return err
			}
		}
	}
	return nil
}

func formatArrival(arrival obaArrival, off int) string {
	in := arrival.time.Sub(time.Now())
	inFmt := fmt.Sprintf("%dm", int(in.Round(time.Minute).Minutes()))
	if in.Minutes() < 0 {
		inFmt = "NOW"
	}
	shortName := fmtField(arrival.shortName, 4, 0, leftPad)
	headSign := fmtField(arrival.headsign, 11, off, rightPad)
	inFmt = fmtField(inFmt, 3, 0, rightPad)

	// 1234 67890123456 890
	return fmt.Sprintf("%s %s %s", shortName, headSign, inFmt)
}

type padDir int

const (
	leftPad padDir = iota
	rightPad
)

func fmtField(s string, size int, off int, pad padDir) string {
	switch {
	case len(s) == size:
		return s
	case len(s) > size:
		s = marquee(s, off)
		return s[:size]
	case pad == leftPad:
		return strings.Repeat(" ", size-len(s)) + s
	case pad == rightPad:
		return s + strings.Repeat(" ", size-len(s))
	}
	return ""
}

func marquee(s string, off int) string {
	s = s + "    "
	off = off % len(s)
	return s[off:] + s[:off]
}

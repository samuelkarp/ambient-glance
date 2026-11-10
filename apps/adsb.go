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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"go.sbk.wtf/ambient-glance/display"
	"go.sbk.wtf/ambient-glance/scheduler"
)

// https://github.com/wiedehopf/readsb/blob/dev/README-json.md
// http://<tar1090>:30152/?closest=<lat>,<lon>,3

type Closest struct {
	Aircraft []Aircraft `json:"aircraft"`
}

type Aircraft struct {
	Hex          string  `json:"hex"`
	RecordType   string  `json:"type"`
	Flight       string  `json:"flight"`
	Registration string  `json:"r"`
	Type         string  `json:"t"`
	Description  string  `json:"desc"`
	Owner        string  `json:"ownOp"`
	Year         string  `json:"year"`
	Lat          float64 `json:"lat"`
	Lon          float64 `json:"lon"`
	GroundSpeed  float64 `json:"gs"`
}

type adsb struct {
	tar1090Endpoint string
	lat             string
	lon             string
	dist            string
	log             *log.Logger
	pendingIntent   atomic.Bool
	intents         chan<- scheduler.Intent
	routes          *routeData
}

type BackgroundApp interface {
	scheduler.App
	Run(ctx context.Context) error
}

// NewADSB returns the ADS-B app. Route data files are kept in routeDataDir.
func NewADSB(tar1090Endpoint, lat, lon, dist, routeDataDir string, log *log.Logger, intents chan<- scheduler.Intent) BackgroundApp {
	return &adsb{
		tar1090Endpoint: tar1090Endpoint,
		lat:             lat,
		lon:             lon,
		dist:            dist,
		log:             log,
		intents:         intents,
		routes:          newRouteData(routeDataDir, log),
	}
}

func (a *adsb) Name() string {
	return "adsb"
}

func (a *adsb) Activate(id string) (scheduler.Activity, error) {
	return &adsbActivity{}, nil
}

func (a *adsb) Stop(id string) error {
	return nil
}

func (a *adsb) Run(ctx context.Context) error {
	go a.routes.Run(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			time.Sleep(1 * time.Second)
		}
		pendingIntent := a.pendingIntent.Load()
		if pendingIntent {
			time.Sleep(10 * time.Second)
		}
		closest, err := a.getClosest()
		if err != nil {
			a.log.Println("adsb: err getting closest:", err)
			time.Sleep(10 * time.Second)
			continue
		}
		if closest == nil || len(closest.Aircraft) == 0 {
			continue
		}
		a.log.Println("adsb closest:", closest)
		if err := a.sendIntentAndWait(); err != nil {
			a.log.Println("adsb: err sending intent:", err)
			time.Sleep(10 * time.Second)
		}
	}
}

func (a *adsb) sendIntentAndWait() error {
	a.pendingIntent.Store(true)
	defer a.pendingIntent.Store(false)
	done := make(chan struct{})
	a.intents <- scheduler.Intent{
		Name: "adsb",
		Activity: &adsbActivity{
			done:    done,
			closest: a.getClosest,
			route:   a.routes.Lookup,
			log:     a.log,
		},
	}
	<-done

	time.Sleep(time.Minute)
	return nil
}

func (a *adsb) closestURL() string {
	return fmt.Sprintf("%s/?closest=%s,%s,%s", a.tar1090Endpoint, a.lat, a.lon, a.dist)
}

func (a *adsb) getClosest() (*Closest, error) {
	r, err := http.Get(a.closestURL())
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = r.Body.Close()
	}()
	var closest Closest
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&closest); err != nil {
		return nil, err
	}
	return &closest, nil
}

type adsbActivity struct {
	done    chan struct{}
	closest func() (*Closest, error)
	route   func(callsign string) (*Route, error)
	log     *log.Logger
}

func (a *adsbActivity) Run(ctx context.Context, d display.Display) error {
	defer close(a.done)
	if err := d.Reset(); err != nil {
		return err
	}
	if _, err := d.Write([]byte("        ADS-B")); err != nil {
		return err
	}
	if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
		return err
	}
	if _, err := d.Write([]byte("     loading...")); err != nil {
	}
	var (
		lastCallsign string
		lastRoute    *Route
	)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		c, err := a.closest()
		if err != nil {
			return err
		}
		if len(c.Aircraft) == 0 {
			return nil
		}
		aircraft := &c.Aircraft[0]
		if aircraft.Flight != lastCallsign {
			lastCallsign = aircraft.Flight
			lastRoute, err = a.route(aircraft.Flight)
			if err != nil {
				a.log.Println("adsb: err getting route:", err)
			} else if lastRoute != nil {
				a.log.Printf("adsb: found route: %s", lastRoute.Codes())
			}

			if err := d.Reset(); err != nil {
				return err
			}

		}
		if err := a.display(aircraft, lastRoute, d); err != nil {
			return err
		}

		time.Sleep(10 * time.Second)
	}
}

func (a *adsbActivity) display(aircraft *Aircraft, route *Route, d display.Display) error {
	callsign := fmt.Sprintf("%-8s", aircraft.Flight)[:8]
	typ := fmt.Sprintf("%4s", aircraft.Type)[:4]
	line1 := fmt.Sprintf("%s %s %4.0fkt", callsign, typ, aircraft.GroundSpeed)
	var line2 string
	if route != nil && len(route.Airports) > 0 {
		if len(route.Airports) == 2 {
			locs := fmt.Sprintf("%s - %s", route.Airports[0].Location, route.Airports[1].Location)
			if len(locs) > 20 {
				if len(route.Airports[0].Location) > len(route.Airports[1].Location) {
					locs = fmt.Sprintf("%s - %s", route.Airports[0].IATA, route.Airports[1].Location)
				} else {
					locs = fmt.Sprintf("%s - %s", route.Airports[0].Location, route.Airports[1].IATA)
				}
			}
			if len(locs) > 20 {
				locs = route.Codes()
			}
			line2 = fmt.Sprintf("%-20s", locs)[:20]
		} else {
			line2 = fmt.Sprintf("%-20s", route.Codes())
		}
	} else {
		line2 = fmt.Sprintf("%-20s", aircraft.Owner)[:20]
	}
	if _, err := d.Write([]byte(line1)); err != nil {
		return err
	}
	if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
		return err
	}
	if _, err := d.Write([]byte(line2)); err != nil {
		return err
	}
	return nil
}

func (a *adsbActivity) displaySpeed(aircraft *Aircraft, d display.Display) error {
	speed := fmt.Sprintf("%4.0fkt", aircraft.GroundSpeed)
	if err := d.MoveCursorCR(15, 1); err != nil {
		return err
	}
	if _, err := d.Write([]byte(speed)); err != nil {
		return err
	}
	return nil
}

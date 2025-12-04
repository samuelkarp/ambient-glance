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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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

// https://api.adsb.lol/docs#/v0/api_routeset_api_0_routeset_post

type RoutesetRequest struct {
	Planes []Plane `json:"planes"`
}

type Plane struct {
	Callsign string  `json:"callsign"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

type RoutesetResponse struct {
	Callsign         string    `json:"callsign"`
	AirportCodesIATA string    `json:"_airport_codes_iata"`
	Airports         []Airport `json:"_airports"`
}

type Airport struct {
	IATA     string `json:"iata"`
	ICAO     string `json:"icao"`
	Location string `json:"location"`
}

type adsb struct {
	tar1090Endpoint string
	lat             string
	lon             string
	dist            string
	log             *log.Logger
	pendingIntent   atomic.Bool
	intents         chan<- scheduler.Intent
}

type BackgroundApp interface {
	scheduler.App
	Run(ctx context.Context) error
}

func NewADSB(tar1090Endpoint, lat, lon, dist string, log *log.Logger, intents chan<- scheduler.Intent) BackgroundApp {
	return &adsb{
		tar1090Endpoint: tar1090Endpoint,
		lat:             lat,
		lon:             lon,
		dist:            dist,
		log:             log,
		intents:         intents,
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
	a.log.Println("starting adsb", a.closestURL())
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
		lastRoute    *RoutesetResponse
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
			if route, err := a.getRoute(aircraft); err != nil {
				lastRoute = nil
				a.log.Println("adsb: err getting route:", err)
			} else {
				a.log.Printf("adsb: found route: %v", route)
				lastRoute = route
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

func (a *adsbActivity) display(aircraft *Aircraft, route *RoutesetResponse, d display.Display) error {
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
				locs = route.AirportCodesIATA
			}
			line2 = fmt.Sprintf("%-20s", locs)[:20]
		} else {
			line2 = fmt.Sprintf("%-20s", route.AirportCodesIATA)
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

func (a *adsbActivity) getRoute(aircraft *Aircraft) (*RoutesetResponse, error) {
	req := &RoutesetRequest{
		Planes: []Plane{{
			Callsign: strings.TrimSpace(aircraft.Flight),
			Lat:      aircraft.Lat,
			Lng:      aircraft.Lon,
		}},
	}
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	r, err := http.Post("https://api.adsb.lol/api/0/routeset", "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer func() { _ = r.Body.Close() }()
	var res []RoutesetResponse
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&res); err != nil {
		return nil, err
	}
	if len(res) != 1 {
		return nil, fmt.Errorf("adsb: got %d routesets", len(res))
	}
	return &res[0], nil
}

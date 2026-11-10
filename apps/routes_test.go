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
	"compress/gzip"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const (
	testRoutesCSV = "\xEF\xBB\xBFCallsign,Code,Number,AirlineCode,AirportCodes\n" +
		"ASA184,ASA,184,ASA,PANC-PACD-PADK\n" +
		"CPA844,CPA,844,CPA,VHHH-KJFK\n" +
		"XXX1,XXX,1,XXX,KT00-KJFK\n" +
		"XXX2,XXX,2,XXX,ZZZZ-KJFK\n"
	testAirportsCSV = "\xEF\xBB\xBFCode,Name,ICAO,IATA,Location,CountryISO2,Latitude,Longitude,AltitudeFeet\n" +
		"PANC,Ted Stevens Anchorage International Airport,PANC,ANC,Anchorage,US,61.1744,-149.9960,152\n" +
		"PACD,Cold Bay Airport,PACD,CDB,Cold Bay,US,55.2061,-162.7250,96\n" +
		"PADK,Adak Airport,PADK,ADK,Adak Island,US,51.8780,-176.6460,18\n" +
		"VHHH,Hong Kong International Airport,VHHH,HKG,Hong Kong,HK,22.3089,113.9150,28\n" +
		"KJFK,John F Kennedy International Airport,KJFK,JFK,\"New York, Queens\",US,40.6398,-73.7789,13\n" +
		"KT00,Chambers County Airport,KT00,,Anahuac,US,29.7701,-94.6624,21\n"
)

func gz(t *testing.T, s string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := gzip.NewWriter(&b)
	if _, err := w.Write([]byte(s)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// routeServer serves the two data files and counts the responses that carried
// a body.
type routeServer struct {
	*httptest.Server
	routes   []byte
	airports []byte
	modified time.Time
	bodies   int
}

func newRouteServer(t *testing.T) *routeServer {
	t.Helper()
	s := &routeServer{
		routes:   gz(t, testRoutesCSV),
		airports: gz(t, testAirportsCSV),
		modified: time.Date(2026, 9, 20, 18, 47, 40, 0, time.UTC),
	}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := s.routes
		if r.URL.Path == "/airports.csv.gz" {
			body = s.airports
		}
		cw := &countingWriter{ResponseWriter: w}
		http.ServeContent(cw, r, "", s.modified, bytes.NewReader(body))
		if cw.n > 0 {
			s.bodies++
		}
	}))
	t.Cleanup(s.Close)
	return s
}

type countingWriter struct {
	http.ResponseWriter
	n int
}

func (c *countingWriter) Write(p []byte) (int, error) {
	c.n += len(p)
	return c.ResponseWriter.Write(p)
}

func newTestRouteData(t *testing.T, s *routeServer) *routeData {
	t.Helper()
	r := newRouteData(t.TempDir(), log.New(io.Discard, "", 0))
	r.routesURL = s.URL + "/routes.csv.gz"
	r.airportsURL = s.URL + "/airports.csv.gz"
	r.minRoutes, r.minAirports = 1, 1
	return r
}

func TestRouteDataLookup(t *testing.T) {
	r := newTestRouteData(t, newRouteServer(t))
	if err := r.refresh(context.Background()); err != nil {
		t.Fatalf("refresh() = %v", err)
	}
	tests := []struct {
		callsign  string
		codes     string // empty when no route is expected
		locations []string
	}{
		{"ASA184  ", "ANC-CDB-ADK", []string{"Anchorage", "Cold Bay", "Adak Island"}},
		{"cpa844", "HKG-JFK", []string{"Hong Kong", "New York, Queens"}},
		// A stop without an IATA code makes the chain ICAO.
		{"XXX1", "KT00-KJFK", []string{"Anahuac", "New York, Queens"}},
		// One stop is absent from the airports file.
		{"XXX2", "ZZZZ-KJFK", []string{"ZZZZ", "New York, Queens"}},
		{"N123AB", "", nil},
		{"", "", nil},
	}
	for _, tt := range tests {
		got, err := r.Lookup(tt.callsign)
		if err != nil {
			t.Errorf("Lookup(%q) = %v", tt.callsign, err)
			continue
		}
		if tt.codes == "" {
			if got != nil {
				t.Errorf("Lookup(%q) = %+v, want nil", tt.callsign, got)
			}
			continue
		}
		if got == nil {
			t.Errorf("Lookup(%q) = nil, want route %s", tt.callsign, tt.codes)
			continue
		}
		if c := got.Codes(); c != tt.codes {
			t.Errorf("Lookup(%q).Codes() = %q, want %q", tt.callsign, c, tt.codes)
		}
		var locs []string
		for _, a := range got.Airports {
			locs = append(locs, a.Location)
		}
		if len(locs) != len(tt.locations) {
			t.Errorf("Lookup(%q) locations = %q, want %q", tt.callsign, locs, tt.locations)
			continue
		}
		for i := range locs {
			if locs[i] != tt.locations[i] {
				t.Errorf("Lookup(%q) locations = %q, want %q", tt.callsign, locs, tt.locations)
				break
			}
		}
	}
}

func TestRouteDataLookupBeforeDownload(t *testing.T) {
	r := newTestRouteData(t, newRouteServer(t))
	if got, err := r.Lookup("ASA184"); err == nil {
		t.Errorf("Lookup(%q) = %+v, nil; want an error before the first download", "ASA184", got)
	}
}

func TestRouteDataLoadFromDisk(t *testing.T) {
	s := newRouteServer(t)
	r := newTestRouteData(t, s)
	if err := r.refresh(context.Background()); err != nil {
		t.Fatalf("refresh() = %v", err)
	}
	s.Close()

	restarted := newTestRouteData(t, s)
	restarted.dir = r.dir
	restarted.load()
	got, err := restarted.Lookup("ASA184")
	if err != nil || got == nil || got.Codes() != "ANC-CDB-ADK" {
		t.Errorf("Lookup(%q) after load from disk = %+v, %v; want route ANC-CDB-ADK", "ASA184", got, err)
	}
}

func TestRouteDataLoadRemovesBadFile(t *testing.T) {
	r := newTestRouteData(t, newRouteServer(t))
	path := filepath.Join(r.dir, routesFile)
	if err := os.WriteFile(path, []byte("not gzip"), 0o644); err != nil {
		t.Fatal(err)
	}
	r.load()
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Stat(%s) after load = %v, want not exist", routesFile, err)
	}
}

func TestRouteDataRefreshUnchanged(t *testing.T) {
	s := newRouteServer(t)
	r := newTestRouteData(t, s)
	for i := range 2 {
		if err := r.refresh(context.Background()); err != nil {
			t.Fatalf("refresh() #%d = %v", i+1, err)
		}
	}
	if s.bodies != 2 {
		t.Errorf("server sent %d file bodies over two refreshes, want 2 (one per file)", s.bodies)
	}
}

func TestRouteDataRefreshKeepsFilesOnBadDownload(t *testing.T) {
	s := newRouteServer(t)
	r := newTestRouteData(t, s)
	if err := r.refresh(context.Background()); err != nil {
		t.Fatalf("refresh() = %v", err)
	}

	bad := []struct {
		name   string
		routes []byte
	}{
		{"not gzip", []byte("<html>error</html>")},
		{"wrong header", gz(t, "Callsign,AirportCodes\nASA184,KSEA-KPDX\n")},
		{"too few rows", gz(t, "Callsign,Code,Number,AirlineCode,AirportCodes\n")},
	}
	for _, tt := range bad {
		t.Run(tt.name, func(t *testing.T) {
			s.routes = tt.routes
			s.modified = s.modified.Add(24 * time.Hour)
			if err := r.refresh(context.Background()); err == nil {
				t.Error("refresh() = nil, want an error")
			}
			got, err := r.Lookup("ASA184")
			if err != nil || got == nil || got.Codes() != "ANC-CDB-ADK" {
				t.Errorf("Lookup(%q) after failed refresh = %+v, %v; want route ANC-CDB-ADK", "ASA184", got, err)
			}
		})
	}
}

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
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
)

// Route data is the Virtual Radar Server standing data. adsb.lol mirrors it as
// two gzipped CSV files.
// https://github.com/vradarserver/standing-data
const (
	defaultRoutesURL   = "https://vrs-standing-data.adsb.lol/routes.csv.gz"
	defaultAirportsURL = "https://vrs-standing-data.adsb.lol/airports.csv.gz"

	routesFile   = "routes.csv.gz"
	airportsFile = "airports.csv.gz"

	// routeRefreshInterval is how often the mirror is asked for a newer file.
	// The server responds 304 Not Modified for an unchanged file.
	routeRefreshInterval = 6 * time.Hour
	routeRefreshTimeout  = 10 * time.Minute

	// maxRouteFileBytes caps one download. The files are about 4.5 MB and
	// 1 MB.
	maxRouteFileBytes = 64 << 20

	// refresh rejects a file with fewer rows than these, such as a truncated
	// file or an error page. The files have about 620,000 routes and 34,000
	// airports.
	minRouteRows   = 400_000
	minAirportRows = 20_000
)

var (
	routesHeader   = []string{"Callsign", "Code", "Number", "AirlineCode", "AirportCodes"}
	airportsHeader = []string{"Code", "Name", "ICAO", "IATA", "Location", "CountryISO2", "Latitude", "Longitude", "AltitudeFeet"}

	utf8BOM = []byte{0xEF, 0xBB, 0xBF}
)

// Route is a callsign's itinerary: origin first, destination last, with any
// intermediate stops in between.
type Route struct {
	Airports []Airport
}

type Airport struct {
	IATA     string
	ICAO     string
	Location string
}

// Codes returns the itinerary as hyphen-joined IATA codes, or as ICAO codes
// when any stop has no IATA code.
func (r *Route) Codes() string {
	iata := make([]string, len(r.Airports))
	icao := make([]string, len(r.Airports))
	allIATA := true
	for i, a := range r.Airports {
		iata[i], icao[i] = a.IATA, a.ICAO
		if a.IATA == "" {
			allIATA = false
		}
	}
	if allIATA {
		return strings.Join(iata, "-")
	}
	return strings.Join(icao, "-")
}

// routeData resolves routes from the standing data files. It keeps a copy of
// the files in dir and their rows in memory.
type routeData struct {
	dir         string
	routesURL   string
	airportsURL string
	client      *http.Client
	log         *log.Logger
	minRoutes   int
	minAirports int

	mu       sync.RWMutex
	routes   *table // callsign to airport codes
	airports *table // airport code to ICAO, IATA, location
}

func newRouteData(dir string, log *log.Logger) *routeData {
	return &routeData{
		dir:         dir,
		routesURL:   defaultRoutesURL,
		airportsURL: defaultAirportsURL,
		client:      http.DefaultClient,
		log:         log,
		minRoutes:   minRouteRows,
		minAirports: minAirportRows,
	}
}

// dataFile describes one of the two standing data files.
type dataFile struct {
	name    string
	url     string
	header  []string
	minRows int
	// row returns the table key and fields for one CSV record, or an empty
	// key for a record to skip.
	row func(rec []string) (key string, fields []string)
	// dst is where the file's table is stored.
	dst **table
}

func (r *routeData) files() []dataFile {
	return []dataFile{
		{routesFile, r.routesURL, routesHeader, r.minRoutes, routeRow, &r.routes},
		{airportsFile, r.airportsURL, airportsHeader, r.minAirports, airportRow, &r.airports},
	}
}

func routeRow(rec []string) (string, []string) {
	var codes []string
	for _, c := range strings.Split(rec[4], "-") {
		if c = strings.ToUpper(strings.TrimSpace(c)); c != "" {
			codes = append(codes, c)
		}
	}
	if len(codes) < 2 {
		return "", nil
	}
	return strings.ToUpper(strings.TrimSpace(rec[0])), codes
}

func airportRow(rec []string) (string, []string) {
	return strings.ToUpper(strings.TrimSpace(rec[0])), []string{
		strings.ToUpper(strings.TrimSpace(rec[2])),
		strings.ToUpper(strings.TrimSpace(rec[3])),
		strings.TrimSpace(rec[4]),
	}
}

// read builds f's table from the file at path.
func (f dataFile) read(path string) (*table, error) {
	t := &table{}
	_, err := scanCSV(path, f.header, func(rec []string) bool {
		if len(rec) != len(f.header) {
			return true
		}
		if key, fields := f.row(rec); key != "" {
			t.add(key, fields)
		}
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("%s: %w", f.name, err)
	}
	if t.len() < f.minRows {
		return nil, fmt.Errorf("%s: %d rows, want at least %d", f.name, t.len(), f.minRows)
	}
	t.sort()
	return t, nil
}

// Run loads the files already in dir, then refreshes them immediately and
// every routeRefreshInterval until ctx is done.
func (r *routeData) Run(ctx context.Context) {
	r.load()
	t := time.NewTicker(routeRefreshInterval)
	defer t.Stop()
	for {
		rctx, cancel := context.WithTimeout(ctx, routeRefreshTimeout)
		err := r.refresh(rctx)
		cancel()
		if err != nil && ctx.Err() == nil {
			r.log.Println("adsb: route data refresh failed:", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// load reads the files in dir into memory. It removes a file that fails its
// checks, which makes the next refresh download it again.
func (r *routeData) load() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, f := range r.files() {
		path := filepath.Join(r.dir, f.name)
		t, err := f.read(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			r.log.Println("adsb: removing route data file:", err)
			_ = os.Remove(path)
			continue
		}
		*f.dst = t
	}
}

// refresh downloads whichever files changed upstream, checks them, and
// replaces the copies on disk and in memory. If a download or a check fails,
// nothing is replaced.
func (r *routeData) refresh(ctx context.Context) error {
	if err := os.MkdirAll(r.dir, 0o755); err != nil {
		return err
	}
	type update struct {
		dataFile
		*fetchedFile
		table *table
	}
	var updates []update
	defer func() {
		for _, u := range updates {
			_ = os.Remove(u.tmp)
		}
	}()
	for _, f := range r.files() {
		got, err := r.fetch(ctx, f.url, filepath.Join(r.dir, f.name))
		if err != nil {
			return fmt.Errorf("%s: %w", f.name, err)
		}
		if got == nil {
			continue
		}
		updates = append(updates, update{dataFile: f, fetchedFile: got})
		t, err := f.read(got.tmp)
		if err != nil {
			return err
		}
		updates[len(updates)-1].table = t
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range updates {
		// fetch sends the file's mtime as If-Modified-Since.
		if err := os.Chtimes(u.tmp, u.modified, u.modified); err != nil {
			return err
		}
		if err := os.Rename(u.tmp, u.dest); err != nil {
			return err
		}
		*u.dst = u.table
		r.log.Printf("adsb: loaded %s: %d rows, modified %s", u.name, u.table.len(), u.modified.Format(time.DateOnly))
	}
	return nil
}

type fetchedFile struct {
	tmp      string
	dest     string
	modified time.Time
}

// fetch downloads url to a temporary file beside dest. It returns nil when
// the server responds 304 Not Modified.
func (r *routeData) fetch(ctx context.Context, url, dest string) (*fetchedFile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if fi, err := os.Stat(dest); err == nil {
		req.Header.Set("If-Modified-Since", fi.ModTime().UTC().Format(http.TimeFormat))
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotModified {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get %s: %s", url, resp.Status)
	}
	modified, err := http.ParseTime(resp.Header.Get("Last-Modified"))
	if err != nil {
		modified = time.Now()
	}

	tmp, err := os.CreateTemp(filepath.Dir(dest), filepath.Base(dest)+".*.tmp")
	if err != nil {
		return nil, err
	}
	n, err := io.Copy(tmp, io.LimitReader(resp.Body, maxRouteFileBytes+1))
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > maxRouteFileBytes {
		err = fmt.Errorf("get %s: larger than %d bytes", url, maxRouteFileBytes)
	}
	if err != nil {
		_ = os.Remove(tmp.Name())
		return nil, err
	}
	return &fetchedFile{tmp: tmp.Name(), dest: dest, modified: modified}, nil
}

// Lookup returns the route for callsign, or nil when the data does not have
// a route for it. It returns an error until the data is loaded.
func (r *routeData) Lookup(callsign string) (*Route, error) {
	callsign = strings.ToUpper(strings.TrimSpace(callsign))
	if callsign == "" {
		return nil, nil
	}
	r.mu.RLock()
	routes, airports := r.routes, r.airports
	r.mu.RUnlock()
	if routes == nil || airports == nil {
		return nil, errors.New("route data is not loaded")
	}

	codes, ok := routes.lookup(callsign)
	if !ok {
		return nil, nil
	}
	route := &Route{}
	for _, c := range codes {
		// A stop that is missing from the airports file keeps its code as
		// ICAO and Location.
		a := Airport{ICAO: c, Location: c}
		if f, ok := airports.lookup(c); ok {
			a.IATA = f[1]
			if f[0] != "" {
				a.ICAO = f[0]
			}
			if f[2] != "" {
				a.Location = f[2]
			}
		}
		route.Airports = append(route.Airports, a)
	}
	return route, nil
}

// table is a read-only map from a key to a list of fields. Its rows are
// stored in one byte slice as "key\tfield\tfield\n". The 620,000 routes take
// about 14 MB.
type table struct {
	data []byte
	rows []uint32 // offset of each row in data, ordered by key after sort
}

var tableSeparators = strings.NewReplacer("\t", " ", "\n", " ")

func (t *table) add(key string, fields []string) {
	t.rows = append(t.rows, uint32(len(t.data)))
	t.data = append(t.data, tableSeparators.Replace(key)...)
	for _, f := range fields {
		t.data = append(t.data, '\t')
		t.data = append(t.data, tableSeparators.Replace(f)...)
	}
	t.data = append(t.data, '\n')
}

func (t *table) len() int {
	return len(t.rows)
}

// key returns the key of the row at offset off.
func (t *table) key(off uint32) []byte {
	row := t.data[off:]
	return row[:bytes.IndexByte(row, '\t')]
}

// sort orders the rows by key. lookup requires it.
func (t *table) sort() {
	slices.SortFunc(t.rows, func(a, b uint32) int {
		return bytes.Compare(t.key(a), t.key(b))
	})
}

func (t *table) lookup(key string) ([]string, bool) {
	k := []byte(key)
	i, found := slices.BinarySearchFunc(t.rows, k, func(off uint32, k []byte) int {
		return bytes.Compare(t.key(off), k)
	})
	if !found {
		return nil, false
	}
	row := t.data[t.rows[i]:]
	row = row[len(k)+1 : bytes.IndexByte(row, '\n')]
	return strings.Split(string(row), "\t"), true
}

// scanCSV reads the gzipped CSV file at path, checks its header, and calls fn
// with each data row until fn returns false. It returns the number of rows
// passed to fn. The record passed to fn is reused between calls.
func scanCSV(path string, header []string, fn func(rec []string) bool) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return 0, err
	}
	defer func() { _ = gz.Close() }()

	br := bufio.NewReader(gz)
	if b, err := br.Peek(len(utf8BOM)); err == nil && string(b) == string(utf8BOM) {
		_, _ = br.Discard(len(utf8BOM))
	}
	cr := csv.NewReader(br)
	// The caller checks each row's width. A line that does not parse as CSV
	// is skipped.
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	cr.ReuseRecord = true

	head, err := cr.Read()
	if err != nil {
		return 0, fmt.Errorf("read header: %w", err)
	}
	if len(head) != len(header) {
		return 0, fmt.Errorf("header has %d columns, want %d", len(head), len(header))
	}
	for i := range header {
		if strings.TrimSpace(head[i]) != header[i] {
			return 0, fmt.Errorf("header column %d is %q, want %q", i+1, head[i], header[i])
		}
	}

	rows := 0
	for {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			return rows, nil
		}
		if err != nil {
			var pe *csv.ParseError
			if errors.As(err, &pe) {
				continue
			}
			return rows, err
		}
		rows++
		if !fn(rec) {
			return rows, nil
		}
	}
}

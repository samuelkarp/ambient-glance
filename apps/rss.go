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
	"container/list"
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/SlyMarbo/rss/v2"
	"go.sbk.wtf/ambient-glance/display"
	"go.sbk.wtf/ambient-glance/scheduler"
)

type rssApp struct {
	log  *log.Logger
	urls map[string]string

	mu    sync.Mutex
	items *list.List
	seen  map[string]struct{}

	intents chan<- scheduler.Intent
}

type rssItemState struct {
	Feed  string
	Title string
	Date  time.Time
	ID    string
}

func NewRSS(feeds map[string]string, log *log.Logger) *rssApp {
	return &rssApp{
		log:   log,
		urls:  feeds,
		items: list.New(),
		seen:  make(map[string]struct{}),
	}
}

func (r *rssApp) Name() string {
	return "rss"
}

func (r *rssApp) Activate(id string) (scheduler.Activity, error) {
	return &rssActivity{app: r}, nil
}

func (r *rssApp) Stop(id string) error {
	return nil
}

func (r *rssApp) WithIntents(intents chan<- scheduler.Intent) IntentApp {
	r.intents = intents
	return r
}

func (r *rssApp) SignalIntent() error {
	if r.intents == nil {
		return errors.New("no intents set")
	}
	r.intents <- scheduler.Intent{
		Name:     r.Name(),
		Activity: &rssActivity{app: r},
	}
	return nil
}

func (r *rssApp) Run(ctx context.Context) error {
	s := slog.New(&LogLoggerHandler{out: r.log})
	reader, err := rss.NewReader(rss.WithLogger(s))
	if err != nil {
		return err
	}
	feeds := make(map[string]*rss.Feed)

	for name, u := range r.urls {
		feed, err := reader.Fetch(ctx, u)
		if err != nil {
			r.log.Printf("rss %s %v", u, err)
			continue
		}
		feeds[name] = feed
		r.processFeedItems(name, feed)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Minute):
		}

		for name, feed := range feeds {
			if err := reader.Update(ctx, feed); err != nil {
				if err != rss.ErrTooSoon {
					r.log.Printf("rss update error %v", err)
				}
				continue
			}
			r.processFeedItems(name, feed)
		}
	}
}

func (r *rssApp) processFeedItems(name string, feed *rss.Feed) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)
	discard, dupe, added := 0, 0, 0

	for _, item := range feed.Items {
		if item.Date.Before(cutoff) {
			discard++
			continue
		}
		id := item.ID
		if id == "" && len(item.Links) > 0 {
			id = item.Links[0].Href
		}
		if id == "" {
			id = item.Title
		}

		if _, ok := r.seen[id]; ok {
			dupe++
			continue
		}
		added++
		r.seen[id] = struct{}{}
		r.items.PushFront(&rssItemState{
			Feed:  name,
			Title: item.Title,
			Date:  item.Date,
			ID:    id,
		})
	}
	r.log.Printf("rss %s: added %d, dupes %d, discarded %d", name, added, dupe, discard)
}

type rssActivity struct {
	app *rssApp
}

func (a *rssActivity) Run(ctx context.Context, d display.Display) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		a.app.mu.Lock()
		var selectedItem *rssItemState
		expired := 0
		for {
			el := a.app.items.Front()
			if el == nil {
				break
			}
			item := a.app.items.Remove(el).(*rssItemState)
			if item.Date.Before(time.Now().Add(-24 * time.Hour)) {
				expired++
				delete(a.app.seen, item.ID)
				continue
			}
			selectedItem = item
			break
		}
		if expired > 0 {
			a.app.log.Printf("rss: expired %d items", expired)
		}
		a.app.mu.Unlock()

		if selectedItem == nil {
			a.app.log.Println("rss: no valid items to display")
			return nil
		}

		if err := d.Reset(); err != nil {
			return err
		}

		a.app.log.Printf("rss: displaying %s: %s", selectedItem.Feed, selectedItem.Title)

		lines := wordwrap(fmt.Sprintf("%s: %s", selectedItem.Feed, selectedItem.Title))
		for i, line := range lines {
			if i%2 == 0 {
				if err := d.Clear(); err != nil {
					return err
				}
			} else {
				if err := d.ClearLine(); err != nil {
					return err
				}
			}
			if _, err := d.Write([]byte(line)); err != nil {
				return err
			}
			if i%2 == 0 {
				if err := d.MoveCursor(display.CursorBottomLeft); err != nil {
					return err
				}
			} else {
				if err := d.MoveCursor(display.CursorTopLeft); err != nil {
					return err
				}
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(3 * time.Second):
				}
			}
		}
		if len(lines)%2 != 0 {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(3 * time.Second):
			}
		}

		a.app.mu.Lock()
		a.app.items.PushBack(selectedItem)
		a.app.mu.Unlock()
	}
}

type LogLoggerHandler struct {
	out   *log.Logger
	attrs []slog.Attr
	group string
}

func (l LogLoggerHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return true
}

func (l LogLoggerHandler) Handle(ctx context.Context, r slog.Record) error {
	attrs := make(map[string]string)
	for _, a := range l.attrs {
		attrs[a.Key] = a.Value.String()
	}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.String()
		return true
	})
	builder := strings.Builder{}
	if l.group != "" {
		builder.WriteString("[" + l.group + "] ")
	}
	for k, v := range attrs {
		builder.WriteString(fmt.Sprintf("%s=%q ", k, v))
	}
	builder.WriteString(r.Message)
	l.out.Println(builder.String())
	return nil
}

func (l LogLoggerHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &LogLoggerHandler{out: l.out, attrs: append(l.attrs, attrs...), group: l.group}
}

func (l LogLoggerHandler) WithGroup(name string) slog.Handler {
	return &LogLoggerHandler{out: l.out, attrs: l.attrs, group: name}
}
